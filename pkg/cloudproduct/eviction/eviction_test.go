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

package eviction

import (
	"testing"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSetEviction(t *testing.T) {
	emptyPodAnd := func(f func(*corev1.Pod)) *corev1.Pod {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{},
				Labels:      map[string]string{},
			},
		}
		f(pod)
		return pod
	}
	for desc, tc := range map[string]struct {
		eviction *agonesv1.Eviction
		pod      *corev1.Pod
		wantPod  *corev1.Pod
	}{
		"eviction: safe: Always, no incoming labels/annotations": {
			eviction: &agonesv1.Eviction{Safe: agonesv1.EvictionSafeAlways},
			pod:      emptyPodAnd(func(*corev1.Pod) {}),
			wantPod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation] = agonesv1.True
				pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = agonesv1.True
			}),
		},
		"eviction: safe: OnUpgrade, no incoming labels/annotations": {
			eviction: &agonesv1.Eviction{Safe: agonesv1.EvictionSafeOnUpgrade},
			pod:      emptyPodAnd(func(*corev1.Pod) {}),
			wantPod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation] = agonesv1.False
				pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = agonesv1.True
			}),
		},
		"eviction: safe: Never, no incoming labels/annotations": {
			eviction: &agonesv1.Eviction{Safe: agonesv1.EvictionSafeNever},
			pod:      emptyPodAnd(func(*corev1.Pod) {}),
			wantPod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation] = agonesv1.False
				pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = agonesv1.False
			}),
		},
		"eviction: safe: Always, incoming labels/annotations": {
			eviction: &agonesv1.Eviction{Safe: agonesv1.EvictionSafeAlways},
			pod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation] = "just don't touch, ok?"
				pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = "seriously, leave it"
			}),
			wantPod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation] = "just don't touch, ok?"
				pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = "seriously, leave it"
			}),
		},
		"eviction: safe: OnUpgrade, incoming labels/annotations": {
			eviction: &agonesv1.Eviction{Safe: agonesv1.EvictionSafeOnUpgrade},
			pod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation] = "better not touch"
				pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = "not another one"
			}),
			wantPod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation] = "better not touch"
				pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = "not another one"
			}),
		},
		"eviction: safe: Never, incoming labels/annotations": {
			eviction: &agonesv1.Eviction{Safe: agonesv1.EvictionSafeNever},
			pod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation] = "a passthrough"
				pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = "or is it passthru?"
			}),
			wantPod: emptyPodAnd(func(pod *corev1.Pod) {
				pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation] = "a passthrough"
				pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = "or is it passthru?"
			}),
		},
	} {
		t.Run(desc, func(t *testing.T) {
			assert.NoError(t, SetEviction(tc.eviction, tc.pod))
			assert.Equal(t, tc.wantPod, tc.pod)
		})
	}
}
