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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// generateTestCA creates a self-signed CA cert and key for testing.
func generateTestCA(t *testing.T) (*x509.Certificate, *ecdsa.PrivateKey, []byte) {
	t.Helper()

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test-requestheader-ca",
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)

	caCert, err := x509.ParseCertificate(caCertDER)
	require.NoError(t, err)

	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})

	return caCert, caKey, caPEM
}

// generateTestClientCert creates a client cert signed by the given CA.
func generateTestClientCert(t *testing.T, caCert *x509.Certificate, caKey *ecdsa.PrivateKey, cn string) tls.Certificate {
	t.Helper()

	clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName: cn,
		},
		NotBefore: time.Now().Add(-1 * time.Hour),
		NotAfter:  time.Now().Add(24 * time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageClientAuth,
		},
	}

	clientCertDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caCert, &clientKey.PublicKey, caKey)
	require.NoError(t, err)

	return tls.Certificate{
		Certificate: [][]byte{clientCertDER},
		PrivateKey:  clientKey,
	}
}

func TestLoadRequestHeaderConfig(t *testing.T) {
	t.Parallel()

	_, _, caPEM := generateTestCA(t)

	t.Run("success", func(t *testing.T) {
		kubeClient := fake.NewSimpleClientset(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      extensionAPIServerAuthenticationCM,
				Namespace: kubeSystemNamespace,
			},
			Data: map[string]string{
				"requestheader-client-ca-file": string(caPEM),
				"requestheader-allowed-names":  `["front-proxy-client"]`,
				"requestheader-username-headers": `["X-Remote-User"]`,
				"requestheader-group-headers":    `["X-Remote-Group"]`,
			},
		})

		config, err := LoadRequestHeaderConfig(context.Background(), kubeClient)
		require.NoError(t, err)
		assert.NotNil(t, config.ClientCAPool)
		assert.Equal(t, []string{"front-proxy-client"}, config.AllowedNames)
		assert.Equal(t, []string{"X-Remote-User"}, config.UsernameHeaders)
		assert.Equal(t, []string{"X-Remote-Group"}, config.GroupHeaders)
	})

	t.Run("configmap not found", func(t *testing.T) {
		kubeClient := fake.NewSimpleClientset()
		_, err := LoadRequestHeaderConfig(context.Background(), kubeClient)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get")
	})

	t.Run("no ca file in configmap", func(t *testing.T) {
		kubeClient := fake.NewSimpleClientset(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      extensionAPIServerAuthenticationCM,
				Namespace: kubeSystemNamespace,
			},
			Data: map[string]string{},
		})
		_, err := LoadRequestHeaderConfig(context.Background(), kubeClient)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "requestheader-client-ca-file")
	})
}

func TestAuthenticateRequest(t *testing.T) {
	t.Parallel()

	caCert, caKey, _ := generateTestCA(t)
	pool := x509.NewCertPool()
	pool.AddCert(caCert)

	config := &RequestHeaderConfig{
		ClientCAPool:    pool,
		AllowedNames:    []string{"front-proxy-client"},
		UsernameHeaders: []string{"X-Remote-User"},
		GroupHeaders:    []string{"X-Remote-Group"},
	}

	t.Run("valid request", func(t *testing.T) {
		clientCert := generateTestClientCert(t, caCert, caKey, "front-proxy-client")
		peerCert, err := x509.ParseCertificate(clientCert.Certificate[0])
		require.NoError(t, err)

		r := &http.Request{
			TLS: &tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{peerCert},
			},
			Header: http.Header{
				"X-Remote-User":  []string{"system:admin"},
				"X-Remote-Group": []string{"system:masters"},
			},
		}

		username, groups, err := config.AuthenticateRequest(r)
		assert.NoError(t, err)
		assert.Equal(t, "system:admin", username)
		assert.Contains(t, groups, "system:masters")
	})

	t.Run("no client certificate", func(t *testing.T) {
		r := &http.Request{
			TLS:    &tls.ConnectionState{},
			Header: http.Header{},
		}
		_, _, err := config.AuthenticateRequest(r)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no client certificate")
	})

	t.Run("no TLS at all", func(t *testing.T) {
		r := &http.Request{
			Header: http.Header{},
		}
		_, _, err := config.AuthenticateRequest(r)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no client certificate")
	})

	t.Run("wrong CN", func(t *testing.T) {
		clientCert := generateTestClientCert(t, caCert, caKey, "evil-client")
		peerCert, err := x509.ParseCertificate(clientCert.Certificate[0])
		require.NoError(t, err)

		r := &http.Request{
			TLS: &tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{peerCert},
			},
			Header: http.Header{
				"X-Remote-User": []string{"system:admin"},
			},
		}
		_, _, err = config.AuthenticateRequest(r)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not in requestheader-allowed-names")
	})

	t.Run("cert not signed by requestheader CA", func(t *testing.T) {
		// Generate a different CA (not the one in config.ClientCAPool)
		otherCACert, otherCAKey, _ := generateTestCA(t)
		_ = otherCACert
		clientCert := generateTestClientCert(t, otherCACert, otherCAKey, "front-proxy-client")
		peerCert, err := x509.ParseCertificate(clientCert.Certificate[0])
		require.NoError(t, err)

		r := &http.Request{
			TLS: &tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{peerCert},
			},
			Header: http.Header{
				"X-Remote-User": []string{"system:admin"},
			},
		}
		_, _, err = config.AuthenticateRequest(r)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "verification failed")
	})

	t.Run("no username header", func(t *testing.T) {
		clientCert := generateTestClientCert(t, caCert, caKey, "front-proxy-client")
		peerCert, err := x509.ParseCertificate(clientCert.Certificate[0])
		require.NoError(t, err)

		r := &http.Request{
			TLS: &tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{peerCert},
			},
			Header: http.Header{}, // no X-Remote-User
		}
		_, _, err = config.AuthenticateRequest(r)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no username found")
	})

	t.Run("empty allowedNames means any CN accepted", func(t *testing.T) {
		configNoNames := &RequestHeaderConfig{
			ClientCAPool:    pool,
			AllowedNames:    nil, // empty = accept any
			UsernameHeaders: []string{"X-Remote-User"},
			GroupHeaders:    []string{"X-Remote-Group"},
		}

		clientCert := generateTestClientCert(t, caCert, caKey, "any-random-cn")
		peerCert, err := x509.ParseCertificate(clientCert.Certificate[0])
		require.NoError(t, err)

		r := &http.Request{
			TLS: &tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{peerCert},
			},
			Header: http.Header{
				"X-Remote-User": []string{"user1"},
			},
		}
		username, _, err := configNoNames.AuthenticateRequest(r)
		assert.NoError(t, err)
		assert.Equal(t, "user1", username)
	})
}

func TestParseJSONStringArray(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"json array", `["a","b","c"]`, []string{"a", "b", "c"}},
		{"single value", `["front-proxy-client"]`, []string{"front-proxy-client"}},
		{"empty array", `[]`, nil},
		{"empty string", "", nil},
		{"bare value", `front-proxy-client`, []string{"front-proxy-client"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseJSONStringArray(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
