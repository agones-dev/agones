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
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"agones.dev/agones/pkg"
	"agones.dev/agones/pkg/apis"
	"agones.dev/agones/pkg/util/runtime"
)

func TestFleetGameServerSetGameServer(t *testing.T) {
	t.Parallel()

	f := Fleet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "namespace",
			UID:       "1234",
		},
		Spec: FleetSpec{
			Replicas:   10,
			Scheduling: apis.Packed,
			Template: GameServerTemplateSpec{
				Spec: GameServerSpec{
					Ports: []GameServerPort{{ContainerPort: 1234}},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{Name: "container", Image: "myimage"}},
						},
					},
				},
			},
		},
	}

	gsSet := f.GameServerSet()
	assert.Equal(t, "", gsSet.ObjectMeta.Name)
	assert.Equal(t, f.ObjectMeta.Namespace, gsSet.ObjectMeta.Namespace)
	assert.Equal(t, f.ObjectMeta.Name+"-", gsSet.ObjectMeta.GenerateName)
	assert.Equal(t, f.ObjectMeta.Name, gsSet.ObjectMeta.Labels[FleetNameLabel])
	assert.Equal(t, int32(0), gsSet.Spec.Replicas)
	assert.Equal(t, f.Spec.Scheduling, gsSet.Spec.Scheduling)
	assert.Equal(t, f.Spec.Template, gsSet.Spec.Template)
	assert.True(t, metav1.IsControlledBy(gsSet, &f))

	runtime.FeatureTestMutex.Lock()
	defer runtime.FeatureTestMutex.Unlock()

	runtime.Must(runtime.ParseFeatures(fmt.Sprintf("%s=true", runtime.FeatureCountsAndLists)))
	gsSet = f.GameServerSet()
	assert.Nil(t, gsSet.Spec.AllocationOverflow)

	f.Spec.AllocationOverflow = &AllocationOverflow{
		Labels:      map[string]string{"stuff": "things"},
		Annotations: nil,
	}

	assert.Nil(t, f.Spec.Priorities)
	f.Spec.Priorities = []Priority{
		{Type: "Counter",
			Key:   "Foo",
			Order: "Ascending"}}
	assert.NotNil(t, f.Spec.Priorities)
	assert.Equal(t, f.Spec.Priorities[0], Priority{Type: "Counter", Key: "Foo", Order: "Ascending"})

	gsSet = f.GameServerSet()
	assert.NotNil(t, gsSet.Spec.AllocationOverflow)
	assert.Equal(t, "things", gsSet.Spec.AllocationOverflow.Labels["stuff"])

	assert.Equal(t, gsSet.Spec.Priorities[0], Priority{Type: "Counter", Key: "Foo", Order: "Ascending"})
}

func TestFleetApplyDefaults(t *testing.T) {
	f := &Fleet{}

	// gate
	assert.EqualValues(t, "", f.Spec.Strategy.Type)
	assert.EqualValues(t, "", f.Spec.Scheduling)
	assert.EqualValues(t, 0, f.Spec.Replicas)

	f.ApplyDefaults()
	assert.Equal(t, appsv1.RollingUpdateDeploymentStrategyType, f.Spec.Strategy.Type)
	assert.Equal(t, "25%", f.Spec.Strategy.RollingUpdate.MaxUnavailable.String())
	assert.Equal(t, "25%", f.Spec.Strategy.RollingUpdate.MaxSurge.String())
	assert.Equal(t, apis.Packed, f.Spec.Scheduling)
	assert.Equal(t, int32(0), f.Spec.Replicas)
	assert.Equal(t, pkg.Version, f.ObjectMeta.Annotations[VersionAnnotation])

	// Test apply defaults is idempotent -- calling ApplyDefaults more than one time does not change the original result.
	f.ApplyDefaults()
	assert.Equal(t, appsv1.RollingUpdateDeploymentStrategyType, f.Spec.Strategy.Type)
	assert.Equal(t, "25%", f.Spec.Strategy.RollingUpdate.MaxUnavailable.String())
	assert.Equal(t, "25%", f.Spec.Strategy.RollingUpdate.MaxSurge.String())
	assert.Equal(t, apis.Packed, f.Spec.Scheduling)
	assert.Equal(t, int32(0), f.Spec.Replicas)
	assert.Equal(t, pkg.Version, f.ObjectMeta.Annotations[VersionAnnotation])
}

func TestFleetUpperBoundReplicas(t *testing.T) {
	f := &Fleet{Spec: FleetSpec{Replicas: 10}}

	assert.Equal(t, int32(10), f.UpperBoundReplicas(12))
	assert.Equal(t, int32(10), f.UpperBoundReplicas(10))
	assert.Equal(t, int32(5), f.UpperBoundReplicas(5))
}

func TestFleetLowerBoundReplicas(t *testing.T) {
	f := &Fleet{Spec: FleetSpec{Replicas: 10}}

	assert.Equal(t, int32(5), f.LowerBoundReplicas(5))
	assert.Equal(t, int32(0), f.LowerBoundReplicas(0))
	assert.Equal(t, int32(0), f.LowerBoundReplicas(-5))
}

func TestSumStatusAllocatedReplicas(t *testing.T) {
	f := Fleet{}
	gsSet1 := f.GameServerSet()
	gsSet1.Status.AllocatedReplicas = 2

	gsSet2 := f.GameServerSet()
	gsSet2.Status.AllocatedReplicas = 3

	assert.Equal(t, int32(5), SumGameServerSets([]*GameServerSet{gsSet1, gsSet2}, func(gsSet *GameServerSet) int32 {
		return gsSet.Status.AllocatedReplicas
	}))
}

func TestFleetGameserverSpec(t *testing.T) {
	f := defaultFleet()
	f.ApplyDefaults()
	errs := f.Validate(fakeAPIHooks{})
	assert.Len(t, errs, 0)

	f.Spec.Template.Spec.Template =
		corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: "container", Image: "myimage"}, {Name: "container2", Image: "myimage"}},
			},
		}

	errs = f.Validate(fakeAPIHooks{})
	assert.Len(t, errs, 1)
	assert.Equal(t, "spec.template.spec.container", errs[0].Field)

	f.Spec.Template.Spec.Container = "testing"
	errs = f.Validate(fakeAPIHooks{})
	assert.Len(t, errs, 1)
	assert.Equal(t, "Could not find a container named testing", errs[0].Detail)

	f.Spec.Template.Spec.Container = "container"
	errs = f.Validate(fakeAPIHooks{})
	assert.Len(t, errs, 0)

	// Verify RollingUpdate parameters validation
	percent := intstr.FromString("0%")
	f.Spec.Strategy.RollingUpdate.MaxUnavailable = &percent
	f.Spec.Strategy.RollingUpdate.MaxSurge = &percent
	errs = f.Validate(fakeAPIHooks{})
	assert.Len(t, errs, 2)

	intParam := intstr.FromInt(0)
	f.Spec.Strategy.RollingUpdate.MaxUnavailable = &intParam
	f.Spec.Strategy.RollingUpdate.MaxSurge = &intParam
	errs = f.Validate(fakeAPIHooks{})
	assert.Len(t, errs, 2)

	percent = intstr.FromString("2a")
	f.Spec.Strategy.RollingUpdate.MaxUnavailable = &percent
	f.Spec.Strategy.RollingUpdate.MaxSurge = &percent
	errs = f.Validate(fakeAPIHooks{})
	assert.Len(t, errs, 2)

	longName := strings.Repeat("f", validation.LabelValueMaxLength+1)
	f = defaultFleet()
	f.ApplyDefaults()
	f.Spec.Template.ObjectMeta.Labels = make(map[string]string)
	f.Spec.Template.ObjectMeta.Labels["label"] = longName
	errs = f.Validate(fakeAPIHooks{})
	assert.Len(t, errs, 1)

	f = defaultFleet()
	f.ApplyDefaults()
	f.Spec.Template.Spec.Template.ObjectMeta.Labels = make(map[string]string)
	f.Spec.Template.Spec.Template.ObjectMeta.Labels["label"] = longName
	errs = f.Validate(fakeAPIHooks{})
	assert.Len(t, errs, 1)

	// Annotations test
	f = defaultFleet()
	f.ApplyDefaults()
	f.Spec.Template.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
	f.Spec.Template.Spec.Template.ObjectMeta.Annotations[longName] = ""
	errs = f.Validate(fakeAPIHooks{})
	assert.Len(t, errs, 1)

	// Strategy Type validation test
	f = defaultFleet()
	f.ApplyDefaults()
	f.Spec.Strategy.Type = appsv1.DeploymentStrategyType("")
	errs = f.Validate(fakeAPIHooks{})
	assert.Len(t, errs, 1)
}

func TestFleetAllocationOverflow(t *testing.T) {
	t.Parallel()

	f := defaultFleet()
	f.ApplyDefaults()

	errs := f.Validate(fakeAPIHooks{})
	require.Empty(t, errs)

	f.Spec.AllocationOverflow = &AllocationOverflow{
		Labels:      map[string]string{"$$$nope": "value"},
		Annotations: nil,
	}

	errs = f.Validate(fakeAPIHooks{})
	require.Len(t, errs, 1)
	require.Equal(t, field.ErrorTypeInvalid, errs[0].Type)

}

func TestFleetName(t *testing.T) {
	f := defaultFleet()
	f.ApplyDefaults()

	longName := strings.Repeat("f", validation.LabelValueMaxLength+1)
	f.Name = longName
	errs := f.Validate(fakeAPIHooks{})
	assert.Len(t, errs, 1)
	assert.Equal(t, "metadata.name", errs[0].Field)

	f.Name = ""
	f.GenerateName = longName
	errs = f.Validate(fakeAPIHooks{})
	assert.Len(t, errs, 0)
}

func TestSumStatusReplicas(t *testing.T) {
	fixture := []*GameServerSet{
		{Status: GameServerSetStatus{Replicas: 10}},
		{Status: GameServerSetStatus{Replicas: 15}},
		{Status: GameServerSetStatus{Replicas: 5}},
	}

	assert.Equal(t, int32(30), SumGameServerSets(fixture, func(gsSet *GameServerSet) int32 {
		return gsSet.Status.Replicas
	}))
}

func TestSumSpecReplicas(t *testing.T) {
	fixture := []*GameServerSet{
		{Spec: GameServerSetSpec{Replicas: 11}},
		{Spec: GameServerSetSpec{Replicas: 14}},
		{Spec: GameServerSetSpec{Replicas: 100}},
		nil,
	}

	assert.Equal(t, int32(125), SumGameServerSets(fixture, func(gsSet *GameServerSet) int32 {
		return gsSet.Spec.Replicas
	}))
}

func TestGetReadyReplicaCountForGameServerSets(t *testing.T) {
	fixture := []*GameServerSet{
		{Status: GameServerSetStatus{ReadyReplicas: 1000}},
		{Status: GameServerSetStatus{ReadyReplicas: 15}},
		{Status: GameServerSetStatus{ReadyReplicas: 5}},
		nil,
	}

	assert.Equal(t, int32(1020), SumGameServerSets(fixture, func(gsSet *GameServerSet) int32 {
		return gsSet.Status.ReadyReplicas
	}))
}

func TestSumGameServerSets(t *testing.T) {
	fixture := []*GameServerSet{
		{Status: GameServerSetStatus{ReadyReplicas: 1000}},
		{Status: GameServerSetStatus{ReadyReplicas: 15}},
		{Status: GameServerSetStatus{ReadyReplicas: 5}},
		nil,
	}

	assert.Equal(t, int32(1020), SumGameServerSets(fixture, func(gsSet *GameServerSet) int32 {
		return gsSet.Status.ReadyReplicas
	}))

	assert.Equal(t, int32(0), SumGameServerSets(fixture, func(gsSet *GameServerSet) int32 {
		return gsSet.Status.Replicas
	}))
}

func defaultFleet() *Fleet {
	gs := GameServer{
		Spec: GameServerSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "testing", Image: "testing/image"}}}}},
	}
	return &Fleet{
		ObjectMeta: metav1.ObjectMeta{GenerateName: "simple-fleet-", Namespace: "defaultNs"},
		Spec: FleetSpec{
			Replicas: 2,
			Template: GameServerTemplateSpec{
				Spec: gs.Spec,
			},
		},
	}
}
