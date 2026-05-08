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
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/net/context"

	alpha "agones.dev/agones/test/sdk/restapi/alpha/swagger"
	beta "agones.dev/agones/test/sdk/restapi/beta/swagger"
	"agones.dev/agones/test/sdk/restapi/swagger"
)

func main() {
	log.Println("Client is starting")
	conf := swagger.NewConfiguration()
	portStr := os.Getenv("AGONES_SDK_HTTP_PORT")
	conf.BasePath = "http://localhost:" + portStr
	cli := swagger.NewAPIClient(conf)

	log.Println("Alpha Client is starting")
	alphaConf := alpha.NewConfiguration()
	alphaConf.BasePath = "http://localhost:" + portStr
	alphaCli := alpha.NewAPIClient(alphaConf)

	log.Println("Beta Client is starting")
	betaConf := beta.NewConfiguration()
	betaConf.BasePath = "http://localhost:" + portStr
	betaCli := beta.NewAPIClient(betaConf)

	ctx := context.Background()

	// Wait for SDK server to start the test (15 seconds)
	for c := 0; c < 15; c++ {
		_, _, err := cli.SDKApi.Ready(ctx, swagger.SdkEmpty{})
		if err == nil {
			break
		} else {
			log.Printf("Could not send Ready: %v\n", err)
		}
		time.Sleep(1 * time.Second)
	}

	c := make(chan string)

	once := true
	go func() {
		for {
			gs, _, err := cli.SDKApi.WatchGameServer(ctx)
			log.Printf("Watch response: %+v", gs)
			if err != nil {
				log.Printf("Error in WatchGameServer: %v\n", err)
				return
			} else {
				if gs.Result.ObjectMeta != nil {
					uid := gs.Result.ObjectMeta.Uid
					if once {
						c <- uid
						once = false
					}
				} else {
					log.Printf("Could not read GS Uid \n")
				}
			}
		}
	}()

	_, _, err := cli.SDKApi.Health(ctx, swagger.SdkEmpty{})
	if err != nil {
		log.Fatalf("Could not send health check: %v\n", err)
	}

	_, _, err = cli.SDKApi.Reserve(ctx, swagger.SdkDuration{"5"})
	if err != nil {
		log.Fatalf("Could not send Reserve: %v\n", err)
	}

	_, _, err = cli.SDKApi.Allocate(ctx, swagger.SdkEmpty{})
	if err != nil {
		log.Fatalf("Could not send Allocate: %v\n", err)
	}

	gs, _, err := cli.SDKApi.GetGameServer(ctx)
	if err != nil {
		log.Fatalf("Could not GetGameserver: %v\n", err)
	}

	creationTS := gs.ObjectMeta.CreationTimestamp

	_, _, err = cli.SDKApi.SetLabel(ctx, swagger.SdkKeyValue{"creationTimestamp", creationTS})
	if err != nil {
		log.Fatalf("Could not SetLabel: %v\n", err)
	}

	// TODO: fix WatchGameServer() HTTP API Swagger definition and remove the following lines
	go func() {
		c <- gs.ObjectMeta.Uid
	}()

	uid := <-c
	_, _, err = cli.SDKApi.SetAnnotation(ctx, swagger.SdkKeyValue{"UID", uid})
	if err != nil {
		log.Fatalf("Could not SetAnnotation: %v\n", err)
	}

	// easy feature flag check
	if strings.Contains(os.Getenv("FEATURE_GATES"), "PlayerTracking=true") {
		testPlayers(ctx, alphaCli)
	} else {
		log.Print("Player Tracking not enabled, skipping.")
	}

	if strings.Contains(os.Getenv("FEATURE_GATES"), "CountsAndLists=true") {
		testCounters(ctx, betaCli)
		testLists(ctx, betaCli)
	} else {
		log.Print("Counts and Lists not enabled, skipping.")
	}

	_, _, err = cli.SDKApi.Shutdown(ctx, swagger.SdkEmpty{})
	if err != nil {
		log.Fatalf("Could not GetGameserver: %v\n", err)
	}
	log.Println("REST API test finished, all queries were performed")
}

func testPlayers(ctx context.Context, alphaCli *alpha.APIClient) {
	capacity := "10"
	if _, _, err := alphaCli.SDKApi.SetPlayerCapacity(ctx, alpha.AlphaCount{Count: capacity}); err != nil {
		log.Fatalf("Could not set Capacity: %v\n", err)
	}

	count, _, err := alphaCli.SDKApi.GetPlayerCapacity(ctx)
	if err != nil {
		log.Fatalf("Could not get Capacity: %v\n", err)
	}
	if count.Count != capacity {
		log.Fatalf("Player Capacity should be %s, but is %s", capacity, count.Count)
	}

	playerID := "1234"
	if ok, _, err := alphaCli.SDKApi.PlayerConnect(ctx, alpha.AlphaPlayerId{PlayerID: playerID}); err != nil {
		log.Fatalf("Error registering player as connected: %s", err)
	} else if !ok.Bool_ {
		log.Fatalf("PlayerConnect returned false")
	}

	if ok, _, err := alphaCli.SDKApi.IsPlayerConnected(ctx, playerID); err != nil {
		log.Fatalf("Error checking if player is connected: %s", err)
	} else if !ok.Bool_ {
		log.Fatalf("IsPlayerConnected returned false")
	}

	if list, _, err := alphaCli.SDKApi.GetConnectedPlayers(ctx); err != nil {
		log.Fatalf("Error getting connected player: %s", err)
	} else if len(list.List) == 0 {
		log.Fatalf("No connected players returned")
	}

	if ok, _, err := alphaCli.SDKApi.PlayerDisconnect(ctx, alpha.AlphaPlayerId{PlayerID: playerID}); err != nil {
		log.Fatalf("Error registering player as disconnected: %s", err)
	} else if !ok.Bool_ {
		log.Fatalf("PlayerDisconnect returned false")
	}

	if count, _, err := alphaCli.SDKApi.GetPlayerCount(ctx); err != nil {
		log.Fatalf("Error retrieving player count: %s", err)
	} else if count.Count != "0" {
		log.Fatalf("Player Count should be 0, but is %v", count)
	}
}

func testCounters(ctx context.Context, betaCli *beta.APIClient) {
	// Tests are expected to run sequentially on the same pre-defined Counter in the localsdk server
	counterName := "rooms"

	expectedCounter := beta.BetaCounter{Name: counterName, Count: "1", Capacity: "10"}
	if counter, _, err := betaCli.SDKApi.GetCounter(ctx, counterName); err != nil {
		log.Fatalf("Error getting Counter: %s", err)
	} else {
		if !cmp.Equal(expectedCounter, counter) {
			log.Fatalf("GetCounter expected Counter: %v, got Counter: %v", expectedCounter, counter)
		}
	}

	// Test updatecounter, setcapacitycounter
	expectedCounter = beta.BetaCounter{Name: counterName, Count: "0", Capacity: "42"}
	if counter, _, err := betaCli.SDKApi.UpdateCounter(ctx, beta.TheRequestedUpdateToMakeToTheCounter{CountDiff: "-1", Capacity: "42"}, counterName); err != nil {
		log.Fatalf("Error getting Counter: %s", err)
	} else {
		if !cmp.Equal(expectedCounter, counter) {
			log.Fatalf("UpdateCounter expected Counter: %v, got Counter: %v", expectedCounter, counter)
		}
	}

	// Test setcountcounter
	expectedCounter = beta.BetaCounter{Name: counterName, Count: "40", Capacity: "42"}
	if counter, _, err := betaCli.SDKApi.UpdateCounter(ctx, beta.TheRequestedUpdateToMakeToTheCounter{Count: "40", Capacity: "42"}, counterName); err != nil {
		log.Fatalf("Error getting Counter: %s", err)
	} else {
		if !cmp.Equal(expectedCounter, counter) {
			log.Fatalf("UpdateCounter expected Counter: %v, got Counter: %v", expectedCounter, counter)
		}
	}
}

func testLists(ctx context.Context, betaCli *beta.APIClient) {
	// Tests are expected to run sequentially on the same pre-defined List in the localsdk server
	listName := "players"

	expectedList := beta.BetaList{Name: listName, Values: []string{"test0", "test1", "test2"}, Capacity: "100"}
	if list, _, err := betaCli.SDKApi.GetList(ctx, listName); err != nil {
		log.Fatalf("Error getting List: %s", err)
	} else {
		if !cmp.Equal(expectedList, list) {
			log.Fatalf("GetList expected List: %v, got List: %v", expectedList, list)
		}
	}

	expectedList = beta.BetaList{Name: listName, Values: []string{"test123", "test456"}, Capacity: "10"}
	if list, _, err := betaCli.SDKApi.UpdateList(ctx, beta.TheListToUpdate{Values: []string{"test123", "test456"}, Capacity: "10"}, listName); err != nil {
		log.Fatalf("Error getting List: %s", err)
	} else {
		if !cmp.Equal(expectedList, list) {
			log.Fatalf("UpdateList expected List: %v, got List: %v", expectedList, list)
		}
	}

	expectedList = beta.BetaList{Name: listName, Values: []string{"test123", "test456", "test789"}, Capacity: "10"}
	if list, _, err := betaCli.SDKApi.AddListValue(ctx, beta.SdkAddListValueBody{Value: "test789"}, listName); err != nil {
		log.Fatalf("Error getting List: %s", err)
	} else {
		if !cmp.Equal(expectedList, list) {
			log.Fatalf("AddListValue expected List: %v, got List: %v", expectedList, list)
		}
	}

	expectedList = beta.BetaList{Name: listName, Values: []string{"test123", "test789"}, Capacity: "10"}
	if list, _, err := betaCli.SDKApi.RemoveListValue(ctx, beta.SdkRemoveListValueBody{Value: "test456"}, listName); err != nil {
		log.Fatalf("Error getting List: %s", err)
	} else {
		if !cmp.Equal(expectedList, list) {
			log.Fatalf("RemoveListValue expected List: %v, got List: %v", expectedList, list)
		}
	}
}
