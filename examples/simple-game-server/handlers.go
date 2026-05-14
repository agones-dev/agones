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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	coresdk "agones.dev/agones/pkg/sdk"
	sdk "agones.dev/agones/sdks/go"
)

type responseHandler func(s *sdk.SDK, parts []string, cancel ...context.CancelFunc) (response string, addACK bool, responseError error)

var responseMap = map[string]responseHandler{
	"EXIT":                 handleExit,
	"UNHEALTHY":            handleUnhealthy,
	"GAMESERVER":           handleGameServer,
	"READY":                handleReady,
	"ALLOCATE":             handleAllocate,
	"RESERVE":              handleReserve,
	"WATCH":                handleWatch,
	"LABEL":                handleLabel,
	"CRASH":                handleCrash,
	"ANNOTATION":           handleAnnotation,
	"PLAYER_CAPACITY":      handlePlayerCapacity,
	"PLAYER_CONNECT":       handlePlayerConnect,
	"PLAYER_DISCONNECT":    handlePlayerDisconnect,
	"PLAYER_CONNECTED":     handlePlayerConnected,
	"GET_PLAYERS":          handleGetPlayers,
	"PLAYER_COUNT":         handlePlayerCount,
	"GET_COUNTER_COUNT":    handleGetCounterCount,
	"INCREMENT_COUNTER":    handleIncrementCounter,
	"DECREMENT_COUNTER":    handleDecrementCounter,
	"SET_COUNTER_COUNT":    handleSetCounterCount,
	"GET_COUNTER_CAPACITY": handleGetCounterCapacity,
	"SET_COUNTER_CAPACITY": handleSetCounterCapacity,
	"GET_LIST_CAPACITY":    handleGetListCapacity,
	"SET_LIST_CAPACITY":    handleSetListCapacity,
	"LIST_CONTAINS":        handleListContains,
	"GET_LIST_LENGTH":      handleGetListLength,
	"GET_LIST_VALUES":      handleGetListValues,
	"APPEND_LIST_VALUE":    handleAppendListValue,
	"DELETE_LIST_VALUE":    handleDeleteListValue,
}

func handleResponse(txt string, s *sdk.SDK, cancel context.CancelFunc) (response string, addACK bool, responseError error) {
	parts := strings.Split(strings.TrimSpace(txt), " ")
	response = txt
	addACK = true
	responseError = nil

	handler, exists := responseMap[parts[0]]
	if !exists {
		return response, addACK, responseError
	}
	if parts[0] == "UNHEALTHY" {
		return handler(s, parts, cancel)
	}

	return handler(s, parts)
}

func handleExit(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	// handle elsewhere, as we respond before exiting
	response, addACK = defaultReply(parts)
	return
}

func handleUnhealthy(s *sdk.SDK, parts []string, cancel ...context.CancelFunc) (response string, addACK bool, responseError error) {
	response, addACK = defaultReply(parts)
	if len(cancel) > 0 {
		cancel[0]() // Invoke cancel function if provided
	}
	return
}

// handleGameServer returns the GameServer name
func handleGameServer(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	gs, err := s.GameServer()
	if err != nil {
		log.Fatalf("Could not retrieve GameServer: %v", err)
	}
	var j []byte
	j, err = json.Marshal(gs)
	if err != nil {
		log.Fatalf("error mashalling GameServer to JSON: %v", err)
	}
	log.Printf("GameServer: %s \n", string(j))
	response = "NAME: " + gs.ObjectMeta.Name + "\n"
	addACK = false
	return
}

// handleReady attempts to mark this gameserver as ready
func handleReady(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	response, addACK = defaultReply(parts)
	err := s.Ready()
	if err != nil {
		log.Fatalf("Could not send ready message: %v", err)
	}
	return
}

// handleAllocate attempts to allocate this gameserver
func handleAllocate(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	response, addACK = defaultReply(parts)
	err := s.Allocate()
	if err != nil {
		log.Fatalf("could not allocate gameserver: %v", err)
	}
	return
}

// handleReserve reserve for 10 seconds
func handleReserve(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	response, addACK = defaultReply(parts)
	if len(parts) != 2 {
		response = "Invalid RESERVE, should have 1 argument"
		responseError = fmt.Errorf("Invalid RESERVE, should have 1 argument")
		addACK = false
		return
	}
	if dur, err := time.ParseDuration(parts[1]); err != nil {
		response = fmt.Sprintf("%s\n", err)
		responseError = err
		addACK = false
		return
	} else {
		err := s.Reserve(dur)
		if err != nil {
			log.Fatalf("could not reserve gameserver: %v", err)
		}
	}
	return
}

// handleWatch creates a callback to log when
// gameserver events occur
func handleWatch(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	response, addACK = defaultReply(parts)
	err := s.WatchGameServer(func(gs *coresdk.GameServer) {
		j, err := json.Marshal(gs)
		if err != nil {
			log.Fatalf("error mashalling GameServer to JSON: %v", err)
		}
		log.Printf("GameServer Event: %s \n", string(j))
	})
	if err != nil {
		log.Fatalf("Could not watch Game Server events, %v", err)
	}
	return
}

func handleLabel(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	response, addACK = defaultReply(parts)
	switch len(parts) {
	case 1:
		// Legacy format
		setLabel(s, "timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	case 3:
		setLabel(s, parts[1], parts[2])
	default:
		response = "Invalid LABEL command, must use zero or 2 arguments"
		responseError = fmt.Errorf("Invalid LABEL command, must use zero or 2 arguments")
	}
	return
}

func handleCrash(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	log.Print("Crashing.")
	exitCode := 1
	if len(parts) > 1 {
		if code, err := strconv.Atoi(parts[1]); err == nil {
			exitCode = code
		}
	}
	os.Exit(exitCode)
	return "", false, nil
}

func handleAnnotation(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	response, addACK = defaultReply(parts)
	switch len(parts) {
	case 1:
		// Legacy format
		setAnnotation(s, "timestamp", time.Now().UTC().String())
	case 3:
		setAnnotation(s, parts[1], parts[2])
	default:
		response = "Invalid ANNOTATION command, must use zero or 2 arguments"
		responseError = fmt.Errorf("Invalid ANNOTATION command, must use zero or 2 arguments")
	}
	return
}

// handlePlayerCapacity sets the player capacity to the given value
// or returns the current player capacity as a string
func handlePlayerCapacity(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	response, addACK = defaultReply(parts)
	switch len(parts) {
	case 1:
		log.Print("Getting Player Capacity")
		capacity, err := s.Alpha().GetPlayerCapacity()
		if err != nil {
			log.Fatalf("could not get capacity: %v", err)
		}
		response = strconv.FormatInt(capacity, 10) + "\n"
		addACK = false
	case 2:
		if cap, err := strconv.Atoi(parts[1]); err != nil {
			response = fmt.Sprintf("%s", err)
			responseError = err
		} else {
			log.Printf("Setting Player Capacity to %d", int64(cap))
			if err := s.Alpha().SetPlayerCapacity(int64(cap)); err != nil {
				log.Fatalf("could not set capacity: %v", err)
			}
		}
	default:
		response = "Invalid PLAYER_CAPACITY, should have 0 or 1 arguments"
		responseError = fmt.Errorf("Invalid PLAYER_CAPACITY, should have 0 or 1 arguments")
	}
	return
}

// handlePlayerConnect connects a given player
func handlePlayerConnect(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	response, addACK = defaultReply(parts)
	if len(parts) < 2 {
		response = "Invalid PLAYER_CONNECT, should have 1 argument"
		responseError = fmt.Errorf("Invalid PLAYER_CONNECT, should have 1 argument")
		return
	}
	log.Printf("Connecting Player: %s", parts[1])
	if _, err := s.Alpha().PlayerConnect(parts[1]); err != nil {
		log.Fatalf("could not connect player: %v", err)
	}
	return
}

// handlePlayerDisconnect disconnects a given player
func handlePlayerDisconnect(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	response, addACK = defaultReply(parts)
	if len(parts) < 2 {
		response = "Invalid PLAYER_DISCONNECT, should have 1 argument"
		responseError = fmt.Errorf("Invalid PLAYER_DISCONNECT, should have 1 argument")
		return
	}
	log.Printf("Disconnecting Player: %s", parts[1])
	if _, err := s.Alpha().PlayerDisconnect(parts[1]); err != nil {
		log.Fatalf("could not disconnect player: %v", err)
	}
	return
}

// handlePlayerConnected returns a bool as a string if a player is connected
func handlePlayerConnected(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 2 {
		response = "Invalid PLAYER_CONNECTED, should have 1 argument"
		responseError = fmt.Errorf("Invalid PLAYER_CONNECTED, should have 1 argument")
		return
	}
	log.Printf("Checking if player %s is connected", parts[1])
	connected, err := s.Alpha().IsPlayerConnected(parts[1])
	if err != nil {
		log.Fatalf("could not retrieve if player is connected: %v", err)
	}
	response = strconv.FormatBool(connected) + "\n"
	addACK = false
	return
}

// handleGetPlayers returns a comma delimited list of connected players
func handleGetPlayers(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	log.Print("Retrieving connected player list")
	list, err := s.Alpha().GetConnectedPlayers()
	if err != nil {
		log.Fatalf("could not retrieve connected players: %s", err)
	}
	response = strings.Join(list, ",") + "\n"
	addACK = false
	return
}

// handlePlayerCount returns the count of connected players as a string
func handlePlayerCount(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	log.Print("Retrieving connected player count")
	count, err := s.Alpha().GetPlayerCount()
	if err != nil {
		log.Fatalf("could not retrieve player count: %s", err)
	}
	response = strconv.FormatInt(count, 10) + "\n"
	addACK = false
	return
}

// handleGetCounterCount returns the Count of the given Counter as a string
func handleGetCounterCount(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 2 {
		response = "Invalid GET_COUNTER_COUNT, should have 1 argument"
		responseError = fmt.Errorf("Invalid GET_COUNTER_COUNT, should have 1 argument")
		return
	}
	log.Printf("Retrieving Counter %s Count", parts[1])
	count, err := s.Beta().GetCounterCount(parts[1])
	if err != nil {
		log.Printf("Error getting Counter %s Count: %s", parts[1], err)
		response, responseError = strconv.FormatInt(count, 10), err
		return
	}
	response, responseError = "COUNTER: "+strconv.FormatInt(count, 10)+"\n", nil
	addACK = false
	return
}

// handleIncrementCounter returns the if the Counter Count was incremented successfully or not
func handleIncrementCounter(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 3 {
		response = "Invalid INCREMENT_COUNTER, should have 2 arguments"
		responseError = fmt.Errorf("Invalid INCREMENT_COUNTER, should have 2 arguments")
		return
	}
	amountInt, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		response, responseError = err.Error(), fmt.Errorf("Could not increment Counter %s by unparseable amount %s: %s", parts[1], parts[2], err)
		return
	}
	log.Printf("Incrementing Counter %s Count by amount %d", parts[1], amountInt)
	err = s.Beta().IncrementCounter(parts[1], amountInt)
	if err != nil {
		log.Printf("Error incrementing Counter %s Count by amount %d: %s", parts[1], amountInt, err)
		response, responseError = err.Error(), err
		return
	}
	response, responseError = "SUCCESS\n", nil
	return
}

// handleDecrementCounter returns if the Counter Count was decremented successfully or not
func handleDecrementCounter(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 3 {
		response = "Invalid DECREMENT_COUNTER, should have 2 arguments"
		responseError = fmt.Errorf("Invalid DECREMENT_COUNTER, should have 2 arguments")
		return
	}
	amountInt, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		response, responseError = err.Error(), fmt.Errorf("could not decrement Counter %s by unparseable amount %s: %s", parts[1], parts[2], err)
		return
	}
	log.Printf("Decrementing Counter %s Count by amount %d", parts[1], amountInt)
	err = s.Beta().DecrementCounter(parts[1], amountInt)
	if err != nil {
		log.Printf("Error decrementing Counter %s Count by amount %d: %s", parts[1], amountInt, err)
		response, responseError = err.Error(), err
		return
	}
	response, responseError = "SUCCESS\n", nil
	return
}

// handleSetCounterCount returns the if the Counter was set to a new Count successfully or not
func handleSetCounterCount(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 3 {
		response = "Invalid SET_COUNTER_COUNT, should have 2 arguments"
		responseError = fmt.Errorf("Invalid SET_COUNTER_COUNT, should have 2 arguments")
		return
	}
	amountInt, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		response, responseError = err.Error(), fmt.Errorf("could not set Counter %s to unparseable amount %s: %s", parts[1], parts[2], err)
		return
	}
	log.Printf("Setting Counter %s Count to amount %d", parts[1], amountInt)
	err = s.Beta().SetCounterCount(parts[1], amountInt)
	if err != nil {
		log.Printf("Error setting Counter %s Count by amount %d: %s", parts[1], amountInt, err)
		response, responseError = err.Error(), err
		return
	}
	response, responseError = "SUCCESS\n", nil
	return
}

// handleGetCounterCapacity returns the Capacity of the given Counter as a string
func handleGetCounterCapacity(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 2 {
		response = "Invalid GET_COUNTER_CAPACITY, should have 1 argument"
		responseError = fmt.Errorf("Invalid GET_COUNTER_CAPACITY, should have 1 argument")
		return
	}
	log.Printf("Retrieving Counter %s Capacity", parts[1])
	count, err := s.Beta().GetCounterCapacity(parts[1])
	if err != nil {
		log.Printf("Error getting Counter %s Capacity: %s", parts[1], err)
		response, responseError = strconv.FormatInt(count, 10), err
		return
	}
	response, responseError = "CAPACITY: "+strconv.FormatInt(count, 10)+"\n", nil
	return
}

// handleSetCounterCapacity returns the if the Counter was set to a new Capacity successfully or not
func handleSetCounterCapacity(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 3 {
		response = "Invalid SET_COUNTER_CAPACITY, should have 2 arguments"
		responseError = fmt.Errorf("Invalid SET_COUNTER_CAPACITY, should have 2 arguments")
		return
	}
	amountInt, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		response, responseError = err.Error(), fmt.Errorf("could not set Counter %s to unparseable amount %s: %s", parts[1], parts[2], err)
		return
	}
	log.Printf("Setting Counter %s Capacity to amount %d", parts[1], amountInt)
	err = s.Beta().SetCounterCapacity(parts[1], amountInt)
	if err != nil {
		log.Printf("Error setting Counter %s Capacity to amount %d: %s", parts[1], amountInt, err)
		response, responseError = err.Error(), err
		return
	}
	response, responseError = "SUCCESS\n", nil
	return
}

// handleGetListCapacity returns the Capacity of the given List as a string
func handleGetListCapacity(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 2 {
		response = "Invalid GET_LIST_CAPACITY, should have 1 argument"
		responseError = fmt.Errorf("Invalid GET_LIST_CAPACITY, should have 1 argument")
		return
	}
	log.Printf("Retrieving List %s Capacity", parts[1])
	capacity, err := s.Beta().GetListCapacity(parts[1])
	if err != nil {
		log.Printf("Error getting List %s Capacity: %s", parts[1], err)
		response, responseError = strconv.FormatInt(capacity, 10), err
		return
	}
	response, responseError = "CAPACITY: "+strconv.FormatInt(capacity, 10)+"\n", nil
	return
}

// handleSetListCapacity returns if the List was set to a new Capacity successfully or not
func handleSetListCapacity(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 3 {
		response = "Invalid SET_LIST_CAPACITY, should have 2 arguments"
		responseError = fmt.Errorf("Invalid SET_LIST_CAPACITY, should have 2 arguments")
		return
	}
	amountInt, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		response, responseError = err.Error(), fmt.Errorf("could not set List %s to unparseable amount %s: %s", parts[1], parts[2], err)
		return
	}
	log.Printf("Setting List %s Capacity to amount %d", parts[1], amountInt)
	err = s.Beta().SetListCapacity(parts[1], amountInt)
	if err != nil {
		log.Printf("Error setting List %s Capacity to amount %d: %s", parts[1], amountInt, err)
		response, responseError = err.Error(), err
		return
	}
	response, responseError = "SUCCESS\n", nil
	return
}

// handleListContains returns true if the given value is in the given List, false otherwise
func handleListContains(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 3 {
		response = "Invalid LIST_CONTAINS, should have 2 arguments"
		responseError = fmt.Errorf("Invalid LIST_CONTAINS, should have 2 arguments")
		return
	}
	log.Printf("Getting List %s contains value %s", parts[1], parts[2])
	ok, err := s.Beta().ListContains(parts[1], parts[2])
	if err != nil {
		log.Printf("Error getting List %s contains value %s: %s", parts[1], parts[2], err)
		response, responseError = strconv.FormatBool(ok), err
		return
	}
	response, responseError = "FOUND: "+strconv.FormatBool(ok)+"\n", nil
	return
}

// handleGetListLength returns the length (number of values) of the given List as a string
func handleGetListLength(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 2 {
		response = "Invalid GET_LIST_LENGTH, should have 1 argument"
		responseError = fmt.Errorf("Invalid GET_LIST_LENGTH, should have 1 argument")
		return
	}
	log.Printf("Getting List %s length", parts[1])
	length, err := s.Beta().GetListLength(parts[1])
	if err != nil {
		log.Printf("Error getting List %s length: %s", parts[1], err)
		response, responseError = strconv.Itoa(length), err
		return
	}
	response, responseError = "LENGTH: "+strconv.Itoa(length)+"\n", nil
	return
}

// handleGetListValues return the values in the given List as a comma delineated string
func handleGetListValues(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 2 {
		response = "Invalid GET_LIST_VALUES, should have 1 argument"
		responseError = fmt.Errorf("Invalid GET_LIST_VALUES, should have 1 argument")
		return
	}
	log.Printf("Getting List %s values", parts[1])
	values, err := s.Beta().GetListValues(parts[1])
	if err != nil {
		log.Printf("Error getting List %s values: %s", parts[1], err)
		response, responseError = "INVALID LIST NAME", err
		return
	}
	if len(values) > 0 {
		response, responseError = "VALUES: "+strings.Join(values, ",")+"\n", nil
		return
	}
	response, responseError = "VALUES: <none>\n", nil
	return
}

// handleAppendListValue returns if the given value was successfully added to the List or not
func handleAppendListValue(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 3 {
		response = "Invalid APPEND_LIST_VALUE, should have 2 arguments"
		responseError = fmt.Errorf("Invalid APPEND_LIST_VALUE, should have 2 arguments")
		return
	}
	log.Printf("Appending Value %s to List %s", parts[2], parts[1])
	err := s.Beta().AppendListValue(parts[1], parts[2])
	if err != nil {
		log.Printf("Error appending Value %s to List %s: %s", parts[2], parts[1], err)
		response, responseError = err.Error(), err
		return
	}
	response, responseError = "SUCCESS\n", nil
	return
}

// handleDeleteListValue returns if the given value was successfully deleted from the List or not
func handleDeleteListValue(s *sdk.SDK, parts []string, _ ...context.CancelFunc) (response string, addACK bool, responseError error) {
	if len(parts) < 3 {
		response = "Invalid DELETE_LIST_VALUE, should have 2 arguments"
		responseError = fmt.Errorf("Invalid DELETE_LIST_VALUE, should have 2 arguments")
		return
	}
	log.Printf("Deleting Value %s from List %s", parts[2], parts[1])
	err := s.Beta().DeleteListValue(parts[1], parts[2])
	if err != nil {
		log.Printf("Error deleting Value %s to List %s: %s", parts[2], parts[1], err)
		response, responseError = err.Error(), err
		return
	}
	response, responseError = "SUCCESS\n", nil
	return
}

// setAnnotation sets a given annotation
func setAnnotation(s *sdk.SDK, key, value string) {
	log.Printf("Setting annotation %v=%v", key, value)
	err := s.SetAnnotation(key, value)
	if err != nil {
		log.Fatalf("could not set annotation: %v", err)
	}
}

// setLabel sets a given label
func setLabel(s *sdk.SDK, key, value string) {
	log.Printf("Setting label %v=%v", key, value)
	// label values can only be alpha, - and .
	err := s.SetLabel(key, value)
	if err != nil {
		log.Fatalf("could not set label: %v", err)
	}
}

// exit shutdowns the server
func exit(s *sdk.SDK) {
	log.Printf("Received EXIT command. Exiting.")
	// This tells Agones to shutdown this Game Server
	shutdownErr := s.Shutdown()
	if shutdownErr != nil {
		log.Printf("Could not shutdown")
	}
	// The process will exit when Agones removes the pod and the
	// container receives the SIGTERM signal
}

// defaultReply is default handler response
func defaultReply(parts []string) (string, bool) {
	response := strings.Join(parts, " ")
	addACK := true
	return response, addACK
}
