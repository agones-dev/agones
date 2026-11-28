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
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
)

func testLogger() *logrus.Entry {
	return logrus.NewEntry(logrus.New())
}

// generateTestCA returns a self-signed CA certificate and its PEM encoding.
func generateTestCA(t *testing.T, cn string) (*x509.Certificate, *rsa.PrivateKey, []byte) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return cert, key, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

// generateTestClientCert returns a client certificate signed by the given CA.
func generateTestClientCert(t *testing.T, ca *x509.Certificate, caKey *rsa.PrivateKey, cn string, notAfter time.Time) (*x509.Certificate, tls.Certificate) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, ca, &key.PublicKey, caKey)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return cert, tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  key,
		Leaf:        cert,
	}
}

// generateTestServerCert returns a server certificate signed by the given CA.
func generateTestServerCert(t *testing.T, ca *x509.Certificate, caKey *rsa.PrivateKey) tls.Certificate {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost"},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, ca, &key.PublicKey, caKey)
	require.NoError(t, err)
	return tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  key,
	}
}

func requestHeaderConfigMap(caPEM []byte, allowedNames []string) *corev1.ConfigMap {
	names, err := json.Marshal(allowedNames)
	if err != nil {
		panic(err)
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      requestHeaderAuthConfigMap,
			Namespace: requestHeaderAuthNamespace,
		},
		Data: map[string]string{
			requestHeaderClientCAKey:     string(caPEM),
			requestHeaderAllowedNamesKey: string(names),
		},
	}
}

func requestWithClientCert(cert *x509.Certificate) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/apis/allocation.agones.dev/v1/namespaces/default/gameserverallocations", nil)
	if cert != nil {
		req.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}}
	}
	return req
}

func TestRequestHeaderAuthenticatorNoCALoaded(t *testing.T) {
	t.Parallel()
	auth := NewRequestHeaderAuthenticator(testLogger())
	ca, caKey, _ := generateTestCA(t, "test-ca")
	clientCert, _ := generateTestClientCert(t, ca, caKey, "system:auth-proxy", time.Now().Add(time.Hour))
	_, ok, err := auth.AuthenticateRequest(requestWithClientCert(clientCert))
	require.NoError(t, err)
	assert.False(t, ok, "no CA loaded, so nothing should authenticate")
}

func TestRequestHeaderAuthenticatorUpdateFromConfigMap(t *testing.T) {
	t.Parallel()
	ca, caKey, caPEM := generateTestCA(t, "test-ca")
	proxyCert, _ := generateTestClientCert(t, ca, caKey, "system:auth-proxy", time.Now().Add(time.Hour))
	otherCert, _ := generateTestClientCert(t, ca, caKey, "someone-else", time.Now().Add(time.Hour))

	auth := NewRequestHeaderAuthenticator(testLogger())

	// Missing CA key is an error, and the authenticator stays unloaded.
	err := auth.UpdateFromConfigMap(&corev1.ConfigMap{Data: map[string]string{}})
	require.Error(t, err)
	_, ok, err := auth.AuthenticateRequest(requestWithClientCert(proxyCert))
	require.NoError(t, err)
	assert.False(t, ok)

	// Bad PEM is an error, and the authenticator stays unloaded.
	err = auth.UpdateFromConfigMap(&corev1.ConfigMap{Data: map[string]string{requestHeaderClientCAKey: "not a pem"}})
	require.Error(t, err)

	// Bad allowed-names JSON is an error, and the authenticator stays unloaded.
	err = auth.UpdateFromConfigMap(&corev1.ConfigMap{Data: map[string]string{
		requestHeaderClientCAKey:     string(caPEM),
		requestHeaderAllowedNamesKey: "not json",
	}})
	require.Error(t, err)

	// Valid ConfigMap loads; only the allowed common name authenticates.
	err = auth.UpdateFromConfigMap(requestHeaderConfigMap(caPEM, []string{"system:auth-proxy"}))
	require.NoError(t, err)
	_, ok, err = auth.AuthenticateRequest(requestWithClientCert(proxyCert))
	require.NoError(t, err)
	assert.True(t, ok)
	_, ok, err = auth.AuthenticateRequest(requestWithClientCert(otherCert))
	require.Error(t, err)
	assert.False(t, ok, "common name is not in the allowed list")

	// No allowed-names key means any certificate verified against the CA is allowed.
	err = auth.UpdateFromConfigMap(&corev1.ConfigMap{Data: map[string]string{requestHeaderClientCAKey: string(caPEM)}})
	require.NoError(t, err)
	_, ok, err = auth.AuthenticateRequest(requestWithClientCert(otherCert))
	require.NoError(t, err)
	assert.True(t, ok)

	// A bad update keeps the previous configuration.
	err = auth.UpdateFromConfigMap(&corev1.ConfigMap{Data: map[string]string{requestHeaderClientCAKey: "not a pem"}})
	require.Error(t, err)
	_, ok, err = auth.AuthenticateRequest(requestWithClientCert(proxyCert))
	require.NoError(t, err)
	assert.True(t, ok, "previous CA must be kept after a failed update")

	// No client certificate, and no TLS at all, do not authenticate.
	_, ok, err = auth.AuthenticateRequest(requestWithClientCert(nil))
	require.NoError(t, err)
	assert.False(t, ok)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.TLS = nil
	_, ok, err = auth.AuthenticateRequest(req)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestRequestHeaderAuthenticatorUnknownCAAndExpired(t *testing.T) {
	t.Parallel()
	ca, caKey, caPEM := generateTestCA(t, "test-ca")
	otherCA, otherCAKey, _ := generateTestCA(t, "other-ca")
	foreignCert, _ := generateTestClientCert(t, otherCA, otherCAKey, "system:auth-proxy", time.Now().Add(time.Hour))
	expiredCert, _ := generateTestClientCert(t, ca, caKey, "system:auth-proxy", time.Now().Add(-time.Hour))

	auth := NewRequestHeaderAuthenticator(testLogger())
	require.NoError(t, auth.UpdateFromConfigMap(requestHeaderConfigMap(caPEM, []string{"system:auth-proxy"})))

	_, ok, err := auth.AuthenticateRequest(requestWithClientCert(foreignCert))
	require.Error(t, err)
	assert.False(t, ok, "certificate from an unknown CA must not authenticate")

	_, ok, err = auth.AuthenticateRequest(requestWithClientCert(expiredCert))
	require.Error(t, err)
	assert.False(t, ok, "expired certificate must not authenticate")
}

func TestRequestHeaderAuthenticatorRotation(t *testing.T) {
	t.Parallel()
	ca1, ca1Key, ca1PEM := generateTestCA(t, "ca-one")
	cert1, _ := generateTestClientCert(t, ca1, ca1Key, "system:auth-proxy", time.Now().Add(time.Hour))
	ca2, ca2Key, ca2PEM := generateTestCA(t, "ca-two")
	cert2, _ := generateTestClientCert(t, ca2, ca2Key, "system:auth-proxy", time.Now().Add(time.Hour))

	auth := NewRequestHeaderAuthenticator(testLogger())
	require.NoError(t, auth.UpdateFromConfigMap(requestHeaderConfigMap(ca1PEM, []string{"system:auth-proxy"})))
	_, ok, err := auth.AuthenticateRequest(requestWithClientCert(cert1))
	require.NoError(t, err)
	assert.True(t, ok)
	_, ok, err = auth.AuthenticateRequest(requestWithClientCert(cert2))
	require.Error(t, err)
	assert.False(t, ok)

	// Rotate to a bundle containing both CAs (as Kubernetes publishes during
	// rotation): both certificates authenticate, with no restart.
	bundle := append(append([]byte{}, ca1PEM...), ca2PEM...)
	require.NoError(t, auth.UpdateFromConfigMap(requestHeaderConfigMap(bundle, []string{"system:auth-proxy"})))
	_, ok, err = auth.AuthenticateRequest(requestWithClientCert(cert1))
	require.NoError(t, err)
	assert.True(t, ok)
	_, ok, err = auth.AuthenticateRequest(requestWithClientCert(cert2))
	require.NoError(t, err)
	assert.True(t, ok)

	// Rotate fully to the new CA: only the new certificate authenticates.
	require.NoError(t, auth.UpdateFromConfigMap(requestHeaderConfigMap(ca2PEM, []string{"system:auth-proxy"})))
	_, ok, err = auth.AuthenticateRequest(requestWithClientCert(cert2))
	require.NoError(t, err)
	assert.True(t, ok)
	_, ok, err = auth.AuthenticateRequest(requestWithClientCert(cert1))
	require.Error(t, err)
	assert.False(t, ok)
}

func TestRequestHeaderAuthenticatorLoadAndWatch(t *testing.T) {
	t.Parallel()
	ca, caKey, caPEM := generateTestCA(t, "test-ca")
	proxyCert, _ := generateTestClientCert(t, ca, caKey, "system:auth-proxy", time.Now().Add(time.Hour))

	ctx := context.Background()
	cm := requestHeaderConfigMap(caPEM, []string{"system:auth-proxy"})
	kubeClient := fake.NewSimpleClientset(cm)

	auth := NewRequestHeaderAuthenticator(testLogger())
	require.NoError(t, auth.Load(ctx, kubeClient))
	_, ok, err := auth.AuthenticateRequest(requestWithClientCert(proxyCert))
	require.NoError(t, err)
	assert.True(t, ok)

	// Loading when the ConfigMap does not exist is an error.
	missing := NewRequestHeaderAuthenticator(testLogger())
	err = missing.Load(ctx, fake.NewSimpleClientset())
	require.Error(t, err)

	// Updates delivered through the watch are applied without a restart.
	ca2, ca2Key, ca2PEM := generateTestCA(t, "ca-two")
	cert2, _ := generateTestClientCert(t, ca2, ca2Key, "system:auth-proxy", time.Now().Add(time.Hour))
	watchCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	factory := informers.NewSharedInformerFactory(kubeClient, 0)
	auth.Watch(watchCtx, factory)
	updated := requestHeaderConfigMap(ca2PEM, []string{"system:auth-proxy"})
	updated.ResourceVersion = "2"
	_, err = kubeClient.CoreV1().ConfigMaps(requestHeaderAuthNamespace).Update(ctx, updated, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		_, ok, err := auth.AuthenticateRequest(requestWithClientCert(cert2))
		return err == nil && ok
	}, 5*time.Second, 100*time.Millisecond, "rotated CA from the watch was not applied")
}

// TestRequestHeaderAuthenticatorIgnoresOtherConfigMaps checks that the
// shared cluster-wide ConfigMap informer only applies the
// extension-apiserver-authentication ConfigMap in kube-system: any other
// ConfigMap, even one carrying a valid CA, must not change authentication.
func TestRequestHeaderAuthenticatorIgnoresOtherConfigMaps(t *testing.T) {
	t.Parallel()
	ca, caKey, caPEM := generateTestCA(t, "test-ca")
	proxyCert, _ := generateTestClientCert(t, ca, caKey, "system:auth-proxy", time.Now().Add(time.Hour))
	ca2, ca2Key, ca2PEM := generateTestCA(t, "ca-two")
	cert2, _ := generateTestClientCert(t, ca2, ca2Key, "system:auth-proxy", time.Now().Add(time.Hour))

	auth := NewRequestHeaderAuthenticator(testLogger())
	require.NoError(t, auth.UpdateFromConfigMap(requestHeaderConfigMap(caPEM, []string{"system:auth-proxy"})))
	_, ok, err := auth.AuthenticateRequest(requestWithClientCert(proxyCert))
	require.NoError(t, err)
	assert.True(t, ok)

	// Same name, wrong namespace: ignored.
	otherNamespace := requestHeaderConfigMap(ca2PEM, []string{"system:auth-proxy"})
	otherNamespace.Namespace = "other-namespace"
	auth.handleConfigMapEvent(otherNamespace)

	// Right namespace, wrong name: ignored.
	otherName := requestHeaderConfigMap(ca2PEM, []string{"system:auth-proxy"})
	otherName.Name = "some-other-config"
	auth.handleConfigMapEvent(otherName)

	// Neither changed the CA: the original certificate still authenticates,
	// and the other CA still does not.
	_, ok, err = auth.AuthenticateRequest(requestWithClientCert(proxyCert))
	require.NoError(t, err)
	assert.True(t, ok)
	_, ok, err = auth.AuthenticateRequest(requestWithClientCert(cert2))
	require.Error(t, err)
	assert.False(t, ok)
}

func TestAPIServerResourceHandlerRequestHeaderAuth(t *testing.T) {
	t.Parallel()
	ca, caKey, caPEM := generateTestCA(t, "test-ca")
	_, proxyPair := generateTestClientCert(t, ca, caKey, "system:auth-proxy", time.Now().Add(time.Hour))
	serverCert := generateTestServerCert(t, ca, caKey)

	pool := x509.NewCertPool()
	require.True(t, pool.AppendCertsFromPEM(caPEM))

	start := func(t *testing.T, auth *RequestHeaderAuthenticator) (*httptest.Server, string, *bool) {
		t.Helper()
		mux := http.NewServeMux()
		api := NewAPIServer(mux)
		if auth != nil {
			api.SetRequestHeaderAuthenticator(auth)
		}
		handled := new(bool)
		api.AddAPIResource(gv.String(), resource, func(_ http.ResponseWriter, _ *http.Request, _ string) error {
			*handled = true
			return nil
		})
		ts := httptest.NewUnstartedServer(mux)
		ts.TLS = &tls.Config{
			Certificates: []tls.Certificate{serverCert},
			ClientAuth:   tls.RequestClientCert,
		}
		ts.StartTLS()
		t.Cleanup(ts.Close)
		return ts, ts.URL + "/apis/allocation.agones.dev/v1/namespaces/default/gameserverallocations", handled
	}

	t.Run("no authenticator, no client cert", func(t *testing.T) {
		ts, path, handled := start(t, nil)
		resp, err := ts.Client().Get(path)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		require.NoError(t, resp.Body.Close())
		assert.True(t, *handled)
	})

	t.Run("authenticator, no client cert", func(t *testing.T) {
		auth := NewRequestHeaderAuthenticator(testLogger())
		require.NoError(t, auth.UpdateFromConfigMap(requestHeaderConfigMap(caPEM, []string{"system:auth-proxy"})))
		ts, path, handled := start(t, auth)
		resp, err := ts.Client().Get(path)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		require.NoError(t, resp.Body.Close())
		assert.False(t, *handled)
	})

	t.Run("authenticator, valid client cert", func(t *testing.T) {
		auth := NewRequestHeaderAuthenticator(testLogger())
		require.NoError(t, auth.UpdateFromConfigMap(requestHeaderConfigMap(caPEM, []string{"system:auth-proxy"})))
		_, path, handled := start(t, auth)
		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:      pool,
					Certificates: []tls.Certificate{proxyPair},
				},
			},
		}
		resp, err := client.Get(path)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		require.NoError(t, resp.Body.Close())
		assert.True(t, *handled)
	})
}
