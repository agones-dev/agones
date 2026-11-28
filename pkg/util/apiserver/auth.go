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
	"encoding/json"
	"net/http"
	"slices"
	"sync/atomic"

	"agones.dev/agones/pkg/util/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	requestx509 "k8s.io/apiserver/pkg/authentication/request/x509"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	// requestHeaderAuthConfigMap is the ConfigMap published by kube-apiserver
	// holding the aggregation layer requestheader CA and allowed names.
	requestHeaderAuthConfigMap = "extension-apiserver-authentication"
	// requestHeaderAuthNamespace is the namespace of the requestheader ConfigMap.
	requestHeaderAuthNamespace = "kube-system"
	// requestHeaderClientCAKey is the ConfigMap key holding the PEM-encoded
	// requestheader client CA bundle.
	requestHeaderClientCAKey = "requestheader-client-ca-file"
	// requestHeaderAllowedNamesKey is the ConfigMap key holding the JSON-encoded
	// list of allowed proxy certificate common names.
	requestHeaderAllowedNamesKey = "requestheader-allowed-names"
)

// requestHeaderAuthState is an immutable snapshot of the requestheader
// authentication configuration. It is swapped atomically so that CA rotation
// takes effect for new requests without restarting the server or pausing
// request processing. Requests already in flight keep the authenticator they
// started with.
type requestHeaderAuthState struct {
	// caPEM is the raw PEM bundle the pool was built from, used to detect changes.
	caPEM string
	// pool is the requestheader CA pool, or nil if no CA has been loaded yet.
	pool *x509.CertPool
	// allowedNames is the list of allowed proxy certificate common names.
	// Empty means any certificate verified against the CA is allowed.
	allowedNames []string
}

// RequestHeaderAuthenticator authenticates requests to the extensions apiserver
// against the Kubernetes aggregation layer requestheader CA, using the
// standard k8s.io/apiserver x509 authenticator. The CA bundle and allowed
// names are loaded from the extension-apiserver-authentication ConfigMap in
// kube-system and watched for changes, so certificate rotation is applied
// without restarting the server: the authenticator reads the current CA pool
// on every request.
type RequestHeaderAuthenticator struct {
	logger *logrus.Entry
	errs   *errors.Errors
	state  atomic.Value // *requestHeaderAuthState
	auth   authenticator.Request
}

// NewRequestHeaderAuthenticator returns a RequestHeaderAuthenticator with no
// CA loaded. Call Load to load the initial ConfigMap, and Watch to keep it
// updated. Until a CA is loaded, AuthenticateRequest rejects every request.
func NewRequestHeaderAuthenticator(logger *logrus.Entry) *RequestHeaderAuthenticator {
	a := &RequestHeaderAuthenticator{logger: logger}
	a.errs = errors.FromStruct(a)
	a.state.Store(&requestHeaderAuthState{})
	verifyOptionsFn := func() (x509.VerifyOptions, bool) {
		state := a.currentState()
		if state == nil || state.pool == nil {
			// No CA loaded yet: express no opinion, so the request is not authenticated.
			return x509.VerifyOptions{}, false
		}
		opts := requestx509.DefaultVerifyOptions()
		opts.Roots = state.pool
		return opts, true
	}
	allowedNamesFn := func() []string {
		return a.currentState().allowedNames
	}
	// The outer verifier checks the client certificate against the current
	// CA and the allowed common names, then the inner authenticator converts
	// the verified chain into user info. Both read the current CA through
	// verifyOptionsFn, so rotation applies to both checks at once.
	inner := requestx509.NewDynamic(verifyOptionsFn, requestx509.CommonNameUserConversion)
	a.auth = requestx509.NewDynamicCAVerifier(verifyOptionsFn, inner, requestx509.StringSliceProviderFunc(allowedNamesFn))
	return a
}

func (a *RequestHeaderAuthenticator) currentState() *requestHeaderAuthState {
	state, _ := a.state.Load().(*requestHeaderAuthState)
	return state
}

// AuthenticateRequest implements authenticator.Request, verifying the client
// certificate presented on the request against the current requestheader CA
// and allowed common names.
func (a *RequestHeaderAuthenticator) AuthenticateRequest(req *http.Request) (*authenticator.Response, bool, error) {
	return a.auth.AuthenticateRequest(req)
}

// UpdateFromConfigMap rebuilds the authenticator from the given ConfigMap.
// If the CA bundle or allowed names are unchanged, it is a no-op. If the
// ConfigMap cannot be parsed, the previous configuration is kept and an
// error is returned, so a bad update can never break authentication.
func (a *RequestHeaderAuthenticator) UpdateFromConfigMap(cm *corev1.ConfigMap) error {
	caPEM, ok := cm.Data[requestHeaderClientCAKey]
	if !ok || caPEM == "" {
		return a.errs.Errorf("ConfigMap %s/%s has no %q entry", requestHeaderAuthNamespace, requestHeaderAuthConfigMap, requestHeaderClientCAKey)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM([]byte(caPEM)) {
		return a.errs.Errorf("no certificates parsed from %q in ConfigMap %s/%s", requestHeaderClientCAKey, requestHeaderAuthNamespace, requestHeaderAuthConfigMap)
	}
	var allowedNames []string
	if raw, ok := cm.Data[requestHeaderAllowedNamesKey]; ok && raw != "" {
		if err := json.Unmarshal([]byte(raw), &allowedNames); err != nil {
			return a.errs.Wrap(err, "could not parse requestheader-allowed-names")
		}
	}
	current := a.currentState()
	if current != nil && current.caPEM == caPEM && slices.Equal(current.allowedNames, allowedNames) {
		return nil
	}
	a.state.Store(&requestHeaderAuthState{caPEM: caPEM, pool: pool, allowedNames: allowedNames})
	a.logger.WithField("allowedNames", allowedNames).Info("requestheader authentication CA updated")
	return nil
}

// Load loads the requestheader CA from the extension-apiserver-authentication
// ConfigMap. Call once at startup, before serving requests, so authentication
// is active from the first request.
func (a *RequestHeaderAuthenticator) Load(ctx context.Context, kubeClient kubernetes.Interface) error {
	cm, err := kubeClient.CoreV1().ConfigMaps(requestHeaderAuthNamespace).Get(ctx, requestHeaderAuthConfigMap, metav1.GetOptions{})
	if err != nil {
		return a.errs.Wrap(err, "could not load extension-apiserver-authentication ConfigMap")
	}
	return a.UpdateFromConfigMap(cm)
}

// Watch keeps the authenticator updated from the extension-apiserver-authentication
// ConfigMap, so certificate rotation is applied without restarting the server.
// It registers on the extensions server's shared InformerFactory, so there is
// still only one factory (and one watch) for the process: the ConfigMap is
// filtered out by name and namespace in the event handler, since the shared
// informer watches ConfigMaps cluster-wide. It returns immediately; the watch
// stops when ctx is done.
func (a *RequestHeaderAuthenticator) Watch(ctx context.Context, factory informers.SharedInformerFactory) {
	informer := factory.Core().V1().ConfigMaps().Informer()
	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) { a.handleConfigMapEvent(obj) },
		UpdateFunc: func(_, newObj interface{}) {
			a.handleConfigMapEvent(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			if !isRequestHeaderAuthConfigMap(obj) {
				return
			}
			a.logger.Warn("extension-apiserver-authentication ConfigMap deleted, keeping the last loaded CA")
		},
	})
	if err != nil {
		a.logger.WithError(err).Error("could not watch extension-apiserver-authentication ConfigMap")
		return
	}
	factory.Start(ctx.Done())
}

// isRequestHeaderAuthConfigMap reports whether obj is the
// extension-apiserver-authentication ConfigMap in kube-system.
func isRequestHeaderAuthConfigMap(obj interface{}) bool {
	cm, ok := obj.(*corev1.ConfigMap)
	if !ok {
		if deleted, ok := obj.(cache.DeletedFinalStateUnknown); ok {
			cm, ok = deleted.Obj.(*corev1.ConfigMap)
		}
		if !ok {
			return false
		}
	}
	return cm.Namespace == requestHeaderAuthNamespace && cm.Name == requestHeaderAuthConfigMap
}

func (a *RequestHeaderAuthenticator) handleConfigMapEvent(obj interface{}) {
	cm, ok := obj.(*corev1.ConfigMap)
	if !ok {
		return
	}
	// The shared informer watches ConfigMaps cluster-wide, so ignore
	// everything that is not the requestheader ConfigMap.
	if cm.Namespace != requestHeaderAuthNamespace || cm.Name != requestHeaderAuthConfigMap {
		return
	}
	if err := a.UpdateFromConfigMap(cm); err != nil {
		a.logger.WithError(err).Error("could not update requestheader authentication from ConfigMap, keeping the previous CA")
	}
}
