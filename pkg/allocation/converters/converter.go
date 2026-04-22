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

// Package converters includes API conversions between GameServerAllocation API and the Allocation proto APIs.
package converters

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pb "agones.dev/agones/pkg/allocation/go"
	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	"agones.dev/agones/pkg/util/runtime"
)

// ConvertAllocationRequestToGSA converts AllocationRequest to GameServerAllocation V1 (GSA)
func ConvertAllocationRequestToGSA(in *pb.AllocationRequest) *allocationv1.GameServerAllocation {
	if in == nil {
		return nil
	}

	gsa := &allocationv1.GameServerAllocation{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: in.GetNamespace(),
		},
		Spec: allocationv1.GameServerAllocationSpec{
			// nolint:staticcheck
			Preferred:  convertGameServerSelectorsToInternalGameServerSelectors(in.GetPreferredGameServerSelectors()),
			Selectors:  convertGameServerSelectorsToInternalGameServerSelectors(in.GetGameServerSelectors()),
			Scheduling: convertAllocationSchedulingToGSASchedulingStrategy(in.GetScheduling()),
		},
	}

	if in.GetMultiClusterSetting() != nil {
		gsa.Spec.MultiClusterSetting = allocationv1.MultiClusterSetting{
			Enabled: in.GetMultiClusterSetting().GetEnabled(),
		}
		if ls := convertLabelSelectorToInternalLabelSelector(in.GetMultiClusterSetting().GetPolicySelector()); ls != nil {
			gsa.Spec.MultiClusterSetting.PolicySelector = *ls
		}
	}

	// Accept both metadata (preferred) and metapatch until metapatch is fully removed.
	metadata := in.GetMetadata()
	if metadata == nil {
		metadata = in.GetMetaPatch()
	}

	if metadata != nil {
		gsa.Spec.MetaPatch = allocationv1.MetaPatch{
			Labels:      metadata.GetLabels(),
			Annotations: metadata.GetAnnotations(),
		}
	}

	// nolint:staticcheck
	if selector := convertGameServerSelectorToInternalGameServerSelector(in.GetRequiredGameServerSelector()); selector != nil {
		// nolint:staticcheck
		gsa.Spec.Required = *selector
	}

	if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		if in.Priorities != nil {
			gsa.Spec.Priorities = convertAllocationPrioritiesToGSAPriorities(in.GetPriorities())
		}
		if in.Counters != nil {
			gsa.Spec.Counters = convertAllocationCountersToGSACounterActions(in.GetCounters())
		}
		if in.Lists != nil {
			gsa.Spec.Lists = convertAllocationListsToGSAListActions(in.GetLists())
		}
	}

	return gsa
}

// ConvertGSAToAllocationRequest converts AllocationRequest to GameServerAllocation V1 (GSA)
func ConvertGSAToAllocationRequest(in *allocationv1.GameServerAllocation) *pb.AllocationRequest {
	if in == nil {
		return nil
	}

	out := &pb.AllocationRequest{
		Namespace:           in.GetNamespace(),
		Scheduling:          convertGSASchedulingStrategyToAllocationScheduling(in.Spec.Scheduling),
		GameServerSelectors: convertInternalLabelSelectorsToLabelSelectors(in.Spec.Selectors),
		MultiClusterSetting: &pb.MultiClusterSetting{
			Enabled: in.Spec.MultiClusterSetting.Enabled,
		},
		Metadata: &pb.MetaPatch{
			Labels:      in.Spec.MetaPatch.Labels,
			Annotations: in.Spec.MetaPatch.Annotations,
		},
		// MetaPatch is deprecated, but we do a double write here to both metapatch and metadata
		// to ensure that multi-cluster allocation still works when one cluster has the field
		// and another one does not have the field yet.
		MetaPatch: &pb.MetaPatch{
			Labels:      in.Spec.MetaPatch.Labels,
			Annotations: in.Spec.MetaPatch.Annotations,
		},
	}

	l := len(out.GameServerSelectors)
	if l > 0 {
		// nolint:staticcheck
		// Sets all but the last GameServerSelector as PreferredGameServerSelectors
		out.PreferredGameServerSelectors = out.GameServerSelectors[:l-1]
		// nolint:staticcheck
		// Sets the last GameServerSelector as RequiredGameServerSelector
		out.RequiredGameServerSelector = out.GameServerSelectors[l-1]
	}

	if in.Spec.MultiClusterSetting.Enabled {
		out.MultiClusterSetting.PolicySelector = convertInternalLabelSelectorToLabelSelector(&in.Spec.MultiClusterSetting.PolicySelector)
	}

	if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		if in.Spec.Priorities != nil {
			out.Priorities = convertGSAPrioritiesToAllocationPriorities(in.Spec.Priorities)
		}
		if in.Spec.Counters != nil {
			out.Counters = convertGSACounterActionsToAllocationCounters(in.Spec.Counters)
		}
		if in.Spec.Lists != nil {
			out.Lists = convertGSAListActionsToAllocationLists(in.Spec.Lists)
		}
	}

	return out
}

// convertAllocationSchedulingToGSASchedulingStrategy converts AllocationRequest_SchedulingStrategy to apis.SchedulingStrategy
func convertAllocationSchedulingToGSASchedulingStrategy(in pb.AllocationRequest_SchedulingStrategy) apis.SchedulingStrategy {
	switch in {
	case pb.AllocationRequest_Packed:
		return apis.Packed
	case pb.AllocationRequest_Distributed:
		return apis.Distributed
	}
	return apis.Packed
}

// convertGSASchedulingStrategyToAllocationScheduling converts  apis.SchedulingStrategy to pb.AllocationRequest_SchedulingStrategy
func convertGSASchedulingStrategyToAllocationScheduling(in apis.SchedulingStrategy) pb.AllocationRequest_SchedulingStrategy {
	switch in {
	case apis.Packed:
		return pb.AllocationRequest_Packed
	case apis.Distributed:
		return pb.AllocationRequest_Distributed
	}
	return pb.AllocationRequest_Packed
}

func convertLabelSelectorToInternalLabelSelector(in *pb.LabelSelector) *metav1.LabelSelector {
	if in == nil {
		return nil
	}
	return &metav1.LabelSelector{MatchLabels: in.GetMatchLabels()}
}

func convertGameServerSelectorToInternalGameServerSelector(in *pb.GameServerSelector) *allocationv1.GameServerSelector {
	if in == nil {
		return nil
	}
	result := &allocationv1.GameServerSelector{
		LabelSelector: metav1.LabelSelector{MatchLabels: in.GetMatchLabels()},
	}

	switch in.GameServerState {
	case pb.GameServerSelector_ALLOCATED:
		allocated := agonesv1.GameServerStateAllocated
		result.GameServerState = &allocated
	case pb.GameServerSelector_READY:
		ready := agonesv1.GameServerStateReady
		result.GameServerState = &ready
	}

	if runtime.FeatureEnabled(runtime.FeaturePlayerAllocationFilter) && in.Players != nil {
		result.Players = &allocationv1.PlayerSelector{
			MinAvailable: int64(in.Players.MinAvailable),
			MaxAvailable: int64(in.Players.MaxAvailable),
		}
	}

	if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		if in.Counters != nil {
			result.Counters = map[string]allocationv1.CounterSelector{}
			for k, v := range in.GetCounters() {
				result.Counters[k] = allocationv1.CounterSelector{
					MinCount:     v.MinCount,
					MaxCount:     v.MaxCount,
					MinAvailable: v.MinAvailable,
					MaxAvailable: v.MaxAvailable,
				}
			}
		}
		if in.Lists != nil {
			result.Lists = map[string]allocationv1.ListSelector{}
			for k, v := range in.GetLists() {
				result.Lists[k] = allocationv1.ListSelector{
					ContainsValue: v.ContainsValue,
					MinAvailable:  v.MinAvailable,
					MaxAvailable:  v.MaxAvailable,
				}
			}
		}
	}

	return result
}

func convertInternalGameServerSelectorToGameServer(in *allocationv1.GameServerSelector) *pb.GameServerSelector {
	if in == nil {
		return nil
	}
	result := &pb.GameServerSelector{
		MatchLabels: in.MatchLabels,
	}

	if in.GameServerState != nil {
		switch *in.GameServerState {
		case agonesv1.GameServerStateReady:
			result.GameServerState = pb.GameServerSelector_READY
		case agonesv1.GameServerStateAllocated:
			result.GameServerState = pb.GameServerSelector_ALLOCATED
		}
	}

	if runtime.FeatureEnabled(runtime.FeaturePlayerAllocationFilter) && in.Players != nil {
		result.Players = &pb.PlayerSelector{
			MinAvailable: uint64(in.Players.MinAvailable),
			MaxAvailable: uint64(in.Players.MaxAvailable),
		}
	}

	if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		if in.Counters != nil {
			result.Counters = map[string]*pb.CounterSelector{}
			for k, v := range in.Counters {
				result.Counters[k] = &pb.CounterSelector{
					MinCount:     v.MinCount,
					MaxCount:     v.MaxCount,
					MinAvailable: v.MinAvailable,
					MaxAvailable: v.MaxAvailable,
				}
			}
		}
		if in.Lists != nil {
			result.Lists = map[string]*pb.ListSelector{}
			for k, v := range in.Lists {
				result.Lists[k] = &pb.ListSelector{
					ContainsValue: v.ContainsValue,
					MinAvailable:  v.MinAvailable,
					MaxAvailable:  v.MaxAvailable,
				}
			}
		}
	}

	return result
}

func convertInternalLabelSelectorToLabelSelector(in *metav1.LabelSelector) *pb.LabelSelector {
	if in == nil {
		return nil
	}
	return &pb.LabelSelector{MatchLabels: in.MatchLabels}
}

func convertInternalLabelSelectorsToLabelSelectors(in []allocationv1.GameServerSelector) []*pb.GameServerSelector {
	var result []*pb.GameServerSelector
	for _, l := range in {
		c := convertInternalGameServerSelectorToGameServer(&l)
		result = append(result, c)
	}
	return result
}

func convertGameServerSelectorsToInternalGameServerSelectors(in []*pb.GameServerSelector) []allocationv1.GameServerSelector {
	var result []allocationv1.GameServerSelector
	for _, l := range in {
		if selector := convertGameServerSelectorToInternalGameServerSelector(l); selector != nil {
			result = append(result, *selector)
		}
	}
	return result
}

// ConvertGSAToAllocationResponse converts GameServerAllocation V1 (GSA) to AllocationResponse
func ConvertGSAToAllocationResponse(in *allocationv1.GameServerAllocation, grpcUnallocatedStatusCode codes.Code) (*pb.AllocationResponse, error) {
	if in == nil {
		return nil, nil
	}

	if err := convertStateV1ToError(in.Status.State, grpcUnallocatedStatusCode); err != nil {
		return nil, err
	}

	res := &pb.AllocationResponse{
		GameServerName: in.Status.GameServerName,
		Address:        in.Status.Address,
		Addresses:      convertGSAAddressesToAllocationAddresses(in.Status.Addresses),
		NodeName:       in.Status.NodeName,
		Ports:          convertGSAAgonesPortsToAllocationPorts(in.Status.Ports),
		Source:         in.Status.Source,
		Metadata:       convertGSAMetadataToAllocationMetadata(in.Status.Metadata),
	}
	if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		if in.Status.Counters != nil {
			res.Counters = convertGSACountersToAllocationCounters(in.Status.Counters)
		}
		if in.Status.Lists != nil {
			res.Lists = convertGSAListsToAllocationLists(in.Status.Lists)
		}
	}

	return res, nil
}

// convertGSACountersToAllocationCounters converts a map of GameServerStatusCounter to AllocationResponse_CounterStatus
func convertGSACountersToAllocationCounters(in map[string]agonesv1.CounterStatus) map[string]*pb.AllocationResponse_CounterStatus {
	out := map[string]*pb.AllocationResponse_CounterStatus{}
	for k, v := range in {
		out[k] = &pb.AllocationResponse_CounterStatus{
			Count:    wrapperspb.Int64(v.Count),
			Capacity: wrapperspb.Int64(v.Capacity),
		}
	}
	return out
}

// convertGSAListsToAllocationLists converts a map of GameServerStatusList to AllocationResponse_ListStatus
func convertGSAListsToAllocationLists(in map[string]agonesv1.ListStatus) map[string]*pb.AllocationResponse_ListStatus {
	out := map[string]*pb.AllocationResponse_ListStatus{}
	for k, v := range in {
		out[k] = &pb.AllocationResponse_ListStatus{
			Values:   v.Values,
			Capacity: wrapperspb.Int64(v.Capacity),
		}
	}
	return out
}

// ConvertAllocationResponseToGSA converts AllocationResponse to GameServerAllocation V1 (GSA)
func ConvertAllocationResponseToGSA(in *pb.AllocationResponse, rs string) *allocationv1.GameServerAllocation {
	if in == nil {
		return nil
	}

	out := &allocationv1.GameServerAllocation{
		Status: allocationv1.GameServerAllocationStatus{
			State:          allocationv1.GameServerAllocationAllocated,
			GameServerName: in.GameServerName,
			Address:        in.Address,
			Addresses:      convertAllocationAddressesToGSAAddresses(in.Addresses),
			NodeName:       in.NodeName,
			Ports:          convertAllocationPortsToGSAAgonesPorts(in.Ports),
			Source:         rs,
			Metadata:       convertAllocationMetadataToGSAMetadata(in.Metadata),
		},
	}
	if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		if in.Counters != nil {
			out.Status.Counters = convertAllocationCountersToGSACounters(in.Counters)
		}
		if in.Lists != nil {
			out.Status.Lists = convertAllocationListsToGSALists(in.Lists)
		}
	}
	out.SetGroupVersionKind(allocationv1.SchemeGroupVersion.WithKind("GameServerAllocation"))

	return out
}

// convertGSAAddressesToAllocationAddresses converts corev1.NodeAddress to AllocationResponse_GameServerStatusAddress
func convertGSAAddressesToAllocationAddresses(in []corev1.NodeAddress) []*pb.AllocationResponse_GameServerStatusAddress {
	var addresses []*pb.AllocationResponse_GameServerStatusAddress
	for _, addr := range in {
		addresses = append(addresses, &pb.AllocationResponse_GameServerStatusAddress{
			Type:    string(addr.Type),
			Address: addr.Address,
		})
	}
	return addresses
}

// convertAllocationAddressesToGSAAddresses converts AllocationResponse_GameServerStatusAddress to corev1.NodeAddress
func convertAllocationAddressesToGSAAddresses(in []*pb.AllocationResponse_GameServerStatusAddress) []corev1.NodeAddress {
	var addresses []corev1.NodeAddress
	for _, addr := range in {
		addresses = append(addresses, corev1.NodeAddress{
			Type:    corev1.NodeAddressType(addr.Type),
			Address: addr.Address,
		})
	}
	return addresses
}

// convertGSAAgonesPortsToAllocationPorts converts GameServerStatusPort V1 (GSA) to AllocationResponse_GameServerStatusPort
func convertGSAAgonesPortsToAllocationPorts(in []agonesv1.GameServerStatusPort) []*pb.AllocationResponse_GameServerStatusPort {
	var pbPorts []*pb.AllocationResponse_GameServerStatusPort
	for _, port := range in {
		pbPort := &pb.AllocationResponse_GameServerStatusPort{
			Name: port.Name,
			Port: port.Port,
		}
		pbPorts = append(pbPorts, pbPort)
	}
	return pbPorts
}

// convertAllocationPortsToGSAAgonesPorts converts AllocationResponse_GameServerStatusPort to GameServerStatusPort V1 (GSA)
func convertAllocationPortsToGSAAgonesPorts(in []*pb.AllocationResponse_GameServerStatusPort) []agonesv1.GameServerStatusPort {
	var out []agonesv1.GameServerStatusPort
	for _, port := range in {
		p := &agonesv1.GameServerStatusPort{
			Name: port.Name,
			Port: port.Port,
		}
		out = append(out, *p)
	}
	return out
}

func convertGSAMetadataToAllocationMetadata(in *allocationv1.GameServerMetadata) *pb.AllocationResponse_GameServerMetadata {
	if in == nil {
		return nil
	}
	metadata := &pb.AllocationResponse_GameServerMetadata{}
	metadata.Labels = in.Labels
	metadata.Annotations = in.Annotations
	return metadata
}

func convertAllocationMetadataToGSAMetadata(in *pb.AllocationResponse_GameServerMetadata) *allocationv1.GameServerMetadata {
	if in == nil {
		return nil
	}
	metadata := &allocationv1.GameServerMetadata{}
	metadata.Labels = in.Labels
	metadata.Annotations = in.Annotations
	return metadata
}

func convertAllocationCountersToGSACounters(in map[string]*pb.AllocationResponse_CounterStatus) map[string]agonesv1.CounterStatus {
	out := map[string]agonesv1.CounterStatus{}
	for k, v := range in {
		out[k] = agonesv1.CounterStatus{
			Count:    v.Count.GetValue(),
			Capacity: v.Capacity.GetValue(),
		}
	}
	return out
}

func convertAllocationListsToGSALists(in map[string]*pb.AllocationResponse_ListStatus) map[string]agonesv1.ListStatus {
	out := map[string]agonesv1.ListStatus{}
	for k, v := range in {
		out[k] = agonesv1.ListStatus{
			Values:   v.Values,
			Capacity: v.Capacity.GetValue(),
		}
	}
	return out
}

// convertStateV1ToError converts GameServerAllocationState V1 (GSA) to AllocationResponse_GameServerAllocationState
func convertStateV1ToError(in allocationv1.GameServerAllocationState, grpcUnallocatedStatusCode codes.Code) error {

	switch in {
	case allocationv1.GameServerAllocationAllocated:
		return nil
	case allocationv1.GameServerAllocationUnAllocated:
		return status.Error(grpcUnallocatedStatusCode, "there is no available GameServer to allocate")
	case allocationv1.GameServerAllocationContention:
		return status.Error(codes.Aborted, "too many concurrent requests have overwhelmed the system")
	}
	return status.Error(codes.Unknown, "unknown issue")
}

// convertAllocationPrioritiesToGSAPriorities converts a list of AllocationRequest_Priorities to a
// list of GameServerAllocationSpec (GSA.Spec) Priorities
func convertAllocationPrioritiesToGSAPriorities(in []*pb.Priority) []agonesv1.Priority {
	var out []agonesv1.Priority
	for _, p := range in {
		var t string
		var o string
		switch p.Type {
		case pb.Priority_List:
			t = agonesv1.GameServerPriorityList
		default: // case pb.Priority_Counter and case nil
			t = agonesv1.GameServerPriorityCounter
		}
		switch p.Order {
		case pb.Priority_Descending:
			o = agonesv1.GameServerPriorityDescending
		default: // case pb.Priority_Ascending and case nil
			o = agonesv1.GameServerPriorityAscending
		}
		priority := agonesv1.Priority{
			Type:  t,
			Key:   p.Key,
			Order: o,
		}
		out = append(out, priority)
	}
	return out
}

// convertAllocationPrioritiesToGSAPriorities converts a list of GameServerAllocationSpec (GSA.Spec)
// Priorities to a list of AllocationRequest_Priorities
func convertGSAPrioritiesToAllocationPriorities(in []agonesv1.Priority) []*pb.Priority {
	var out []*pb.Priority
	for _, p := range in {
		var pt pb.Priority_Type
		var po pb.Priority_Order
		switch p.Type {
		case agonesv1.GameServerPriorityList:
			pt = pb.Priority_List
		default: // case agonesv1.GameServerPriorityCounter and case nil
			pt = pb.Priority_Counter
		}
		switch p.Order {
		case agonesv1.GameServerPriorityDescending:
			po = pb.Priority_Descending
		default: // case agonesv1.GameServerPriorityAscending and case nil
			po = pb.Priority_Ascending
		}
		priority := pb.Priority{
			Type:  pt,
			Key:   p.Key,
			Order: po,
		}
		out = append(out, &priority)
	}
	return out
}

// convertAllocationCountersToGSACounterActions converts a map of AllocationRequest_Counters to a
// map of GameServerAllocationSpec CounterActions
func convertAllocationCountersToGSACounterActions(in map[string]*pb.CounterAction) map[string]allocationv1.CounterAction {
	out := map[string]allocationv1.CounterAction{}
	for k, v := range in {
		ca := allocationv1.CounterAction{}

		if v.Action != nil {
			action := v.Action.GetValue()
			ca.Action = &action
		}
		if v.Amount != nil {
			amount := v.Amount.GetValue()
			ca.Amount = &amount
		}
		if v.Capacity != nil {
			capacity := v.Capacity.GetValue()
			ca.Capacity = &capacity
		}

		out[k] = ca
	}
	return out
}

// convertGSACounterActionsToAllocationCounters converts a map of GameServerAllocationSpec CounterActions
// to a map of AllocationRequest_Counters
func convertGSACounterActionsToAllocationCounters(in map[string]allocationv1.CounterAction) map[string]*pb.CounterAction {
	out := map[string]*pb.CounterAction{}

	for k, v := range in {
		ca := pb.CounterAction{}

		if v.Action != nil {
			ca.Action = wrapperspb.String(*v.Action)
		}
		if v.Amount != nil {
			ca.Amount = wrapperspb.Int64(*v.Amount)
		}
		if v.Capacity != nil {
			ca.Capacity = wrapperspb.Int64(*v.Capacity)
		}

		out[k] = &ca
	}
	return out
}

// convertAllocationListsToGSAListActions converts a map of AllocationRequest_Lists to a
// map of GameServerAllocationSpec ListActions
func convertAllocationListsToGSAListActions(in map[string]*pb.ListAction) map[string]allocationv1.ListAction {
	out := map[string]allocationv1.ListAction{}

	for k, v := range in {
		la := allocationv1.ListAction{}

		if v.AddValues != nil {
			addValues := v.GetAddValues()
			copyValues := make([]string, len(addValues))
			copy(copyValues, addValues)
			la.AddValues = copyValues
		}
		if v.DeleteValues != nil {
			deleteValues := v.GetDeleteValues()
			copyValues := make([]string, len(deleteValues))
			copy(copyValues, deleteValues)
			la.DeleteValues = copyValues
		}
		if v.Capacity != nil {
			capacity := v.Capacity.GetValue()
			la.Capacity = &capacity
		}

		out[k] = la
	}

	return out
}

// convertGSAListActionsToAllocationLists converts a map of GameServerAllocationSpec ListActions
// to a map of AllocationRequest_Lists
func convertGSAListActionsToAllocationLists(in map[string]allocationv1.ListAction) map[string]*pb.ListAction {
	out := map[string]*pb.ListAction{}

	for k, v := range in {
		la := pb.ListAction{}

		if v.AddValues != nil {
			copyValues := make([]string, len(v.AddValues))
			copy(copyValues, v.AddValues)
			la.AddValues = copyValues
		}
		if v.DeleteValues != nil {
			copyValues := make([]string, len(v.DeleteValues))
			copy(copyValues, v.DeleteValues)
			la.DeleteValues = copyValues
		}
		if v.Capacity != nil {
			la.Capacity = wrapperspb.Int64(*v.Capacity)
		}

		out[k] = &la
	}
	return out
}
