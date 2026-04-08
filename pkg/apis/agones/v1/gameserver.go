// Copyright 2017 Google LLC All Rights Reserved.
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
	"encoding/json"
	"fmt"
	"net"
	"slices"
	"strings"

	"agones.dev/agones/pkg"
	"agones.dev/agones/pkg/apis"
	"agones.dev/agones/pkg/apis/agones"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/pkg/errors"
	"gomodules.xyz/jsonpatch/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	apimachineryvalidation "k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// GameServerState is the state for the GameServer
type GameServerState string

const (
	// ListMaxCapacity is the maximum capacity for List in the gamerserver spec and status CRDs.
	ListMaxCapacity = int64(1000)
)

const (
	// GameServerStatePortAllocation is for when a dynamically allocating GameServer
	// is being created, an open port needs to be allocated
	GameServerStatePortAllocation GameServerState = "PortAllocation"
	// GameServerStateCreating is before the Pod for the GameServer is being created
	GameServerStateCreating GameServerState = "Creating"
	// GameServerStateStarting is for when the Pods for the GameServer are being
	// created but are not yet Scheduled
	GameServerStateStarting GameServerState = "Starting"
	// GameServerStateScheduled is for when we have determined that the Pod has been
	// scheduled in the cluster -- basically, we have a NodeName
	GameServerStateScheduled GameServerState = "Scheduled"
	// GameServerStateRequestReady is when the GameServer has declared that it is ready
	GameServerStateRequestReady GameServerState = "RequestReady"
	// GameServerStateReady is when a GameServer is ready to take connections
	// from Game clients
	GameServerStateReady GameServerState = "Ready"
	// GameServerStateShutdown is when the GameServer has shutdown and everything needs to be
	// deleted from the cluster
	GameServerStateShutdown GameServerState = "Shutdown"
	// GameServerStateError is when something has gone wrong with the Gameserver and
	// it cannot be resolved
	GameServerStateError GameServerState = "Error"
	// GameServerStateUnhealthy is when the GameServer has failed its health checks
	GameServerStateUnhealthy GameServerState = "Unhealthy"
	// GameServerStateReserved is for when a GameServer is reserved and therefore can be allocated but not removed
	GameServerStateReserved GameServerState = "Reserved"
	// GameServerStateAllocated is when the GameServer has been allocated to a session
	GameServerStateAllocated GameServerState = "Allocated"
)

// PortPolicy is the port policy for the GameServer
type PortPolicy string

const (
	// Static PortPolicy means that the user defines the hostPort to be used
	// in the configuration.
	Static PortPolicy = "Static"
	// Dynamic PortPolicy means that the system will choose an open
	// port for the GameServer in question
	Dynamic PortPolicy = "Dynamic"
	// Passthrough dynamically sets the `containerPort` to the same value as the dynamically selected hostPort.
	// This will mean that users will need to lookup what port has been opened through the server side SDK.
	Passthrough PortPolicy = "Passthrough"
	// None means the `hostPort` is ignored and if defined, the `containerPort` (optional) is used to set the port on the GameServer instance.
	None PortPolicy = "None"
)

// EvictionSafe specified whether the game server supports termination via SIGTERM
type EvictionSafe string

const (
	// EvictionSafeAlways means the game server supports termination via SIGTERM, and wants eviction signals
	// from Cluster Autoscaler scaledown and node upgrades.
	EvictionSafeAlways EvictionSafe = "Always"
	// EvictionSafeOnUpgrade means the game server supports termination via SIGTERM, and wants eviction signals
	// from node upgrades, but not Cluster Autoscaler scaledown.
	EvictionSafeOnUpgrade EvictionSafe = "OnUpgrade"
	// EvictionSafeNever means the game server should run to completion and may not understand SIGTERM. Eviction
	// from ClusterAutoscaler and upgrades should both be blocked.
	EvictionSafeNever EvictionSafe = "Never"
)

// SdkServerLogLevel is the log level for SDK server (sidecar) logs
type SdkServerLogLevel string

const (
	// SdkServerLogLevelInfo will cause the SDK server to output all messages except for debug messages.
	SdkServerLogLevelInfo SdkServerLogLevel = "Info"
	// SdkServerLogLevelDebug will cause the SDK server to output all messages including debug messages.
	SdkServerLogLevelDebug SdkServerLogLevel = "Debug"
	// SdkServerLogLevelError will cause the SDK server to only output error messages.
	SdkServerLogLevelError SdkServerLogLevel = "Error"
	// SdkServerLogLevelTrace will cause the SDK server to output all messages, including detailed tracing information.
	SdkServerLogLevelTrace SdkServerLogLevel = "Trace"
)

const (
	// ProtocolTCPUDP Protocol exposes the hostPort allocated for this container for both TCP and UDP.
	ProtocolTCPUDP corev1.Protocol = "TCPUDP"

	// DefaultPortRange is the name of the default port range.
	DefaultPortRange = "default"

	// RoleLabel is the label in which the Agones role is specified.
	// Pods from a GameServer will have the value "gameserver"
	RoleLabel = agones.GroupName + "/role"
	// GameServerLabelRole is the GameServer label value for RoleLabel
	GameServerLabelRole = "gameserver"
	// GameServerPodLabel is the label that the name of the GameServer
	// is set on the Pod the GameServer controls
	GameServerPodLabel = agones.GroupName + "/gameserver"
	// GameServerPortPolicyPodLabel is the label to identify the port policy
	// of the pod
	GameServerPortPolicyPodLabel = agones.GroupName + "/port"
	// GameServerContainerAnnotation is the annotation that stores
	// which container is the container that runs the dedicated game server
	GameServerContainerAnnotation = agones.GroupName + "/container"
	// DevAddressAnnotation is an annotation to indicate that a GameServer hosted outside of Agones.
	// A locally hosted GameServer is not managed by Agones it is just simply registered.
	DevAddressAnnotation = "agones.dev/dev-address"
	// GameServerReadyContainerIDAnnotation is an annotation that is set on the GameServer
	// becomes ready, so we can track when restarts should occur and when a GameServer
	// should be moved to Unhealthy.
	GameServerReadyContainerIDAnnotation = agones.GroupName + "/ready-container-id"
	// PodSafeToEvictAnnotation is an annotation that the Kubernetes cluster autoscaler uses to
	// determine if a pod can safely be evicted to compact a cluster by moving pods between nodes
	// and scaling down nodes.
	PodSafeToEvictAnnotation = "cluster-autoscaler.kubernetes.io/safe-to-evict"
	// SafeToEvictLabel is a label that, when "false", matches the restrictive PDB agones-gameserver-safe-to-evict-false.
	SafeToEvictLabel = agones.GroupName + "/safe-to-evict"
	// GameServerErroredAtAnnotation is an annotation that records the timestamp the GameServer entered the
	// error state. The timestamp is encoded in RFC3339 format.
	GameServerErroredAtAnnotation = agones.GroupName + "/errored-at"
	// FinalizerName is the domain name and finalizer path used to manage garbage collection of the GameServer.
	FinalizerName = agones.GroupName + "/controller"

	// NodePodIP identifies an IP address from a pod.
	NodePodIP corev1.NodeAddressType = "PodIP"

	// PassthroughPortAssignmentAnnotation is an annotation to keep track of game server container and its Passthrough ports indices
	PassthroughPortAssignmentAnnotation = "agones.dev/container-passthrough-port-assignment"

	// True is the string "true" to appease the goconst lint.
	True = "true"
	// False is the string "false" to appease the goconst lint.
	False = "false"
)

var (
	// GameServerRolePodSelector is the selector to get all GameServer Pods
	GameServerRolePodSelector = labels.SelectorFromSet(labels.Set{RoleLabel: GameServerLabelRole})

	// TerminalGameServerStates is a set (map[GameServerState]bool) of states from which a GameServer will not recover.
	// From state diagram at https://agones.dev/site/docs/reference/gameserver/
	TerminalGameServerStates = map[GameServerState]bool{
		GameServerStateShutdown:  true,
		GameServerStateError:     true,
		GameServerStateUnhealthy: true,
	}
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GameServer is the data structure for a GameServer resource.
// It is worth noting that while there is a `GameServerStatus` Status entry for the `GameServer`, it is not
// defined as a subresource - unlike `Fleet` and other Agones resources.
// This is so that we can retain the ability to change multiple aspects of a `GameServer` in a single atomic operation,
// which is particularly useful for operations such as allocation.
type GameServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GameServerSpec   `json:"spec"`
	Status GameServerStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GameServerList is a list of GameServer resources
type GameServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []GameServer `json:"items"`
}

// GameServerTemplateSpec is a template for GameServers
type GameServerTemplateSpec struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              GameServerSpec `json:"spec"`
}

// GameServerSpec is the spec for a GameServer resource
type GameServerSpec struct {
	// Container specifies which Pod container is the game server. Only required if there is more than one
	// container defined
	Container string `json:"container,omitempty"`
	// Ports are the array of ports that can be exposed via the game server
	Ports []GameServerPort `json:"ports,omitempty"`
	// Health configures health checking
	Health Health `json:"health,omitempty"`
	// Scheduling strategy. Defaults to "Packed"
	Scheduling apis.SchedulingStrategy `json:"scheduling,omitempty"`
	// SdkServer specifies parameters for the Agones SDK Server sidecar container
	SdkServer SdkServer `json:"sdkServer,omitempty"`
	// Template describes the Pod that will be created for the GameServer
	Template corev1.PodTemplateSpec `json:"template"`
	// (Alpha, PlayerTracking feature flag) Players provides the configuration for player tracking features.
	// +optional
	Players *PlayersSpec `json:"players,omitempty"`
	// (Beta, CountsAndLists feature flag) Counters provides the configuration for tracking of int64 values against a GameServer.
	// Keys must be declared at GameServer creation time.
	// +optional
	Counters map[string]CounterStatus `json:"counters,omitempty"`
	// (Beta, CountsAndLists feature flag) Lists provides the configuration for tracking of lists of up to 1000 values against a GameServer.
	// Keys must be declared at GameServer creation time.
	// +optional
	Lists map[string]ListStatus `json:"lists,omitempty"`
	// Eviction specifies the eviction tolerance of the GameServer. Defaults to "Never".
	// +optional
	Eviction *Eviction `json:"eviction,omitempty"`
	// immutableReplicas is present in gameservers.agones.dev but omitted here (it's always 1).
}

// PlayersSpec tracks the initial player capacity
type PlayersSpec struct {
	InitialCapacity int64 `json:"initialCapacity,omitempty"`
}

// Eviction specifies the eviction tolerance of the GameServer
type Eviction struct {
	// Game server supports termination via SIGTERM:
	// - Always: Allow eviction for both Cluster Autoscaler and node drain for upgrades
	// - OnUpgrade: Allow eviction for upgrades alone
	// - Never (default): Pod should run to completion
	Safe EvictionSafe `json:"safe,omitempty"`
}

// Health configures health checking on the GameServer
type Health struct {
	// Disabled is whether health checking is disabled or not
	Disabled bool `json:"disabled,omitempty"`
	// PeriodSeconds is the number of seconds each health ping has to occur in
	PeriodSeconds int32 `json:"periodSeconds,omitempty"`
	// FailureThreshold how many failures in a row constitutes unhealthy
	FailureThreshold int32 `json:"failureThreshold,omitempty"`
	// InitialDelaySeconds initial delay before checking health
	InitialDelaySeconds int32 `json:"initialDelaySeconds,omitempty"`
}

// GameServerPort defines a set of Ports that
// are to be exposed via the GameServer
type GameServerPort struct {
	// Name is the descriptive name of the port
	Name string `json:"name,omitempty"`
	// (Alpha, PortRanges feature flag) Range is the port range name from which to select a port when using a
	// 'Dynamic' or 'Passthrough' port policy.
	// +optional
	Range string `json:"range,omitempty"`
	// PortPolicy defines the policy for how the HostPort is populated.
	// Dynamic port will allocate a HostPort within the selected MIN_PORT and MAX_PORT range passed to the controller
	// at installation time.
	// When `Static` portPolicy is specified, `HostPort` is required, to specify the port that game clients will
	// connect to
	// `Passthrough` dynamically sets the `containerPort` to the same value as the dynamically selected hostPort.
	// `None` portPolicy ignores `HostPort` and the `containerPort` (optional) is used to set the port on the GameServer instance.
	PortPolicy PortPolicy `json:"portPolicy,omitempty"`
	// Container is the name of the container or sidecar container on which to open the port. Defaults to the game server container.
	// +optional
	Container *string `json:"container,omitempty"`
	// ContainerPort is the port that is being opened on the specified container's process
	ContainerPort int32 `json:"containerPort,omitempty"`
	// HostPort the port exposed on the host for clients to connect to
	HostPort int32 `json:"hostPort,omitempty"`
	// Protocol is the network protocol being used. Defaults to UDP. TCP and TCPUDP are other options.
	Protocol corev1.Protocol `json:"protocol,omitempty"`
}

// SdkServer specifies parameters for the Agones SDK Server sidecar container
type SdkServer struct {
	// LogLevel for SDK server (sidecar) logs. Defaults to "Info"
	LogLevel SdkServerLogLevel `json:"logLevel,omitempty"`
	// GRPCPort is the port on which the SDK Server binds the gRPC server to accept incoming connections
	GRPCPort int32 `json:"grpcPort,omitempty"`
	// HTTPPort is the port on which the SDK Server binds the HTTP gRPC gateway server to accept incoming connections
	HTTPPort int32 `json:"httpPort,omitempty"`
}

// GameServerStatus is the status for a GameServer resource
type GameServerStatus struct {
	// GameServerState is the current state of a GameServer, e.g. Creating, Starting, Ready, etc
	State   GameServerState        `json:"state"`
	Ports   []GameServerStatusPort `json:"ports"`
	Address string                 `json:"address"`
	// Addresses is the array of addresses at which the GameServer can be reached; copy of Node.Status.addresses.
	// +optional
	Addresses     []corev1.NodeAddress `json:"addresses"`
	NodeName      string               `json:"nodeName"`
	ReservedUntil *metav1.Time         `json:"reservedUntil"`
	// [Stage:Alpha]
	// [FeatureFlag:PlayerTracking]
	// +optional
	Players *PlayerStatus `json:"players"`
	// (Beta, CountsAndLists feature flag) Counters and Lists provides the configuration for generic tracking features.
	// +optional
	Counters map[string]CounterStatus `json:"counters,omitempty"`
	// +optional
	Lists map[string]ListStatus `json:"lists,omitempty"`
	// Eviction specifies the eviction tolerance of the GameServer.
	// +optional
	Eviction *Eviction `json:"eviction,omitempty"`
	// immutableReplicas is present in gameservers.agones.dev but omitted here (it's always 1).
}

// GameServerStatusPort shows the port that was allocated to a
// GameServer.
type GameServerStatusPort struct {
	Name string `json:"name,omitempty"`
	Port int32  `json:"port"`
}

// PlayerStatus stores the current player capacity values
type PlayerStatus struct {
	Count    int64    `json:"count"`
	Capacity int64    `json:"capacity"`
	IDs      []string `json:"ids"`
}

// CounterStatus stores the current counter values and maximum capacity
type CounterStatus struct {
	Count    int64 `json:"count"`
	Capacity int64 `json:"capacity"`
}

// ListStatus stores the current list values and maximum capacity
type ListStatus struct {
	Capacity int64    `json:"capacity"`
	Values   []string `json:"values"`
}

// ApplyDefaults applies default values to the GameServer if they are not already populated
func (gs *GameServer) ApplyDefaults() {
	// VersionAnnotation is the annotation that stores
	// the version of sdk which runs in a sidecar
	if gs.ObjectMeta.Annotations == nil {
		gs.ObjectMeta.Annotations = map[string]string{}
	}
	gs.ObjectMeta.Annotations[VersionAnnotation] = pkg.Version
	gs.ObjectMeta.Finalizers = append(gs.ObjectMeta.Finalizers, FinalizerName)

	gs.Spec.ApplyDefaults()
	gs.applyStatusDefaults()
}

// ApplyDefaults applies default values to the GameServerSpec if they are not already populated
func (gss *GameServerSpec) ApplyDefaults() {
	gss.applyContainerDefaults()
	gss.applyPortDefaults()
	gss.applyHealthDefaults()
	gss.applyEvictionDefaults()
	gss.applySchedulingDefaults()
	gss.applySdkServerDefaults()
}

// applySdkServerDefaults applies the default log level ("Info") for the sidecar
func (gss *GameServerSpec) applySdkServerDefaults() {
	if gss.SdkServer.LogLevel == "" {
		gss.SdkServer.LogLevel = SdkServerLogLevelInfo
	}
	if gss.SdkServer.GRPCPort == 0 {
		gss.SdkServer.GRPCPort = 9357
	}
	if gss.SdkServer.HTTPPort == 0 {
		gss.SdkServer.HTTPPort = 9358
	}
}

// applyContainerDefaults applies the container defaults
func (gss *GameServerSpec) applyContainerDefaults() {
	if len(gss.Template.Spec.Containers) == 1 {
		gss.Container = gss.Template.Spec.Containers[0].Name
	}
}

// applyHealthDefaults applies health checking defaults
func (gss *GameServerSpec) applyHealthDefaults() {
	if !gss.Health.Disabled {
		if gss.Health.PeriodSeconds <= 0 {
			gss.Health.PeriodSeconds = 5
		}
		if gss.Health.FailureThreshold <= 0 {
			gss.Health.FailureThreshold = 3
		}
		if gss.Health.InitialDelaySeconds <= 0 {
			gss.Health.InitialDelaySeconds = 5
		}
	}
}

// applyStatusDefaults applies Status defaults
func (gs *GameServer) applyStatusDefaults() {
	if gs.Status.State == "" {
		gs.Status.State = GameServerStateCreating
		// applyStatusDefaults() should be called after applyPortDefaults()
		if gs.HasPortPolicy(Dynamic) || gs.HasPortPolicy(Passthrough) {
			gs.Status.State = GameServerStatePortAllocation
		}
	}

	if runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		// set value if enabled, otherwise very easy to accidentally panic
		// when gs.Status.Players is nil
		if gs.Status.Players == nil {
			gs.Status.Players = &PlayerStatus{}
		}
		if gs.Spec.Players != nil {
			gs.Status.Players.Capacity = gs.Spec.Players.InitialCapacity
		}
	}

	gs.applyEvictionStatus()
	gs.applyCountsListsStatus()
}

// applyPortDefaults applies default values for all ports
func (gss *GameServerSpec) applyPortDefaults() {
	for i, p := range gss.Ports {
		// basic spec
		if p.PortPolicy == "" {
			gss.Ports[i].PortPolicy = Dynamic
		}

		if p.Range == "" {
			gss.Ports[i].Range = DefaultPortRange
		}

		if p.Protocol == "" {
			gss.Ports[i].Protocol = "UDP"
		}

		if p.Container == nil || *p.Container == "" {
			gss.Ports[i].Container = &gss.Container
		}
	}
}

func (gss *GameServerSpec) applySchedulingDefaults() {
	if gss.Scheduling == "" {
		gss.Scheduling = apis.Packed
	}
}

func (gss *GameServerSpec) applyEvictionDefaults() {
	if gss.Eviction == nil {
		gss.Eviction = &Eviction{}
	}
	if gss.Eviction.Safe == "" {
		gss.Eviction.Safe = EvictionSafeNever
	}
}

func (gs *GameServer) applyEvictionStatus() {
	gs.Status.Eviction = gs.Spec.Eviction.DeepCopy()
	if gs.Spec.Template.ObjectMeta.Annotations[PodSafeToEvictAnnotation] == "true" {
		if gs.Status.Eviction == nil {
			gs.Status.Eviction = &Eviction{}
		}
		gs.Status.Eviction.Safe = EvictionSafeAlways
	}
}

func (gs *GameServer) applyCountsListsStatus() {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		return
	}
	if gs.Spec.Counters != nil {
		countersCopy := make(map[string]CounterStatus, len(gs.Spec.Counters))
		for key, val := range gs.Spec.Counters {
			countersCopy[key] = *val.DeepCopy()
		}
		gs.Status.Counters = countersCopy
	}
	if gs.Spec.Lists != nil {
		listsCopy := make(map[string]ListStatus, len(gs.Spec.Lists))
		for key, val := range gs.Spec.Lists {
			listsCopy[key] = *val.DeepCopy()
		}
		gs.Status.Lists = listsCopy
	}
}

// validateFeatureGates checks if fields are set when the associated feature gate is not set.
func (gss *GameServerSpec) validateFeatureGates(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		if gss.Players != nil {
			allErrs = append(allErrs, field.Forbidden(fldPath.Child("players"), fmt.Sprintf("Value cannot be set unless feature flag %s is enabled", runtime.FeaturePlayerTracking)))
		}
	}

	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		if gss.Counters != nil {
			allErrs = append(allErrs, field.Forbidden(fldPath.Child("counters"), fmt.Sprintf("Value cannot be set unless feature flag %s is enabled", runtime.FeatureCountsAndLists)))
		}
		if gss.Lists != nil {
			allErrs = append(allErrs, field.Forbidden(fldPath.Child("lists"), fmt.Sprintf("Value cannot be set unless feature flag %s is enabled", runtime.FeatureCountsAndLists)))
		}
	}

	if !runtime.FeatureEnabled(runtime.FeaturePortPolicyNone) {
		for i, p := range gss.Ports {
			if p.PortPolicy == None {
				allErrs = append(allErrs, field.Forbidden(fldPath.Child("ports").Index(i).Child("portPolicy"), fmt.Sprintf("Value cannot be set to %s unless feature flag %s is enabled", None, runtime.FeaturePortPolicyNone)))
			}
		}
	}

	return allErrs
}

// Validate validates the GameServerSpec configuration.
// devAddress is a specific IP address used for local Gameservers, for fleets "" is used
// If a GameServer Spec is invalid there will be > 0 values in the returned array
func (gss *GameServerSpec) Validate(apiHooks APIHooks, devAddress string, fldPath *field.Path) field.ErrorList {
	allErrs := gss.validateFeatureGates(fldPath)
	if len(devAddress) > 0 {
		// verify that the value is a valid IP address.
		if net.ParseIP(devAddress) == nil {
			// Authentication is only required if the gameserver is created directly.
			allErrs = append(allErrs, field.Invalid(field.NewPath("metadata", "annotations", DevAddressAnnotation), devAddress, "must be a valid IP address"))
		}

		for i, p := range gss.Ports {
			if p.HostPort == 0 {
				allErrs = append(allErrs, field.Required(fldPath.Child("ports").Index(i).Child("hostPort"), DevAddressAnnotation))
			}
			if p.PortPolicy != Static {
				allErrs = append(allErrs, field.Required(fldPath.Child("ports").Index(i).Child("portPolicy"), ErrPortPolicyStatic))
			}
		}

		allErrs = append(allErrs, validateObjectMeta(&gss.Template.ObjectMeta, fldPath.Child("template", "metadata"))...)
		return allErrs
	}

	// make sure a name is specified when there is multiple containers in the pod.
	if gss.Container == "" && len(gss.Template.Spec.Containers) > 1 {
		allErrs = append(allErrs, field.Required(fldPath.Child("container"), ErrContainerRequired))
	}

	// make sure the container value points to a valid container
	if !gss.HasContainer(gss.Container, false) {
		allErrs = append(allErrs,
			field.Invalid(
				fldPath.Child("container"),
				gss.Container,
				"Could not find a container named "+gss.Container,
			),
		)
	}

	// no host port when using dynamic PortPolicy
	for i, p := range gss.Ports {
		path := fldPath.Child("ports").Index(i)
		if p.PortPolicy == Dynamic || p.PortPolicy == Static {
			if p.ContainerPort <= 0 {
				allErrs = append(allErrs, field.Required(path.Child("containerPort"), ErrContainerPortRequired))
			}
		}

		if p.PortPolicy == Passthrough && p.ContainerPort > 0 {
			allErrs = append(allErrs, field.Required(path.Child("containerPort"), ErrContainerPortPassthrough))
		}

		if p.HostPort > 0 && (p.PortPolicy == Dynamic || p.PortPolicy == Passthrough) {
			allErrs = append(allErrs, field.Forbidden(path.Child("hostPort"), ErrHostPort))
		}

		if p.Container != nil && gss.Container != "" {
			if !gss.HasContainer(*p.Container, true) {
				allErrs = append(allErrs, field.Invalid(path.Child("container"), *p.Container, ErrContainerNameInvalid))
			}
		}
	}
	for i, c := range gss.Template.Spec.InitContainers {
		path := fldPath.Child("template", "spec", "initContainers").Index(i)
		allErrs = append(allErrs, ValidateResourceRequirements(&c.Resources, path.Child("resources"))...)
	}
	for i, c := range gss.Template.Spec.Containers {
		path := fldPath.Child("template", "spec", "containers").Index(i)
		allErrs = append(allErrs, ValidateResourceRequirements(&c.Resources, path.Child("resources"))...)
	}

	allErrs = append(allErrs, apiHooks.ValidateGameServerSpec(gss, fldPath)...)
	allErrs = append(allErrs, validateObjectMeta(&gss.Template.ObjectMeta, fldPath.Child("template", "metadata"))...)
	return allErrs
}

// ValidateResourceRequirements Validates resource requirement spec.
func ValidateResourceRequirements(requirements *corev1.ResourceRequirements, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	limPath := fldPath.Child("limits")
	reqPath := fldPath.Child("requests")

	for resourceName, quantity := range requirements.Limits {
		fldPath := limPath.Key(string(resourceName))
		// Validate resource quantity.
		allErrs = append(allErrs, ValidateNonnegativeQuantity(quantity, fldPath)...)

	}

	for resourceName, quantity := range requirements.Requests {
		fldPath := reqPath.Key(string(resourceName))
		// Validate resource quantity.
		allErrs = append(allErrs, ValidateNonnegativeQuantity(quantity, fldPath)...)

		// Check that request <= limit.
		limitQuantity, exists := requirements.Limits[resourceName]
		if exists && quantity.Cmp(limitQuantity) > 0 {
			allErrs = append(allErrs, field.Invalid(reqPath, quantity.String(), fmt.Sprintf("must be less than or equal to %s limit of %s", resourceName, limitQuantity.String())))
		}
	}
	return allErrs
}

// ValidateNonnegativeQuantity Validates that a Quantity is not negative
func ValidateNonnegativeQuantity(value resource.Quantity, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if value.Cmp(resource.Quantity{}) < 0 {
		allErrs = append(allErrs, field.Invalid(fldPath, value.String(), apimachineryvalidation.IsNegativeErrorMsg))
	}
	return allErrs
}

// Validate validates the GameServer configuration.
// If a GameServer is invalid there will be > 0 values in
// the returned array
func (gs *GameServer) Validate(apiHooks APIHooks) field.ErrorList {
	allErrs := validateName(gs, field.NewPath("metadata"))

	// make sure the host port is specified if this is a development server
	devAddress, _ := gs.GetDevAddress()
	allErrs = append(allErrs, gs.Spec.Validate(apiHooks, devAddress, field.NewPath("spec"))...)
	return allErrs
}

// GetDevAddress returns the address for game server.
func (gs *GameServer) GetDevAddress() (string, bool) {
	devAddress, hasDevAddress := gs.ObjectMeta.Annotations[DevAddressAnnotation]
	return devAddress, hasDevAddress
}

// IsDeletable returns false if the server is currently allocated/reserved and is not already in the
// process of being deleted
func (gs *GameServer) IsDeletable() bool {
	if gs.Status.State == GameServerStateAllocated || gs.Status.State == GameServerStateReserved {
		return !gs.ObjectMeta.DeletionTimestamp.IsZero()
	}

	return true
}

// IsBeingDeleted returns true if the server is in the process of being deleted.
func (gs *GameServer) IsBeingDeleted() bool {
	return !gs.ObjectMeta.DeletionTimestamp.IsZero() || gs.Status.State == GameServerStateShutdown
}

// IsBeforeReady returns true if the GameServer Status has yet to move to or past the Ready
// state in its lifecycle, such as Allocated or Reserved, or any of the Error/Unhealthy states
func (gs *GameServer) IsBeforeReady() bool {
	switch gs.Status.State {
	case GameServerStatePortAllocation:
		return true
	case GameServerStateCreating:
		return true
	case GameServerStateStarting:
		return true
	case GameServerStateScheduled:
		return true
	case GameServerStateRequestReady:
		return true
	}

	return false
}

// IsActive returns true if the GameServer status is Ready, Reserved, or Allocated state.
func (gs *GameServer) IsActive() bool {
	switch gs.Status.State {
	case GameServerStateAllocated:
		return true
	case GameServerStateReady:
		return true
	case GameServerStateReserved:
		return true
	}

	return false
}

// HasContainer determines if the GameServerSpec has a container with the specified name.
// Init containers with RestartPolicy `Always` will be considered if `includeSidecar` is true.
func (gss *GameServerSpec) HasContainer(name string, includeSidecar bool) bool {
	for _, c := range gss.Template.Spec.Containers {
		if c.Name == name {
			return true
		}
	}
	if !includeSidecar {
		return false
	}

	for _, c := range gss.Template.Spec.InitContainers {
		if c.RestartPolicy == nil || *c.RestartPolicy != corev1.ContainerRestartPolicyAlways {
			continue
		}
		if c.Name == name {
			return true
		}
	}
	return false
}

// ApplyToPodContainer applies func(v1.Container) to the specified container in the pod.
// Init containers with RestartPolicy Always will also be considered.
// Returns an error if the container is not found.
func (gs *GameServer) ApplyToPodContainer(pod *corev1.Pod, containerName string, f func(corev1.Container) corev1.Container) error {
	for i, c := range pod.Spec.Containers {
		if c.Name == containerName {
			pod.Spec.Containers[i] = f(c)
			return nil
		}
	}
	for i, c := range pod.Spec.InitContainers {
		if c.RestartPolicy == nil || *c.RestartPolicy != corev1.ContainerRestartPolicyAlways {
			continue
		}
		if c.Name == containerName {
			pod.Spec.InitContainers[i] = f(c)
			return nil
		}
	}
	return errors.Errorf("failed to find container named %s in pod spec", containerName)
}

// Pod creates a new Pod from the PodTemplateSpec
// attached to the GameServer resource
func (gs *GameServer) Pod(apiHooks APIHooks, sidecars ...corev1.Container) (*corev1.Pod, error) {
	pod := &corev1.Pod{
		ObjectMeta: *gs.Spec.Template.ObjectMeta.DeepCopy(),
		Spec:       *gs.Spec.Template.Spec.DeepCopy(),
	}

	if len(pod.Spec.Hostname) == 0 {
		// replace . with - since it must match RFC 1123
		pod.Spec.Hostname = strings.ReplaceAll(gs.ObjectMeta.Name, ".", "-")
	}

	gs.podObjectMeta(pod)

	passthroughContainerPortMap := make(map[string][]int)
	for _, p := range gs.Spec.Ports {
		var hostPort int32
		portIdx := 0

		if !runtime.FeatureEnabled(runtime.FeaturePortPolicyNone) || p.PortPolicy != None {
			hostPort = p.HostPort
		}

		cp := corev1.ContainerPort{
			ContainerPort: p.ContainerPort,
			HostPort:      hostPort,
			Protocol:      p.Protocol,
		}
		err := gs.ApplyToPodContainer(pod, *p.Container, func(c corev1.Container) corev1.Container {
			portIdx = len(c.Ports)
			c.Ports = append(c.Ports, cp)

			return c
		})
		if err != nil {
			return nil, err
		}
		if p.PortPolicy == Passthrough {
			passthroughContainerPortMap[*p.Container] = append(passthroughContainerPortMap[*p.Container], portIdx)
		}
	}

	if len(passthroughContainerPortMap) != 0 {
		containerToPassthroughMapJSON, err := json.Marshal(passthroughContainerPortMap)
		if err != nil {
			return nil, err
		}
		pod.ObjectMeta.Annotations[PassthroughPortAssignmentAnnotation] = string(containerToPassthroughMapJSON)
	}

	if runtime.FeatureEnabled(runtime.FeatureSidecarContainers) {
		// make sure all sidecars have a restart policy of Always, so they are valid sidecar containers.
		// https://kubernetes.io/docs/concepts/workloads/pods/sidecar-containers/#sidecar-containers-and-pod-lifecycle
		always := corev1.ContainerRestartPolicyAlways
		for i := range sidecars {
			sidecars[i].RestartPolicy = &always
		}

		// addSidecarsAsInitContainers puts the sidecars in the initContainers list so that they can have their own independent
		// container restart policies.
		pod.Spec.InitContainers = slices.Concat(sidecars, pod.Spec.InitContainers)

		// default GameServer container should also be Restart: Never
		if len(pod.Spec.RestartPolicy) == 0 {
			pod.Spec.RestartPolicy = corev1.RestartPolicyNever
		}
	} else {
		gs.addSidecarsAsContainers(sidecars, pod)
	}

	gs.podScheduling(pod)

	if err := apiHooks.MutateGameServerPod(&gs.Spec, pod); err != nil {
		return nil, err
	}
	if err := apiHooks.SetEviction(gs.Status.Eviction, pod); err != nil {
		return nil, err
	}

	return pod, nil
}

// addSidecarsAsContainers puts the sidecars at the start of the general list of containers so that the kubelet starts
// them first
func (gs *GameServer) addSidecarsAsContainers(sidecars []corev1.Container, pod *corev1.Pod) {
	containers := make([]corev1.Container, 0, len(sidecars)+len(pod.Spec.Containers))
	containers = append(containers, sidecars...)
	containers = append(containers, pod.Spec.Containers...)
	pod.Spec.Containers = containers
}

// podObjectMeta configures the pod ObjectMeta details
func (gs *GameServer) podObjectMeta(pod *corev1.Pod) {
	pod.ObjectMeta.GenerateName = ""
	// Pods inherit the name of their gameserver. It's safe since there's
	// a guarantee that pod won't outlive its parent.
	pod.ObjectMeta.Name = gs.ObjectMeta.Name
	// Pods for GameServers need to stay in the same namespace
	pod.ObjectMeta.Namespace = gs.ObjectMeta.Namespace
	// Make sure these are blank, just in case
	pod.ObjectMeta.ResourceVersion = ""
	pod.ObjectMeta.UID = ""
	if pod.ObjectMeta.Labels == nil {
		pod.ObjectMeta.Labels = make(map[string]string, 2)
	}
	if pod.ObjectMeta.Annotations == nil {
		pod.ObjectMeta.Annotations = make(map[string]string, 2)
	}
	pod.ObjectMeta.Labels[RoleLabel] = GameServerLabelRole
	// store the GameServer name as a label, for easy lookup later on
	pod.ObjectMeta.Labels[GameServerPodLabel] = gs.ObjectMeta.Name
	// store the GameServer container as an annotation, to make lookup at a Pod level easier
	pod.ObjectMeta.Annotations[GameServerContainerAnnotation] = gs.Spec.Container
	ref := metav1.NewControllerRef(gs, SchemeGroupVersion.WithKind("GameServer"))
	pod.ObjectMeta.OwnerReferences = append(pod.ObjectMeta.OwnerReferences, *ref)

	// Add Agones version into Pod Annotations
	pod.ObjectMeta.Annotations[VersionAnnotation] = pkg.Version
}

// podScheduling applies the Fleet scheduling strategy to the passed in Pod
// this sets the a PreferredDuringSchedulingIgnoredDuringExecution for GameServer
// pods to a host topology. Basically doing a half decent job of packing GameServer
// pods together.
func (gs *GameServer) podScheduling(pod *corev1.Pod) {
	if gs.Spec.Scheduling == apis.Packed {
		if pod.Spec.Affinity == nil {
			pod.Spec.Affinity = &corev1.Affinity{}
		}
		if pod.Spec.Affinity.PodAffinity == nil {
			pod.Spec.Affinity.PodAffinity = &corev1.PodAffinity{}
		}

		wpat := corev1.WeightedPodAffinityTerm{
			Weight: 100,
			PodAffinityTerm: corev1.PodAffinityTerm{
				TopologyKey:   "kubernetes.io/hostname",
				LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{RoleLabel: GameServerLabelRole}},
			},
		}

		pod.Spec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(pod.Spec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution, wpat)
	}
}

// DisableServiceAccount disables the service account for the gameserver container
func (gs *GameServer) DisableServiceAccount(pod *corev1.Pod) error {
	// gameservers don't get access to the k8s api.
	emptyVol := corev1.Volume{Name: "empty", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}
	pod.Spec.Volumes = append(pod.Spec.Volumes, emptyVol)
	mount := corev1.VolumeMount{MountPath: "/var/run/secrets/kubernetes.io/serviceaccount", Name: emptyVol.Name, ReadOnly: true}

	return gs.ApplyToPodContainer(pod, gs.Spec.Container, func(c corev1.Container) corev1.Container {
		c.VolumeMounts = append(c.VolumeMounts, mount)

		return c
	})
}

// HasPortPolicy checks if there is a port with a given
// PortPolicy
func (gs *GameServer) HasPortPolicy(policy PortPolicy) bool {
	for _, p := range gs.Spec.Ports {
		if p.PortPolicy == policy {
			return true
		}
	}
	return false
}

// Status returns a GameServerStatusPort for this GameServerPort
func (p GameServerPort) Status() GameServerStatusPort {
	if runtime.FeatureEnabled(runtime.FeaturePortPolicyNone) && p.PortPolicy == None {
		return GameServerStatusPort{Name: p.Name, Port: p.ContainerPort}
	}

	return GameServerStatusPort{Name: p.Name, Port: p.HostPort}
}

// CountPorts returns the number of
// ports that match condition function
func (gs *GameServer) CountPorts(f func(policy PortPolicy) bool) int {
	count := 0
	for _, p := range gs.Spec.Ports {
		if f(p.PortPolicy) {
			count++
		}
	}
	return count
}

// CountPortsForRange returns the number of ports that match condition function and range name.
func (gs *GameServer) CountPortsForRange(name string, f func(policy PortPolicy) bool) int {
	count := 0
	for _, p := range gs.Spec.Ports {
		if p.Range == name && f(p.PortPolicy) {
			count++
		}
	}
	return count
}

// Patch creates a JSONPatch to move the current GameServer to the passed in delta GameServer.
// Returned Patch includes a "test" operation that will cause the GameServers.Patch() operation to
// fail if the Game Server has been updated (ResourceVersion has changed) in between when the Patch
// was created and applied.
func (gs *GameServer) Patch(delta *GameServer) ([]byte, error) {
	var result []byte

	oldJSON, err := json.Marshal(gs)
	if err != nil {
		return result, errors.Wrapf(err, "error marshalling to json current GameServer %s", gs.ObjectMeta.Name)
	}

	newJSON, err := json.Marshal(delta)
	if err != nil {
		return result, errors.Wrapf(err, "error marshalling to json delta GameServer %s", delta.ObjectMeta.Name)
	}

	patch, err := jsonpatch.CreatePatch(oldJSON, newJSON)
	if err != nil {
		return result, errors.Wrapf(err, "error creating patch for GameServer %s", gs.ObjectMeta.Name)
	}

	// Per https://jsonpatch.com/ "Tests that the specified value is set in the document. If the test
	// fails, then the patch as a whole should not apply."
	// Used here to check the object has not been updated (has not changed ResourceVersion).
	patches := []jsonpatch.JsonPatchOperation{{Operation: "test", Path: "/metadata/resourceVersion", Value: gs.ObjectMeta.ResourceVersion}}
	patches = append(patches, patch...)

	result, err = json.Marshal(patches)
	return result, errors.Wrapf(err, "error creating json for patch for GameServer %s", gs.ObjectMeta.Name)
}

// UpdateCount increments or decrements a CounterStatus on a Game Server by the given amount.
func (gs *GameServer) UpdateCount(name string, action string, amount int64) error {
	if !(action == GameServerPriorityIncrement || action == GameServerPriorityDecrement) {
		return errors.Errorf("unable to UpdateCount with Name %s, Action %s, Amount %d. Allocation action must be one of %s or %s", name, action, amount, GameServerPriorityIncrement, GameServerPriorityDecrement)
	}
	if amount < 0 {
		return errors.Errorf("unable to UpdateCount with Name %s, Action %s, Amount %d. Amount must be greater than 0", name, action, amount)
	}
	if counter, ok := gs.Status.Counters[name]; ok {
		cnt := counter.Count
		if action == GameServerPriorityIncrement {
			cnt += amount
		} else {
			cnt -= amount
		}
		// Truncate to Capacity if Count > Capacity
		if cnt > counter.Capacity {
			cnt = counter.Capacity
		}
		// Truncate to Zero if Count is negative
		if cnt < 0 {
			cnt = 0
		}
		counter.Count = cnt
		gs.Status.Counters[name] = counter
		return nil
	}
	return errors.Errorf("unable to UpdateCount with Name %s, Action %s, Amount %d. Counter not found in GameServer %s", name, action, amount, gs.ObjectMeta.GetName())
}

// UpdateCounterCapacity updates the CounterStatus Capacity to the given capacity.
func (gs *GameServer) UpdateCounterCapacity(name string, capacity int64) error {
	if capacity < 0 {
		return errors.Errorf("unable to UpdateCounterCapacity: Name %s, Capacity %d. Capacity must be greater than or equal to 0", name, capacity)
	}
	if counter, ok := gs.Status.Counters[name]; ok {
		counter.Capacity = capacity
		// If Capacity is now less than Count, reset Count here to equal Capacity
		if counter.Count > counter.Capacity {
			counter.Count = counter.Capacity
		}
		gs.Status.Counters[name] = counter
		return nil
	}
	return errors.Errorf("unable to UpdateCounterCapacity: Name %s, Capacity %d. Counter not found in GameServer %s", name, capacity, gs.ObjectMeta.GetName())
}

// UpdateListCapacity updates the ListStatus Capacity to the given capacity.
func (gs *GameServer) UpdateListCapacity(name string, capacity int64) error {

	if capacity < 0 || capacity > ListMaxCapacity {
		return errors.Errorf("unable to UpdateListCapacity: Name %s, Capacity %d. Capacity must be between 0 and 1000, inclusive", name, capacity)
	}
	if list, ok := gs.Status.Lists[name]; ok {
		list.Capacity = capacity
		list.Values = truncateList(list.Capacity, list.Values)
		gs.Status.Lists[name] = list
		return nil
	}
	return errors.Errorf("unable to UpdateListCapacity: Name %s, Capacity %d. List not found in GameServer %s", name, capacity, gs.ObjectMeta.GetName())
}

// AppendListValues adds unique values to the ListStatus Values list.
func (gs *GameServer) AppendListValues(name string, values []string) error {
	if values == nil {
		return errors.Errorf("unable to AppendListValues: Name %s, Values %s. Values must not be nil", name, values)
	}
	if list, ok := gs.Status.Lists[name]; ok {
		mergedList := MergeRemoveDuplicates(list.Values, values)
		// Any duplicate values are silently dropped.
		list.Values = mergedList
		list.Values = truncateList(list.Capacity, list.Values)
		gs.Status.Lists[name] = list
		return nil
	}
	return errors.Errorf("unable to AppendListValues: Name %s, Values %s. List not found in GameServer %s", name, values, gs.ObjectMeta.GetName())
}

// DeleteListValues removes values from the ListStatus Values list. Values in the DeleteListValues
// list that are not in the ListStatus Values list are ignored.
func (gs *GameServer) DeleteListValues(name string, values []string) error {
	if values == nil {
		return errors.Errorf("unable to DeleteListValues: Name %s, Values %s. Values must not be nil", name, values)
	}
	if list, ok := gs.Status.Lists[name]; ok {
		deleteValuesMap := make(map[string]bool)
		for _, value := range values {
			deleteValuesMap[value] = true
		}
		newList := deleteValues(list.Values, deleteValuesMap)
		list.Values = newList
		gs.Status.Lists[name] = list
		return nil
	}
	return errors.Errorf("unable to DeleteListValues: Name %s, Values %s. List not found in GameServer %s", name, values, gs.ObjectMeta.GetName())
}

// deleteValues returns a new list with all the values in valuesList that are not keys in deleteValuesMap.
func deleteValues(valuesList []string, deleteValuesMap map[string]bool) []string {
	newValuesList := []string{}
	for _, value := range valuesList {
		if _, ok := deleteValuesMap[value]; ok {
			continue
		}
		newValuesList = append(newValuesList, value)
	}
	return newValuesList
}

// truncateList truncates the list to the given capacity
func truncateList(capacity int64, list []string) []string {
	if list == nil || len(list) <= int(capacity) {
		return list
	}
	list = append([]string{}, list[:capacity]...)
	return list
}

// MergeRemoveDuplicates merges two lists and removes any duplicate values.
// Maintains ordering, so new values from list2 are appended to the end of list1.
// Returns a new list with unique values only.
func MergeRemoveDuplicates(list1 []string, list2 []string) []string {
	uniqueList := []string{}
	listMap := make(map[string]bool)
	for _, v1 := range list1 {
		if _, ok := listMap[v1]; !ok {
			uniqueList = append(uniqueList, v1)
			listMap[v1] = true
		}
	}
	for _, v2 := range list2 {
		if _, ok := listMap[v2]; !ok {
			uniqueList = append(uniqueList, v2)
			listMap[v2] = true
		}
	}
	return uniqueList
}

// CompareCountAndListPriorities compares two game servers based on a list of CountsAndLists Priorities using available
// capacity as the comparison.
func (gs *GameServer) CompareCountAndListPriorities(priorities []Priority, other *GameServer) *bool {
	for _, priority := range priorities {
		res := gs.compareCountAndListPriority(&priority, other)
		if res != nil {
			// reverse if descending
			if priority.Order == GameServerPriorityDescending {
				flip := !*res
				return &flip
			}

			return res
		}
	}

	return nil
}

// compareCountAndListPriority compares two game servers based on a CountsAndLists Priority using available
// capacity (Capacity - Count for Counters, and Capacity - len(Values) for Lists) as the comparison.
// Returns true if gs1 < gs2; false if gs1 > gs2; nil if gs1 == gs2; nil if neither gamer server has the Priority.
// If only one game server has the Priority, prefer that server. I.e. nil < gsX when Priority
// Order is Descending (3, 2, 1, 0, nil), and nil > gsX when Order is Ascending (0, 1, 2, 3, nil).
func (gs *GameServer) compareCountAndListPriority(p *Priority, other *GameServer) *bool {
	var gs1ok, gs2ok bool
	t := true
	f := false
	switch p.Type {
	case GameServerPriorityCounter:
		// Check if both game servers contain the Counter.
		counter1, ok1 := gs.Status.Counters[p.Key]
		counter2, ok2 := other.Status.Counters[p.Key]
		// If both game servers have the Counter
		if ok1 && ok2 {
			availCapacity1 := counter1.Capacity - counter1.Count
			availCapacity2 := counter2.Capacity - counter2.Count
			if availCapacity1 < availCapacity2 {
				return &t
			}
			if availCapacity1 > availCapacity2 {
				return &f
			}
			if availCapacity1 == availCapacity2 {
				return nil
			}
		}
		gs1ok = ok1
		gs2ok = ok2
	case GameServerPriorityList:
		// Check if both game servers contain the List.
		list1, ok1 := gs.Status.Lists[p.Key]
		list2, ok2 := other.Status.Lists[p.Key]
		// If both game servers have the List
		if ok1 && ok2 {
			availCapacity1 := list1.Capacity - int64(len(list1.Values))
			availCapacity2 := list2.Capacity - int64(len(list2.Values))
			if availCapacity1 < availCapacity2 {
				return &t
			}
			if availCapacity1 > availCapacity2 {
				return &f
			}
			if availCapacity1 == availCapacity2 {
				return nil
			}
		}
		gs1ok = ok1
		gs2ok = ok2
	}
	// If only one game server has the Priority, prefer that server. I.e. nil < gsX when Order is
	// Descending (3, 2, 1, 0, nil), and nil > gsX when Order is Ascending (0, 1, 2, 3, nil).
	if (gs1ok && p.Order == GameServerPriorityDescending) ||
		(gs2ok && p.Order == GameServerPriorityAscending) {
		return &f
	}
	if (gs1ok && p.Order == GameServerPriorityAscending) ||
		(gs2ok && p.Order == GameServerPriorityDescending) {
		return &t
	}
	// If neither game server has the Priority
	return nil
}
