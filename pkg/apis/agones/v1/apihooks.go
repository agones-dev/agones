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

package v1

import (
	"agones.dev/agones/pkg/apis"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// APIHooks is a subset of the cloudproduct.CloudProduct interface for cloud product hooks specific to this package.
type APIHooks interface {
	// ValidateGameServerSpec is called by GameServer.Validate to allow for product specific validation.
	ValidateGameServerSpec(*GameServerSpec, *field.Path) field.ErrorList

	// ValidateScheduling is called by Fleet and GameServerSet Validate() to allow for product specific validation of scheduling strategy.
	ValidateScheduling(apis.SchedulingStrategy, *field.Path) field.ErrorList

	// MutateGameServerPod is called by createGameServerPod to allow for product specific pod mutation.
	MutateGameServerPod(*GameServerSpec, *corev1.Pod) error

	// SetEviction is called by gs.Pod to enforce GameServer.Status.Eviction.
	SetEviction(*Eviction, *corev1.Pod) error
}
