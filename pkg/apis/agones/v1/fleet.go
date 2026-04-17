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
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"agones.dev/agones/pkg"
	"agones.dev/agones/pkg/apis"
	"agones.dev/agones/pkg/apis/agones"
	"agones.dev/agones/pkg/util/runtime"
)

const (
	// FleetNameLabel is the label that the name of the Fleet
	// is set to on GameServerSet and GameServer  the Fleet controls
	FleetNameLabel = agones.GroupName + "/fleet"
)

// +genclient
// +genclient:method=GetScale,verb=get,subresource=scale,result=k8s.io/api/autoscaling/v1.Scale
// +genclient:method=UpdateScale,verb=update,subresource=scale,input=k8s.io/api/autoscaling/v1.Scale,result=k8s.io/api/autoscaling/v1.Scale
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Fleet is the data structure for a Fleet resource
type Fleet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FleetSpec   `json:"spec"`
	Status FleetStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FleetList is a list of Fleet resources
type FleetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Fleet `json:"items"`
}

// FleetSpec is the spec for a Fleet
type FleetSpec struct {
	// Replicas are the number of GameServers that should be in this set. Defaults to 0.
	Replicas int32 `json:"replicas"`
	// Labels and/or Annotations to apply to overflowing GameServers when the number of Allocated GameServers is more
	// than the desired replicas on the underlying `GameServerSet`
	// +optional
	AllocationOverflow *AllocationOverflow `json:"allocationOverflow,omitempty"`
	// Deployment strategy
	Strategy appsv1.DeploymentStrategy `json:"strategy"`
	// Scheduling strategy. Defaults to "Packed".
	Scheduling apis.SchedulingStrategy `json:"scheduling"`
	// [Stage: Beta]
	// [FeatureFlag:CountsAndLists]
	// `Priorities` configuration alters scale down logic in Fleets based on the configured available capacity order under that key.
	//
	// Priority of sorting is in descending importance. I.e. The position 0 `priority` entry is checked first.
	//
	// For `Packed` strategy scale down, this priority list will be the tie-breaker within the node, to ensure optimal
	// infrastructure usage while also allowing some custom prioritisation of `GameServers`.
	//
	// For `Distributed` strategy scale down, the entire `Fleet` will be sorted by this priority list to provide the
	// order of `GameServers` to delete on scale down.
	// +optional
	Priorities []Priority `json:"priorities,omitempty"`
	// Template the GameServer template to apply for this Fleet
	Template GameServerTemplateSpec `json:"template"`
}

// FleetStatus is the status of a Fleet
type FleetStatus struct {
	// Replicas the total number of current GameServer replicas
	Replicas int32 `json:"replicas"`
	// ReadyReplicas are the number of Ready GameServer replicas
	ReadyReplicas int32 `json:"readyReplicas"`
	// ReservedReplicas are the total number of Reserved GameServer replicas in this fleet.
	// Reserved instances won't be deleted on scale down, but won't cause an autoscaler to scale up.
	ReservedReplicas int32 `json:"reservedReplicas"`
	// AllocatedReplicas are the number of Allocated GameServer replicas
	AllocatedReplicas int32 `json:"allocatedReplicas"`
	// [Stage:Alpha]
	// [FeatureFlag:PlayerTracking]
	// Players are the current total player capacity and count for this Fleet
	// +optional
	Players *AggregatedPlayerStatus `json:"players,omitempty"`
	// (Beta, CountsAndLists feature flag) Counters provides aggregated Counter capacity and Counter
	// count for this Fleet.
	// +optional
	Counters map[string]AggregatedCounterStatus `json:"counters,omitempty"`
	// (Beta, CountsAndLists feature flag) Lists provides aggregated List capacityv and List values
	// for this Fleet.
	// +optional
	Lists map[string]AggregatedListStatus `json:"lists,omitempty"`
}

// GameServerSet returns a single GameServerSet for this Fleet definition
func (f *Fleet) GameServerSet() *GameServerSet {
	gsSet := &GameServerSet{
		ObjectMeta: *f.Spec.Template.ObjectMeta.DeepCopy(),
		Spec: GameServerSetSpec{
			Template:   f.Spec.Template,
			Scheduling: f.Spec.Scheduling,
		},
	}

	// Switch to GenerateName, so that we always get a Unique name for the GameServerSet, and there
	// can be no collisions
	gsSet.ObjectMeta.GenerateName = f.ObjectMeta.Name + "-"
	gsSet.ObjectMeta.Name = ""
	gsSet.ObjectMeta.Namespace = f.ObjectMeta.Namespace
	gsSet.ObjectMeta.ResourceVersion = ""
	gsSet.ObjectMeta.UID = ""

	ref := metav1.NewControllerRef(f, SchemeGroupVersion.WithKind("Fleet"))
	gsSet.ObjectMeta.OwnerReferences = append(gsSet.ObjectMeta.OwnerReferences, *ref)

	if gsSet.ObjectMeta.Labels == nil {
		gsSet.ObjectMeta.Labels = make(map[string]string, 1)
	}

	gsSet.ObjectMeta.Labels[FleetNameLabel] = f.ObjectMeta.Name

	if f.Spec.AllocationOverflow != nil {
		gsSet.Spec.AllocationOverflow = f.Spec.AllocationOverflow.DeepCopy()
	}

	if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) && f.Spec.Priorities != nil {
		// DeepCopy done manually here as f.Spec.Priorities does not have a DeepCopy() method.
		gsSet.Spec.Priorities = make([]Priority, len(f.Spec.Priorities))
		copy(gsSet.Spec.Priorities, f.Spec.Priorities)
	}

	return gsSet
}

// ApplyDefaults applies default values to the Fleet
func (f *Fleet) ApplyDefaults() {
	if f.Spec.Strategy.Type == "" {
		f.Spec.Strategy.Type = appsv1.RollingUpdateDeploymentStrategyType
	}

	if f.Spec.Scheduling == "" {
		f.Spec.Scheduling = apis.Packed
	}

	if f.Spec.Strategy.Type == appsv1.RollingUpdateDeploymentStrategyType {
		if f.Spec.Strategy.RollingUpdate == nil {
			f.Spec.Strategy.RollingUpdate = &appsv1.RollingUpdateDeployment{}
		}

		def := intstr.FromString("25%")
		if f.Spec.Strategy.RollingUpdate.MaxSurge == nil {
			f.Spec.Strategy.RollingUpdate.MaxSurge = &def
		}
		if f.Spec.Strategy.RollingUpdate.MaxUnavailable == nil {
			f.Spec.Strategy.RollingUpdate.MaxUnavailable = &def
		}
	}
	// Add Agones version into Fleet Annotations
	if f.ObjectMeta.Annotations == nil {
		f.ObjectMeta.Annotations = make(map[string]string, 1)
	}
	f.ObjectMeta.Annotations[VersionAnnotation] = pkg.Version

}

// GetGameServerSpec get underlying Gameserver specification
func (f *Fleet) GetGameServerSpec() *GameServerSpec {
	return &f.Spec.Template.Spec
}

func (f *Fleet) validateRollingUpdate(value *intstr.IntOrString, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	r, err := intstr.GetValueFromIntOrPercent(value, 100, true)
	if value.Type == intstr.String {
		if err != nil || r < 1 || r > 99 {
			allErrs = append(allErrs, field.Invalid(fldPath, value, "must be between 1% and 99%"))
		}
	} else if r < 1 {
		allErrs = append(allErrs, field.Invalid(fldPath, value, "must be at least 1"))
	}
	return allErrs
}

// Validate validates the Fleet configuration.
// If a Fleet is invalid there will be > 0 values in
// the returned array
func (f *Fleet) Validate(apiHooks APIHooks) field.ErrorList {
	allErrs := validateName(f, field.NewPath("metadata"))

	strategyPath := field.NewPath("spec", "strategy")
	if f.Spec.Strategy.Type == appsv1.RollingUpdateDeploymentStrategyType {
		allErrs = append(allErrs, f.validateRollingUpdate(f.Spec.Strategy.RollingUpdate.MaxUnavailable, strategyPath.Child("rollingUpdate", "maxUnavailable"))...)
		allErrs = append(allErrs, f.validateRollingUpdate(f.Spec.Strategy.RollingUpdate.MaxSurge, strategyPath.Child("rollingUpdate", "maxSurge"))...)
	} else if f.Spec.Strategy.Type != appsv1.RecreateDeploymentStrategyType {
		allErrs = append(allErrs, field.NotSupported(strategyPath.Child("type"), f.Spec.Strategy.Type, []string{"RollingUpdate", "Recreate"}))
	}

	// check Gameserver specification in a Fleet
	allErrs = append(allErrs, validateGSSpec(apiHooks, f, field.NewPath("spec", "template", "spec"))...)
	allErrs = append(allErrs, apiHooks.ValidateScheduling(f.Spec.Scheduling, field.NewPath("spec", "scheduling"))...)
	allErrs = append(allErrs, validateObjectMeta(&f.Spec.Template.ObjectMeta, field.NewPath("spec", "template", "metadata"))...)

	if f.Spec.AllocationOverflow != nil {
		allErrs = append(allErrs, f.Spec.AllocationOverflow.Validate(field.NewPath("spec", "allocationOverflow"))...)
	}

	if f.Spec.Priorities != nil && !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "priorities"), "FeatureCountsAndLists is not enabled"))
	}

	return allErrs
}

// UpperBoundReplicas returns whichever is smaller,
// the value i, or the f.Spec.Replicas.
func (f *Fleet) UpperBoundReplicas(i int32) int32 {
	if i > f.Spec.Replicas {
		return f.Spec.Replicas
	}
	return i
}

// LowerBoundReplicas returns 0 (the minimum value for
// replicas) if i is < 0
func (f *Fleet) LowerBoundReplicas(i int32) int32 {
	if i < 0 {
		return 0
	}
	return i
}

// SumGameServerSets calculates a total from the value returned from the passed in function.
// Useful for calculating totals based on status value(s), such as gsSet.Status.Replicas
func SumGameServerSets(list []*GameServerSet, f func(gsSet *GameServerSet) int32) int32 {
	var total int32
	for _, gsSet := range list {
		if gsSet != nil {
			total += f(gsSet)
		}
	}

	return total
}
