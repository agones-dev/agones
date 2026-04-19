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

package testing

import (
	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// FakeAPIHooks is a no-op, fake implementation of APIHooks
type FakeAPIHooks struct {
}

var _ agonesv1.APIHooks = FakeAPIHooks{}

// ValidateGameServerSpec is called by GameServer.Validate to allow for product specific validation.
func (f FakeAPIHooks) ValidateGameServerSpec(_ *agonesv1.GameServerSpec, _ *field.Path) field.ErrorList {
	return nil
}

// ValidateScheduling is called by Fleet and GameServerSet Validate() to allow for product specific validation of scheduling strategy.
func (f FakeAPIHooks) ValidateScheduling(_ apis.SchedulingStrategy, _ *field.Path) field.ErrorList {
	return nil
}

// MutateGameServerPod is called by createGameServerPod to allow for product specific pod mutation.
func (f FakeAPIHooks) MutateGameServerPod(_ *agonesv1.GameServerSpec, _ *corev1.Pod) error {
	return nil
}

// SetEviction is called by gs.Pod to enforce GameServer.Status.Eviction.
func (f FakeAPIHooks) SetEviction(_ *agonesv1.Eviction, _ *corev1.Pod) error {
	return nil
}
