/*
 * Copyright Contributors to Agones a Series of LF Projects, LLC.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package fleetautoscalers

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	extism "github.com/extism/go-sdk"
	"github.com/pkg/errors"
	"github.com/robfig/cron/v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	autoscalingv1 "agones.dev/agones/pkg/apis/autoscaling/v1"
	listeragonesv1 "agones.dev/agones/pkg/client/listers/agones/v1"
	"agones.dev/agones/pkg/fleets"
	"agones.dev/agones/pkg/gameservers"
	gssets "agones.dev/agones/pkg/gameserversets"
	"agones.dev/agones/pkg/util/runtime"
)

const (
	maxDuration = "2540400h" // 290 Years
)

// InactiveScheduleError denotes an error for schedules that are not currently active.
type InactiveScheduleError struct{}

func (InactiveScheduleError) Error() string {
	return "inactive schedule, policy not applicable"
}

// computeDesiredFleetSize computes the new desired size of the given fleet
func computeDesiredFleetSize(ctx context.Context, state *fasState, pol autoscalingv1.FleetAutoscalerPolicy, f *agonesv1.Fleet,
	gameServerNamespacedLister listeragonesv1.GameServerNamespaceLister, nodeCounts map[string]gameservers.NodeCount, fasLog *FasLogger) (int32, bool, error) {

	var (
		replicas int32
		limited  bool
		err      error
	)

	switch pol.Type {
	case autoscalingv1.BufferPolicyType:
		replicas, limited, err = applyBufferPolicy(state, pol.Buffer, f, fasLog)
	case autoscalingv1.WebhookPolicyType:
		replicas, limited, err = applyWebhookPolicy(state, pol.Webhook, f, fasLog)
	case autoscalingv1.CounterPolicyType:
		replicas, limited, err = applyCounterOrListPolicyWrapper(state, pol.Counter, nil, f, gameServerNamespacedLister, nodeCounts, fasLog)
	case autoscalingv1.ListPolicyType:
		replicas, limited, err = applyCounterOrListPolicyWrapper(state, nil, pol.List, f, gameServerNamespacedLister, nodeCounts, fasLog)
	case autoscalingv1.SchedulePolicyType:
		replicas, limited, err = applySchedulePolicy(ctx, state, pol.Schedule, f, gameServerNamespacedLister, nodeCounts, time.Now(), fasLog)
	case autoscalingv1.ChainPolicyType:
		replicas, limited, err = applyChainPolicy(ctx, state, pol.Chain, f, gameServerNamespacedLister, nodeCounts, time.Now(), fasLog)
	case autoscalingv1.WasmPolicyType:
		replicas, limited, err = applyWasmPolicy(ctx, state, pol.Wasm, f, fasLog)

	default:
		err = errors.New("wrong policy type, should be one of: Buffer, Webhook, Counter, List, Schedule, Chain")
	}

	if err != nil && !errors.Is(err, InactiveScheduleError{}) {
		loggerForFleetAutoscalerKey(fasLog.fas.ObjectMeta.Name, fasLog.baseLogger).
			Debugf("Failed to apply policy type %q: %v", pol.Type, err)
	}

	return replicas, limited, err
}

func applyWasmPolicy(ctx context.Context, state *fasState, wp *autoscalingv1.WasmPolicy, f *agonesv1.Fleet, log *FasLogger) (int32, bool, error) {
	if !runtime.FeatureEnabled(runtime.FeatureWasmAutoscaler) {
		return 0, false, errors.Errorf("cannot apply WasmPolicy unless feature flag %s is enabled", runtime.FeatureWasmAutoscaler)
	}

	if wp == nil {
		return 0, false, errors.New("wasmPolicy parameter must not be nil")
	}

	if f == nil {
		return 0, false, errors.New("fleet parameter must not be nil")
	}

	if state.wasmPlugin == nil {
		// Build URL from the WasmPolicy
		u, err := buildURLFromConfiguration(state, wp.From.URL)
		if err != nil {
			return 0, false, err
		}

		if state.httpClient == nil {
			return 0, false, errors.New("http client not set")
		}

		res, err := state.httpClient.Get(u.String())
		if err != nil {
			return 0, false, errors.Wrapf(err, "failed to fetch Wasm module from %s", u.String())
		}
		defer res.Body.Close() //nolint:errcheck

		if res.StatusCode != http.StatusOK {
			return 0, false, fmt.Errorf("bad status code %d from the server: %s", res.StatusCode, u.String())
		}

		b, err := io.ReadAll(res.Body)
		if err != nil {
			return 0, false, errors.Wrapf(err, "failed to read Wasm module from %s", u.String())
		}

		data := extism.WasmData{Data: b}
		if len(wp.Hash) > 0 {
			data.Hash = wp.Hash
		}
		manifest := extism.Manifest{
			Wasm: []extism.Wasm{
				data,
			},
			Config: wp.Config,
		}

		config := extism.PluginConfig{
			EnableWasi: true,
		}
		plugin, err := extism.NewPlugin(ctx, manifest, config, []extism.HostFunction{})
		if err != nil {
			return 0, false, errors.Wrapf(err, "failed to create Wasm plugin from %s", u.String())
		}
		state.wasmPlugin = plugin // Store the plugin in the state map
	}

	// Create FleetAutoscaleReview
	review := autoscalingv1.FleetAutoscaleReview{
		Request: &autoscalingv1.FleetAutoscaleRequest{
			UID:       uuid.NewUUID(),
			Name:      f.Name,
			Namespace: f.Namespace,
			Status:    f.Status,
		},
		Response: nil,
	}

	if runtime.FeatureEnabled(runtime.FeatureFleetAutoscaleRequestMetaData) {
		review.Request.Annotations = f.ObjectMeta.Annotations
		review.Request.Labels = f.ObjectMeta.Labels
	}

	b, err := json.Marshal(review)
	if err != nil {
		return 0, false, errors.Wrap(err, "failed to marshal autoscaling request")
	}

	_, b, err = state.wasmPlugin.CallWithContext(ctx, wp.Function, b)
	if err != nil {
		return 0, false, errors.Wrapf(err, "failed to call Wasm plugin function %s", wp.Function)
	}

	if err := json.Unmarshal(b, &review); err != nil {
		return 0, false, errors.Wrap(err, "failed to unmarshal autoscaling response")
	}

	loggerForFleetAutoscalerKey(log.fas.ObjectMeta.Name, log.baseLogger).Debugf(
		"Fleet Autoscaler operation completed for fleet: %s, with was function: %s", f.ObjectMeta.Name, wp.Function)

	if review.Response.Scale {
		return review.Response.Replicas, false, nil
	}

	return f.Status.Replicas, false, nil
}

// buildURLFromConfiguration - build URL for Webhook and set CARoot for client Transport
func buildURLFromConfiguration(state *fasState, w *autoscalingv1.URLConfiguration) (u *url.URL, err error) {
	if w.URL != nil && w.Service != nil {
		return nil, errors.New("service and URL cannot be used simultaneously")
	}

	// if we haven't created the http state yet, let's create the http client, with appropriate tls configuration.
	if state.httpClient == nil {
		config := &tls.Config{}
		state.httpClient = &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: config,
			},
		}

		if w.CABundle != nil {
			if err := setCABundle(config, w.CABundle); err != nil {
				return nil, err
			}
		}
	}

	scheme := "http"
	if w.CABundle != nil {
		scheme = "https"
	}

	if w.URL != nil {
		if *w.URL == "" {
			return nil, errors.New("URL was not provided")
		}

		return url.ParseRequestURI(*w.URL)
	}

	if w.Service == nil {
		return nil, errors.New("service was not provided, either URL or Service must be provided")
	}

	if w.Service.Name == "" {
		return nil, errors.New("service name was not provided")
	}

	if w.Service.Path == nil {
		empty := ""
		w.Service.Path = &empty
	}

	if w.Service.Namespace == "" {
		w.Service.Namespace = "default"
	}

	return createURL(scheme, w.Service.Name, w.Service.Namespace, *w.Service.Path, w.Service.Port), nil
}

// moved to a separate method to cover it with unit tests and check that URL corresponds to a proper pattern
func createURL(scheme, name, namespace, path string, port *int32) *url.URL {
	var hostPort int32 = 8000
	if port != nil {
		hostPort = *port
	}

	return &url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s.%s.svc:%d", name, namespace, hostPort),
		Path:   path,
	}
}

func setCABundle(tls *tls.Config, caBundle []byte) error {
	rootCAs := x509.NewCertPool()
	if ok := rootCAs.AppendCertsFromPEM(caBundle); !ok {
		return errors.New("no certs were appended from caBundle")
	}
	tls.RootCAs = rootCAs
	return nil
}

func applyWebhookPolicy(state *fasState, w *autoscalingv1.URLConfiguration, f *agonesv1.Fleet, fasLog *FasLogger) (replicas int32, limited bool, err error) {
	if w == nil {
		return 0, false, errors.New("webhookPolicy parameter must not be nil")
	}

	if f == nil {
		return 0, false, errors.New("fleet parameter must not be nil")
	}

	u, err := buildURLFromConfiguration(state, w)
	if err != nil {
		return 0, false, err
	}
	if state.httpClient == nil {
		return 0, false, errors.New("http client not set")
	}

	faReq := autoscalingv1.FleetAutoscaleReview{
		Request: &autoscalingv1.FleetAutoscaleRequest{
			UID:       uuid.NewUUID(),
			Name:      f.Name,
			Namespace: f.Namespace,
			Status:    f.Status,
		},
		Response: nil,
	}

	if runtime.FeatureEnabled(runtime.FeatureFleetAutoscaleRequestMetaData) {
		faReq.Request.Annotations = f.ObjectMeta.Annotations
		faReq.Request.Labels = f.ObjectMeta.Labels
	}

	b, err := json.Marshal(faReq)
	if err != nil {
		return 0, false, err
	}

	res, err := state.httpClient.Post(
		u.String(),
		"application/json",
		strings.NewReader(string(b)),
	)
	if err != nil {
		return 0, false, err
	}
	defer func() {
		// Drain any unread body so the underlying connection can be reused by http.Transport.
		_, _ = io.Copy(io.Discard, res.Body)
		if cerr := res.Body.Close(); cerr != nil {
			if err != nil {
				err = errors.Wrap(err, cerr.Error())
			} else {
				err = cerr
			}
		}
	}()

	if res.StatusCode != http.StatusOK {
		return 0, false, fmt.Errorf("bad status code %d from the server: %s", res.StatusCode, u.String())
	}
	result, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, false, err
	}

	var faResp autoscalingv1.FleetAutoscaleReview
	err = json.Unmarshal(result, &faResp)
	if err != nil {
		return 0, false, err
	}

	// Log Fleet Autoscaler operation, handling nil or empty Name in one line
	webhookPolicyName := "<nil>"
	if w.Service != nil && w.Service.Name != "" {
		webhookPolicyName = w.Service.Name
	}
	loggerForFleetAutoscalerKey(fasLog.fas.ObjectMeta.Name, fasLog.baseLogger).Debugf(
		"Fleet Autoscaler operation completed for fleet: %s, with WebhookPolicy: %s", f.ObjectMeta.Name, webhookPolicyName)

	if faResp.Response.Scale {
		return faResp.Response.Replicas, false, nil
	}

	return f.Status.Replicas, false, nil
}

func applyBufferPolicy(_ *fasState, b *autoscalingv1.BufferPolicy, f *agonesv1.Fleet, fasLog *FasLogger) (int32, bool, error) {
	var replicas int32

	if b.BufferSize.Type == intstr.Int {
		replicas = f.Status.AllocatedReplicas + int32(b.BufferSize.IntValue())
	} else {
		// the percentage value is a little more complex, as we can't apply
		// the desired percentage to any current value, but to the future one
		// Example: we have 8 allocated replicas, 10 total replicas and bufferSize set to 30%.
		// 30% means that we must have 30% ready instances in the fleet
		// Right now there are 20%, so we must increase the fleet until we reach 30%
		// To compute the new size, we start from the other end: if ready must be 30%
		// it means that allocated must be 70% and adjust the fleet size to make that true.
		bufferPercent, err := intstr.GetValueFromIntOrPercent(&b.BufferSize, 100, true)
		if err != nil {
			return 0, false, err
		}
		// use Math.Ceil to round the result up
		replicas = int32(math.Ceil(float64(f.Status.AllocatedReplicas*100) / float64(100-bufferPercent)))
	}

	scalingInLimited := false
	scalingOutLimited := false

	if replicas < b.MinReplicas {
		replicas = b.MinReplicas
		scalingInLimited = true
	}
	if replicas > b.MaxReplicas {
		replicas = b.MaxReplicas
		scalingOutLimited = true
	}

	loggerForFleetAutoscalerKey(fasLog.fas.ObjectMeta.Name, fasLog.baseLogger).Debugf(
		"Fleet Autoscaler operation completed for fleet: %s, with BufferPolicy: %v", f.ObjectMeta.Name, b.BufferSize)

	return replicas, scalingInLimited || scalingOutLimited, nil
}

// New function to call applyCounterOrListPolicy
func applyCounterOrListPolicyWrapper(_ *fasState, c *autoscalingv1.CounterPolicy, l *autoscalingv1.ListPolicy,
	f *agonesv1.Fleet, gameServerNamespacedLister listeragonesv1.GameServerNamespaceLister,
	nodeCounts map[string]gameservers.NodeCount, fasLog *FasLogger) (int32, bool, error) {

	// Call applyCounterOrListPolicy inside the wrapper
	desiredReplicas, scalingLimited, err := applyCounterOrListPolicy(c, l, f, gameServerNamespacedLister, nodeCounts)

	if err == nil {
		// Log directly based on which policy is used, with a description of the key
		if c != nil {
			// Log the Key from CounterPolicy with a description
			loggerForFleetAutoscalerKey(fasLog.fas.ObjectMeta.Name, fasLog.baseLogger).Debugf(
				"Fleet Autoscaler operation completed for fleet: %s, with CounterPolicy - Key: %v", f.ObjectMeta.Name, c.Key)
		} else if l != nil {
			// Log the Key from ListPolicy with a description
			loggerForFleetAutoscalerKey(fasLog.fas.ObjectMeta.Name, fasLog.baseLogger).Debugf(
				"Fleet Autoscaler operation completed for fleet: %s, with ListPolicy - Key: %v", f.ObjectMeta.Name, l.Key)
		}
	}

	return desiredReplicas, scalingLimited, err
}

func applyCounterOrListPolicy(c *autoscalingv1.CounterPolicy, l *autoscalingv1.ListPolicy,
	f *agonesv1.Fleet, gameServerNamespacedLister listeragonesv1.GameServerNamespaceLister,
	nodeCounts map[string]gameservers.NodeCount) (int32, bool, error) {

	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		return 0, false, errors.Errorf("cannot apply CounterPolicy unless feature flag %s is enabled", runtime.FeatureCountsAndLists)
	}

	var isCounter bool          // True if a CounterPolicy False if a ListPolicy
	var key string              // The specified Counter or List
	var count int64             // The Count or number of Values in the template Game Server
	var capacity int64          // The Capacity in the template Game Server
	var aggCount int64          // The Aggregate Count of the specified Counter or List of all GameServers across the GameServerSet in the Fleet
	var aggCapacity int64       // The Aggregate Capacity of the specified Counter or List of all GameServers across the GameServerSet in the Fleet
	var aggAllocatedCount int64 // The Aggregate Count of the specified Counter or List of GameServers in an Allocated state across the GameServerSet in the Fleet
	var minCapacity int64       // The Minimum Aggregate Capacity
	var maxCapacity int64       // The Maximum Aggregate Capacity
	var bufferSize intstr.IntOrString

	if c != nil {
		isCounter = true
		counter, ok := f.Spec.Template.Spec.Counters[c.Key]
		if !ok {
			return 0, false, errors.Errorf("cannot apply CounterPolicy as Counter key %s does not exist in the Fleet Spec", c.Key)
		}

		aggCounter, ok := f.Status.Counters[c.Key]
		if !ok {
			return 0, false, errors.Errorf("cannot apply CounterPolicy as Counter key %s does not exist in the Fleet Status", c.Key)
		}

		key = c.Key
		count = counter.Count
		capacity = counter.Capacity
		aggCount = aggCounter.Count
		aggCapacity = aggCounter.Capacity
		aggAllocatedCount = aggCounter.AllocatedCount
		minCapacity = c.MinCapacity
		maxCapacity = c.MaxCapacity
		bufferSize = c.BufferSize

	} else {
		isCounter = false
		list, ok := f.Spec.Template.Spec.Lists[l.Key]
		if !ok {
			return 0, false, errors.Errorf("cannot apply ListPolicy as List key %s does not exist in the Fleet Spec", l.Key)
		}

		aggList, ok := f.Status.Lists[l.Key]
		if !ok {
			return 0, false, errors.Errorf("cannot apply ListPolicy as List key %s does not exist in the Fleet Status", l.Key)
		}

		key = l.Key
		count = int64(len(list.Values))
		capacity = list.Capacity
		aggCount = aggList.Count
		aggCapacity = aggList.Capacity
		aggAllocatedCount = aggList.AllocatedCount
		minCapacity = l.MinCapacity
		maxCapacity = l.MaxCapacity
		bufferSize = l.BufferSize
	}

	// Checks if we've limited by TOTAL capacity
	limited, scale := isLimited(aggCapacity, minCapacity, maxCapacity)

	// Total current number of Replicas
	replicas := f.Status.Replicas

	// The buffer is the desired available capacity
	var buffer int64

	switch {
	// Desired replicas based on BufferSize specified as an absolute value (i.e. 5)
	case bufferSize.Type == intstr.Int:
		buffer = int64(bufferSize.IntValue())
	// Desired replicas based on BufferSize specified as a percent (i.e. 5%)
	case bufferSize.Type == intstr.String:
		bufferPercent, err := intstr.GetValueFromIntOrPercent(&bufferSize, 100, isCounter)
		if err != nil {
			return 0, false, err
		}
		// If the Aggregated Allocated Counts is 0 then desired capacity gets calculated as 0. If the
		// capacity of 1 replica is equal to or greater than minimum capacity we can exit early.
		if aggAllocatedCount <= 0 && capacity >= minCapacity {
			return 1, true, nil
		}

		// The desired TOTAL capacity based on the Aggregated Allocated Counts (see applyBufferPolicy for explanation)
		desiredCapacity := int64(math.Ceil(float64(aggAllocatedCount*100) / float64(100-bufferPercent)))
		// Convert into a desired AVAILABLE capacity aka the buffer
		buffer = desiredCapacity - aggAllocatedCount
	}

	// Current available capacity across the TOTAL fleet
	switch availableCapacity := aggCapacity - aggCount; {
	case availableCapacity == buffer:
		if limited {
			return scaleLimited(scale, f, gameServerNamespacedLister, nodeCounts, key, isCounter, replicas,
				capacity, aggCapacity, minCapacity, maxCapacity)
		}
		return replicas, false, nil
	case availableCapacity < buffer: // Scale Up
		if limited { // Case where we want to scale up, but we're already limited by MaxCapacity.
			return scaleLimited(scale, f, gameServerNamespacedLister, nodeCounts, key, isCounter, replicas,
				capacity, aggCapacity, minCapacity, maxCapacity)
		}
		return scaleUp(replicas, capacity, count, aggCapacity, availableCapacity, maxCapacity,
			minCapacity, buffer)
	case availableCapacity > buffer: // Scale Down
		if limited && scale == 1 { // Case where we want to scale down but we're already limited by MinCapacity
			return scaleLimited(scale, f, gameServerNamespacedLister, nodeCounts, key, isCounter, replicas,
				capacity, aggCapacity, minCapacity, maxCapacity)
		}
		return scaleDown(f, gameServerNamespacedLister, nodeCounts, key, isCounter, replicas, aggCount,
			aggCapacity, minCapacity, buffer)
	}

	if isCounter {
		return 0, false, errors.Errorf("unable to apply CounterPolicy %v", c)
	}
	return 0, false, errors.Errorf("unable to apply ListPolicy %v", l)
}

func applySchedulePolicy(ctx context.Context, state *fasState, s *autoscalingv1.SchedulePolicy, f *agonesv1.Fleet, gameServerNamespacedLister listeragonesv1.GameServerNamespaceLister, nodeCounts map[string]gameservers.NodeCount, currentTime time.Time, fasLog *FasLogger) (int32, bool, error) {
	// Ensure the scheduled autoscaler feature gate is enabled
	if !runtime.FeatureEnabled(runtime.FeatureScheduledAutoscaler) {
		return 0, false, errors.Errorf("cannot apply SchedulePolicy unless feature flag %s is enabled", runtime.FeatureScheduledAutoscaler)
	}

	if isScheduleActive(s, currentTime) {
		return computeDesiredFleetSize(ctx, state, s.Policy, f, gameServerNamespacedLister, nodeCounts, fasLog)
	}

	loggerForFleetAutoscalerKey(fasLog.fas.ObjectMeta.Name, fasLog.baseLogger).Debugf(
		"Fleet autoscaler check: Schedule not active for fleet %s", f.ObjectMeta.Name)

	// If the schedule wasn't active then return the current replica amount of the fleet
	return f.Status.Replicas, false, &InactiveScheduleError{}
}

func applyChainPolicy(ctx context.Context, state *fasState, c autoscalingv1.ChainPolicy, f *agonesv1.Fleet, gameServerNamespacedLister listeragonesv1.GameServerNamespaceLister, nodeCounts map[string]gameservers.NodeCount, currentTime time.Time, fasLog *FasLogger) (int32, bool, error) {
	// Ensure the scheduled autoscaler feature gate is enabled
	if !runtime.FeatureEnabled(runtime.FeatureScheduledAutoscaler) {
		return 0, false, errors.Errorf("cannot apply ChainPolicy unless feature flag %s is enabled", runtime.FeatureScheduledAutoscaler)
	}

	replicas := f.Status.Replicas
	var limited bool
	var err error
	var chainEntry autoscalingv1.FleetAutoscalerPolicyType

	// Loop over all entries in the chain
	for _, entry := range c {
		switch entry.Type {
		case autoscalingv1.SchedulePolicyType:
			replicas, limited, err = applySchedulePolicy(ctx, state, entry.Schedule, f, gameServerNamespacedLister, nodeCounts, currentTime, fasLog)

			if err != nil {
				loggerForFleetAutoscalerKey(fasLog.fas.ObjectMeta.Name, fasLog.baseLogger).Debugf(
					"Failed to apply SchedulePolicy ID=%s in ChainPolicy: %v", entry.ID, err)
			}
		case autoscalingv1.WebhookPolicyType:
			replicas, limited, err = applyWebhookPolicy(state, entry.Webhook, f, fasLog)

			if err != nil {
				loggerForFleetAutoscalerKey(fasLog.fas.ObjectMeta.Name, fasLog.baseLogger).Debugf(
					"Failed to apply WebhookPolicy ID=%s in ChainPolicy: %v", entry.ID, err)
			}
		default:
			// Every other policy type we just want to compute the desired fleet and return it
			replicas, limited, err = computeDesiredFleetSize(ctx, state, entry.FleetAutoscalerPolicy, f, gameServerNamespacedLister, nodeCounts, fasLog)

			if err != nil && !errors.Is(err, InactiveScheduleError{}) {
				loggerForFleetAutoscalerKey(fasLog.fas.ObjectMeta.Name, fasLog.baseLogger).Debugf(
					"Failed to apply %s ID=%s in ChainPolicy: %v", entry.Type, entry.ID, err)
			}
		}

		// If no error occurred, exit the loop early
		if err == nil {
			chainEntry = autoscalingv1.FleetAutoscalerPolicyType(fmt.Sprintf("%s:%s:%s", autoscalingv1.ChainPolicyType, entry.ID, entry.Type))
			break
		}
	}

	if err != nil && !errors.Is(err, InactiveScheduleError{}) {
		emitChainPolicyEvent(fasLog, "Unknown", "")
		loggerForFleetAutoscalerKey(fasLog.fas.ObjectMeta.Name, fasLog.baseLogger).Debug("Failed to apply ChainPolicy: no valid policy applied")
		return replicas, limited, err
	}

	currChainEntry := strings.Split(string(chainEntry), ":")

	// Handle the final state of the chain and update status if necessary
	if lastAppliedPolicy := fasLog.fas.Status.LastAppliedPolicy; strings.Contains(string(lastAppliedPolicy), string(autoscalingv1.ChainPolicyType)) {
		prevChainEntry := strings.Split(string(lastAppliedPolicy), ":")

		// Only log if there is a change in the policy
		if len(prevChainEntry) > 2 && (currChainEntry[1] != prevChainEntry[1] || currChainEntry[2] != prevChainEntry[2]) {
			fasLog.currChainEntry = &chainEntry
			emitChainPolicyEvent(fasLog, currChainEntry[1], currChainEntry[2])
		}
	} else {
		fasLog.currChainEntry = &chainEntry
		emitChainPolicyEvent(fasLog, currChainEntry[1], currChainEntry[2])
	}

	return replicas, limited, nil
}

// isScheduleActive checks if a chain entry's is active and returns a boolean, true if active, false otherwise
func isScheduleActive(s *autoscalingv1.SchedulePolicy, currentTime time.Time) bool {
	// Used for checking ahead of the schedule for daylight savings purposes
	cronDelta := (time.Minute * -1) + (time.Second * -30)

	// If the current time is before the start time, the schedule is inactive so return false
	startTime := s.Between.Start.Time
	if currentTime.Before(startTime) {
		return false
	}

	// If an end time is present and the current time is after the end time, the schedule is inactive so return false
	endTime := s.Between.End.Time
	if !endTime.IsZero() && currentTime.After(endTime) {
		return false
	}

	// If no startCron field is specified, then it's automatically true (duration is no longer relevant since we're always running)
	if s.ActivePeriod.StartCron == "" {
		return true
	}

	// Ignore the error as validation is already done within the validateChainPolicy after being unmarshalled
	location, _ := time.LoadLocation(s.ActivePeriod.Timezone)

	// Ignore the error as validation is already done within the validateChainPolicy after being unmarshalled
	startCron, _ := cron.ParseStandard(s.ActivePeriod.StartCron)

	// Ignore the error as validation is already done within the validateChainPolicy after being unmarshalled.
	// If the duration is empty set it to the largest duration possible (290 years)
	duration, _ := time.ParseDuration(s.ActivePeriod.Duration)
	if s.ActivePeriod.Duration == "" {
		duration, _ = time.ParseDuration(maxDuration)
	}

	// Get the current time - duration
	currentTimeMinusDuration := currentTime.Add(duration * -1)
	// Take (current time - duration) to get the first available start time
	cronStartTime := startCron.Next(currentTimeMinusDuration.In(location))
	// Take the (cronStartTime + duration) to get the end time
	cronEndTime := cronStartTime.Add(duration)

	// If the current time is after the cronStartTime - 90 seconds (for daylight saving purposes) AND the current time before the cronEndTime
	// then return true
	// Example: startCron = 0 14 * * * // 2:00 PM Everyday | duration = 1 hr | cronDelta = 90 seconds | currentTime = 2024-08-01T14:30:00Z | currentTimeMinusDuration = 2024-08-01T13:30:00Z
	// then cronStartTime = 2024-08-01T14:00:00Z and cronEndTime = 2024-08-01T15:00:00Z
	// and since currentTime > cronStartTime + cronDelta AND currentTime < cronEndTime, we return true
	if currentTime.After(cronStartTime.Add(cronDelta)) && currentTime.Before(cronEndTime) {
		return true
	}

	return false
}

// getSortedGameServers returns the list of Game Servers for the Fleet in the order in which the
// Game Servers would be deleted.
func getSortedGameServers(f *agonesv1.Fleet, gameServerNamespacedLister listeragonesv1.GameServerNamespaceLister,
	nodeCounts map[string]gameservers.NodeCount) ([]*agonesv1.GameServer, error) {
	gsList, err := fleets.ListGameServersByFleetOwner(gameServerNamespacedLister, f)
	if err != nil {
		return nil, err
	}

	gameServers := gssets.SortGameServersByStrategy(f.Spec.Scheduling, gsList, nodeCounts, f.Spec.Priorities)
	return gameServers, nil
}

// isLimited indicates that the calculated scale would be above or below the range defined by
// MinCapacity and MaxCapacity in the ListPolicy or CounterPolicy.
// Return 1 if the fleet needs to scale up, -1 if the fleets need to scale down, 0 if the fleet does
// not need to scale, or if the fleet is not limited.
func isLimited(aggCapacity, minCapacity, maxCapacity int64) (bool, int) {
	if aggCapacity < minCapacity { // Scale up
		return true, 1
	}
	if aggCapacity > maxCapacity { // Scale down
		return true, -1
	}
	return false, 0
}

// scaleUpLimited scales up the fleet to meet the MinCapacity
func scaleUpLimited(replicas int32, capacity, aggCapacity, minCapacity int64) (int32, bool, error) {
	if capacity == 0 {
		return 0, false, errors.Errorf("cannot scale up as Capacity is equal to 0")
	}
	for aggCapacity < minCapacity {
		aggCapacity += capacity
		replicas++
	}
	return replicas, true, nil
}

// scaleDownLimited scales down the fleet to meet the MaxCapacity
func scaleDownLimited(f *agonesv1.Fleet, gameServerNamespacedLister listeragonesv1.GameServerNamespaceLister,
	nodeCounts map[string]gameservers.NodeCount, key string, isCounter bool, replicas int32,
	aggCapacity, maxCapacity int64) (int32, bool, error) {
	// Game Servers in order of deletion on scale down
	gameServers, err := getSortedGameServers(f, gameServerNamespacedLister, nodeCounts)
	if err != nil {
		return 0, false, err
	}
	for _, gs := range gameServers {
		if aggCapacity <= maxCapacity {
			break
		}
		switch isCounter {
		case true:
			if counter, ok := gs.Status.Counters[key]; ok {
				aggCapacity -= counter.Capacity
			}
		case false:
			if list, ok := gs.Status.Lists[key]; ok {
				aggCapacity -= list.Capacity
			}
		}
		replicas--
	}

	// We are not currently able to scale down to zero replicas, so one replica is the minimum allowed
	if replicas < 1 {
		replicas = 1
	}

	return replicas, true, nil
}

func scaleLimited(scale int, f *agonesv1.Fleet, gameServerNamespacedLister listeragonesv1.GameServerNamespaceLister,
	nodeCounts map[string]gameservers.NodeCount, key string, isCounter bool, replicas int32,
	capacity, aggCapacity, minCapacity, maxCapacity int64) (int32, bool, error) {

	switch scale {
	case 1: // scale up
		return scaleUpLimited(replicas, capacity, aggCapacity, minCapacity)
	case -1: // scale down
		return scaleDownLimited(f, gameServerNamespacedLister, nodeCounts, key, isCounter, replicas,
			aggCapacity, maxCapacity)
	case 0:
		return replicas, false, nil
	}

	return 0, false, errors.Errorf("cannot scale due to error in scaleLimited function")
}

// scaleUp scales up for either Integer or Percentage Buffer.
func scaleUp(replicas int32, capacity, count, aggCapacity, availableCapacity, maxCapacity,
	minCapacity, buffer int64) (int32, bool, error) {

	// How much capacity is gained by adding one more replica to the fleet.
	replicaCapacity := capacity - count
	if replicaCapacity <= 0 {
		return 0, false, errors.Errorf("cannot scale up as adding additional replicas does not increase available Capacity")
	}

	additionalReplicas := int32(math.Ceil((float64(buffer) - float64(availableCapacity)) / float64(replicaCapacity)))

	// Check to make sure we're not limited (over Max Capacity)
	limited, _ := isLimited(aggCapacity+(int64(additionalReplicas)*capacity), minCapacity, maxCapacity)
	if limited {
		additionalReplicas = int32((maxCapacity - aggCapacity) / capacity)
	}

	return replicas + additionalReplicas, limited, nil
}

// scaleDown scales down for either Integer or Percentage Buffer.
func scaleDown(f *agonesv1.Fleet, gameServerNamespacedLister listeragonesv1.GameServerNamespaceLister,
	nodeCounts map[string]gameservers.NodeCount, key string, isCounter bool, replicas int32,
	aggCount, aggCapacity, minCapacity, buffer int64) (int32, bool, error) {
	// Exit early if we're already at MinCapacity to avoid calling getSortedGameServers if unnecessary
	if aggCapacity == minCapacity {
		return replicas, true, nil
	}

	// We first need to get the individual game servers in order of deletion on scale down, as any
	// game server may have a unique value for counts and / or capacity.
	gameServers, err := getSortedGameServers(f, gameServerNamespacedLister, nodeCounts)
	if err != nil {
		return 0, false, err
	}

	var availableCapacity int64

	// "Remove" one game server at a time in order of potential deletion. (Not actually removed here,
	// that's done later, if possible, by the fleetautoscaler controller.)
	for _, gs := range gameServers {
		replicas--
		switch isCounter {
		case true:
			if counter, ok := gs.Status.Counters[key]; ok {
				aggCount -= counter.Count
				aggCapacity -= counter.Capacity
			} else {
				continue
			}
		case false:
			if list, ok := gs.Status.Lists[key]; ok {
				aggCount -= int64(len(list.Values))
				aggCapacity -= list.Capacity
			} else {
				continue
			}
		}
		availableCapacity = aggCapacity - aggCount
		// Check if we've overshot our buffer
		if availableCapacity < buffer {
			return replicas + 1, false, nil
		}
		// Check if we're Limited (Below MinCapacity)
		if aggCapacity < minCapacity {
			return replicas + 1, true, nil
		}
		// Check if we're at our desired Buffer
		if availableCapacity == buffer {
			return replicas, false, nil
		}
		// Check if we're at Limited
		if aggCapacity == minCapacity {
			return replicas, true, nil
		}
	}

	// We are not currently able to scale down to zero replicas, so one replica is the minimum allowed.
	if replicas < 1 {
		replicas = 1
	}

	return replicas, false, nil
}

func emitChainPolicyEvent(fasLog *FasLogger, chainID string, chainType string) {
	if fasLog.recorder == nil {
		return
	}

	var eventMessage string
	var eventType string

	if chainID == "Unknown" {
		eventMessage = fmt.Sprintf("FleetAutoscaler '%s' failed to apply ChainPolicy | ID: %s | Type: %s",
			fasLog.fas.ObjectMeta.Name, chainID, chainType)
		eventType = corev1.EventTypeWarning // Use Warning for failure
	} else {
		eventMessage = fmt.Sprintf("FleetAutoscaler '%s' successfully applied ChainPolicy | ID: %s | Type: %s",
			fasLog.fas.ObjectMeta.Name, chainID, chainType)
		eventType = corev1.EventTypeNormal // Use Normal for success
	}

	// Emit the event
	fasLog.recorder.Eventf(fasLog.fas, eventType, "ChainPolicy", eventMessage)
}
