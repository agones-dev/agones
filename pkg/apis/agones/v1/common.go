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

package v1

import (
	"math"

	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1validation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// Block of const Error messages and GameServerAllocation Counter actions
const (
	ErrContainerRequired        = "Container is required when using multiple containers in the pod template"
	ErrHostPort                 = "HostPort cannot be specified with a Dynamic or Passthrough PortPolicy"
	ErrPortPolicyStatic         = "PortPolicy must be Static"
	ErrContainerPortRequired    = "ContainerPort must be defined for Dynamic and Static PortPolicies"
	ErrContainerPortPassthrough = "ContainerPort cannot be specified with Passthrough PortPolicy"
	ErrContainerNameInvalid     = "Container must be empty or the name of a container in the pod template"
	// GameServerPriorityIncrement is a Counter Action that indiciates the Counter's Count should be incremented at Allocation.
	GameServerPriorityIncrement string = "Increment"
	// GameServerPriorityDecrement is a Counter Action that indiciates the Counter's Count should be decremented at Allocation.
	GameServerPriorityDecrement string = "Decrement"
	// GameServerPriorityCounter is a Type for sorting Game Servers by Counter
	GameServerPriorityCounter string = "Counter"
	// GameServerPriorityList is a Type for sorting Game Servers by List
	GameServerPriorityList string = "List"
	// GameServerPriorityAscending is a Priority Order where the smaller count is preferred in sorting.
	GameServerPriorityAscending string = "Ascending"
	// GameServerPriorityDescending is a Priority Order where the larger count is preferred in sorting.
	GameServerPriorityDescending string = "Descending"
)

// AggregatedPlayerStatus stores total player tracking values
type AggregatedPlayerStatus struct {
	Count    int64 `json:"count"`
	Capacity int64 `json:"capacity"`
}

// AggregatedCounterStatus stores total and allocated Counter tracking values
type AggregatedCounterStatus struct {
	AllocatedCount    int64 `json:"allocatedCount"`
	AllocatedCapacity int64 `json:"allocatedCapacity"`
	Count             int64 `json:"count"`
	Capacity          int64 `json:"capacity"`
}

// AggregatedListStatus stores total and allocated List tracking values
type AggregatedListStatus struct {
	AllocatedCount    int64 `json:"allocatedCount"`
	AllocatedCapacity int64 `json:"allocatedCapacity"`
	Count             int64 `json:"count"`
	Capacity          int64 `json:"capacity"`
}

// crd is an interface to get Name and Kind of CRD
type crd interface {
	GetName() string
	GetObjectKind() schema.ObjectKind
}

// validateName Check NameSize of a CRD
func validateName(c crd, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	name := c.GetName()
	// make sure the Name of a Fleet does not oversize the Label size in GSS and GS
	if len(name) > validation.LabelValueMaxLength {
		allErrs = append(allErrs, field.TooLongMaxLength(fldPath.Child("name"), name, 63))
	}
	return allErrs
}

// gsSpec is an interface which contains all necessary
// functions to perform common validations against it
type gsSpec interface {
	GetGameServerSpec() *GameServerSpec
}

// validateGSSpec Check GameServerSpec of a CRD
// Used by Fleet and GameServerSet
func validateGSSpec(apiHooks APIHooks, gs gsSpec, fldPath *field.Path) field.ErrorList {
	gsSpec := gs.GetGameServerSpec()
	gsSpec.ApplyDefaults()
	allErrs := gsSpec.Validate(apiHooks, "", fldPath)
	return allErrs
}

// validateObjectMeta Check ObjectMeta specification
// Used by Fleet, GameServerSet and GameServer
func validateObjectMeta(objMeta *metav1.ObjectMeta, fldPath *field.Path) field.ErrorList {
	allErrs := metav1validation.ValidateLabels(objMeta.Labels, fldPath.Child("labels"))
	allErrs = append(allErrs, apivalidation.ValidateAnnotations(objMeta.Annotations, fldPath.Child("annotations"))...)
	return allErrs
}

// AllocationOverflow specifies what labels and/or annotations to apply on Allocated GameServers
// if the desired number of the underlying `GameServerSet` drops below the number of Allocated GameServers
// attached to it.
type AllocationOverflow struct {
	// Labels to be applied to the `GameServer`
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations to be applied to the `GameServer`
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// Validate validates the label and annotation values
func (ao *AllocationOverflow) Validate(fldPath *field.Path) field.ErrorList {
	allErrs := metav1validation.ValidateLabels(ao.Labels, fldPath.Child("labels"))
	allErrs = append(allErrs, apivalidation.ValidateAnnotations(ao.Annotations, fldPath.Child("annotations"))...)
	return allErrs
}

// CountMatches returns the number of Allocated GameServers that match the labels and annotations, and
// the set of GameServers left over.
func (ao *AllocationOverflow) CountMatches(list []*GameServer) (int32, []*GameServer) {
	count := int32(0)
	var rest []*GameServer
	labelSelector := labels.Set(ao.Labels).AsSelector()
	annotationSelector := labels.Set(ao.Annotations).AsSelector()

	for _, gs := range list {
		if gs.Status.State != GameServerStateAllocated {
			continue
		}
		if !labelSelector.Matches(labels.Set(gs.ObjectMeta.Labels)) {
			rest = append(rest, gs)
			continue
		}
		if !annotationSelector.Matches(labels.Set(gs.ObjectMeta.Annotations)) {
			rest = append(rest, gs)
			continue
		}
		count++
	}

	return count, rest
}

// Apply applies the labels and annotations to the passed in GameServer
func (ao *AllocationOverflow) Apply(gs *GameServer) {
	if ao.Annotations != nil {
		if gs.ObjectMeta.Annotations == nil {
			gs.ObjectMeta.Annotations = map[string]string{}
		}
		for k, v := range ao.Annotations {
			gs.ObjectMeta.Annotations[k] = v
		}
	}
	if ao.Labels != nil {
		if gs.ObjectMeta.Labels == nil {
			gs.ObjectMeta.Labels = map[string]string{}
		}
		for k, v := range ao.Labels {
			gs.ObjectMeta.Labels[k] = v
		}
	}
}

// Priority is a sorting option for GameServers with Counters or Lists based on the available capacity,
// i.e. the current Capacity value, minus either the Count value or List length.
type Priority struct {
	// Type: Sort by a "Counter" or a "List".
	Type string `json:"type"`
	// Key: The name of the Counter or List. If not found on the GameServer, has no impact.
	Key string `json:"key"`
	// Order: Sort by "Ascending" or "Descending". "Descending" a bigger available capacity is preferred.
	// "Ascending" would be smaller available capacity is preferred.
	// The default sort order is "Ascending"
	Order string `json:"order"`
}

// SafeAdd prevents overflow by limiting the sum to math.MaxInt64.
func SafeAdd(x, y int64) int64 {
	if x > math.MaxInt64-y {
		return math.MaxInt64
	}
	return x + y
}
