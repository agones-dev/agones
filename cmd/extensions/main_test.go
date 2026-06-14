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

package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

func selfSignedCAPEM(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)

	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}))
}

func TestLoadRequestHeaderCA(t *testing.T) {
	t.Parallel()

	t.Run("valid CA in ConfigMap", func(t *testing.T) {
		t.Parallel()
		caPEM := selfSignedCAPEM(t)
		client := kubefake.NewSimpleClientset(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "extension-apiserver-authentication",
				Namespace: "kube-system",
			},
			Data: map[string]string{
				"requestheader-client-ca-file": caPEM,
			},
		})
		pool, err := loadRequestHeaderCA(context.Background(), client)
		require.NoError(t, err)
		assert.NotNil(t, pool)
	})

	t.Run("ConfigMap not found", func(t *testing.T) {
		t.Parallel()
		client := kubefake.NewSimpleClientset()
		pool, err := loadRequestHeaderCA(context.Background(), client)
		assert.Error(t, err)
		assert.Nil(t, pool)
	})

	t.Run("requestheader-client-ca-file key missing", func(t *testing.T) {
		t.Parallel()
		client := kubefake.NewSimpleClientset(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "extension-apiserver-authentication",
				Namespace: "kube-system",
			},
			Data: map[string]string{
				"other-key": "value",
			},
		})
		pool, err := loadRequestHeaderCA(context.Background(), client)
		assert.Error(t, err)
		assert.Nil(t, pool)
	})

	t.Run("invalid PEM data", func(t *testing.T) {
		t.Parallel()
		client := kubefake.NewSimpleClientset(&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "extension-apiserver-authentication",
				Namespace: "kube-system",
			},
			Data: map[string]string{
				"requestheader-client-ca-file": "not-valid-pem",
			},
		})
		pool, err := loadRequestHeaderCA(context.Background(), client)
		assert.Error(t, err)
		assert.Nil(t, pool)
	})
}
