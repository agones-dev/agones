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

package e2e

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedv1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

func TestPingHTTP(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	kubeCore := framework.KubeClient.CoreV1()
	svc, err := kubeCore.Services("agones-system").Get(ctx, "agones-ping-http-service", metav1.GetOptions{})
	require.NoError(t, err)

	ip, err := externalIP(t, kubeCore, svc)
	require.NoError(t, err)

	port := svc.Spec.Ports[0]
	// gate
	assert.Equal(t, "http", port.Name)
	assert.Equal(t, corev1.ProtocolTCP, port.Protocol)
	p, err := externalPort(svc, port)
	require.NoError(t, err)

	response, err := http.Get(fmt.Sprintf("http://%s:%d", ip, p))
	require.NoError(t, err)
	defer response.Body.Close() // nolint: errcheck

	assert.Equal(t, http.StatusOK, response.StatusCode)
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	assert.Equal(t, []byte("ok"), body)
}

func externalPort(svc *corev1.Service, port corev1.ServicePort) (int32, error) {
	switch svc.Spec.Type {
	case corev1.ServiceTypeNodePort:
		return port.NodePort, nil

	case corev1.ServiceTypeLoadBalancer:
		return port.Port, nil
	}

	return 0, errors.New("could not find external port")
}

func TestPingUDP(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	kubeCore := framework.KubeClient.CoreV1()
	svc, err := kubeCore.Services("agones-system").Get(ctx, "agones-ping-udp-service", metav1.GetOptions{})
	require.NoError(t, err)

	externalIP, err := externalIP(t, kubeCore, svc)
	require.NoError(t, err)

	port := svc.Spec.Ports[0]
	// gate
	assert.Equal(t, "udp", port.Name)
	assert.Equal(t, corev1.ProtocolUDP, port.Protocol)
	p, err := externalPort(svc, port)
	require.NoError(t, err)

	expected := "hello"
	reply, err := framework.SendUDP(t, fmt.Sprintf("%s:%d", externalIP, p), expected)
	require.NoError(t, err)
	assert.Equal(t, expected, reply)
}

func externalIP(t *testing.T, kubeCore typedv1.NodesGetter, svc *corev1.Service) (string, error) {
	externalIP := ""
	ctx := context.Background()

	logrus.WithField("svc", svc).Info("load balancer")

	// likely this is minikube, so go get the node ip
	if svc.Spec.Type == corev1.ServiceTypeNodePort {
		nodes, err := kubeCore.Nodes().List(ctx, metav1.ListOptions{})
		assert.Nil(t, err)
		assert.Len(t, nodes.Items, 1, "Should only be 1 node on minikube")

		addresses := nodes.Items[0].Status.Addresses
		for _, a := range addresses {
			if a.Type == corev1.NodeInternalIP {
				externalIP = a.Address
			}
		}
	} else {
		externalIP = svc.Status.LoadBalancer.Ingress[0].IP
	}

	var err error
	if externalIP == "" {
		err = errors.New("could not find external ip")
	}
	return externalIP, err
}
