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

package sdkserver

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/mennanov/fmutils"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
	"k8s.io/apimachinery/pkg/util/yaml"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/sdk/alpha"
	"agones.dev/agones/pkg/sdk/beta"
	"agones.dev/agones/pkg/util/runtime"
)

var (
	_ sdk.SDKServer   = &LocalSDKServer{}
	_ alpha.SDKServer = &LocalSDKServer{}
	_ beta.SDKServer  = &LocalSDKServer{}
)

func defaultGs() *sdk.GameServer {
	gs := &sdk.GameServer{
		ObjectMeta: &sdk.GameServer_ObjectMeta{
			Name:              "local",
			Namespace:         "default",
			Uid:               "1234",
			Generation:        1,
			ResourceVersion:   "v1",
			CreationTimestamp: time.Now().Unix(),
			Labels:            map[string]string{"islocal": "true"},
			Annotations:       map[string]string{"annotation": "true"},
		},
		Spec: &sdk.GameServer_Spec{
			Health: &sdk.GameServer_Spec_Health{
				Disabled:            false,
				PeriodSeconds:       3,
				FailureThreshold:    5,
				InitialDelaySeconds: 10,
			},
		},
		Status: &sdk.GameServer_Status{
			State:   "Ready",
			Address: "127.0.0.1",
			Ports:   []*sdk.GameServer_Status_Port{{Name: "default", Port: 7777}},
		},
	}

	if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		gs.Status.Counters = map[string]*sdk.GameServer_Status_CounterStatus{
			"rooms": {Count: 1, Capacity: 10},
		}
		gs.Status.Lists = map[string]*sdk.GameServer_Status_ListStatus{
			"players": {Values: []string{"test0", "test1", "test2"}, Capacity: 100},
		}
	}

	return gs
}

// LocalSDKServer type is the SDKServer implementation for when the sidecar
// is being run for local development, and doesn't connect to the
// Kubernetes cluster
//
//nolint:govet // ignore fieldalignment, singleton
type LocalSDKServer struct {
	gsMutex           sync.RWMutex
	gs                *sdk.GameServer
	logger            *logrus.Entry
	update            chan struct{}
	updateObservers   sync.Map
	testMutex         sync.Mutex
	requestSequence   []string
	expectedSequence  []string
	gsState           agonesv1.GameServerState
	gsReserveDuration *time.Duration
	reserveTimer      *time.Timer
	testMode          bool
	testSdkName       string
}

// NewLocalSDKServer returns the default LocalSDKServer
func NewLocalSDKServer(filePath string, testSdkName string) (*LocalSDKServer, error) {
	l := &LocalSDKServer{
		gsMutex:         sync.RWMutex{},
		gs:              defaultGs(),
		update:          make(chan struct{}),
		updateObservers: sync.Map{},
		testMutex:       sync.Mutex{},
		requestSequence: make([]string, 0),
		testMode:        false,
		testSdkName:     testSdkName,
		gsState:         agonesv1.GameServerStateScheduled,
	}
	l.logger = runtime.NewLoggerWithType(l)

	if filePath != "" {
		err := l.setGameServerFromFilePath(filePath)
		if err != nil {
			return l, err
		}

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return l, err
		}

		go func() {
			for event := range watcher.Events {
				if event.Op != fsnotify.Write {
					continue
				}
				l.logger.WithField("event", event).Info("File has been changed!")
				err := l.setGameServerFromFilePath(filePath)
				if err != nil {
					l.logger.WithError(err).Error("error setting GameServer from file")
					continue
				}
				l.logger.Info("Sending watched GameServer!")
				l.update <- struct{}{}
			}
		}()

		err = watcher.Add(filePath)
		if err != nil {
			l.logger.WithError(err).WithField("filePath", filePath).Error("error adding watcher")
		}
	}
	if runtime.FeatureEnabled(runtime.FeaturePlayerTracking) && l.gs.Status.Players == nil {
		l.gs.Status.Players = &sdk.GameServer_Status_PlayerStatus{}
	}

	if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		if l.gs.Status.Counters == nil {
			l.gs.Status.Counters = make(map[string]*sdk.GameServer_Status_CounterStatus)
		}
		if l.gs.Status.Lists == nil {
			l.gs.Status.Lists = make(map[string]*sdk.GameServer_Status_ListStatus)
		}
	}

	go func() {
		for value := range l.update {
			l.logger.Info("Gameserver update received")
			l.updateObservers.Range(func(observer, _ interface{}) bool {
				observer.(chan struct{}) <- value
				return true
			})
		}
	}()

	return l, nil
}

// GenerateUID - generate gameserver UID at random for testing
func (l *LocalSDKServer) GenerateUID() {
	// Generating Random UID
	seededRand := rand.New(
		rand.NewSource(time.Now().UnixNano()))
	UID := fmt.Sprintf("%d", seededRand.Int())
	l.gs.ObjectMeta.Uid = UID
}

// SetTestMode set test mode to collect the sequence of performed requests
func (l *LocalSDKServer) SetTestMode(testMode bool) {
	l.testMode = testMode
}

// SetExpectedSequence set expected request sequence which would be
// verified against after run was completed
func (l *LocalSDKServer) SetExpectedSequence(sequence []string) {
	l.expectedSequence = sequence
}

// SetSdkName set SDK name to be added to the logs
func (l *LocalSDKServer) SetSdkName(sdkName string) {
	l.testSdkName = sdkName
	l.logger = l.logger.WithField("sdkName", l.testSdkName)
}

// recordRequest append request name to slice
func (l *LocalSDKServer) recordRequest(request string) {
	if l.testMode {
		l.testMutex.Lock()
		defer l.testMutex.Unlock()
		l.requestSequence = append(l.requestSequence, request)
	}
	if l.testSdkName != "" {
		l.logger.Debugf("Received %s request", request)
	}
}

// recordRequestWithValue append request name to slice only if
// value equals to objMetaField: creationTimestamp or UID
func (l *LocalSDKServer) recordRequestWithValue(request string, value string, objMetaField string) {
	if l.testMode {
		fieldVal := ""
		switch objMetaField {
		case "CreationTimestamp":
			fieldVal = strconv.FormatInt(l.gs.ObjectMeta.CreationTimestamp, 10)
		case "UID":
			fieldVal = l.gs.ObjectMeta.Uid
		case "PlayerCapacity":
			if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
				return
			}
			fieldVal = strconv.FormatInt(l.gs.Status.Players.Capacity, 10)
		case "PlayerIDs":
			if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
				return
			}
			fieldVal = strings.Join(l.gs.Status.Players.Ids, ",")
		default:
			l.logger.Error("unexpected Field to compare")
		}

		if value == fieldVal {
			l.testMutex.Lock()
			defer l.testMutex.Unlock()
			l.requestSequence = append(l.requestSequence, request)
		} else {
			l.logger.Errorf("expected to receive '%s' as value for '%s' request but received '%s'", fieldVal, request, value)
		}
	}
}

func (l *LocalSDKServer) updateState(newState agonesv1.GameServerState) {
	l.gsState = newState
	l.gs.Status.State = string(l.gsState)
}

// Ready logs that the Ready request has been received
func (l *LocalSDKServer) Ready(context.Context, *sdk.Empty) (*sdk.Empty, error) {
	l.logger.Info("Ready request has been received!")
	l.recordRequest("ready")
	l.gsMutex.Lock()
	defer l.gsMutex.Unlock()

	// Follow the GameServer state diagram
	l.updateState(agonesv1.GameServerStateReady)
	l.stopReserveTimer()
	l.update <- struct{}{}
	return &sdk.Empty{}, nil
}

// Allocate logs that an allocate request has been received
func (l *LocalSDKServer) Allocate(context.Context, *sdk.Empty) (*sdk.Empty, error) {
	l.logger.Info("Allocate request has been received!")
	l.recordRequest("allocate")
	l.gsMutex.Lock()
	defer l.gsMutex.Unlock()
	l.updateState(agonesv1.GameServerStateAllocated)
	l.stopReserveTimer()
	l.update <- struct{}{}

	return &sdk.Empty{}, nil
}

// Shutdown logs that the shutdown request has been received
func (l *LocalSDKServer) Shutdown(context.Context, *sdk.Empty) (*sdk.Empty, error) {
	l.logger.Info("Shutdown request has been received!")
	l.recordRequest("shutdown")
	l.gsMutex.Lock()
	defer l.gsMutex.Unlock()
	l.updateState(agonesv1.GameServerStateShutdown)
	l.stopReserveTimer()
	l.update <- struct{}{}
	return &sdk.Empty{}, nil
}

// Health logs each health ping that comes down the stream
func (l *LocalSDKServer) Health(stream sdk.SDK_HealthServer) error {
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			l.logger.Info("Health stream closed.")
			return stream.SendAndClose(&sdk.Empty{})
		}
		if err != nil {
			return errors.Wrap(err, "Error with Health check")
		}
		l.recordRequest("health")
		l.logger.Info("Health Ping Received!")
	}
}

// SetLabel applies a Label to the backing GameServer metadata
func (l *LocalSDKServer) SetLabel(_ context.Context, kv *sdk.KeyValue) (*sdk.Empty, error) {
	l.logger.WithField("values", kv).Info("Setting label")
	l.gsMutex.Lock()
	defer l.gsMutex.Unlock()

	if l.gs.ObjectMeta == nil {
		l.gs.ObjectMeta = &sdk.GameServer_ObjectMeta{}
	}
	if l.gs.ObjectMeta.Labels == nil {
		l.gs.ObjectMeta.Labels = map[string]string{}
	}

	l.gs.ObjectMeta.Labels[metadataPrefix+kv.Key] = kv.Value
	l.update <- struct{}{}
	l.recordRequestWithValue("setlabel", kv.Value, "CreationTimestamp")
	return &sdk.Empty{}, nil
}

// SetAnnotation applies a Annotation to the backing GameServer metadata
func (l *LocalSDKServer) SetAnnotation(_ context.Context, kv *sdk.KeyValue) (*sdk.Empty, error) {
	l.logger.WithField("values", kv).Info("Setting annotation")
	l.gsMutex.Lock()
	defer l.gsMutex.Unlock()

	if l.gs.ObjectMeta == nil {
		l.gs.ObjectMeta = &sdk.GameServer_ObjectMeta{}
	}
	if l.gs.ObjectMeta.Annotations == nil {
		l.gs.ObjectMeta.Annotations = map[string]string{}
	}

	l.gs.ObjectMeta.Annotations[metadataPrefix+kv.Key] = kv.Value
	l.update <- struct{}{}
	l.recordRequestWithValue("setannotation", kv.Value, "UID")
	return &sdk.Empty{}, nil
}

// GetGameServer returns current GameServer configuration.
func (l *LocalSDKServer) GetGameServer(context.Context, *sdk.Empty) (*sdk.GameServer, error) {
	l.logger.Info("Getting GameServer details")
	l.recordRequest("gameserver")
	l.gsMutex.RLock()
	defer l.gsMutex.RUnlock()
	return l.gs, nil
}

// WatchGameServer will return current GameServer configuration, 3 times, every 5 seconds
func (l *LocalSDKServer) WatchGameServer(_ *sdk.Empty, stream sdk.SDK_WatchGameServerServer) error {
	l.logger.Info("Connected to watch GameServer...")
	observer := make(chan struct{}, 1)

	defer func() {
		l.updateObservers.Delete(observer)
	}()

	l.updateObservers.Store(observer, true)

	l.recordRequest("watch")

	// send initial game server state
	observer <- struct{}{}

	for range observer {
		l.gsMutex.RLock()
		err := stream.Send(l.gs)
		l.gsMutex.RUnlock()
		if err != nil {
			l.logger.WithError(err).Error("error sending gameserver")
			return err
		}
	}

	return nil
}

// Reserve moves this GameServer to the Reserved state for the Duration specified
func (l *LocalSDKServer) Reserve(ctx context.Context, d *sdk.Duration) (*sdk.Empty, error) {
	l.logger.WithField("duration", d).Info("Reserve request has been received!")
	l.recordRequest("reserve")
	l.gsMutex.Lock()
	defer l.gsMutex.Unlock()
	if d.Seconds > 0 {
		duration := time.Duration(d.Seconds) * time.Second
		l.gsReserveDuration = &duration
		l.resetReserveAfter(ctx, *l.gsReserveDuration)
	}

	l.updateState(agonesv1.GameServerStateReserved)
	l.update <- struct{}{}

	return &sdk.Empty{}, nil
}

func (l *LocalSDKServer) resetReserveAfter(ctx context.Context, duration time.Duration) {
	if l.reserveTimer != nil {
		l.reserveTimer.Stop()
	}

	l.reserveTimer = time.AfterFunc(duration, func() {
		if _, err := l.Ready(ctx, &sdk.Empty{}); err != nil {
			l.logger.WithError(err).Error("error returning to Ready after reserved ")
		}
	})
}

func (l *LocalSDKServer) stopReserveTimer() {
	if l.reserveTimer != nil {
		l.reserveTimer.Stop()
	}
	l.gsReserveDuration = nil
}

// PlayerConnect should be called when a player connects.
// [Stage:Alpha]
// [FeatureFlag:PlayerTracking]
func (l *LocalSDKServer) PlayerConnect(_ context.Context, id *alpha.PlayerID) (*alpha.Bool, error) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		return &alpha.Bool{Bool: false}, errors.Errorf("%s not enabled", runtime.FeaturePlayerTracking)
	}
	l.logger.WithField("playerID", id.PlayerID).Info("Player Connected")
	l.gsMutex.Lock()
	defer l.gsMutex.Unlock()

	if l.gs.Status.Players == nil {
		l.gs.Status.Players = &sdk.GameServer_Status_PlayerStatus{}
	}

	// the player is already connected, return false.
	for _, playerID := range l.gs.Status.Players.Ids {
		if playerID == id.PlayerID {
			return &alpha.Bool{Bool: false}, nil
		}
	}

	if l.gs.Status.Players.Count >= l.gs.Status.Players.Capacity {
		return &alpha.Bool{Bool: false}, errors.New("Players are already at capacity")
	}

	l.gs.Status.Players.Ids = append(l.gs.Status.Players.Ids, id.PlayerID)
	l.gs.Status.Players.Count = int64(len(l.gs.Status.Players.Ids))

	l.update <- struct{}{}
	l.recordRequestWithValue("playerconnect", "1234", "PlayerIDs")
	return &alpha.Bool{Bool: true}, nil
}

// PlayerDisconnect should be called when a player disconnects.
// [Stage:Alpha]
// [FeatureFlag:PlayerTracking]
func (l *LocalSDKServer) PlayerDisconnect(_ context.Context, id *alpha.PlayerID) (*alpha.Bool, error) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		return &alpha.Bool{Bool: false}, errors.Errorf("%s not enabled", runtime.FeaturePlayerTracking)
	}
	l.logger.WithField("playerID", id.PlayerID).Info("Player Disconnected")
	l.gsMutex.Lock()
	defer l.gsMutex.Unlock()

	if l.gs.Status.Players == nil {
		l.gs.Status.Players = &sdk.GameServer_Status_PlayerStatus{}
	}

	found := -1
	for i, playerID := range l.gs.Status.Players.Ids {
		if playerID == id.PlayerID {
			found = i
			break
		}
	}
	if found == -1 {
		return &alpha.Bool{Bool: false}, nil
	}

	l.gs.Status.Players.Ids = append(l.gs.Status.Players.Ids[:found], l.gs.Status.Players.Ids[found+1:]...)
	l.gs.Status.Players.Count = int64(len(l.gs.Status.Players.Ids))

	l.update <- struct{}{}
	l.recordRequestWithValue("playerdisconnect", "", "PlayerIDs")
	return &alpha.Bool{Bool: true}, nil
}

// IsPlayerConnected returns if the playerID is currently connected to the GameServer.
// [Stage:Alpha]
// [FeatureFlag:PlayerTracking]
func (l *LocalSDKServer) IsPlayerConnected(_ context.Context, id *alpha.PlayerID) (*alpha.Bool, error) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		return &alpha.Bool{Bool: false}, errors.Errorf("%s not enabled", runtime.FeaturePlayerTracking)
	}

	result := &alpha.Bool{Bool: false}
	l.logger.WithField("playerID", id.PlayerID).Info("Is a Player Connected?")
	l.gsMutex.Lock()
	defer l.gsMutex.Unlock()

	l.recordRequestWithValue("isplayerconnected", id.PlayerID, "PlayerIDs")

	if l.gs.Status.Players == nil {
		return result, nil
	}

	for _, playerID := range l.gs.Status.Players.Ids {
		if id.PlayerID == playerID {
			result.Bool = true
			break
		}
	}

	return result, nil
}

// GetConnectedPlayers returns the list of the currently connected player ids.
// [Stage:Alpha]
// [FeatureFlag:PlayerTracking]
func (l *LocalSDKServer) GetConnectedPlayers(_ context.Context, _ *alpha.Empty) (*alpha.PlayerIDList, error) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		return nil, errors.Errorf("%s not enabled", runtime.FeaturePlayerTracking)
	}
	l.logger.Info("Getting Connected Players")

	result := &alpha.PlayerIDList{List: []string{}}

	l.gsMutex.Lock()
	defer l.gsMutex.Unlock()
	l.recordRequest("getconnectedplayers")

	if l.gs.Status.Players == nil {
		return result, nil
	}
	result.List = l.gs.Status.Players.Ids
	return result, nil
}

// GetPlayerCount returns the current player count.
// [Stage:Alpha]
// [FeatureFlag:PlayerTracking]
func (l *LocalSDKServer) GetPlayerCount(_ context.Context, _ *alpha.Empty) (*alpha.Count, error) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		return nil, errors.Errorf("%s not enabled", runtime.FeaturePlayerTracking)
	}
	l.logger.Info("Getting Player Count")
	l.recordRequest("getplayercount")
	l.gsMutex.RLock()
	defer l.gsMutex.RUnlock()

	result := &alpha.Count{}
	if l.gs.Status.Players != nil {
		result.Count = l.gs.Status.Players.Count
	}

	return result, nil
}

// SetPlayerCapacity to change the game server's player capacity.
// [Stage:Alpha]
// [FeatureFlag:PlayerTracking]
func (l *LocalSDKServer) SetPlayerCapacity(_ context.Context, count *alpha.Count) (*alpha.Empty, error) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		return nil, errors.Errorf("%s not enabled", runtime.FeaturePlayerTracking)
	}

	l.logger.WithField("capacity", count.Count).Info("Setting Player Capacity")
	l.gsMutex.Lock()
	defer l.gsMutex.Unlock()

	if l.gs.Status.Players == nil {
		l.gs.Status.Players = &sdk.GameServer_Status_PlayerStatus{}
	}

	l.gs.Status.Players.Capacity = count.Count

	l.update <- struct{}{}
	l.recordRequestWithValue("setplayercapacity", strconv.FormatInt(count.Count, 10), "PlayerCapacity")
	return &alpha.Empty{}, nil
}

// GetPlayerCapacity returns the current player capacity.
// [Stage:Alpha]
// [FeatureFlag:PlayerTracking]
func (l *LocalSDKServer) GetPlayerCapacity(_ context.Context, _ *alpha.Empty) (*alpha.Count, error) {
	if !runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		return nil, errors.Errorf("%s not enabled", runtime.FeaturePlayerTracking)
	}
	l.logger.Info("Getting Player Capacity")
	l.recordRequest("getplayercapacity")
	l.gsMutex.RLock()
	defer l.gsMutex.RUnlock()

	// SDK.GetPlayerCapacity() has a contract of always return a number,
	// so if we're nil, then let's always return a value, and
	// remove lots of special cases upstream.
	result := &alpha.Count{}
	if l.gs.Status.Players != nil {
		result.Count = l.gs.Status.Players.Capacity
	}

	return result, nil
}

// GetCounter returns a Counter. Returns not found if the counter does not exist.
// [Stage:Beta]
// [FeatureFlag:CountsAndLists]
func (l *LocalSDKServer) GetCounter(_ context.Context, in *beta.GetCounterRequest) (*beta.Counter, error) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		return nil, errors.Errorf("%s not enabled", runtime.FeatureCountsAndLists)
	}

	if in == nil {
		return nil, errors.Errorf("invalid argument. GetCounterRequest cannot be nil")
	}

	l.logger.WithField("name", in.Name).Info("Getting Counter")
	l.recordRequest("getcounter")
	l.gsMutex.RLock()
	defer l.gsMutex.RUnlock()

	if counter, ok := l.gs.Status.Counters[in.Name]; ok {
		return &beta.Counter{Name: in.Name, Count: counter.Count, Capacity: counter.Capacity}, nil
	}
	return nil, errors.Errorf("not found. %s Counter not found", in.Name)
}

// UpdateCounter updates the given Counter. Unlike the SDKServer, this LocalSDKServer UpdateCounter
// does not batch requests, and directly updates the localsdk gameserver.
// Returns error if the Counter does not exist (name cannot be updated).
// Returns error if the Count is out of range [0,Capacity].
// [Stage:Beta]
// [FeatureFlag:CountsAndLists]
func (l *LocalSDKServer) UpdateCounter(_ context.Context, in *beta.UpdateCounterRequest) (*beta.Counter, error) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		return nil, errors.Errorf("%s not enabled", runtime.FeatureCountsAndLists)
	}

	if in.CounterUpdateRequest == nil {
		return nil, errors.Errorf("invalid argument. CounterUpdateRequest cannot be nil")
	}

	name := in.CounterUpdateRequest.Name

	l.logger.WithField("name", name).Info("Updating Counter")
	l.gsMutex.Lock()
	defer l.gsMutex.Unlock()

	counter, ok := l.gs.Status.Counters[name]
	if !ok {
		return nil, errors.Errorf("not found. %s Counter not found", name)
	}

	tmpCounter := beta.Counter{Name: name, Count: counter.Count, Capacity: counter.Capacity}
	// Set Capacity
	if in.CounterUpdateRequest.Capacity != nil {
		l.recordRequest("setcapacitycounter")
		tmpCounter.Capacity = in.CounterUpdateRequest.Capacity.GetValue()
		if tmpCounter.Capacity < 0 {
			return nil, errors.Errorf("out of range. Capacity must be greater than or equal to 0. Found Capacity: %d",
				tmpCounter.Capacity)
		}
	}
	// Set Count
	if in.CounterUpdateRequest.Count != nil {
		l.recordRequest("setcountcounter")
		tmpCounter.Count = in.CounterUpdateRequest.Count.GetValue()
		if tmpCounter.Count < 0 || tmpCounter.Count > tmpCounter.Capacity {
			return nil, errors.Errorf("out of range. Count must be within range [0,Capacity]. Found Count: %d, Capacity: %d",
				tmpCounter.Count, tmpCounter.Capacity)
		}
	}
	// Increment or Decrement Count
	if in.CounterUpdateRequest.CountDiff != 0 {
		l.recordRequest("updatecounter")
		tmpCounter.Count += in.CounterUpdateRequest.CountDiff
		if tmpCounter.Count < 0 || tmpCounter.Count > tmpCounter.Capacity {
			return nil, errors.Errorf("out of range. Count must be within range [0,Capacity]. Found Count: %d, Capacity: %d",
				tmpCounter.Count, tmpCounter.Capacity)
		}
	}

	// Write newly updated List to gameserverstatus.
	counter.Capacity = tmpCounter.Capacity
	counter.Count = tmpCounter.Count
	l.gs.Status.Counters[name] = counter
	return &tmpCounter, nil
}

// GetList returns a List. Returns not found if the List does not exist.
// [Stage:Beta]
// [FeatureFlag:CountsAndLists]
func (l *LocalSDKServer) GetList(_ context.Context, in *beta.GetListRequest) (*beta.List, error) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		return nil, errors.Errorf("%s not enabled", runtime.FeatureCountsAndLists)
	}

	l.logger.WithField("name", in.Name).Info("Getting List")
	l.recordRequest("getlist")
	l.gsMutex.RLock()
	defer l.gsMutex.RUnlock()

	if list, ok := l.gs.Status.Lists[in.Name]; ok {
		return &beta.List{Name: in.Name, Capacity: list.Capacity, Values: list.Values}, nil
	}
	return nil, errors.Errorf("not found. %s List not found", in.Name)
}

// UpdateList returns the updated List. Returns not found if the List does not exist (name cannot be updated).
// **THIS WILL OVERWRITE ALL EXISTING LIST.VALUES WITH ANY REQUEST LIST.VALUES**
// Use AddListValue() or RemoveListValue() for modifying the List.Values field.
// Returns invalid argument if the field mask path(s) are not field(s) of the List.
// If a field mask path(s) is specified, but the value is not set in the request List object,
// then the default value for the variable will be set (i.e. 0 for "capacity", empty list for "values").
// [Stage:Beta]
// [FeatureFlag:CountsAndLists]
func (l *LocalSDKServer) UpdateList(_ context.Context, in *beta.UpdateListRequest) (*beta.List, error) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		return nil, errors.Errorf("%s not enabled", runtime.FeatureCountsAndLists)
	}

	if in.List == nil || in.UpdateMask == nil {
		return nil, errors.Errorf("invalid argument. List: %v and UpdateMask %v cannot be nil", in.List, in.UpdateMask)
	}

	l.logger.WithField("name", in.List.Name).Info("Updating List")
	l.recordRequest("updatelist")
	l.gsMutex.Lock()
	defer l.gsMutex.Unlock()

	// TODO: https://google.aip.dev/134, "Update masks must support a special value *, meaning full replacement."
	// Check if the UpdateMask paths are valid, return invalid argument if not.
	if !in.UpdateMask.IsValid(in.List.ProtoReflect().Interface()) {
		return nil, errors.Errorf("invalid argument. Field Mask Path(s): %v are invalid for List. Use valid field name(s): %v", in.UpdateMask.GetPaths(), in.List.ProtoReflect().Descriptor().Fields())
	}

	if GameServerListMaxCapacity == 0 {
		err := l.GsLocalListsMaxItems()
		if err != nil {
			return nil, fmt.Errorf("%v", err)
		}
	}

	if in.List.Capacity < 0 || in.List.Capacity > GameServerListMaxCapacity {
		return nil, errors.Errorf("out of range. Capacity must be within range [0,1000]. Found Capacity: %d", in.List.Capacity)
	}

	name := in.List.Name
	if list, ok := l.gs.Status.Lists[name]; ok {
		// Create *beta.List from *sdk.GameServer_Status_ListStatus for merging.
		tmpList := &beta.List{Name: name, Capacity: list.Capacity, Values: list.Values}
		// Removes any fields from the request object that are not included in the FieldMask Paths.
		fmutils.Filter(in.List, in.UpdateMask.Paths)
		// Removes any fields from the existing gameserver object that are included in the FieldMask Paths.
		fmutils.Prune(tmpList, in.UpdateMask.Paths)
		// Due due filtering and pruning all gameserver object field(s) contained in the FieldMask are overwritten by the request object field(s).
		proto.Merge(tmpList, in.List)
		// Silently truncate list values if Capacity < len(Values)
		if tmpList.Capacity < int64(len(tmpList.Values)) {
			tmpList.Values = append([]string{}, tmpList.Values[:tmpList.Capacity]...)
		}
		// Write newly updated List to gameserverstatus.
		l.gs.Status.Lists[name].Capacity = tmpList.Capacity
		l.gs.Status.Lists[name].Values = tmpList.Values
		return &beta.List{Name: name, Capacity: l.gs.Status.Lists[name].Capacity, Values: l.gs.Status.Lists[name].Values}, nil
	}
	return nil, errors.Errorf("not found. %s List not found", name)
}

// AddListValue appends a value to the end of a List and returns updated List.
// Returns not found if the List does not exist.
// Returns already exists if the value is already in the List.
// Returns out of range if the List is already at Capacity.
// [Stage:Beta]
// [FeatureFlag:CountsAndLists]
func (l *LocalSDKServer) AddListValue(_ context.Context, in *beta.AddListValueRequest) (*beta.List, error) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		return nil, errors.Errorf("%s not enabled", runtime.FeatureCountsAndLists)
	}

	l.logger.WithField("name", in.Name).Info("Adding Value to List")
	l.recordRequest("addlistvalue")
	l.gsMutex.Lock()
	defer l.gsMutex.Unlock()

	if list, ok := l.gs.Status.Lists[in.Name]; ok {
		// Verify room to add another value
		if list.Capacity <= int64(len(list.Values)) {
			return nil, errors.Errorf("out of range. No available capacity. Current Capacity: %d, List Size: %d", list.Capacity, len(list.Values))
		}
		// Verify value does not already exist in the list
		for _, val := range l.gs.Status.Lists[in.Name].Values {
			if in.Value == val {
				return nil, errors.Errorf("already exists. Value: %s already in List: %s", in.Value, in.Name)
			}
		}
		// Add new value to gameserverstatus.
		l.gs.Status.Lists[in.Name].Values = append(l.gs.Status.Lists[in.Name].Values, in.Value)
		return &beta.List{Name: in.Name, Capacity: l.gs.Status.Lists[in.Name].Capacity, Values: l.gs.Status.Lists[in.Name].Values}, nil
	}
	return nil, errors.Errorf("not found. %s List not found", in.Name)
}

// RemoveListValue removes a value from a List and returns updated List.
// Returns not found if the List does not exist.
// Returns not found if the value is not in the List.
// [Stage:Beta]
// [FeatureFlag:CountsAndLists]
func (l *LocalSDKServer) RemoveListValue(_ context.Context, in *beta.RemoveListValueRequest) (*beta.List, error) {
	if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		return nil, errors.Errorf("%s not enabled", runtime.FeatureCountsAndLists)
	}

	l.logger.WithField("name", in.Name).Info("Removing Value from List")
	l.recordRequest("removelistvalue")
	l.gsMutex.Lock()
	defer l.gsMutex.Unlock()

	if list, ok := l.gs.Status.Lists[in.Name]; ok {
		// Verify value exists in the list
		for i, val := range l.gs.Status.Lists[in.Name].Values {
			if in.Value == val {
				// Remove value (maintains list ordering and modifies underlying gameserverstatus List.Values array).
				list.Values = append(list.Values[:i], list.Values[i+1:]...)
				return &beta.List{Name: in.Name, Capacity: l.gs.Status.Lists[in.Name].Capacity, Values: l.gs.Status.Lists[in.Name].Values}, nil
			}
		}
		return nil, errors.Errorf("not found. Value: %s not found in List: %s", in.Value, in.Name)
	}
	return nil, errors.Errorf("not found. %s List not found", in.Name)
}

// Close tears down all the things
func (l *LocalSDKServer) Close() {
	l.updateObservers.Range(func(observer, _ interface{}) bool {
		close(observer.(chan struct{}))
		return true
	})
	l.compare()
}

// EqualSets tells whether expected and received slices contain the same elements.
// A nil argument is equivalent to an empty slice.
func (l *LocalSDKServer) EqualSets(expected, received []string) bool {
	aSet := make(map[string]bool)
	bSet := make(map[string]bool)
	for _, v := range expected {
		aSet[v] = true
	}
	for _, v := range received {
		if _, ok := aSet[v]; !ok {
			l.logger.WithField("request", v).Error("Found a request which was not expected")
			return false
		}
		bSet[v] = true
	}
	for _, v := range expected {
		if _, ok := bSet[v]; !ok {
			l.logger.WithField("request", v).Error("Could not find a request which was expected")
			return false
		}
	}
	return true
}

// compare the results of a test run
func (l *LocalSDKServer) compare() {
	if l.testMode {
		l.testMutex.Lock()
		defer l.testMutex.Unlock()
		if !l.EqualSets(l.expectedSequence, l.requestSequence) {
			l.logger.WithField("expected", l.expectedSequence).WithField("received", l.requestSequence).Info("Testing Failed")
			// we don't care if the mutex gets unlocked on exit, so ignore the warning.
			// nolint: gocritic
			os.Exit(1)
		}
		l.logger.Info("Received requests match expected list. Test run was successful")
	}
}

func (l *LocalSDKServer) setGameServerFromFilePath(filePath string) error {
	l.logger.WithField("filePath", filePath).Info("Reading GameServer configuration")

	reader, err := os.Open(filePath) // nolint: gosec
	defer reader.Close()             // nolint: staticcheck,errcheck

	if err != nil {
		return err
	}

	var gs agonesv1.GameServer
	// 4096 is the number of bytes the YAMLOrJSONDecoder goes looking
	// into the file to determine if it's JSON or YAML
	// (JSON == has whitespace followed by an open brace).
	// The Kubernetes uses 4096 bytes as its default, so that's what we'll
	// use as well.
	// https://github.com/kubernetes/kubernetes/blob/master/plugin/pkg/admission/podnodeselector/admission.go#L86
	decoder := yaml.NewYAMLOrJSONDecoder(reader, 4096)
	err = decoder.Decode(&gs)
	if err != nil {
		return err
	}

	l.gsMutex.Lock()
	defer l.gsMutex.Unlock()
	l.gs = convert(&gs)

	// Set LogLevel if specified
	logLevel := agonesv1.SdkServerLogLevelInfo
	if gs.Spec.SdkServer.LogLevel != "" {
		logLevel = gs.Spec.SdkServer.LogLevel
	}
	l.logger.WithField("logLevel", logLevel).Debug("Setting LogLevel configuration")
	level, err := logrus.ParseLevel(strings.ToLower(string(logLevel)))
	if err == nil {
		l.logger.Logger.SetLevel(level)
	} else {
		l.logger.WithError(err).Warn("Specified wrong Logging.SdkServer. Setting default loglevel - Info")
		l.logger.Logger.SetLevel(logrus.InfoLevel)
	}
	return nil
}

// GsLocalListsMaxItems retrieves or sets the maximum number of items allowed
// in any GameServer list for the local SDK.
func (l *LocalSDKServer) GsLocalListsMaxItems() error {

	if playersList, found := l.gs.Status.Lists["players"]; found {
		GameServerListMaxCapacity = playersList.Capacity
	} else {
		return fmt.Errorf("no players for list")
	}

	return nil
}
