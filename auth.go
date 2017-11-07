/*
Copyright 2017 The Kubernetes Authors.

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

package main

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/golang/glog"

	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/authenticatorfactory"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/authorization/authorizerfactory"
	clientset "k8s.io/client-go/kubernetes"
	authenticationclient "k8s.io/client-go/kubernetes/typed/authentication/v1beta1"
	authorizationclient "k8s.io/client-go/kubernetes/typed/authorization/v1beta1"
)

type AnonConfig struct {
	Enabled bool
}

type X509Config struct {
	ClientCAFile string
}

type WebhookConfig struct {
	Enabled bool
}

type AuthnConfig struct {
	Anonymous *AnonConfig
	Webhook   *WebhookConfig
	X509      *X509Config
}

type AuthzConfig struct {
	Mode string
}

type AuthConfig struct {
	Authentication *AuthnConfig
	Authorization  *AuthzConfig
}

// kubeStateMetricsAuth implements AuthInterface
type kubeStateMetricsAuth struct {
	// authenticator identifies the user for requests to kube-state-metrics
	authenticator.Request
	// authorizerAttributeGetter builds authorization.Attributes for a request to kube-state-metrics
	authorizer.RequestAttributesGetter
	// authorizer determines whether a given authorization.Attributes is allowed
	authorizer.Authorizer
}

func newKubeStateMetricsAuth(authenticator authenticator.Request, authorizer authorizer.Authorizer) AuthInterface {
	return &kubeStateMetricsAuth{authenticator, newKubeStateMetricsAuthorizerAttributesGetter(), authorizer}
}

// BuildAuth creates an authenticator, an authorizer, and a matching authorizer attributes getter compatible with the kube-state-metrics
func BuildAuth(client clientset.Interface, config AuthConfig) (AuthInterface, error) {
	// Get clients, if provided
	var (
		tokenClient authenticationclient.TokenReviewInterface
		sarClient   authorizationclient.SubjectAccessReviewInterface
	)
	if client != nil && !reflect.ValueOf(client).IsNil() {
		tokenClient = client.AuthenticationV1beta1().TokenReviews()
		sarClient = client.AuthorizationV1beta1().SubjectAccessReviews()
	}

	authenticator, err := buildAuthn(tokenClient, config.Authentication)
	if err != nil {
		return nil, err
	}

	authorizer, err := buildAuthz(sarClient, config.Authorization)
	if err != nil {
		return nil, err
	}

	return newKubeStateMetricsAuth(authenticator, authorizer), nil
}

// buildAuthn creates an authenticator compatible with the kubelet's needs
func buildAuthn(client authenticationclient.TokenReviewInterface, authn *AuthnConfig) (authenticator.Request, error) {
	authenticatorConfig := authenticatorfactory.DelegatingAuthenticatorConfig{
		Anonymous:    authn.Anonymous.Enabled,
		CacheTTL:     2 * time.Minute,
		ClientCAFile: authn.X509.ClientCAFile,
	}

	if authn.Webhook.Enabled {
		if client == nil {
			return nil, errors.New("no client provided, cannot use webhook authentication")
		}
		authenticatorConfig.TokenAccessReviewClient = client
	}

	authenticator, _, err := authenticatorConfig.New()
	return authenticator, err
}

const (
	authorizationModeAlwaysAllow = "AlwaysAllow"
	authorizationModeWebhook     = "Webhook"
)

// buildAuthz creates an authorizer compatible with the kubelet's needs
func buildAuthz(client authorizationclient.SubjectAccessReviewInterface, authz *AuthzConfig) (authorizer.Authorizer, error) {
	switch authz.Mode {
	case authorizationModeAlwaysAllow:
		return authorizerfactory.NewAlwaysAllowAuthorizer(), nil

	case authorizationModeWebhook:
		if client == nil {
			return nil, errors.New("no client provided, cannot use webhook authorization")
		}
		authorizerConfig := authorizerfactory.DelegatingAuthorizerConfig{
			SubjectAccessReviewClient: client,
			AllowCacheTTL:             5 * time.Minute,
			DenyCacheTTL:              30 * time.Second,
		}
		return authorizerConfig.New()

	case "":
		return nil, fmt.Errorf("No authorization mode specified")

	default:
		return nil, fmt.Errorf("Unknown authorization mode %s", authz.Mode)

	}
}

func newKubeStateMetricsAuthorizerAttributesGetter() authorizer.RequestAttributesGetter {
	return ksmAuthorizerAttributesGetter{}
}

type ksmAuthorizerAttributesGetter struct{}

// GetRequestAttributes populates authorizer attributes for the requests to kube-state-metrics.
func (n ksmAuthorizerAttributesGetter) GetRequestAttributes(u user.Info, r *http.Request) authorizer.Attributes {
	apiVerb := ""
	switch r.Method {
	case "POST":
		apiVerb = "create"
	case "GET":
		apiVerb = "get"
	case "PUT":
		apiVerb = "update"
	case "PATCH":
		apiVerb = "patch"
	case "DELETE":
		apiVerb = "delete"
	}

	requestPath := r.URL.Path

	// Default attributes mirror the API attributes that would allow this access to kube-state-metrics
	attrs := authorizer.AttributesRecord{
		User:            u,
		Verb:            apiVerb,
		Namespace:       "",
		APIGroup:        "",
		APIVersion:      "",
		Resource:        "",
		Subresource:     "",
		Name:            "",
		ResourceRequest: false,
		Path:            requestPath,
	}

	glog.V(5).Infof("kube-state-metrics request attributes: attrs=%#v", attrs)

	return attrs
}

func AuthRequest(auth AuthInterface, w http.ResponseWriter, req *http.Request) bool {
	// Authenticate
	u, ok, err := auth.AuthenticateRequest(req)
	if err != nil {
		glog.Errorf("Unable to authenticate the request due to an error: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false
	}
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false
	}

	// Get authorization attributes
	attrs := auth.GetRequestAttributes(u, req)

	// Authorize
	authorized, _, err := auth.Authorize(attrs)
	if err != nil {
		msg := fmt.Sprintf("Authorization error (user=%s, verb=%s, resource=%s, subresource=%s)", u.GetName(), attrs.GetVerb(), attrs.GetResource(), attrs.GetSubresource())
		glog.Errorf(msg, err)
		http.Error(w, msg, http.StatusInternalServerError)
		return false
	}
	if !authorized {
		msg := fmt.Sprintf("Forbidden (user=%s, verb=%s, resource=%s, subresource=%s)", u.GetName(), attrs.GetVerb(), attrs.GetResource(), attrs.GetSubresource())
		glog.V(2).Info(msg)
		http.Error(w, msg, http.StatusForbidden)
		return false
	}

	return true
}
