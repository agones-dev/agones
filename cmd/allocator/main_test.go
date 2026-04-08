// Copyright 2019 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"os"
	"testing"

	pb "agones.dev/agones/pkg/allocation/go"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
)

func TestAllocateHandler(t *testing.T) {
	t.Parallel()

	h := serviceHandler{
		allocationCallback: func(_ context.Context, _ *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
			return &allocationv1.GameServerAllocation{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
				},
				Status: allocationv1.GameServerAllocationStatus{
					State: allocationv1.GameServerAllocationContention,
				},
			}, nil
		},
	}

	request := &pb.AllocationRequest{
		Namespace: "ns",
		MultiClusterSetting: &pb.MultiClusterSetting{
			Enabled: true,
		},
	}

	response, err := h.Allocate(context.Background(), request)
	if !assert.Nil(t, response) {
		return
	}
	st, ok := status.FromError(err)
	if !assert.True(t, ok) {
		return
	}
	assert.Equal(t, st.Code(), codes.Aborted)
}

func TestAllocateHandlerReturnsError(t *testing.T) {
	t.Parallel()

	h := serviceHandler{
		allocationCallback: func(_ context.Context, _ *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
			return nil, k8serror.NewBadRequest("error")
		},
	}

	request := &pb.AllocationRequest{}
	_, err := h.Allocate(context.Background(), request)
	if assert.Error(t, err) {
		assert.Equal(t, "error", err.Error())
	}
}

func TestGetTlsCert(t *testing.T) {
	t.Parallel()
	cert1, err := tls.X509KeyPair(serverCert1, serverKey1)
	assert.Nil(t, err, "expected (serverCert1, serverKey1) to create a cert")

	cert2, err := tls.X509KeyPair(serverCert2, serverKey2)
	assert.Nil(t, err, "expected (serverCert2, serverKey2) to create a cert")

	h := serviceHandler{
		tlsCert: &cert1,
	}

	retrievedCert1, err := h.getTLSCert(nil)
	assert.Nil(t, err, "expected getTlsCert() to not fail")
	assert.Equal(t, cert1.Certificate, retrievedCert1.Certificate, "expected the retrieved cert to be equal to the original one")

	h.tlsCert = &cert2
	retrievedCert2, err := h.getTLSCert(nil)
	assert.Nil(t, err, "expected getTlsCert() to not fail")
	assert.Equal(t, cert2.Certificate, retrievedCert2.Certificate, "expected the retrieved cert to be equal to the original one")
}

func TestHandlingStatus(t *testing.T) {
	t.Parallel()

	errorMessage := "GameServerAllocation is invalid"
	h := serviceHandler{
		allocationCallback: func(_ context.Context, _ *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
			return &metav1.Status{
				Status:  metav1.StatusFailure,
				Message: errorMessage,
				Reason:  metav1.StatusReasonInvalid,
				Details: &metav1.StatusDetails{
					Kind:  "GameServerAllocation",
					Group: allocationv1.SchemeGroupVersion.Group,
				},
				Code: http.StatusUnprocessableEntity,
			}, nil
		},
	}

	request := &pb.AllocationRequest{}
	_, err := h.Allocate(context.Background(), request)
	if !assert.Error(t, err, "expecting failure") {
		return
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Errorf("expecting status error: %v", err)
	}
	assert.Equal(t, 422, int(st.Code()))
	assert.Contains(t, st.Message(), errorMessage)
}

func TestBadReturnType(t *testing.T) {
	t.Parallel()

	h := serviceHandler{
		allocationCallback: func(_ context.Context, _ *allocationv1.GameServerAllocation) (k8sruntime.Object, error) {
			return &corev1.Secret{}, nil
		},
	}

	request := &pb.AllocationRequest{}
	_, err := h.Allocate(context.Background(), request)
	if !assert.Error(t, err, "expecting failure") {
		return
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Errorf("expecting status error: %v", err)
	}
	assert.Equal(t, codes.Internal, st.Code())
	assert.Contains(t, st.Message(), "internal server error")
}

func TestVerifyClientCertificateSucceeds(t *testing.T) {
	t.Parallel()

	crt := []byte(clientCert)
	certPool := x509.NewCertPool()
	assert.True(t, certPool.AppendCertsFromPEM(crt))

	h := serviceHandler{
		caCertPool: certPool,
	}

	block, _ := pem.Decode(crt)
	input := [][]byte{block.Bytes}
	assert.Nil(t, h.verifyClientCertificate(input, nil),
		"verifyClientCertificate failed.")
}

func TestVerifyClientCertificateFails(t *testing.T) {
	t.Parallel()

	crt := []byte(clientCert)
	certPool := x509.NewCertPool()
	h := serviceHandler{
		caCertPool: certPool,
	}

	block, _ := pem.Decode(crt)
	input := [][]byte{block.Bytes}
	assert.Error(t, h.verifyClientCertificate(input, nil),
		"verifyClientCertificate() succeeded, expected error.")
}

func TestGettingCaCert(t *testing.T) {
	t.Parallel()

	file, err := os.CreateTemp(".", "*.crt")
	if assert.Nil(t, err) {
		defer os.Remove(file.Name()) // nolint: errcheck
		_, err = file.WriteString(clientCert)
		if assert.Nil(t, err) {
			certPool, err := getCACertPool("./")
			if assert.Nil(t, err) {
				// linting complaints certPool.Subjects() has been deprecated since Go 1.18.
				// But since this cert doesn't come from SystemCertPool, it doesn't seem behavior
				// should be impacted. So marking the lint as ignored.
				assert.Len(t, certPool.Subjects(), 1) // nolint
			}
		}
	}
}

var clientCert = `-----BEGIN CERTIFICATE-----
MIIDuzCCAqOgAwIBAgIUduDWtqpUsp3rZhCEfUrzI05laVIwDQYJKoZIhvcNAQEL
BQAwbTELMAkGA1UEBhMCR0IxDzANBgNVBAgMBkxvbmRvbjEPMA0GA1UEBwwGTG9u
ZG9uMRgwFgYDVQQKDA9HbG9iYWwgU2VjdXJpdHkxFjAUBgNVBAsMDUlUIERlcGFy
dG1lbnQxCjAIBgNVBAMMASowHhcNMTkwNTAyMjIzMDQ3WhcNMjkwNDI5MjIzMDQ3
WjBtMQswCQYDVQQGEwJHQjEPMA0GA1UECAwGTG9uZG9uMQ8wDQYDVQQHDAZMb25k
b24xGDAWBgNVBAoMD0dsb2JhbCBTZWN1cml0eTEWMBQGA1UECwwNSVQgRGVwYXJ0
bWVudDEKMAgGA1UEAwwBKjCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEB
AKGDasjadVwe0bXUEQfZCkMEAkzn0qTud3RYytympmaS0c01SWFNZwPRO0rpdIOZ
fyXVXVOAhgmgCR6QuXySmyQIoYl/D6tVhc5r9FyWPIBtzQKCJTX0mZOZwMn22qvo
bfnDnVsZ1Ny3RLZIF3um3xovvePXyg1z7D/NvCogNuYpyUUEITPZX6ss5ods/U78
BxLhKrT8iyu61ZC+ZegbHQqFRngbeb348gE1JwKTslDfe4oH7tZ+bNDZxnGcvh9j
eyagpM0zys4gFfQf/vfD2aEsUJ+GesUQC6uGVoGnTFshFhBsAK6vpIQ4ZQujaJ0r
NKgJ/ccBJFiJXMCR44yWFY0CAwEAAaNTMFEwHQYDVR0OBBYEFEe1gDd8JpzgnvOo
1AEloAXxmxHCMB8GA1UdIwQYMBaAFEe1gDd8JpzgnvOo1AEloAXxmxHCMA8GA1Ud
EwEB/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBAI5GyuakVgunerCGCSN7Ghsr
ys9vJytbyT+BLmxNBPSXWQwcm3g9yCDdgf0Y3q3Eef7IEZu4I428318iLzhfKln1
ua4fxvmTFKJ65lQKNkc6Y4e3w1t+C2HOl6fOIVT231qsCoM5SAwQQpqAzEUj6kZl
x+3avw9KSlXqR/mCAkePyoKvprxeb6RVDdq92Ug0qzoAHLpvIkuHdlF0dNp6/kO0
1pVL0BqW+6UTimSSvH8F/cMeYKbkhpE1u2c/NtNwsR2jN4M9kl3KHqkynk67PfZv
pwlCqZx4M8FpdfCbOZeRLzClUBdD5qzev0L3RNUx7UJzEIN+4LCBv37DIojNOyA=
-----END CERTIFICATE-----
`

var serverCert1 = []byte(`-----BEGIN CERTIFICATE-----
MIIFazCCA1OgAwIBAgIUD7ekqktGEe+F+pq3ACvxKLgSsK8wDQYJKoZIhvcNAQEL
BQAwRTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM
GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0yMDA2MjAwNTUxMzZaFw0yMTA2
MjAwNTUxMzZaMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEw
HwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwggIiMA0GCSqGSIb3DQEB
AQUAA4ICDwAwggIKAoICAQCta6IXgSJHiScouNAQxGKiLWVvjdJRAkjW3eiCDq9Y
iOvM44282yBGOHs6RM9PL3egG5pTNMDDLHGEs+37h3nfxSmMav86g7wmOT989VX4
HfvLaOygFS2u2brU/9exHfgqIiYsosG5njh7waQ07t+MmzRyZhH4WC3JXXCE/ixi
9hYa7nFONWRJ7bgwfI1OjW77Bsu8nDJyYhqhJi7y4SC89vs1fxnW15iKGGCH9bs/
jbYJox4egIPcDXm1t75eWP8yr9atyQp6IHYQeLw6UTiLHH7bWpOP5whtVvJAT95o
WfFRBHyrgnHI/q3OBMGpMCPJEWWN6AwBiSb5AqMDjuwrWams/mfR8doAeHihYoAV
4a7Oeo/uR51orsT5xHtsQUyD89O/nvcTG/awz4fqbidUrRuD0e04pxB/uqr6zsAg
RxBto42cgFsABYnVf1GH9uKBFoVxhiR933JXo8XkaRl7LRRKp6Dun8sf85frU+P/
ciFrhC3MzPb2mQa8dnVO4OsYmkl3PlZv6deyZVCAyskEBkCXCjcXAm9T/HopTr6y
y4J5DNDvECs76yAPyVPKKF5F+aLTxhC7604FRTaDO6cNyhG1hOS2xU4qitZyl5Wo
pPE7q8EnXcgVi0k6IV42qCV+vpTB4m77C0ffsbGhzEwjAJi5+JZJ5OlMN1bMp3D3
NQIDAQABo1MwUTAdBgNVHQ4EFgQUPPx1Wuca7S2LCbVWfCstv+a7nAwwHwYDVR0j
BBgwFoAUPPx1Wuca7S2LCbVWfCstv+a7nAwwDwYDVR0TAQH/BAUwAwEB/zANBgkq
hkiG9w0BAQsFAAOCAgEAqFYw/cHnjCo9eF4SJZf6/zUaTWvr/5st7CotnThpMEPz
4HSjhXNW35b9a5a/PW/geFvkvftGCV21iCiNjySgAhoORsOHz6/FtVdmBflW7Rpf
dJ1jXPjJbscYNN1zcwIPOrHHq9TJHFibYI7E3J3L2G61DemBjPondsNlC+0gIJdA
YYvXQSTg+hhs6RjSLshbzS615Yh87N39+LvXE4sN5+uw+WrTqakn0CDvjzEMVrJk
cDHnC0201NYa/hbv+urwOQCY4bdnWqNXOCJGcUVuODs2VO7P99rU30Da0Yc2V0H3
CjWiM4Qy0/ETkE62+1lF34rzRDHxur2FPT8hu8nbN5KhgDTybce1dOUkYOJKEOTT
IitPCVXiVg2kloMT9vhx5oerxi8xqjHK4WaW3yIkbu2oDMjG8R/lKC4WdylDoQqk
K5WkRdrQQp2eMZwXdHtiyaITNE6Rbx7z10uFeOyZyHqsIRZnPz3YMXLrM0oeNtL9
fQpoiBeBJTvlQSjD8nWyeg/xSYLvO0jnWjfPbvmyhzmH8DERxsO1xq3OmVM04QhX
nj+x2nLIp3Ql0rQStHIpkFVKS96WTQFTNBUIWcXgFPZgbRw3BkRPLa4kbyT/n8y+
BABcbxYpuM3TRnIMPai/1Vkg2H3FYladd4UJScSV9YUC23KJKQynrS+EXLamKdc=
-----END CERTIFICATE-----`)
var serverCert2 = []byte(`-----BEGIN CERTIFICATE-----
MIIFazCCA1OgAwIBAgIUJ/m4g8RwmRn5jh7C2wbTbmXGDNMwDQYJKoZIhvcNAQEL
BQAwRTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM
GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0yMDA2MjAwNTU0MDdaFw0yMTA2
MjAwNTU0MDdaMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEw
HwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwggIiMA0GCSqGSIb3DQEB
AQUAA4ICDwAwggIKAoICAQC0bL5NaI0OOcYzyWwtoNfHok2A9r4QdD7BlRwPkJqy
9N+VjMHw44IfLmmP3vmWMBt9tlIWqu8uUt8Hj2ExK+9H1TKsS1sf/FFpAb3m/Lvf
uahqYIu0IyAy+AmXnEkby7pggn2EPiVp5anisICQiyz0k9uh3vthztb+hxhMTkfp
HY4QHj2/yIvwPtIouPZEa3TVl4S89t4csngbZeCNvbHnovV9KmfZe6By4CvLYiBO
ui1RU07jw5NSeAyb/0gZ3HCqJZZzzK6pGpZBMjAQLVkxBjWevm8QZIu+O6RMj1Fe
45dumo3DQe8w119c/pPgYbgNqjWTR3qDyf8LbarKXOhWdSovb5vPXQodKOSC45/b
EyqMpYpeye2puK3Cd9k1+G7V2PYEeRhD6kSjbuikF5xlaEWksTLDHltQgGNrWG6T
/1MldsYzHwjAFVkDEDwpW9Rk2J7u4dFQYUq40StR8878iEZ7G3ljUJPpTUPbZz3L
lrIRWv4xrS5n8sYs33H/CmfkfXHcUvEKr3vd//DxtbIOSvoi0gJnf+BbekOWbN+A
2kYb3Obs0+pEEWW2WvArBAxPkcRZ4UUuQpAK7+IJ/pyE4ZiFT60PxsxktsMTaOqx
7KFfL9uo9hibT/ixcOzvZFQ3j2eeonulc3pNEfqfRu6VmMc3smzkiU/tGKXQRqxT
vQIDAQABo1MwUTAdBgNVHQ4EFgQUN3nVFUq0PIlf0zldkr7/GlqVnyUwHwYDVR0j
BBgwFoAUN3nVFUq0PIlf0zldkr7/GlqVnyUwDwYDVR0TAQH/BAUwAwEB/zANBgkq
hkiG9w0BAQsFAAOCAgEABX2nkrHmp5v731XVWETWgLNpi11pe9rQRlO4sdFgfVYB
GuArap0HsooULPO0AKECELwKZ6NpW0Kqkul0jSQEusDbTsMwCW44HYYnR93suJ2D
7X4SY/G5aSnyj/6JotSKFTCnSu2DGdxaKs7ufKJf0G5n8FLSJJaeCUNrklXSRObG
qrk533Du97RT19us1/48YxuDiSFrlElVVleTZRLwI8zDf3ikXsESUJmMVRw5dxWm
WCNgdYY3hjeMLGHa3l5CzmmcGNaZVjrSxn1aE8Qiau2q/u5ScqFOtT7aMYvXSAEn
fv86V46If/jXZ9MnjMXcwO7Dh/bdgCaNBXsATcQVsyNfAv0PM5TRTLYI+q+tZr6a
TAwx78VoBkD/C3E9wquj2bBxtrjKHwQkVfgOinvHNYU20pW4J8AdrZvuEPyV60zs
uZnCH1E21KO20U0DdBUJ2sVCFjw2tF/T9sFpd10nGthMBrPamSD3anG3C+En2usE
YsJtdsxpNyMnV1cq7E/8rXmarQ3fHtAq1AlcMe0Dtz7tc4JLhNxu1qwjkHW+Wddc
D8ky1I78OMA3rj1Z6etXve+K+3BwP2CNWCN2iTZzlY4k4X1QUX6cBD/UbykEvpp6
jVgXv9Y4mMlGeJOaOqEysMXltsGL0f6E0QBUBMs8z/5NU7WkoMUdH3UY8QP6hLA=
-----END CERTIFICATE-----`)

var serverKey1 = []byte(`-----BEGIN PRIVATE KEY-----
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
kEIJI+7KY0lG3Ig3y5XPRM6kzYKGjA==
-----END PRIVATE KEY-----`)
var serverKey2 = []byte(`-----BEGIN PRIVATE KEY-----
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
xA2QN+Zomk0SRjSeYzZO/DT3LTykJXE=
-----END PRIVATE KEY-----`)
