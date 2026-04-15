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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
)

func TestGameServerSetGameServer(t *testing.T) {
	gsSet := GameServerSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "namespace",
			UID:       "1234",
			Labels: map[string]string{
				FleetNameLabel: "fleetname",
			},
		},
		Spec: GameServerSetSpec{
			Replicas: 10,
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

	gs := gsSet.GameServer()
	assert.Equal(t, "", gs.ObjectMeta.Name)
	assert.Equal(t, gsSet.ObjectMeta.Namespace, gs.ObjectMeta.Namespace)
	assert.Equal(t, gsSet.ObjectMeta.Name+"-", gs.ObjectMeta.GenerateName)
	assert.Equal(t, gsSet.ObjectMeta.Name, gs.ObjectMeta.Labels[GameServerSetGameServerLabel])
	assert.Equal(t, gsSet.ObjectMeta.Labels[FleetNameLabel], gs.ObjectMeta.Labels[FleetNameLabel])

	assert.Equal(t, gs.Spec, gsSet.Spec.Template.Spec)
	assert.True(t, metav1.IsControlledBy(gs, &gsSet))
}

// TestGameServerSetValidateUpdate test GameServerSet Validate() and ValidateUpdate()
func TestGameServerSetValidateUpdate(t *testing.T) {
	gsSpec := defaultGameServer().Spec
	gsSet := GameServerSet{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: GameServerSetSpec{
			Replicas: 10,
			Template: GameServerTemplateSpec{
				Spec: gsSpec,
			},
		},
	}

	errs := gsSet.ValidateUpdate(gsSet.DeepCopy())
	assert.Empty(t, errs)

	newGSS := gsSet.DeepCopy()
	newGSS.Spec.Replicas = 5
	errs = gsSet.ValidateUpdate(newGSS)
	assert.Empty(t, errs)

	newGSS.Spec.Template.Spec.Ports[0].ContainerPort = 321
	errs = gsSet.ValidateUpdate(newGSS)
	assert.Len(t, errs, 1)
	assert.Equal(t, "spec.template", errs[0].Field)

	newGSS = gsSet.DeepCopy()
	longName := strings.Repeat("f", validation.LabelValueMaxLength+1)
	newGSS.Name = longName
	errs = newGSS.Validate(fakeAPIHooks{})
	assert.Len(t, errs, 1)
	assert.Equal(t, "metadata.name", errs[0].Field)

	newGSS.Name = ""
	newGSS.GenerateName = longName
	errs = newGSS.Validate(fakeAPIHooks{})
	assert.Len(t, errs, 0)

	newGSS = gsSet.DeepCopy()
	newGSS.Name = longName
	errs = gsSet.ValidateUpdate(newGSS)
	assert.Len(t, errs, 1)
	assert.Equal(t, "metadata.name", errs[0].Field)

	newGSS = gsSet.DeepCopy()
	newGSS.Spec.Template.ObjectMeta.Labels = make(map[string]string)
	newGSS.Spec.Template.ObjectMeta.Labels[longName] = ""
	errs = newGSS.Validate(fakeAPIHooks{})
	assert.Len(t, errs, 1)
	assert.Equal(t, "spec.template.metadata.labels", errs[0].Field)

	// Same validation applies to nested Labels which applies to GameServer pod
	newGSS = gsSet.DeepCopy()
	newGSS.Spec.Template.Spec.Template.ObjectMeta.Labels = make(map[string]string)
	newGSS.Spec.Template.Spec.Template.ObjectMeta.Labels[longName] = ""
	errs = newGSS.Validate(fakeAPIHooks{})
	assert.Len(t, errs, 1)
	assert.Equal(t, "spec.template.spec.template.metadata.labels", errs[0].Field)

	// Similar Annotations validation check
	newGSS = gsSet.DeepCopy()
	newGSS.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
	newGSS.Spec.Template.ObjectMeta.Annotations[longName] = ""
	errs = newGSS.Validate(fakeAPIHooks{})
	assert.Len(t, errs, 1)
	assert.Equal(t, "spec.template.metadata.annotations", errs[0].Field)

	// Nested GS Spec Annotations
	newGSS = gsSet.DeepCopy()
	newGSS.Spec.Template.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
	newGSS.Spec.Template.Spec.Template.ObjectMeta.Annotations[longName] = ""
	errs = newGSS.Validate(fakeAPIHooks{})
	assert.Len(t, errs, 1)
	assert.Equal(t, "spec.template.spec.template.metadata.annotations", errs[0].Field)

	gsSet.Spec.Template.Spec.Template =
		corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: "container", Image: "myimage"}, {Name: "container2", Image: "myimage"}},
			},
		}

	errs = gsSet.Validate(fakeAPIHooks{})
	assert.Len(t, errs, 2)
	assert.Equal(t, "spec.template.spec.container", errs[0].Field)
}
