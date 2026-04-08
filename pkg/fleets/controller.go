// Copyright 2018 Google LLC All Rights Reserved.
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

package fleets

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"agones.dev/agones/pkg/apis/agones"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	getterv1 "agones.dev/agones/pkg/client/clientset/versioned/typed/agones/v1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	listerv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	"agones.dev/agones/pkg/util/crd"
	"agones.dev/agones/pkg/util/logfields"
	"agones.dev/agones/pkg/util/runtime"
	"agones.dev/agones/pkg/util/webhooks"
	"agones.dev/agones/pkg/util/workerqueue"
	"github.com/google/go-cmp/cmp"
	"github.com/heptiolabs/healthcheck"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apiextclientv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/integer"
)

// Extensions struct contains what is needed to bind webhook handlers
type Extensions struct {
	baseLogger *logrus.Entry
	apiHooks   agonesv1.APIHooks
}

// Controller is a the GameServerSet controller
type Controller struct {
	baseLogger          *logrus.Entry
	crdGetter           apiextclientv1.CustomResourceDefinitionInterface
	gameServerSetGetter getterv1.GameServerSetsGetter
	gameServerSetLister listerv1.GameServerSetLister
	gameServerSetSynced cache.InformerSynced
	fleetGetter         getterv1.FleetsGetter
	fleetLister         listerv1.FleetLister
	fleetSynced         cache.InformerSynced
	workerqueue         *workerqueue.WorkerQueue
	recorder            record.EventRecorder
}

// NewController returns a new fleets crd controller
func NewController(
	health healthcheck.Handler,
	kubeClient kubernetes.Interface,
	extClient extclientset.Interface,
	agonesClient versioned.Interface,
	agonesInformerFactory externalversions.SharedInformerFactory) *Controller {

	gameServerSets := agonesInformerFactory.Agones().V1().GameServerSets()
	gsSetInformer := gameServerSets.Informer()

	fleets := agonesInformerFactory.Agones().V1().Fleets()
	fInformer := fleets.Informer()

	c := &Controller{
		crdGetter:           extClient.ApiextensionsV1().CustomResourceDefinitions(),
		gameServerSetGetter: agonesClient.AgonesV1(),
		gameServerSetLister: gameServerSets.Lister(),
		gameServerSetSynced: gsSetInformer.HasSynced,
		fleetGetter:         agonesClient.AgonesV1(),
		fleetLister:         fleets.Lister(),
		fleetSynced:         fInformer.HasSynced,
	}

	c.baseLogger = runtime.NewLoggerWithType(c)
	c.workerqueue = workerqueue.NewWorkerQueueWithRateLimiter(c.syncFleet, c.baseLogger, logfields.FleetKey, agones.GroupName+".FleetController", workerqueue.FastRateLimiter(3*time.Second))
	health.AddLivenessCheck("fleet-workerqueue", healthcheck.Check(c.workerqueue.Healthy))

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(c.baseLogger.Debugf)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	c.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "fleet-controller"})

	_, _ = fInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.workerqueue.Enqueue,
		UpdateFunc: func(_, newObj interface{}) {
			c.workerqueue.Enqueue(newObj)
		},
	})

	_, _ = gsSetInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.gameServerSetEventHandler,
		UpdateFunc: func(_, newObj interface{}) {
			gsSet := newObj.(*agonesv1.GameServerSet)
			// ignore if already being deleted
			if gsSet.ObjectMeta.DeletionTimestamp.IsZero() {
				c.gameServerSetEventHandler(gsSet)
			}
		},
	})

	return c
}

// NewExtensions binds the handlers to the webhook outside the initialization of the controller
// initializes a new logger for extensions.
func NewExtensions(apiHooks agonesv1.APIHooks, wh *webhooks.WebHook) *Extensions {
	ext := &Extensions{apiHooks: apiHooks}

	ext.baseLogger = runtime.NewLoggerWithType(ext)

	wh.AddHandler("/mutate", agonesv1.Kind("Fleet"), admissionv1.Create, ext.creationMutationHandler)
	wh.AddHandler("/validate", agonesv1.Kind("Fleet"), admissionv1.Create, ext.creationValidationHandler)
	wh.AddHandler("/validate", agonesv1.Kind("Fleet"), admissionv1.Update, ext.creationValidationHandler)

	return ext
}

// creationMutationHandler is the handler for the mutating webhook that sets the
// the default values on the Fleet
// Should only be called on fleet create operations.
// nolint:dupl
func (ext *Extensions) creationMutationHandler(review admissionv1.AdmissionReview) (admissionv1.AdmissionReview, error) {
	ext.baseLogger.WithField("review", review).Debug("creationMutationHandler")

	obj := review.Request.Object
	fleet := &agonesv1.Fleet{}
	err := json.Unmarshal(obj.Raw, fleet)
	if err != nil {
		// If the JSON is invalid during mutation, fall through to validation. This allows OpenAPI schema validation
		// to proceed, resulting in a more user friendly error message.
		return review, nil
	}

	// This is the main logic of this function
	// the rest is really just json plumbing
	fleet.ApplyDefaults()

	newFleet, err := json.Marshal(fleet)
	if err != nil {
		return review, errors.Wrapf(err, "error marshalling default applied Fleet %s to json", fleet.ObjectMeta.Name)
	}

	patch, err := jsonpatch.CreatePatch(obj.Raw, newFleet)
	if err != nil {
		return review, errors.Wrapf(err, "error creating patch for Fleet %s", fleet.ObjectMeta.Name)
	}

	jsn, err := json.Marshal(patch)
	if err != nil {
		return review, errors.Wrapf(err, "error creating json for patch for Fleet %s", fleet.ObjectMeta.Name)
	}

	loggerForFleet(fleet, ext.baseLogger).WithField("patch", string(jsn)).Debug("patch created!")

	pt := admissionv1.PatchTypeJSONPatch
	review.Response.PatchType = &pt
	review.Response.Patch = jsn

	return review, nil
}

// creationValidationHandler that validates a Fleet when it is created
// Should only be called on Fleet create and Update operations.
func (ext *Extensions) creationValidationHandler(review admissionv1.AdmissionReview) (admissionv1.AdmissionReview, error) {
	ext.baseLogger.WithField("review", review).Debug("creationValidationHandler")

	obj := review.Request.Object
	fleet := &agonesv1.Fleet{}
	err := json.Unmarshal(obj.Raw, fleet)
	if err != nil {
		return review, errors.Wrapf(err, "error unmarshalling Fleet json after schema validation: %s", obj.Raw)
	}

	if errs := fleet.Validate(ext.apiHooks); len(errs) > 0 {
		kind := runtimeschema.GroupKind{
			Group: review.Request.Kind.Group,
			Kind:  review.Request.Kind.Kind,
		}
		statusErr := k8serrors.NewInvalid(kind, review.Request.Name, errs)
		review.Response.Allowed = false
		review.Response.Result = &statusErr.ErrStatus
		loggerForFleet(fleet, ext.baseLogger).WithField("review", review).Debug("Invalid Fleet")
	}

	return review, nil
}

// Run the Fleet controller. Will block until stop is closed.
// Runs threadiness number workers to process the rate limited queue
func (c *Controller) Run(ctx context.Context, workers int) error {
	err := crd.WaitForEstablishedCRD(ctx, c.crdGetter, "fleets.agones.dev", c.baseLogger)
	if err != nil {
		return err
	}

	c.baseLogger.Debug("Wait for cache sync")
	if !cache.WaitForCacheSync(ctx.Done(), c.gameServerSetSynced, c.fleetSynced) {
		return errors.New("failed to wait for caches to sync")
	}

	c.workerqueue.Run(ctx, workers)
	return nil
}

func loggerForFleetKey(key string, logger *logrus.Entry) *logrus.Entry {
	return logfields.AugmentLogEntry(logger, logfields.FleetKey, key)
}

func loggerForFleet(f *agonesv1.Fleet, logger *logrus.Entry) *logrus.Entry {
	fleetName := "NilFleet"
	if f != nil {
		fleetName = f.ObjectMeta.Namespace + "/" + f.ObjectMeta.Name
	}
	return loggerForFleetKey(fleetName, logger).WithField("fleet", f)
}

// gameServerSetEventHandler enqueues the owning Fleet for this GameServerSet,
// assuming that it has one
func (c *Controller) gameServerSetEventHandler(obj interface{}) {
	gsSet := obj.(*agonesv1.GameServerSet)
	ref := metav1.GetControllerOf(gsSet)
	if ref == nil {
		return
	}

	fleet, err := c.fleetLister.Fleets(gsSet.ObjectMeta.Namespace).Get(ref.Name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			c.baseLogger.WithField("ref", ref).Warn("Owner Fleet no longer available for syncing")
		} else {
			runtime.HandleError(loggerForFleet(fleet, c.baseLogger).WithField("ref", ref),
				errors.Wrap(err, "error retrieving GameServerSet owner"))
		}
		return
	}
	c.workerqueue.Enqueue(fleet)
}

// syncFleet synchronised the fleet CRDs and configures/updates
// backing GameServerSets
func (c *Controller) syncFleet(ctx context.Context, key string) error {
	loggerForFleetKey(key, c.baseLogger).Debug("Synchronising")

	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		// don't return an error, as we don't want this retried
		runtime.HandleError(loggerForFleetKey(key, c.baseLogger), errors.Wrapf(err, "invalid resource key"))
		return nil
	}

	fleet, err := c.fleetLister.Fleets(namespace).Get(name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			loggerForFleetKey(key, c.baseLogger).Debug("Fleet is no longer available for syncing")
			return nil
		}
		return errors.Wrapf(err, "error retrieving fleet %s from namespace %s", name, namespace)
	}

	// If Fleet is marked for deletion don't do anything.
	if !fleet.DeletionTimestamp.IsZero() {
		return nil
	}

	gameServerSetNamespacedLister := c.gameServerSetLister.GameServerSets(fleet.ObjectMeta.Namespace)
	list, err := ListGameServerSetsByFleetOwner(gameServerSetNamespacedLister, fleet)
	if err != nil {
		return err
	}

	active, rest := c.filterGameServerSetByActive(fleet, list)

	// if there isn't an active gameServerSet, create one (but don't persist yet)
	if active == nil {
		loggerForFleet(fleet, c.baseLogger).Debug("could not find active GameServerSet, creating")
		active = fleet.GameServerSet()
	}

	replicas, err := c.applyDeploymentStrategy(ctx, fleet, active, rest)
	if err != nil {
		return err
	}
	if err := c.deleteEmptyGameServerSets(ctx, fleet, rest); err != nil {
		return err
	}

	if err := c.upsertGameServerSet(ctx, fleet, active, replicas); err != nil {
		return err
	}
	return c.updateFleetStatus(ctx, fleet)
}

// upsertGameServerSet if the GameServerSet is new, insert it
// if the replicas do not match the active
// GameServerSet, then update it
func (c *Controller) upsertGameServerSet(ctx context.Context, fleet *agonesv1.Fleet, active *agonesv1.GameServerSet, replicas int32) error {
	if active.ObjectMeta.UID == "" {
		active.Spec.Replicas = replicas
		gsSets := c.gameServerSetGetter.GameServerSets(active.ObjectMeta.Namespace)
		gsSet, err := gsSets.Create(ctx, active, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrapf(err, "error creating gameserverset for fleet %s", fleet.ObjectMeta.Name)
		}

		// extra step which is needed to set
		// default values for GameServerSet Status Subresource
		gsSetCopy := gsSet.DeepCopy()
		gsSetCopy.Status.ReadyReplicas = 0
		gsSetCopy.Status.Replicas = 0
		gsSetCopy.Status.AllocatedReplicas = 0
		_, err = gsSets.UpdateStatus(ctx, gsSetCopy, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrapf(err, "error updating status of gameserverset for fleet %s",
				fleet.ObjectMeta.Name)
		}

		c.recorder.Eventf(fleet, corev1.EventTypeNormal, "CreatingGameServerSet",
			"Created GameServerSet %s", gsSet.ObjectMeta.Name)
		return nil
	}

	if replicas != active.Spec.Replicas || active.Spec.Scheduling != fleet.Spec.Scheduling {
		gsSetCopy := active.DeepCopy()
		gsSetCopy.Spec.Replicas = replicas
		gsSetCopy.Spec.Scheduling = fleet.Spec.Scheduling
		gsSetCopy, err := c.gameServerSetGetter.GameServerSets(fleet.ObjectMeta.Namespace).Update(ctx, gsSetCopy, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrapf(err, "error updating replicas for gameserverset for fleet %s", fleet.ObjectMeta.Name)
		}
		c.recorder.Eventf(fleet, corev1.EventTypeNormal, "ScalingGameServerSet",
			"Scaling active GameServerSet %s from %d to %d", gsSetCopy.ObjectMeta.Name, active.Spec.Replicas, gsSetCopy.Spec.Replicas)
	}

	// Update GameServerSet Counts and Lists Priorities if not equal to the Priorities on the Fleet
	if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		if !cmp.Equal(active.Spec.Priorities, fleet.Spec.Priorities) {
			gsSetCopy := active.DeepCopy()
			gsSetCopy.Spec.Priorities = fleet.Spec.Priorities
			_, err := c.gameServerSetGetter.GameServerSets(fleet.ObjectMeta.Namespace).Update(ctx, gsSetCopy, metav1.UpdateOptions{})
			if err != nil {
				return errors.Wrapf(err, "error updating priorities for gameserverset for fleet %s", fleet.ObjectMeta.Name)
			}
			c.recorder.Eventf(fleet, corev1.EventTypeNormal, "UpdatingGameServerSet",
				"Updated GameServerSet %s Priorities", gsSetCopy.ObjectMeta.Name)
		}
	}

	return nil
}

// applyDeploymentStrategy applies the Fleet > Spec > Deployment strategy to all the non-active
// GameServerSets that are passed in
func (c *Controller) applyDeploymentStrategy(ctx context.Context, fleet *agonesv1.Fleet, active *agonesv1.GameServerSet, rest []*agonesv1.GameServerSet) (int32, error) {
	// if there is nothing `rest`, then it's either a brand-new Fleet, or we can just jump to the fleet value,
	// since there is nothing else scaling down at this point
	if len(rest) == 0 {
		return fleet.Spec.Replicas, nil
	}

	// if we do have `rest` but all their spec.replicas is zero, we can just do subtraction against whatever is allocated in `rest`.
	if agonesv1.SumGameServerSets(rest, func(gsSet *agonesv1.GameServerSet) int32 {
		return gsSet.Spec.Replicas
	}) == 0 {
		blocked := agonesv1.SumGameServerSets(rest, func(gsSet *agonesv1.GameServerSet) int32 {
			return gsSet.Status.ReservedReplicas + gsSet.Status.AllocatedReplicas
		})
		replicas := fleet.Spec.Replicas - blocked
		if replicas < 0 {
			replicas = 0
		}
		return replicas, nil
	}

	switch fleet.Spec.Strategy.Type {
	case appsv1.RecreateDeploymentStrategyType:
		return c.recreateDeployment(ctx, fleet, rest)
	case appsv1.RollingUpdateDeploymentStrategyType:
		return c.rollingUpdateDeployment(ctx, fleet, active, rest)
	}

	return 0, errors.Errorf("unexpected deployment strategy type: %s", fleet.Spec.Strategy.Type)
}

// deleteEmptyGameServerSets deletes all GameServerServerSets
// That have `Status > Replicas` of 0
func (c *Controller) deleteEmptyGameServerSets(ctx context.Context, fleet *agonesv1.Fleet, list []*agonesv1.GameServerSet) error {
	p := metav1.DeletePropagationBackground
	for _, gsSet := range list {
		if gsSet.Status.Replicas == 0 && gsSet.Status.ShutdownReplicas == 0 {
			err := c.gameServerSetGetter.GameServerSets(gsSet.ObjectMeta.Namespace).Delete(ctx, gsSet.ObjectMeta.Name, metav1.DeleteOptions{PropagationPolicy: &p})
			if err != nil {
				return errors.Wrapf(err, "error updating gameserverset %s", gsSet.ObjectMeta.Name)
			}

			c.recorder.Eventf(fleet, corev1.EventTypeNormal, "DeletingGameServerSet", "Deleting inactive GameServerSet %s", gsSet.ObjectMeta.Name)
		}
	}

	return nil
}

// recreateDeployment applies the recreate deployment strategy to all non-active
// GameServerSets, and return the replica count for the active GameServerSet
func (c *Controller) recreateDeployment(ctx context.Context, fleet *agonesv1.Fleet, rest []*agonesv1.GameServerSet) (int32, error) {
	for _, gsSet := range rest {
		if gsSet.Spec.Replicas == 0 {
			continue
		}
		loggerForFleet(fleet, c.baseLogger).WithField("gameserverset", gsSet.ObjectMeta.Name).Debug("applying recreate deployment: scaling to 0")
		gsSetCopy := gsSet.DeepCopy()
		gsSetCopy.Spec.Replicas = 0
		if _, err := c.gameServerSetGetter.GameServerSets(gsSetCopy.ObjectMeta.Namespace).Update(ctx, gsSetCopy, metav1.UpdateOptions{}); err != nil {
			return 0, errors.Wrapf(err, "error updating gameserverset %s", gsSetCopy.ObjectMeta.Name)
		}
		c.recorder.Eventf(fleet, corev1.EventTypeNormal, "ScalingGameServerSet",
			"Scaling inactive GameServerSet %s from %d to %d", gsSetCopy.ObjectMeta.Name, gsSet.Spec.Replicas, gsSetCopy.Spec.Replicas)
	}

	return fleet.LowerBoundReplicas(fleet.Spec.Replicas - agonesv1.SumGameServerSets(rest, func(gsSet *agonesv1.GameServerSet) int32 {
		return gsSet.Status.AllocatedReplicas
	})), nil
}

// rollingUpdateDeployment will do the rolling update of the old GameServers
// through to the new ones, based on the fleet.Spec.Strategy.RollingUpdate configuration
// and return the replica count for the active GameServerSet
func (c *Controller) rollingUpdateDeployment(ctx context.Context, fleet *agonesv1.Fleet, active *agonesv1.GameServerSet, rest []*agonesv1.GameServerSet) (int32, error) {
	replicas, err := c.rollingUpdateActive(fleet, active, rest)
	if err != nil {
		return 0, err
	}
	if err := c.rollingUpdateRest(ctx, fleet, active, rest); err != nil {
		return 0, err
	}
	return replicas, nil
}

// rollingUpdateActive applies the rolling update to the active GameServerSet
// and returns what its replica value should be set to
func (c *Controller) rollingUpdateActive(fleet *agonesv1.Fleet, active *agonesv1.GameServerSet, rest []*agonesv1.GameServerSet) (int32, error) {
	replicas := active.Spec.Replicas
	// always leave room for Allocated GameServers
	sumAllocated := agonesv1.SumGameServerSets(rest, func(gsSet *agonesv1.GameServerSet) int32 {
		return gsSet.Status.AllocatedReplicas
	})

	// if the active spec replicas don't equal the active status replicas, this means we are
	// in the middle of a rolling update, and should wait for it to complete.
	if active.Spec.Replicas != active.Status.Replicas {
		return replicas, nil
	}

	// if the current number replicas from the fleet is zero, the rolling update can be ignored
	// and the cleanup stage will remove dangling GameServerSets
	if fleet.Spec.Replicas == 0 {
		return 0, nil
	}

	// if the active spec replicas are greater than or equal the fleet spec replicas, then we don't
	// need to do another rolling update upwards.
	if active.Spec.Replicas >= (fleet.Spec.Replicas - sumAllocated) {
		return fleet.Spec.Replicas - sumAllocated, nil
	}

	r, err := intstr.GetValueFromIntOrPercent(fleet.Spec.Strategy.RollingUpdate.MaxSurge, int(fleet.Spec.Replicas), true)
	if err != nil {
		return 0, errors.Wrapf(err, "error parsing MaxSurge value: %s", fleet.ObjectMeta.Name)
	}
	surge := int32(r)

	// make sure we don't end up with more than the configured max surge
	maxSurge := surge + fleet.Spec.Replicas
	replicas = fleet.UpperBoundReplicas(replicas + surge)
	total := agonesv1.SumGameServerSets(rest, func(gsSet *agonesv1.GameServerSet) int32 {
		return gsSet.Status.Replicas
	}) + replicas
	if total > maxSurge {
		replicas = fleet.LowerBoundReplicas(replicas - (total - maxSurge))
	}

	// make room for allocated game servers, but not over the fleet replica count
	if replicas+sumAllocated > fleet.Spec.Replicas {
		replicas = fleet.LowerBoundReplicas(fleet.Spec.Replicas - sumAllocated)
	}

	loggerForFleet(fleet, c.baseLogger).WithField("gameserverset", active.ObjectMeta.Name).WithField("replicas", replicas).
		Debug("applying rolling update to active gameserverset")

	return replicas, nil
}

func (c *Controller) cleanupUnhealthyReplicas(ctx context.Context, rest []*agonesv1.GameServerSet,
	fleet *agonesv1.Fleet, maxCleanupCount int32) ([]*agonesv1.GameServerSet, int32, error) {

	// Safely scale down all old GameServerSets with unhealthy replicas.
	totalScaledDown := int32(0)
	for i, gsSet := range rest {
		if totalScaledDown >= maxCleanupCount {
			break
		}
		if gsSet.Spec.Replicas == 0 {
			// cannot scale down this replica set.
			continue
		}
		if gsSet.Spec.Replicas == gsSet.Status.ReadyReplicas {
			// no unhealthy replicas found, no scaling required.
			continue
		}

		scaledDownCount := int32(integer.IntMin(int(maxCleanupCount-totalScaledDown), int(gsSet.Spec.Replicas-gsSet.Status.ReadyReplicas)))
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

func (c *Controller) rollingUpdateRestFixedOnReady(ctx context.Context, fleet *agonesv1.Fleet, active *agonesv1.GameServerSet, rest []*agonesv1.GameServerSet) error {
	if len(rest) == 0 {
		return nil
	}
	if runtime.FeatureEnabled(runtime.FeatureRollingUpdateFix) {
		return c.rollingUpdateRestFixedOnReadyRollingUpdateFix(ctx, fleet, active, rest)
	}

	// Look at Kubernetes Deployment util ResolveFenceposts() function
	r, err := intstr.GetValueFromIntOrPercent(fleet.Spec.Strategy.RollingUpdate.MaxUnavailable, int(fleet.Spec.Replicas), false)
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
	minAvailable := fleet.Spec.Replicas - unavailable

	// Check if we are ready to scale down
	allPodsCount := agonesv1.SumGameServerSets(allGSS, func(gsSet *agonesv1.GameServerSet) int32 {
		return gsSet.Spec.Replicas
	})
	newGSSUnavailablePodCount := active.Spec.Replicas - active.Status.ReadyReplicas - active.Status.AllocatedReplicas
	maxScaledDown := allPodsCount - minAvailable - newGSSUnavailablePodCount

	if maxScaledDown <= 0 {
		return nil
	}
	rest, _, err = c.cleanupUnhealthyReplicas(ctx, rest, fleet, maxScaledDown)
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

			// No updates on GameServerSet
			if newReplicasCount == gsSet.Spec.Replicas {
				continue
			}

			gsSetCopy.Spec.Replicas = newReplicasCount
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

// rollingUpdateRest applies the rolling update to the inactive GameServerSets
func (c *Controller) rollingUpdateRest(ctx context.Context, fleet *agonesv1.Fleet, active *agonesv1.GameServerSet, rest []*agonesv1.GameServerSet) error {
	return c.rollingUpdateRestFixedOnReady(ctx, fleet, active, rest)
}

// updateFleetStatus gets the GameServerSets for this Fleet and then
// calculates the counts for the status, and updates the Fleet
func (c *Controller) updateFleetStatus(ctx context.Context, fleet *agonesv1.Fleet) error {
	loggerForFleet(fleet, c.baseLogger).Debug("Update Fleet Status")

	gameServerSetNamespacedLister := c.gameServerSetLister.GameServerSets(fleet.ObjectMeta.Namespace)
	list, err := ListGameServerSetsByFleetOwner(gameServerSetNamespacedLister, fleet)
	if err != nil {
		return err
	}

	fCopy, err := c.fleetGetter.Fleets(fleet.ObjectMeta.Namespace).Get(ctx, fleet.ObjectMeta.GetName(), metav1.GetOptions{})
	if err != nil {
		return err
	}
	fCopy.Status.Replicas = 0
	fCopy.Status.ReadyReplicas = 0
	fCopy.Status.ReservedReplicas = 0
	fCopy.Status.AllocatedReplicas = 0
	if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		fCopy.Status.Counters = make(map[string]agonesv1.AggregatedCounterStatus)
		fCopy.Status.Lists = make(map[string]agonesv1.AggregatedListStatus)
	}
	// Drop Counters and Lists status if the feature flag has been set to false
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		if len(fCopy.Status.Counters) != 0 || len(fCopy.Status.Lists) != 0 {
			fCopy.Status.Counters = map[string]agonesv1.AggregatedCounterStatus{}
			fCopy.Status.Lists = map[string]agonesv1.AggregatedListStatus{}
		}
	}

	for _, gsSet := range list {
		fCopy.Status.Replicas += gsSet.Status.Replicas
		fCopy.Status.ReadyReplicas += gsSet.Status.ReadyReplicas
		fCopy.Status.ReservedReplicas += gsSet.Status.ReservedReplicas
		fCopy.Status.AllocatedReplicas += gsSet.Status.AllocatedReplicas
		if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
			fCopy.Status.Counters = mergeCounters(fCopy.Status.Counters, gsSet.Status.Counters)
			fCopy.Status.Lists = mergeLists(fCopy.Status.Lists, gsSet.Status.Lists)
		}
	}
	if runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		// to make this code simpler, while the feature gate is in place,
		// we will loop around the gsSet list twice.
		fCopy.Status.Players = &agonesv1.AggregatedPlayerStatus{}
		// TODO: integrate this extra loop into the above for loop when PlayerTracking moves to GA
		for _, gsSet := range list {
			if gsSet.Status.Players != nil {
				fCopy.Status.Players.Count += gsSet.Status.Players.Count
				fCopy.Status.Players.Capacity += gsSet.Status.Players.Capacity
			}
		}
	}

	_, err = c.fleetGetter.Fleets(fCopy.ObjectMeta.Namespace).UpdateStatus(ctx, fCopy, metav1.UpdateOptions{})
	return errors.Wrapf(err, "error updating status of fleet %s", fCopy.ObjectMeta.Name)
}

// filterGameServerSetByActive returns the active GameServerSet (or nil if it
// doesn't exist) and then the rest of the GameServerSets that are controlled
// by this Fleet.
// If multiple GameServerSets match the Fleet's template (e.g. due to informer
// cache lag during creation), the oldest one by CreationTimestamp is kept as
// active and the duplicates are placed in rest so they can be scaled down and
// cleaned up by the normal deployment strategy.
func (c *Controller) filterGameServerSetByActive(fleet *agonesv1.Fleet, list []*agonesv1.GameServerSet) (*agonesv1.GameServerSet, []*agonesv1.GameServerSet) {
	var active *agonesv1.GameServerSet
	var rest []*agonesv1.GameServerSet

	for _, gsSet := range list {
		if apiequality.Semantic.DeepEqual(gsSet.Spec.Template, fleet.Spec.Template) {
			if active == nil {
				active = gsSet
			} else {
				// Multiple GameServerSets match the fleet template.
				// Keep the oldest as active; treat the newer duplicate as rest.
				if gsSet.ObjectMeta.CreationTimestamp.Before(&active.ObjectMeta.CreationTimestamp) {
					rest = append(rest, active)
					active = gsSet
				} else {
					rest = append(rest, gsSet)
				}
			}
		} else {
			rest = append(rest, gsSet)
		}
	}

	return active, rest
}

// mergeCounters adds the contents of AggregatedCounterStatus c2 into c1.
func mergeCounters(c1, c2 map[string]agonesv1.AggregatedCounterStatus) map[string]agonesv1.AggregatedCounterStatus {
	if c1 == nil {
		c1 = make(map[string]agonesv1.AggregatedCounterStatus)
	}

	for key, val := range c2 {
		// If the Counter exists in both maps, aggregate the values.
		if counter, ok := c1[key]; ok {
			counter.AllocatedCapacity = agonesv1.SafeAdd(counter.AllocatedCapacity, val.AllocatedCapacity)
			counter.AllocatedCount = agonesv1.SafeAdd(counter.AllocatedCount, val.AllocatedCount)
			counter.Capacity = agonesv1.SafeAdd(counter.Capacity, val.Capacity)
			counter.Count = agonesv1.SafeAdd(counter.Count, val.Count)
			c1[key] = counter
		} else {
			c1[key] = *val.DeepCopy()
		}
	}

	return c1
}

// mergeLists adds the contents of AggregatedListStatus l2 into l1.
func mergeLists(l1, l2 map[string]agonesv1.AggregatedListStatus) map[string]agonesv1.AggregatedListStatus {
	if l1 == nil {
		l1 = make(map[string]agonesv1.AggregatedListStatus)
	}

	for key, val := range l2 {
		// If the List exists in both maps, aggregate the values.
		if list, ok := l1[key]; ok {
			list.AllocatedCapacity += val.AllocatedCapacity
			list.AllocatedCount += val.AllocatedCount
			list.Capacity += val.Capacity
			list.Count += val.Count
			l1[key] = list
		} else {
			l1[key] = *val.DeepCopy()
		}
	}

	return l1
}
