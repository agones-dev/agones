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

// Package eviction implements a generic SetEviction interface for cloud products
package eviction

import (
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

// SetEviction sets disruptions controls on a Pod based on GameServer.Status.Eviction.
func SetEviction(eviction *agonesv1.Eviction, pod *corev1.Pod) error {
	if eviction == nil {
		return errors.New("No eviction value set. Should be the default value")
	}
	if _, exists := pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation]; !exists {
		switch eviction.Safe {
		case agonesv1.EvictionSafeAlways:
			pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation] = agonesv1.True
		case agonesv1.EvictionSafeOnUpgrade, agonesv1.EvictionSafeNever:
			// For EvictionSafeOnUpgrade and EvictionSafeNever, we block Cluster Autoscaler
			// (on Autopilot, this enables Extended Duration pods, which is equivalent).
			pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation] = agonesv1.False
		default:
			return errors.Errorf("unknown eviction.safe value %q", string(eviction.Safe))
		}
	}
	if _, exists := pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel]; !exists {
		switch eviction.Safe {
		case agonesv1.EvictionSafeAlways, agonesv1.EvictionSafeOnUpgrade:
			// For EvictionSafeAlways and EvictionSafeOnUpgrade, we use a label value
			// that does not match the agones-gameserver-safe-to-evict-false PDB. But
			// we go ahead and label it, in case someone wants to adopt custom logic
			// for this group of game servers.
			pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = agonesv1.True
		case agonesv1.EvictionSafeNever:
			// For EvictionSafeNever, match gones-gameserver-safe-to-evict-false PDB.
			pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = agonesv1.False
		default:
			return errors.Errorf("unknown eviction.safe value %q", string(eviction.Safe))
		}
	}
	return nil
}
