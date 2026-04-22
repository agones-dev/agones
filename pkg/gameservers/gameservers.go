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

package gameservers

import (
	"net"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// isGameServerPod returns if this Pod is a Pod that comes from a GameServer
func isGameServerPod(pod *corev1.Pod) bool {
	// add some defense in case this happens
	if pod == nil {
		return false
	}

	if agonesv1.GameServerRolePodSelector.Matches(labels.Set(pod.ObjectMeta.Labels)) {
		owner := metav1.GetControllerOf(pod)
		return owner != nil && owner.Kind == "GameServer"
	}

	return false
}

// address returns the "primary" network address that the given Pod is run on,
// and a slice of all network addresses.
//
// The primary address will default to the ExternalDNS, but if the ExternalDNS is
// not set, it will fall back to the ExternalIP then InternalDNS then InternalIP,
// If externalDNS is false, skip ExternalDNS and InternalDNS.
// since we can have clusters that are private, and/or tools like minikube
// that only report an InternalIP.
func address(node *corev1.Node) (string, []corev1.NodeAddress, error) {
	addresses := make([]corev1.NodeAddress, 0, len(node.Status.Addresses))
	for _, a := range node.Status.Addresses {
		addresses = append(addresses, *a.DeepCopy())
	}

	for _, a := range addresses {
		if a.Type == corev1.NodeExternalDNS {
			return a.Address, addresses, nil
		}
	}

	for _, a := range addresses {
		if a.Type == corev1.NodeExternalIP && net.ParseIP(a.Address) != nil {
			return a.Address, addresses, nil
		}
	}

	// There might not be a public DNS/IP, so fall back to the private DNS/IP
	for _, a := range addresses {
		if a.Type == corev1.NodeInternalDNS {
			return a.Address, addresses, nil
		}
	}

	for _, a := range addresses {
		if a.Type == corev1.NodeInternalIP && net.ParseIP(a.Address) != nil {
			return a.Address, addresses, nil
		}
	}

	return "", nil, errors.Errorf("Could not find an address for Node: %s", node.ObjectMeta.Name)
}

// applyGameServerAddressAndPort gathers the address and port details from the node and pod
// and applies them to the GameServer that is passed in, and returns it.
func applyGameServerAddressAndPort(gs *agonesv1.GameServer, node *corev1.Node, pod *corev1.Pod, syncPodPortsToGameServer func(*agonesv1.GameServer, *corev1.Pod) error) (*agonesv1.GameServer, error) {
	addr, addrs, err := address(node)
	if err != nil {
		return gs, errors.Wrapf(err, "error getting external address for GameServer %s", gs.ObjectMeta.Name)
	}

	gs.Status.Address = addr
	gs.Status.Addresses = addrs
	gs.Status.NodeName = pod.Spec.NodeName

	for _, ip := range pod.Status.PodIPs {
		gs.Status.Addresses = append(gs.Status.Addresses, corev1.NodeAddress{
			Type:    agonesv1.NodePodIP,
			Address: ip.IP,
		})
	}

	if err := syncPodPortsToGameServer(gs, pod); err != nil {
		return gs, errors.Wrapf(err, "cloud product error syncing ports on GameServer %s", gs.ObjectMeta.Name)
	}

	// HostPort is always going to be populated, even when dynamic
	// This will be a double up of information, but it will be easier to read
	gs.Status.Ports = make([]agonesv1.GameServerStatusPort, len(gs.Spec.Ports))
	for i, p := range gs.Spec.Ports {
		gs.Status.Ports[i] = p.Status()
	}

	return gs, nil
}

// isBeforePodCreated checks to see if the GameServer is in a state in which the pod could not have been
// created yet. This includes "Starting" in which a pod MAY exist, but may not yet be available, depending on when the
// informer cache updates
func isBeforePodCreated(gs *agonesv1.GameServer) bool {
	state := gs.Status.State
	return state == agonesv1.GameServerStatePortAllocation || state == agonesv1.GameServerStateCreating || state == agonesv1.GameServerStateStarting
}
