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

// Binary sdk-client-test tests the Go SDK client.
package main

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	pkgSdk "agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/util/runtime"
	goSdk "agones.dev/agones/sdks/go"
)

func main() {
	viper.AllowEmptyEnv(true)
	runtime.FeaturesBindFlags()
	pflag.Parse()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	runtime.Must(runtime.FeaturesBindEnv())
	runtime.Must(runtime.ParseFeaturesFromEnv())

	// Use to delays to prevent Game Servers from churning too quickly on a running cluster.
	shutdownDelaySec := 0
	gracefulTerminationDelaySec := 0
	if sds := os.Getenv("SHUTDOWN_DELAY_SECONDS"); sds != "" {
		sec, err := strconv.Atoi(sds)
		if err != nil {
			log.Fatalf("Could not parse SHUTDOWN_DELAY_SECONDS: %v", err)
		}
		shutdownDelaySec = sec
	}
	if gtds := os.Getenv("GRACEFUL_TERMINATION_DELAY_SECONDS"); gtds != "" {
		sec, err := strconv.Atoi(gtds)
		if err != nil {
			log.Fatalf("Could not parse GRACEFUL_TERMINATION_DELAY_SECONDS: %v", err)
		}
		gracefulTerminationDelaySec = sec
	}

	log.SetFlags(log.Lshortfile)
	log.Println("Client is starting")
	log.Printf("Feature Flags: %s\n", runtime.EncodeFeatures())
	time.Sleep(100 * time.Millisecond)
	sdk, err := goSdk.NewSDK()
	if err != nil {
		log.Fatalf("Could not connect to sdk: %v\n", err)
	}

	c := make(chan string)

	once := true
	err = sdk.WatchGameServer(func(gs *pkgSdk.GameServer) {
		log.Println("Received GameServer update")
		log.Println(gs)
		uid := gs.ObjectMeta.Uid
		if once {
			c <- uid
			once = false
		}
	})
	if err != nil {
		log.Fatalf("Error on watch GameServer %s", err)
	}
	err = sdk.Ready()
	if err != nil {
		log.Fatalf("Could not send ready message %s", err)
	}
	if err = sdk.Reserve(5 * time.Second); err != nil {
		log.Fatalf("Could not send Reserve command: %s", err)
	}
	err = sdk.Allocate()
	if err != nil {
		log.Fatalf("Err sending allocate request %s", err)
	}
	err = sdk.Health()
	if err != nil {
		log.Fatalf("Could not send Health check: %s", err)
	}
	gs, err := sdk.GameServer()
	if err != nil {
		log.Fatalf("Could not get gameserver parameters: %s", err)
	}
	log.Println(gs)

	err = sdk.SetLabel("creationTimestamp", strconv.FormatInt(gs.ObjectMeta.CreationTimestamp, 10))
	if err != nil {
		log.Fatalf("Could not set label: %s", err)
	}
	if err != nil {
		log.Fatalf("Error received on watch gameserver %s", err)
	}
	uid := <-c
	err = sdk.SetAnnotation("UID", uid)
	if err != nil {
		log.Fatalf("Could not set annotation: %s", err)
	}

	if runtime.FeatureEnabled(runtime.FeaturePlayerTracking) {
		testPlayerTracking(sdk)
	}

	if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		testCounts(sdk)
		testLists(sdk)
	}

	log.Printf("Waiting %d seconds before shutting down game server", shutdownDelaySec)
	time.Sleep(time.Duration(shutdownDelaySec) * time.Second)

	err = sdk.Shutdown()
	if err != nil {
		log.Fatalf("Could not shutdown GameServer: %s", err)
	}

	log.Printf("Waiting %d seconds before exiting", gracefulTerminationDelaySec)
	time.Sleep(time.Duration(gracefulTerminationDelaySec) * time.Second)
}

func testPlayerTracking(sdk *goSdk.SDK) {
	capacity := int64(10)
	if err := sdk.Alpha().SetPlayerCapacity(capacity); err != nil {
		log.Fatalf("Error setting player capacity: %s", err)
	}

	c, err := sdk.Alpha().GetPlayerCapacity()
	if err != nil {
		log.Fatalf("Error getting player capacity: %s", err)
	}
	if c != capacity {
		log.Fatalf("Player Capacity should be %d, but is %d", capacity, c)
	}

	playerID := "1234"
	if ok, err := sdk.Alpha().PlayerConnect(playerID); err != nil {
		log.Fatalf("Error registering player as connected: %s", err)
	} else if !ok {
		log.Fatalf("PlayerConnect returned false")
	}

	if ok, err := sdk.Alpha().IsPlayerConnected(playerID); err != nil {
		log.Fatalf("Error checking if player is connected: %s", err)
	} else if !ok {
		log.Fatalf("IsPlayerConnected returned false")
	}

	if list, err := sdk.Alpha().GetConnectedPlayers(); err != nil {
		log.Fatalf("Error getting connected player: %s", err)
	} else if len(list) == 0 {
		log.Fatalf("No connected players returned")
	}

	if ok, err := sdk.Alpha().PlayerDisconnect(playerID); err != nil {
		log.Fatalf("Error registering player as disconnected: %s", err)
	} else if !ok {
		log.Fatalf("PlayerDisconnect returned false")
	}

	if c, err = sdk.Alpha().GetPlayerCount(); err != nil {
		log.Fatalf("Error retrieving player count: %s", err)
	} else if c != int64(0) {
		log.Fatalf("Player Count should be 0, but is %d", c)
	}
}

func testCounts(sdk *goSdk.SDK) {
	// LocalSDKServer starting "rooms": {Count: 1, Capacity: 10}
	counter := "rooms"
	count, err := sdk.Beta().GetCounterCount(counter)
	if err != nil {
		log.Fatalf("Error getting Counter count: %s", err)
	} else if count != int64(1) {
		log.Fatalf("Counter count should be 1, but is %d", count)
	}

	err = sdk.Beta().IncrementCounter(counter, 9)
	if err != nil {
		log.Fatalf("Error incrementing Counter: %s", err)
	}

	err = sdk.Beta().DecrementCounter(counter, 10)
	if err != nil {
		log.Fatalf("Error decrementing Counter: %s", err)
	}

	err = sdk.Beta().SetCounterCount(counter, 10)
	if err != nil {
		log.Fatalf("Error setting Counter count: %s", err)
	}

	capacity, err := sdk.Beta().GetCounterCapacity(counter)
	if err != nil {
		log.Fatalf("Error getting Counter capacity: %s", err)
	} else if capacity != int64(10) {
		log.Fatalf("Counter capacity should be 10, but is %d", capacity)
	}

	err = sdk.Beta().SetCounterCapacity(counter, 1)
	if err != nil {
		log.Fatalf("Error setting Counter capacity: %s", err)
	}
}

func testLists(sdk *goSdk.SDK) {
	// LocalSDKServer starting "players": {Values: []string{"test0", "test1", "test2"}, Capacity: 100}}
	list := "players"
	vals := []string{"test0", "test1", "test2"}

	contains, err := sdk.Beta().ListContains(list, "test1")
	if !contains {
		log.Fatalf("List should contain value \"test1\" err: %s", err)
	}

	length, err := sdk.Beta().GetListLength(list)
	if err != nil {
		log.Fatalf("Error getting List length: %s", err)
	} else if int64(length) != 3 {
		log.Fatalf("List length should be 3, but is %d", length)
	}

	values, err := sdk.Beta().GetListValues(list)
	if err != nil {
		log.Fatalf("Error getting List values: %s", err)
	} else if !cmp.Equal(vals, values) {
		log.Fatalf("List values should be %v, but is %v", vals, values)
	}

	err = sdk.Beta().AppendListValue(list, "test3")
	if err != nil {
		log.Fatalf("Unable to append value \"test3\" err: %s", err)
	}

	err = sdk.Beta().DeleteListValue(list, "test2")
	if err != nil {
		log.Fatalf("Unable to delete value \"test2\" err: %s", err)
	}

	capacity, err := sdk.Beta().GetListCapacity(list)
	if err != nil {
		log.Fatalf("Error getting List capacity: %s", err)
	} else if capacity != int64(100) {
		log.Fatalf("List capacity should be 100, but is %d", capacity)
	}

	err = sdk.Beta().SetListCapacity(list, 2)
	if err != nil {
		log.Fatalf("Error setting List capacity: %s", err)
	}
}
