// Copyright Contributors to Agones a Series of LF Projects, LLC.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apiserver

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"

	"agones.dev/agones/pkg/util/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// extensionAPIServerAuthenticationCM is the ConfigMap that the kube-apiserver
	// populates with the RequestHeader CA bundle and allowed CNs.
	extensionAPIServerAuthenticationCM = "extension-apiserver-authentication"

	// kubeSystemNamespace is the namespace where the ConfigMap lives.
	kubeSystemNamespace = "kube-system"

	// defaultUsernameHeader is the default header used by the kube-apiserver
	// aggregator to pass the authenticated username to extension apiservers.
	defaultUsernameHeader = "X-Remote-User"

	// defaultGroupHeader is the default header used by the kube-apiserver
	// aggregator to pass the authenticated groups to extension apiservers.
	defaultGroupHeader = "X-Remote-Group"
)

// RequestHeaderConfig holds the configuration loaded from the
// extension-apiserver-authentication ConfigMap.
type RequestHeaderConfig struct {
	// ClientCAPool is the pool of CA certificates used to verify
	// the client certificate presented by the kube-apiserver aggregator.
	ClientCAPool *x509.CertPool

	// AllowedNames is the list of allowed Common Names for the proxy
	// client certificate. If empty, any CN signed by the CA is accepted.
	AllowedNames []string

	// UsernameHeaders is the list of header names to inspect for the username.
	UsernameHeaders []string

	// GroupHeaders is the list of header names to inspect for groups.
	GroupHeaders []string
}

var authLogger = runtime.NewLoggerWithSource("apiserver-auth")

// LoadRequestHeaderConfig reads the extension-apiserver-authentication ConfigMap
// from kube-system and returns the parsed RequestHeaderConfig.
// This is what aggregated API servers use to verify that the kube-apiserver
// aggregator is the one making the request.
func LoadRequestHeaderConfig(ctx context.Context, kubeClient kubernetes.Interface) (*RequestHeaderConfig, error) {
	cm, err := kubeClient.CoreV1().ConfigMaps(kubeSystemNamespace).Get(
		ctx, extensionAPIServerAuthenticationCM, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get %s/%s ConfigMap", kubeSystemNamespace, extensionAPIServerAuthenticationCM)
	}

	config := &RequestHeaderConfig{
		UsernameHeaders: []string{defaultUsernameHeader},
		GroupHeaders:    []string{defaultGroupHeader},
	}

	// Parse the requestheader-client-ca-file
	caPEM, ok := cm.Data["requestheader-client-ca-file"]
	if !ok || caPEM == "" {
		return nil, fmt.Errorf("ConfigMap %s/%s does not contain requestheader-client-ca-file",
			kubeSystemNamespace, extensionAPIServerAuthenticationCM)
	}

	pool := x509.NewCertPool()
	rest := []byte(caPEM)
	count := 0
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			authLogger.WithError(err).Warn("skipping unparseable certificate in requestheader CA bundle")
			continue
		}
		pool.AddCert(cert)
		count++
	}

	if count == 0 {
		return nil, fmt.Errorf("no valid certificates found in requestheader-client-ca-file")
	}

	config.ClientCAPool = pool
	authLogger.WithField("certCount", count).Info("Loaded requestheader client CA certificates")

	// Parse requestheader-allowed-names (JSON-encoded string array, optional)
	if allowedNamesJSON, ok := cm.Data["requestheader-allowed-names"]; ok && allowedNamesJSON != "" {
		config.AllowedNames = parseJSONStringArray(allowedNamesJSON)
		authLogger.WithField("allowedNames", config.AllowedNames).Info("Loaded requestheader allowed names")
	}

	// Parse username/group headers if present
	if uh, ok := cm.Data["requestheader-username-headers"]; ok && uh != "" {
		if parsed := parseJSONStringArray(uh); len(parsed) > 0 {
			config.UsernameHeaders = parsed
		}
	}
	if gh, ok := cm.Data["requestheader-group-headers"]; ok && gh != "" {
		if parsed := parseJSONStringArray(gh); len(parsed) > 0 {
			config.GroupHeaders = parsed
		}
	}

	return config, nil
}

// parseJSONStringArray attempts to parse a JSON-encoded string array.
// Falls back to treating the whole string as a single-element array.
func parseJSONStringArray(s string) []string {
	// The ConfigMap values are JSON-encoded arrays like: ["front-proxy-client"]
	var result []string
	// Trim whitespace
	s = trimBrackets(s)
	if s == "" {
		return nil
	}
	// Simple split on comma, remove quotes
	for _, part := range splitAndTrim(s) {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// trimBrackets removes surrounding [ ] from a string
func trimBrackets(s string) string {
	if len(s) >= 2 && s[0] == '[' && s[len(s)-1] == ']' {
		return s[1 : len(s)-1]
	}
	return s
}

// splitAndTrim splits on comma and trims quotes and whitespace
func splitAndTrim(s string) []string {
	var result []string
	for _, part := range splitComma(s) {
		part = trimQuotes(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func splitComma(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func trimQuotes(s string) string {
	// Trim whitespace first
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	// Trim surrounding quotes
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// AuthenticateRequest verifies that the request came from the kube-apiserver
// aggregator by checking the TLS peer certificate against the RequestHeader CA.
// Returns the authenticated username and groups, or an error.
func (c *RequestHeaderConfig) AuthenticateRequest(r *http.Request) (username string, groups []string, err error) {
	// 1. Verify the peer certificate was presented and signed by RequestHeader CA
	if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
		return "", nil, fmt.Errorf("no client certificate presented")
	}

	peerCert := r.TLS.PeerCertificates[0]

	// Verify the cert chain against the RequestHeader CA pool
	opts := x509.VerifyOptions{
		Roots:         c.ClientCAPool,
		Intermediates: x509.NewCertPool(),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	for _, cert := range r.TLS.PeerCertificates[1:] {
		opts.Intermediates.AddCert(cert)
	}
	if _, err := peerCert.Verify(opts); err != nil {
		return "", nil, errors.Wrap(err, "client certificate verification failed against requestheader CA")
	}

	// 2. Check the CN against allowedNames (if configured)
	if len(c.AllowedNames) > 0 {
		cnAllowed := false
		for _, allowed := range c.AllowedNames {
			if peerCert.Subject.CommonName == allowed {
				cnAllowed = true
				break
			}
		}
		if !cnAllowed {
			return "", nil, fmt.Errorf("client certificate CN %q is not in requestheader-allowed-names %v",
				peerCert.Subject.CommonName, c.AllowedNames)
		}
	}

	// 3. Extract username from headers (set by the kube-apiserver aggregator)
	for _, header := range c.UsernameHeaders {
		if val := r.Header.Get(header); val != "" {
			username = val
			break
		}
	}
	if username == "" {
		return "", nil, fmt.Errorf("no username found in request headers %v", c.UsernameHeaders)
	}

	// 4. Extract groups from headers
	for _, header := range c.GroupHeaders {
		groups = append(groups, r.Header.Values(header)...)
	}

	return username, groups, nil
}

// AuthorizeAllocation performs a SubjectAccessReview to check if the
// authenticated user has "create" permission on gameserverallocations
// in the given namespace.
func AuthorizeAllocation(ctx context.Context, kubeClient kubernetes.Interface,
	username string, groups []string, namespace string) error {

	sar := &authorizationv1.SubjectAccessReview{
		Spec: authorizationv1.SubjectAccessReviewSpec{
			User:   username,
			Groups: groups,
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace: namespace,
				Verb:      "create",
				Group:     "allocation.agones.dev",
				Resource:  "gameserverallocations",
			},
		},
	}

	result, err := kubeClient.AuthorizationV1().SubjectAccessReviews().Create(ctx, sar, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "SubjectAccessReview failed")
	}

	if !result.Status.Allowed {
		reason := result.Status.Reason
		if reason == "" {
			reason = "no reason given"
		}
		return fmt.Errorf("user %q is not authorized to create gameserverallocations in namespace %q: %s",
			username, namespace, reason)
	}

	authLogger.WithFields(logrus.Fields{
		"user":      username,
		"namespace": namespace,
	}).Debug("Authorization check passed for allocation request")

	return nil
}
