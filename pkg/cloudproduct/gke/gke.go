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

// Package gke implements the GKE cloud product (specifically Autopilot for now)
package gke

import (
	"context"
	"encoding/json"
	"fmt"

	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	"agones.dev/agones/pkg/cloudproduct/eviction"
	"agones.dev/agones/pkg/portallocator"
	"agones.dev/agones/pkg/util/runtime"
	"cloud.google.com/go/compute/metadata"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

const (
	hostPortAssignmentAnnotation = "autopilot.gke.io/host-port-assignment"
	primaryContainerAnnotation   = "autopilot.gke.io/primary-container"

	errRangeInvalid                 = "range must not be used on GKE Autopilot"
	errSchedulingMustBePacked       = "scheduling strategy must be Packed on GKE Autopilot"
	errEvictionSafeOnUpgradeInvalid = "eviction.safe OnUpgrade not supported on GKE Autopilot"
)

var (
	autopilotMutatingWebhooks = []string{
		"workload-defaulter.config.common-webhooks.networking.gke.io", // pre-1.26
		"sasecret-redacter.config.common-webhooks.networking.gke.io",  // 1.26+
	}
	noWorkloadDefaulter = fmt.Sprintf("found no MutatingWebhookConfigurations matching %v", autopilotMutatingWebhooks)

	logger = runtime.NewLoggerWithSource("gke")
)

type gkeAutopilot struct {
	useExtendedDurationPods bool
}

// hostPortAssignment is the JSON structure of the `host-port-assignment` annotation
//
//nolint:govet // API-like, keep consistent
type hostPortAssignment struct {
	Min           int32           `json:"min,omitempty"`
	Max           int32           `json:"max,omitempty"`
	PortsAssigned map[int32]int32 `json:"portsAssigned,omitempty"` // old -> new
}

// Detect whether we're running on GKE and/or Autopilot and return the appropriate
// cloud product string.
func Detect(ctx context.Context, kc *kubernetes.Clientset) string {
	if !metadata.OnGCE() {
		return ""
	}
	// Look for the workload defaulter - this is the current best method to detect Autopilot
	found := false
	for _, webhook := range autopilotMutatingWebhooks {
		if _, err := kc.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(
			ctx, webhook, metav1.GetOptions{}); err != nil {
			logger.WithError(err).WithField("webhook", webhook).Info("Detecting Autopilot MutatingWebhookConfiguration")
		} else {
			found = true
			break
		}
	}
	if !found {
		logger.WithField("reason", noWorkloadDefaulter).Info(
			"Assuming GKE Standard and defaulting to generic provider (expected if not on GKE Autopilot)")
		return "" // GKE standard, but we don't need an interface for it just yet.
	}
	logger.Info("Running on GKE Autopilot (skip detection with --cloud-product=gke-autopilot)")
	return "gke-autopilot"
}

// Autopilot returns a GKE Autopilot cloud product
//
//nolint:revive // ignore the unexported return; implements ControllerHooksInterface
func Autopilot() *gkeAutopilot {
	return &gkeAutopilot{useExtendedDurationPods: runtime.FeatureEnabled(runtime.FeatureGKEAutopilotExtendedDurationPods)}
}

func (*gkeAutopilot) SyncPodPortsToGameServer(gs *agonesv1.GameServer, pod *corev1.Pod) error {
	// If applyGameServerAddressAndPort has already filled in Status, SyncPodPortsToGameServer
	// has already run. Skip syncing from the Pod again - this avoids having to reason
	// about whether we're re-applying the old->new mapping.
	if len(gs.Status.Ports) == len(gs.Spec.Ports) {
		return nil
	}
	annotation, ok := pod.ObjectMeta.Annotations[hostPortAssignmentAnnotation]
	if !ok {
		return nil
	}
	var hpa hostPortAssignment
	if err := json.Unmarshal([]byte(annotation), &hpa); err != nil {
		return errors.Wrapf(err, "could not unmarshal annotation %s (value %q)", hostPortAssignmentAnnotation, annotation)
	}
	for i, p := range gs.Spec.Ports {
		if newPort, ok := hpa.PortsAssigned[p.HostPort]; ok {
			gs.Spec.Ports[i].HostPort = newPort
		}
	}
	return nil
}

func (*gkeAutopilot) NewPortAllocator(portRanges map[string]portallocator.PortRange,
	_ informers.SharedInformerFactory,
	_ externalversions.SharedInformerFactory,
) portallocator.Interface {
	defPortRange := portRanges[agonesv1.DefaultPortRange]
	return &autopilotPortAllocator{minPort: defPortRange.MinPort, maxPort: defPortRange.MaxPort}
}

func (*gkeAutopilot) WaitOnFreePorts() bool { return true }

func (g *gkeAutopilot) ValidateGameServerSpec(gss *agonesv1.GameServerSpec, fldPath *field.Path) field.ErrorList {
	allErrs := g.ValidateScheduling(gss.Scheduling, fldPath.Child("scheduling"))

	// Loop through ports and use the helper function for validation
	for i, p := range gss.Ports {
		allErrs = append(allErrs, validatePortPolicy(p, i, fldPath)...)

	}

	// See SetEviction comment below for why we block EvictionSafeOnUpgrade, if Extended Duration pods aren't supported.
	if !g.useExtendedDurationPods && gss.Eviction.Safe == agonesv1.EvictionSafeOnUpgrade {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("eviction").Child("safe"), string(gss.Eviction.Safe), errEvictionSafeOnUpgradeInvalid))
	}
	return allErrs
}

func (*gkeAutopilot) ValidateScheduling(ss apis.SchedulingStrategy, fldPath *field.Path) field.ErrorList {
	if ss != apis.Packed {
		return field.ErrorList{
			field.Invalid(fldPath, string(ss), errSchedulingMustBePacked),
		}
	}
	return nil
}

func (*gkeAutopilot) MutateGameServerPod(gss *agonesv1.GameServerSpec, pod *corev1.Pod) error {
	setPassthroughLabel(gss, pod)
	setPrimaryContainer(pod, gss.Container)
	podSpecSeccompUnconfined(&pod.Spec)
	return nil
}

// setPassthroughLabel sets the agones.dev/port: "autopilot-passthrough" label to the game server container.
// This will help to back the container port from the allocated port using an objectSelector of this label
// in GameServers that are using Passthrough Port Policy
func setPassthroughLabel(gs *agonesv1.GameServerSpec, pod *corev1.Pod) {
	if hasPortPolicy(gs, agonesv1.Passthrough) {
		pod.ObjectMeta.Labels[agonesv1.GameServerPortPolicyPodLabel] = "autopilot-passthrough"
	}
}

// setPrimaryContainer sets the autopilot.gke.io/primary-container annotation to the game server container.
// This acts as a hint to Autopilot for which container to add resources to during resource adjustment.
// See https://cloud.google.com/kubernetes-engine/docs/concepts/autopilot-resource-requests#autopilot-resource-management
// for more details.
func setPrimaryContainer(pod *corev1.Pod, containerName string) {
	if _, ok := pod.ObjectMeta.Annotations[primaryContainerAnnotation]; ok {
		return
	}
	pod.ObjectMeta.Annotations[primaryContainerAnnotation] = containerName
}

// podSpecSeccompUnconfined sets to seccomp profile to `Unconfined` to avoid serious performance
// degradation possible with seccomp. We only set the pod level seccompProfile, and only set
// it if it hasn't been set - users can then override at either the pod or container level
// in the GameServer spec.
func podSpecSeccompUnconfined(podSpec *corev1.PodSpec) {
	if podSpec.SecurityContext != nil && podSpec.SecurityContext.SeccompProfile != nil {
		return
	}
	if podSpec.SecurityContext == nil {
		podSpec.SecurityContext = &corev1.PodSecurityContext{}
	}
	podSpec.SecurityContext.SeccompProfile = &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeUnconfined}
}

func (g *gkeAutopilot) SetEviction(ev *agonesv1.Eviction, pod *corev1.Pod) error {
	if g.useExtendedDurationPods {
		return eviction.SetEviction(ev, pod)
	}
	return setEvictionNoExtended(ev, pod)
}

// setEvictionNoExtended sets disruption controls based on GameServer.Status.Eviction. For Autopilot:
//   - Since the safe-to-evict pod annotation is not supported if "false", we delete it (if it's set
//     to anything else, we allow it - Autopilot only rejects "false").
//   - OnUpgrade is not supported and rejected by validation above. Since we can't support
//     safe-to-evict=false but can support a restrictive PDB, we can support Never and Always, but
//     OnUpgrade doesn't make sense on Autopilot today. - an overly restrictive PDB prevents
//     any sort of graceful eviction.
func setEvictionNoExtended(ev *agonesv1.Eviction, pod *corev1.Pod) error {
	if safeAnnotation := pod.ObjectMeta.Annotations[agonesv1.PodSafeToEvictAnnotation]; safeAnnotation == agonesv1.False {
		delete(pod.ObjectMeta.Annotations, agonesv1.PodSafeToEvictAnnotation)
	}
	if ev == nil {
		return errors.New("No eviction value set. Should be the default value")
	}
	if _, exists := pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel]; !exists {
		switch ev.Safe {
		case agonesv1.EvictionSafeAlways:
			// For EvictionSafeAlways, we use a label value that does not match the
			// agones-gameserver-safe-to-evict-false PDB. But we go ahead and label
			// it, in case someone wants to adopt custom logic for this group of
			// game servers.
			pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = agonesv1.True
		case agonesv1.EvictionSafeNever:
			pod.ObjectMeta.Labels[agonesv1.SafeToEvictLabel] = agonesv1.False
		default:
			return errors.Errorf("eviction.safe == %s, which webhook should have rejected on Autopilot", ev.Safe)
		}
	}
	return nil
}

func hasPortPolicy(gs *agonesv1.GameServerSpec, portPolicy agonesv1.PortPolicy) bool {
	for _, p := range gs.Ports {
		if p.PortPolicy == portPolicy {
			return true
		}
	}
	return false
}

type autopilotPortAllocator struct {
	minPort int32
	maxPort int32
}

func (*autopilotPortAllocator) Run(_ context.Context) error       { return nil }
func (*autopilotPortAllocator) DeAllocate(_ *agonesv1.GameServer) {}

func (apa *autopilotPortAllocator) Allocate(gs *agonesv1.GameServer) *agonesv1.GameServer {
	if len(gs.Spec.Ports) == 0 {
		return gs // Nothing to do.
	}

	var ports []agonesv1.GameServerPort
	for i, p := range gs.Spec.Ports {
		if !(p.PortPolicy == agonesv1.Dynamic || p.PortPolicy == agonesv1.Passthrough) {
			logger.WithField("gs", gs.Name).WithField("portPolicy", p.PortPolicy).Error(
				"GameServer has invalid PortPolicy for Autopilot - this should have been rejected by webhooks. Refusing to assign ports.")
			return gs
		}
		p.HostPort = int32(i + 1) // Autopilot expects _some_ host port - use a value unique to this GameServer Port.

		if p.Protocol == agonesv1.ProtocolTCPUDP {
			tcp := p
			tcp.Name = p.Name + "-tcp"
			tcp.Protocol = corev1.ProtocolTCP
			ports = append(ports, tcp)

			p.Name += "-udp"
			p.Protocol = corev1.ProtocolUDP
		}
		ports = append(ports, p)
	}

	hpa := hostPortAssignment{Min: apa.minPort, Max: apa.maxPort}
	hpaJSON, err := json.Marshal(hpa)
	if err != nil {
		logger.WithError(err).WithField("hostPort", hpa).WithField("gs", gs.Name).Error("Internal error marshalling hostPortAssignment for GameServer")
		// In error cases, return the original gs - on Autopilot this will result in a policy failure.
		return gs
	}

	// No errors past here.
	gs.Spec.Ports = ports
	if gs.Spec.Template.ObjectMeta.Annotations == nil {
		gs.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
	}
	gs.Spec.Template.ObjectMeta.Annotations[hostPortAssignmentAnnotation] = string(hpaJSON)
	return gs
}

// validatePortPolicy is a helper function to validate a single GameServerPort's PortPolicy
// for GKE Autopilot constraints.
func validatePortPolicy(p agonesv1.GameServerPort, i int, fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	portPath := fldPath.Child("ports").Index(i)

	switch p.PortPolicy {
	case agonesv1.Dynamic, agonesv1.Passthrough:
		// These policies are always valid on GKE Autopilot.
	case agonesv1.None:
		// "None" is valid only if the feature gate FeaturePortPolicyNone is enabled.
		if !runtime.FeatureEnabled(runtime.FeaturePortPolicyNone) {
			allErrs = append(allErrs, field.Invalid(portPath.Child("portPolicy"), p.PortPolicy, "PortPolicy 'None' is not enabled"))
		}
	default:
		// Any other port policy, such as "Static", is considered invalid on GKE Autopilot.
		allErrs = append(allErrs, field.Invalid(portPath.Child("portPolicy"), p.PortPolicy, "portPolicy must be Dynamic, Passthrough, or None on GKE Autopilot"))
	}

	if p.Range != agonesv1.DefaultPortRange && (p.PortPolicy != agonesv1.None || !runtime.FeatureEnabled(runtime.FeaturePortPolicyNone)) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("ports").Index(i).Child("range"), p.Range, errRangeInvalid))
	}

	return allErrs
}
