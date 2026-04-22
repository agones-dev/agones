// Copyright Contributors to Agones a Series of LF Projects, LLC.
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

// Package generic implements generic cloud product hooks
package generic

import (
	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	"agones.dev/agones/pkg/cloudproduct/eviction"
	"agones.dev/agones/pkg/portallocator"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/informers"
)

// New returns a new generic cloud product
//
//nolint:revive // ignore the unexported return; implements ControllerHooksInterface
func New() *generic { return &generic{} }

type generic struct{}

func (*generic) ValidateGameServerSpec(*agonesv1.GameServerSpec, *field.Path) field.ErrorList {
	return nil
}
func (*generic) ValidateScheduling(apis.SchedulingStrategy, *field.Path) field.ErrorList { return nil }
func (*generic) MutateGameServerPod(*agonesv1.GameServerSpec, *corev1.Pod) error         { return nil }

// SetEviction sets disruptions controls based on GameServer.Status.Eviction.
func (*generic) SetEviction(ev *agonesv1.Eviction, pod *corev1.Pod) error {
	return eviction.SetEviction(ev, pod)
}

func (*generic) SyncPodPortsToGameServer(*agonesv1.GameServer, *corev1.Pod) error { return nil }

func (*generic) NewPortAllocator(portRanges map[string]portallocator.PortRange,
	kubeInformerFactory informers.SharedInformerFactory,
	agonesInformerFactory externalversions.SharedInformerFactory) portallocator.Interface {
	return portallocator.New(portRanges, kubeInformerFactory, agonesInformerFactory)
}

func (*generic) WaitOnFreePorts() bool { return false }
