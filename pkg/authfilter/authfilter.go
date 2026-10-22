/*
Copyright 2026 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package authfilter protects HTTP handlers by delegating authentication
// (TokenReview) and authorization (SubjectAccessReview) to the Kubernetes API
// server.
package authfilter

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/cache"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/wait"
	authenticationv1client "k8s.io/client-go/kubernetes/typed/authentication/v1"
	authorizationv1client "k8s.io/client-go/kubernetes/typed/authorization/v1"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

// These match the settings controller-runtime's metrics filter used for the
// delegating authenticator and authorizer of k8s.io/apiserver.
const (
	tokenReviewTimeout  = 10 * time.Second
	tokenCacheTTL       = 1 * time.Minute
	allowCacheTTL       = 5 * time.Minute
	denyCacheTTL        = 30 * time.Second
	cacheSize           = 8192
	allAuthenticatedGrp = "system:authenticated"
)

// retryBackoff is k8s.io/apiserver's DefaultAuthWebhookRetryBackoff.
var retryBackoff = wait.Backoff{
	Duration: 500 * time.Millisecond,
	Factor:   1.5,
	Jitter:   0.2,
	Steps:    5,
}

// Filter authenticates requests with a bearer token through a TokenReview
// and authorizes them through a SubjectAccessReview for the request's path
// and verb, so scrapers need a ClusterRole that allows "get" on the
// nonResourceURLs they read, e.g. /metrics.
type Filter struct {
	tokenReviews          authenticationv1client.TokenReviewInterface
	subjectAccessReviews  authorizationv1client.SubjectAccessReviewInterface
	backoff               wait.Backoff
	tokenCache, authCache *cache.LRUExpireCache
}

// New returns a Filter that talks to the API server described by config.
func New(config *rest.Config, httpClient *http.Client) (*Filter, error) {
	authn, err := authenticationv1client.NewForConfigAndClient(config, httpClient)
	if err != nil {
		return nil, err
	}
	authz, err := authorizationv1client.NewForConfigAndClient(config, httpClient)
	if err != nil {
		return nil, err
	}
	return newFilter(authn.TokenReviews(), authz.SubjectAccessReviews()), nil
}

func newFilter(tokenReviews authenticationv1client.TokenReviewInterface, subjectAccessReviews authorizationv1client.SubjectAccessReviewInterface) *Filter {
	return &Filter{
		tokenReviews:         tokenReviews,
		subjectAccessReviews: subjectAccessReviews,
		backoff:              retryBackoff,
		tokenCache:           cache.NewLRUExpireCache(cacheSize),
		authCache:            cache.NewLRUExpireCache(cacheSize),
	}
}

// Wrap returns a handler that only passes authenticated and authorized
// requests on to next.
func (f *Filter) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		token, ok := bearerToken(req)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		userInfo, ok, err := f.authenticate(req.Context(), token)
		if err != nil {
			klog.ErrorS(err, "Authentication failed")
			http.Error(w, "Authentication failed", http.StatusInternalServerError)
			return
		}
		if !ok {
			klog.V(4).InfoS("Authentication failed")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		// Do not hand the credential on to the wrapped handler.
		req.Header.Del("Authorization")

		spec := authorizationv1.SubjectAccessReviewSpec{
			User:   userInfo.Username,
			UID:    userInfo.UID,
			Groups: userInfo.Groups,
			Extra:  convertExtra(userInfo.Extra),
			NonResourceAttributes: &authorizationv1.NonResourceAttributes{
				Path: req.URL.Path,
				Verb: strings.ToLower(req.Method),
			},
		}
		allowed, reason, err := f.authorize(req.Context(), spec)
		if err != nil {
			msg := fmt.Sprintf("Authorization for user %s failed", userInfo.Username)
			klog.ErrorS(err, msg)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}
		if !allowed {
			msg := fmt.Sprintf("Authorization denied for user %s", userInfo.Username)
			klog.V(4).InfoS(msg, "reason", reason)
			http.Error(w, msg, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, req)
	})
}

// bearerToken extracts the token from an "Authorization: Bearer <token>"
// header.
func bearerToken(req *http.Request) (string, bool) {
	scheme, token, ok := strings.Cut(strings.TrimSpace(req.Header.Get("Authorization")), " ")
	if !ok || !strings.EqualFold(scheme, "bearer") {
		return "", false
	}
	token, _, _ = strings.Cut(token, " ")
	return token, token != ""
}

type tokenResult struct {
	user          authenticationv1.UserInfo
	authenticated bool
}

// authenticate resolves token to a user through a TokenReview. Results are
// cached for tokenCacheTTL; errors are not.
func (f *Filter) authenticate(ctx context.Context, token string) (authenticationv1.UserInfo, bool, error) {
	key := sha256.Sum256([]byte(token))
	if v, ok := f.tokenCache.Get(key); ok {
		r := v.(tokenResult)
		return r.user, r.authenticated, nil
	}

	ctx, cancel := context.WithTimeout(ctx, tokenReviewTimeout)
	defer cancel()
	review := &authenticationv1.TokenReview{Spec: authenticationv1.TokenReviewSpec{Token: token}}
	var result *authenticationv1.TokenReview
	if err := f.withRetry(ctx, func() (err error) {
		result, err = f.tokenReviews.Create(ctx, review, metav1.CreateOptions{})
		return err
	}); err != nil {
		return authenticationv1.UserInfo{}, false, err
	}
	if !result.Status.Authenticated && result.Status.Error != "" {
		return authenticationv1.UserInfo{}, false, errors.New(result.Status.Error)
	}

	r := tokenResult{user: result.Status.User, authenticated: result.Status.Authenticated}
	if r.authenticated && !slices.Contains(r.user.Groups, allAuthenticatedGrp) {
		r.user.Groups = append(slices.Clone(r.user.Groups), allAuthenticatedGrp)
	}
	f.tokenCache.Add(key, r, tokenCacheTTL)
	return r.user, r.authenticated, nil
}

// authorize checks spec through a SubjectAccessReview. Allowed results are
// cached for allowCacheTTL and all others for denyCacheTTL; errors are not.
func (f *Filter) authorize(ctx context.Context, spec authorizationv1.SubjectAccessReviewSpec) (bool, string, error) {
	b, err := json.Marshal(spec)
	if err != nil {
		return false, "", err
	}
	key := string(b)

	var status authorizationv1.SubjectAccessReviewStatus
	if v, ok := f.authCache.Get(key); ok {
		status = v.(authorizationv1.SubjectAccessReviewStatus)
	} else {
		review := &authorizationv1.SubjectAccessReview{Spec: spec}
		var result *authorizationv1.SubjectAccessReview
		if err := f.withRetry(ctx, func() (err error) {
			result, err = f.subjectAccessReviews.Create(ctx, review, metav1.CreateOptions{})
			return err
		}); err != nil {
			return false, "", err
		}
		status = result.Status
		ttl := denyCacheTTL
		if status.Allowed {
			ttl = allowCacheTTL
		}
		f.authCache.Add(key, status, ttl)
	}

	if status.Allowed && status.Denied {
		return false, status.Reason, errors.New("subject access review returned both allow and deny response")
	}
	return status.Allowed, status.Reason, nil
}

// withRetry calls fn until it succeeds, fails with an error that is not
// transient, or the backoff is exhausted. It mirrors k8s.io/apiserver's
// webhook.WithExponentialBackoff with webhook.DefaultShouldRetry.
func (f *Filter) withRetry(ctx context.Context, fn func() error) error {
	var lastErr error
	err := wait.ExponentialBackoffWithContext(ctx, f.backoff, func(context.Context) (bool, error) {
		lastErr = fn()
		if shouldRetry(lastErr) {
			return false, nil
		}
		return lastErr == nil, lastErr
	})
	switch {
	case lastErr != nil:
		return lastErr
	case err != nil:
		return fmt.Errorf("API server call failed: %w", err)
	default:
		return nil
	}
}

func shouldRetry(err error) bool {
	if err == nil {
		return false
	}
	if utilnet.IsConnectionReset(err) || utilnet.IsHTTP2ConnectionLost(err) || apierrors.IsInternalError(err) || apierrors.IsTimeout(err) || apierrors.IsTooManyRequests(err) {
		return true
	}
	_, retry := apierrors.SuggestsClientDelay(err)
	return retry
}

func convertExtra(extra map[string]authenticationv1.ExtraValue) map[string]authorizationv1.ExtraValue {
	if extra == nil {
		return nil
	}
	out := make(map[string]authorizationv1.ExtraValue, len(extra))
	for k, v := range extra {
		out[k] = authorizationv1.ExtraValue(v)
	}
	return out
}
