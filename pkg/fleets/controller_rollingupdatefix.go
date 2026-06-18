package fleets

import (
	"context"
	"fmt"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/integer"
)

func (c *Controller) cleanupUnhealthyReplicasRollingUpdateFix(ctx context.Context, rest []*agonesv1.GameServerSet,
	fleet *agonesv1.Fleet, maxCleanupCount int32) ([]*agonesv1.GameServerSet, int32, error) {

	// Safely scale down all old GameServerSets with unhealthy replicas.
	totalScaledDown := int32(0)
	for i, gsSet := range rest {
		healthy := gsSet.Status.ReadyReplicas + gsSet.Status.AllocatedReplicas + gsSet.Status.ReservedReplicas

		if totalScaledDown >= maxCleanupCount {
			break
		}
		if gsSet.Spec.Replicas == 0 {
			// cannot scale down this replica set.
			continue
		}
		if gsSet.Spec.Replicas <= healthy {
			// no unhealthy replicas found, no scaling required.
			continue
		}

		scaledDownCount := int32(integer.IntMin(int(maxCleanupCount-totalScaledDown), int(gsSet.Spec.Replicas-healthy)))
		newReplicasCount := gsSet.Spec.Replicas - scaledDownCount
		if newReplicasCount > gsSet.Spec.Replicas {
			return nil, 0, fmt.Errorf("when cleaning up unhealthy replicas, got invalid request to scale down %s/%s %d -> %d", gsSet.Namespace, gsSet.Name, gsSet.Spec.Replicas, newReplicasCount)
		}

		gsSetCopy := gsSet.DeepCopy()
		gsSetCopy.Spec.Replicas = newReplicasCount
		totalScaledDown += scaledDownCount
		if _, err := c.gameServerSetGetter.GameServerSets(gsSetCopy.ObjectMeta.Namespace).Update(ctx, gsSetCopy, metav1.UpdateOptions{}); err != nil {
			return nil, totalScaledDown, errors.Wrapf(err, "error updating gameserverset %s", gsSetCopy.ObjectMeta.Name)
		}
		c.recorder.Eventf(fleet, corev1.EventTypeNormal, "ScalingGameServerSet",
			"Scaling inactive GameServerSet %s from %d to %d", gsSetCopy.ObjectMeta.Name, gsSet.Spec.Replicas, gsSetCopy.Spec.Replicas)

		rest[i] = gsSetCopy
	}
	return rest, totalScaledDown, nil
}

func (c *Controller) rollingUpdateRestFixedOnReadyRollingUpdateFix(ctx context.Context, fleet *agonesv1.Fleet, active *agonesv1.GameServerSet, rest []*agonesv1.GameServerSet) error {
	if len(rest) == 0 {
		return nil
	}

	// Look at Kubernetes Deployment util ResolveFenceposts() function
	r, err := intstr.GetValueFromIntOrPercent(fleet.Spec.Strategy.RollingUpdate.MaxUnavailable, int(fleet.Status.ReadyReplicas), false)
	if err != nil {
		return errors.Wrapf(err, "error parsing MaxUnavailable value: %s", fleet.ObjectMeta.Name)
	}
	if r == 0 {
		r = 1
	}
	if r > int(fleet.Spec.Replicas) {
		r = int(fleet.Spec.Replicas)
	}
	unavailable := int32(r)

	totalAlreadyScaledDown := int32(0)

	totalScaleDownCount := int32(0)
	// Check if we can scale down.
	allGSS := rest
	allGSS = append(allGSS, active)
	readyReplicasCount := agonesv1.SumGameServerSets(allGSS, func(gsSet *agonesv1.GameServerSet) int32 {
		return gsSet.Status.ReadyReplicas
	})
	minAvailable := fleet.Status.ReadyReplicas - unavailable
	if minAvailable > fleet.Spec.Replicas {
		minAvailable = fleet.Spec.Replicas
	}

	// Check if we are ready to scale down
	newGSSUnavailablePodCount := active.Spec.Replicas - active.Status.ReadyReplicas - active.Status.ReservedReplicas -
		active.Status.AllocatedReplicas - active.Status.ShutdownReplicas
	maxScaledDown := readyReplicasCount - minAvailable - newGSSUnavailablePodCount

	if maxScaledDown <= 0 {
		return nil
	}
	rest, _, err = c.cleanupUnhealthyReplicasRollingUpdateFix(ctx, rest, fleet, maxScaledDown)
	if err != nil {
		loggerForFleet(fleet, c.baseLogger).WithField("fleet", fleet.ObjectMeta.Name).WithField("maxScaledDown", maxScaledDown).
			Debug("Can not cleanup Unhealth Replicas")
		// There could be the case when GameServerSet would be updated from another place, say Status or Spec would be updated
		// We don't want to propagate such errors further
		// And this set in sync with reconcileOldReplicaSets() Kubernetes code
		return nil
	}
	// Resulting value is readyReplicasCount + unavailable - fleet.Spec.Replicas
	totalScaleDownCount = readyReplicasCount - minAvailable
	if readyReplicasCount <= minAvailable {
		// Cannot scale down.
		return nil
	}
	for _, gsSet := range rest {
		if totalAlreadyScaledDown >= totalScaleDownCount {
			// No further scaling required.
			break
		}

		// Crucial fix if we are using wrong configuration of a fleet,
		// that would lead to Status.Replicas being 0 but number of GameServers would be in a Scheduled or Unhealthy state.
		// Compare with scaleDownOldReplicaSetsForRollingUpdate() for loop.
		// if the Spec.Replicas are less than or equal to 0, then that means we are done
		// scaling this GameServerSet down, and can therefore exit/move to the next one.
		if gsSet.Spec.Replicas <= 0 {
			continue
		}

		// If the Spec.Replicas does not equal the Status.Replicas for this GameServerSet, this means
		// that the rolling down process is currently ongoing, and we should therefore exit so we can wait for it to finish
		if gsSet.Spec.Replicas != gsSet.Status.Replicas {
			break
		}
		gsSetCopy := gsSet.DeepCopy()
		if gsSet.Status.ShutdownReplicas == 0 {
			// Wait for new GameServers to become Ready before scaling down Inactive GameServerset
			// Scale down.
			scaleDownCount := int32(integer.IntMin(int(gsSet.Spec.Replicas), int(totalScaleDownCount-totalAlreadyScaledDown)))

			newReplicasCount := gsSet.Spec.Replicas - scaleDownCount
			if newReplicasCount > gsSet.Spec.Replicas {
				return fmt.Errorf("when scaling down old GameServerSet, got invalid request to scale down %s/%s %d -> %d", gsSet.Namespace, gsSet.Name, gsSet.Spec.Replicas, newReplicasCount)
			}

			switch {
			case gsSet.Status.Replicas == gsSet.Status.AllocatedReplicas:
				gsSetCopy.Spec.Replicas = 0
			case newReplicasCount == gsSet.Spec.Replicas:
				// No updates on GameServerSet
				continue
			default:
				gsSetCopy.Spec.Replicas = newReplicasCount
			}
			loggerForFleet(fleet, c.baseLogger).WithField("gameserverset", gsSet.ObjectMeta.Name).WithField("replicas", gsSetCopy.Spec.Replicas).
				Debug("applying rolling update to inactive gameserverset")

			if _, err := c.gameServerSetGetter.GameServerSets(gsSetCopy.ObjectMeta.Namespace).Update(ctx, gsSetCopy, metav1.UpdateOptions{}); err != nil {
				return errors.Wrapf(err, "error updating gameserverset %s", gsSetCopy.ObjectMeta.Name)
			}
			c.recorder.Eventf(fleet, corev1.EventTypeNormal, "ScalingGameServerSet",
				"Scaling inactive GameServerSet %s from %d to %d", gsSetCopy.ObjectMeta.Name, gsSet.Spec.Replicas, gsSetCopy.Spec.Replicas)

			totalAlreadyScaledDown += scaleDownCount
		}
	}
	return nil
}
