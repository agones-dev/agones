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

// Binary ws-watch-test tests websocket watch in Go.
package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	portStr := os.Getenv("AGONES_SDK_HTTP_PORT")
	watchURL := url.URL{Scheme: "ws", Host: "localhost:" + portStr, Path: "/watch/gameserver"}
	reserveURL := url.URL{Scheme: "http", Host: "localhost:" + portStr, Path: "/reserve"}
	gameServerURL := url.URL{Scheme: "http", Host: "localhost:" + portStr, Path: "/gameserver"}

	// Connect watchGameserver API with websocket
	log.Printf("Connecting to %s", watchURL.String())
	websocketClient, connectResponse, dialErr := websocket.DefaultDialer.Dial(watchURL.String(), nil)
	if dialErr != nil {
		log.Fatalf("Could not dial watch websocket: %v", dialErr)
	}
	defer connectResponse.Body.Close() // nolint: errcheck
	defer websocketClient.Close()      // nolint: errcheck

	// Send reserved request
	log.Printf("Change to status to reserved")
	req, reqErr := http.NewRequest(http.MethodPost, reserveURL.String(), nil)
	if reqErr != nil {
		log.Fatalf("Could not create reserve request: %v", reqErr) // nolint: gocritic
	}

	httpClient := &http.Client{
		Timeout: time.Second * 10,
	}
	response, respErr := httpClient.Do(req)
	if respErr != nil {
		log.Fatalf("Could not post reserve request: %v", reqErr) // nolint: gocritic
	}
	defer response.Body.Close() // nolint: errcheck

	// Wait for gameserver become Reserved (max 10 seconds)
	for c := 0; c < 10; c++ {
		log.Printf("Get GameServer status...util GameServer status become Reserved")
		req, reqErr = http.NewRequest(http.MethodGet, gameServerURL.String(), nil)
		if reqErr != nil {
			log.Fatalf("Could not create gameserver request: %v", reqErr) // nolint: gocritic
		}
		response, respErr = httpClient.Do(req)
		if respErr != nil {
			log.Fatalf("Could not get response: %v", respErr)
		}
		gs, err := io.ReadAll(response.Body)
		if err != nil {
			log.Fatalf("Could not read gameserver response body")
		}
		err = response.Body.Close()
		if err != nil {
			log.Fatalf("Could not close response body")
		}
		if strings.Contains(string(gs), "Reserved") {
			break
		}
		time.Sleep(1 * time.Second)
	}

	// Read message from the sdkserver
	log.Printf("Read message from the websocket server")
	_, message, err := websocketClient.ReadMessage()
	if err != nil {
		log.Fatalf("Unable to read message from websocket: %v", err)
		return
	}
	log.Printf("Received message from websocket: %s", message)

	// Check if the watchGameserver result has the status: Reserved
	switch {
	case strings.Contains(string(message), "Reserved"):
		log.Printf("Found status 'Reserved' in message")
	case strings.Contains(string(message), "Shutdown"):
		// test time out
		log.Println("Found status 'Shutdown' in message")
	default:
		log.Fatalf("Cloud not find status 'Reserved' or 'Shutdown'")
	}

	// Write message to the sdkserver
	log.Printf("Write empty message to the websocket server")
	closeErr := websocketClient.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if closeErr != nil {
		log.Fatalf("Error writing close message: %v", closeErr)
	}
}
