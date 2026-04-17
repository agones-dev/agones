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
	"agones.dev/agones/pkg/apis"
	"agones.dev/agones/pkg/apis/agones"
	"agones.dev/agones/pkg/util/runtime"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

const (
	// GameServerSetGameServerLabel is the label that the name of the GameServerSet
	// is set on the GameServer the GameServerSet controls
	GameServerSetGameServerLabel = agones.GroupName + "/gameserverset"
)

// +genclient
// +genclient:method=GetScale,verb=get,subresource=scale,result=k8s.io/api/autoscaling/v1.Scale
// +genclient:method=UpdateScale,verb=update,subresource=scale,input=k8s.io/api/autoscaling/v1.Scale,result=k8s.io/api/autoscaling/v1.Scale
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GameServerSet is the data structure for a set of GameServers.
// This matches philosophically with the relationship between
// Deployments and ReplicaSets
type GameServerSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GameServerSetSpec   `json:"spec"`
	Status GameServerSetStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GameServerSetList is a list of GameServerSet resources
type GameServerSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []GameServerSet `json:"items"`
}

// GameServerSetSpec the specification for GameServerSet
type GameServerSetSpec struct {
	// Replicas are the number of GameServers that should be in this set
	Replicas int32 `json:"replicas"`
	// Labels and Annotations to apply to GameServers when the number of Allocated GameServers drops below
	// the desired replicas on the underlying `GameServerSet`
	// +optional
	AllocationOverflow *AllocationOverflow `json:"allocationOverflow,omitempty"`
	// Scheduling strategy. Defaults to "Packed".
	Scheduling apis.SchedulingStrategy `json:"scheduling,omitempty"`
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
	// Template the GameServer template to apply for this GameServerSet
	Template GameServerTemplateSpec `json:"template"`
}

// GameServerSetStatus is the status of a GameServerSet
type GameServerSetStatus struct {
	// Replicas is the total number of current GameServer replicas
	Replicas int32 `json:"replicas"`
	// ReadyReplicas is the number of Ready GameServer replicas
	ReadyReplicas int32 `json:"readyReplicas"`
	// ReservedReplicas is the number of Reserved GameServer replicas
	ReservedReplicas int32 `json:"reservedReplicas"`
	// AllocatedReplicas is the number of Allocated GameServer replicas
	AllocatedReplicas int32 `json:"allocatedReplicas"`
	// ShutdownReplicas is the number of Shutdown GameServers replicas
	ShutdownReplicas int32 `json:"shutdownReplicas"`
	// [Stage:Alpha]
	// [FeatureFlag:PlayerTracking]
	// Players is the current total player capacity and count for this GameServerSet
	// +optional
	Players *AggregatedPlayerStatus `json:"players,omitempty"`
	// (Beta, CountsAndLists feature flag) Counters provides aggregated Counter capacity and Counter
	// count for this GameServerSet.
	// +optional
	Counters map[string]AggregatedCounterStatus `json:"counters,omitempty"`
	// (Beta, CountsAndLists feature flag) Lists provides aggregated List capacity and List values
	// for this GameServerSet.
	// +optional
	Lists map[string]AggregatedListStatus `json:"lists,omitempty"`
}

// ValidateUpdate validates when updates occur. The argument
// is the new GameServerSet, being passed into the old GameServerSet
func (gsSet *GameServerSet) ValidateUpdate(newGSS *GameServerSet) field.ErrorList {
	allErrs := validateName(newGSS, field.NewPath("metadata"))
	if !apiequality.Semantic.DeepEqual(gsSet.Spec.Template, newGSS.Spec.Template) {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "template"), gsSet.Spec.Template, "GameServerSet template cannot be updated"))
	}

	return allErrs
}

// Validate validates when Create occurs. Check the name size
func (gsSet *GameServerSet) Validate(apiHooks APIHooks) field.ErrorList {
	allErrs := validateName(gsSet, field.NewPath("metadata"))

	// check GameServer specification in a GameServerSet
	allErrs = append(allErrs, validateGSSpec(apiHooks, gsSet, field.NewPath("spec", "template", "spec"))...)
	allErrs = append(allErrs, apiHooks.ValidateScheduling(gsSet.Spec.Scheduling, field.NewPath("spec", "scheduling"))...)

	if gsSet.Spec.Priorities != nil && !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec", "priorities"), "FeatureCountsAndLists is not enabled"))
	}

	allErrs = append(allErrs, validateObjectMeta(&gsSet.Spec.Template.ObjectMeta, field.NewPath("spec", "template", "metadata"))...)
	return allErrs
}

// GetGameServerSpec get underlying GameServer specification
func (gsSet *GameServerSet) GetGameServerSpec() *GameServerSpec {
	return &gsSet.Spec.Template.Spec
}

// GameServer returns a single GameServer derived
// from the GameServer template
func (gsSet *GameServerSet) GameServer() *GameServer {
	gs := &GameServer{
		ObjectMeta: *gsSet.Spec.Template.ObjectMeta.DeepCopy(),
		Spec:       *gsSet.Spec.Template.Spec.DeepCopy(),
	}

	gs.Spec.Scheduling = gsSet.Spec.Scheduling

	// Switch to GenerateName, so that we always get a Unique name for the GameServer, and there
	// can be no collisions
	gs.ObjectMeta.GenerateName = gsSet.ObjectMeta.Name + "-"
	gs.ObjectMeta.Name = ""
	gs.ObjectMeta.Namespace = gsSet.ObjectMeta.Namespace
	gs.ObjectMeta.ResourceVersion = ""
	gs.ObjectMeta.UID = ""

	ref := metav1.NewControllerRef(gsSet, SchemeGroupVersion.WithKind("GameServerSet"))
	gs.ObjectMeta.OwnerReferences = append(gs.ObjectMeta.OwnerReferences, *ref)

	if gs.ObjectMeta.Labels == nil {
		gs.ObjectMeta.Labels = make(map[string]string, 2)
	}

	gs.ObjectMeta.Labels[GameServerSetGameServerLabel] = gsSet.ObjectMeta.Name
	gs.ObjectMeta.Labels[FleetNameLabel] = gsSet.ObjectMeta.Labels[FleetNameLabel]
	return gs
}
