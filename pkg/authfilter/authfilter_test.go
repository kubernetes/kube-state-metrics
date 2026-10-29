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

package authfilter

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/fake"
	authenticationv1client "k8s.io/client-go/kubernetes/typed/authentication/v1"
	authorizationv1client "k8s.io/client-go/kubernetes/typed/authorization/v1"
	k8stesting "k8s.io/client-go/testing"
)

const validToken = "valid-token"

// fakeAPIServer answers TokenReviews and SubjectAccessReviews: validToken
// belongs to user "scraper", and "scraper" may only get /metrics.
type fakeAPIServer struct {
	tokenReviews, subjectAccessReviews int
	lastSAR                            authorizationv1.SubjectAccessReviewSpec
	tokenReviewErrs                    []error
	tokenReviewStatusError             string
}

func (s *fakeAPIServer) filter() *Filter {
	client := fake.NewClientset()
	client.PrependReactor("create", "tokenreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		s.tokenReviews++
		if len(s.tokenReviewErrs) > 0 {
			err := s.tokenReviewErrs[0]
			s.tokenReviewErrs = s.tokenReviewErrs[1:]
			return true, nil, err
		}
		review := action.(k8stesting.CreateAction).GetObject().(*authenticationv1.TokenReview).DeepCopy()
		if review.Spec.Token == validToken {
			review.Status = authenticationv1.TokenReviewStatus{
				Authenticated: true,
				User: authenticationv1.UserInfo{
					Username: "scraper",
					UID:      "1234",
					Groups:   []string{"monitoring"},
					Extra:    map[string]authenticationv1.ExtraValue{"scope": {"metrics"}},
				},
			}
		} else {
			review.Status.Error = s.tokenReviewStatusError
		}
		return true, review, nil
	})
	client.PrependReactor("create", "subjectaccessreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		s.subjectAccessReviews++
		review := action.(k8stesting.CreateAction).GetObject().(*authorizationv1.SubjectAccessReview).DeepCopy()
		s.lastSAR = review.Spec
		nra := review.Spec.NonResourceAttributes
		review.Status.Allowed = review.Spec.User == "scraper" && nra != nil && nra.Path == "/metrics" && nra.Verb == "get"
		return true, review, nil
	})

	f := newFilter(client.AuthenticationV1().TokenReviews(), client.AuthorizationV1().SubjectAccessReviews())
	f.backoff = wait.Backoff{Duration: time.Millisecond, Factor: 1, Steps: 5}
	return f
}

func serve(t *testing.T, h http.Handler, method, path, authorization string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestWrap(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("Authorization header was passed on to the wrapped handler")
		}
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name          string
		method, path  string
		authorization string
		want          int
	}{
		{name: "no header", method: http.MethodGet, path: "/metrics", want: http.StatusUnauthorized},
		{name: "basic auth", method: http.MethodGet, path: "/metrics", authorization: "Basic dXNlcjpwdw==", want: http.StatusUnauthorized},
		{name: "empty bearer token", method: http.MethodGet, path: "/metrics", authorization: "Bearer ", want: http.StatusUnauthorized},
		{name: "invalid token", method: http.MethodGet, path: "/metrics", authorization: "Bearer nope", want: http.StatusUnauthorized},
		{name: "valid token", method: http.MethodGet, path: "/metrics", authorization: "Bearer " + validToken, want: http.StatusOK},
		{name: "lowercase scheme and trailing newline", method: http.MethodGet, path: "/metrics", authorization: "bearer " + validToken + "\n", want: http.StatusOK},
		{name: "path not allowed", method: http.MethodGet, path: "/debug/pprof/", authorization: "Bearer " + validToken, want: http.StatusForbidden},
		{name: "verb not allowed", method: http.MethodPost, path: "/metrics", authorization: "Bearer " + validToken, want: http.StatusForbidden},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &fakeAPIServer{}
			if got := serve(t, s.filter().Wrap(ok), tc.method, tc.path, tc.authorization).Code; got != tc.want {
				t.Errorf("got status %d, want %d", got, tc.want)
			}
		})
	}
}

func TestSubjectAccessReviewSpec(t *testing.T) {
	s := &fakeAPIServer{}
	serve(t, s.filter().Wrap(http.NotFoundHandler()), http.MethodGet, "/metrics", "Bearer "+validToken)

	want := authorizationv1.SubjectAccessReviewSpec{
		User:                  "scraper",
		UID:                   "1234",
		Groups:                []string{"monitoring", "system:authenticated"},
		Extra:                 map[string]authorizationv1.ExtraValue{"scope": {"metrics"}},
		NonResourceAttributes: &authorizationv1.NonResourceAttributes{Path: "/metrics", Verb: "get"},
	}
	got := s.lastSAR
	if got.User != want.User || got.UID != want.UID || !slices.Equal(got.Groups, want.Groups) ||
		!slices.Equal(got.Extra["scope"], want.Extra["scope"]) || *got.NonResourceAttributes != *want.NonResourceAttributes {
		t.Errorf("got SubjectAccessReview spec %+v, want %+v", got, want)
	}
}

func TestCaching(t *testing.T) {
	s := &fakeAPIServer{}
	h := s.filter().Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	for range 3 {
		serve(t, h, http.MethodGet, "/metrics", "Bearer "+validToken)
		serve(t, h, http.MethodGet, "/debug/pprof/", "Bearer "+validToken)
		serve(t, h, http.MethodGet, "/metrics", "Bearer nope")
	}
	if s.tokenReviews != 2 {
		t.Errorf("got %d TokenReviews, want 2 (one per token)", s.tokenReviews)
	}
	if s.subjectAccessReviews != 2 {
		t.Errorf("got %d SubjectAccessReviews, want 2 (one per path)", s.subjectAccessReviews)
	}
}

func TestErrors(t *testing.T) {
	h := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})

	t.Run("transient errors are retried", func(t *testing.T) {
		s := &fakeAPIServer{tokenReviewErrs: []error{
			apierrors.NewInternalError(errors.New("boom")),
			apierrors.NewTooManyRequests("slow down", 0),
		}}
		if got := serve(t, s.filter().Wrap(h), http.MethodGet, "/metrics", "Bearer "+validToken).Code; got != http.StatusOK {
			t.Errorf("got status %d, want %d", got, http.StatusOK)
		}
		if s.tokenReviews != 3 {
			t.Errorf("got %d TokenReviews, want 3", s.tokenReviews)
		}
	})

	t.Run("other errors are not retried or cached", func(t *testing.T) {
		s := &fakeAPIServer{tokenReviewErrs: []error{apierrors.NewForbidden(authenticationv1.Resource("tokenreviews"), "", errors.New("no"))}}
		f := s.filter().Wrap(h)
		if got := serve(t, f, http.MethodGet, "/metrics", "Bearer "+validToken).Code; got != http.StatusInternalServerError {
			t.Errorf("got status %d, want %d", got, http.StatusInternalServerError)
		}
		if got := serve(t, f, http.MethodGet, "/metrics", "Bearer "+validToken).Code; got != http.StatusOK {
			t.Errorf("got status %d after the error cleared, want %d", got, http.StatusOK)
		}
		if s.tokenReviews != 2 {
			t.Errorf("got %d TokenReviews, want 2", s.tokenReviews)
		}
	})

	t.Run("TokenReview status error", func(t *testing.T) {
		s := &fakeAPIServer{tokenReviewStatusError: "token authenticator unavailable"}
		if got := serve(t, s.filter().Wrap(h), http.MethodGet, "/metrics", "Bearer nope").Code; got != http.StatusInternalServerError {
			t.Errorf("got status %d, want %d", got, http.StatusInternalServerError)
		}
	})
}

// deadlineTokenReviews and deadlineSubjectAccessReviews record whether Create
// was called with a deadline.
type deadlineTokenReviews struct {
	authenticationv1client.TokenReviewInterface
	hasDeadline *bool
}

func (d deadlineTokenReviews) Create(ctx context.Context, review *authenticationv1.TokenReview, opts metav1.CreateOptions) (*authenticationv1.TokenReview, error) {
	_, *d.hasDeadline = ctx.Deadline()
	return d.TokenReviewInterface.Create(ctx, review, opts)
}

type deadlineSubjectAccessReviews struct {
	authorizationv1client.SubjectAccessReviewInterface
	hasDeadline *bool
}

func (d deadlineSubjectAccessReviews) Create(ctx context.Context, review *authorizationv1.SubjectAccessReview, opts metav1.CreateOptions) (*authorizationv1.SubjectAccessReview, error) {
	_, *d.hasDeadline = ctx.Deadline()
	return d.SubjectAccessReviewInterface.Create(ctx, review, opts)
}

// A request context has no deadline of its own, so both reviews must set one
// or a hung API server blocks the request until the client goes away.
func TestReviewTimeout(t *testing.T) {
	s := &fakeAPIServer{}
	f := s.filter()
	var tokenReviewDeadline, subjectAccessReviewDeadline bool
	f.tokenReviews = deadlineTokenReviews{f.tokenReviews, &tokenReviewDeadline}
	f.subjectAccessReviews = deadlineSubjectAccessReviews{f.subjectAccessReviews, &subjectAccessReviewDeadline}

	if got := serve(t, f.Wrap(http.NotFoundHandler()), http.MethodGet, "/metrics", "Bearer "+validToken).Code; got != http.StatusNotFound {
		t.Fatalf("got status %d, want %d", got, http.StatusNotFound)
	}
	if !tokenReviewDeadline {
		t.Error("TokenReview was created without a deadline")
	}
	if !subjectAccessReviewDeadline {
		t.Error("SubjectAccessReview was created without a deadline")
	}
}
