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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"agones.dev/agones/pkg/util/signals"
	sdk "agones.dev/agones/sdks/go"
)

// Main starts a server that serves as an example of how to integrate GenAI endpoints into your dedicated game server.
func main() {
	sigCtx, _ := signals.NewSigKillContext()

	port := flag.String("port", "7654", "The port to listen to traffic on")
	genAiEndpoint := flag.String("GenAiEndpoint", "", "The full base URL to send API requests to simulate computer (NPC) responses to user input")
	genAiContext := flag.String("GenAiContext", "", "Context for the GenAI endpoint")
	prompt := flag.String("Prompt", "", "The first prompt for the GenAI endpoint")
	simEndpoint := flag.String("SimEndpoint", "", "The full base URL to send API requests to simulate user input")
	simContext := flag.String("SimContext", "", "Context for the Sim endpoint")
	stopPhrase := flag.String("StopPhrase", "Bye!", "In autonomous chat, if either side sends this, stop after the next turn.")
	numChats := flag.Int("NumChats", 1, "Number of back and forth chats between the sim and genAI")
	genAiNpc := flag.Bool("GenAiNpc", false, "Set to true if the GenAIEndpoint is the npc-chat-api endpoint")
	simNpc := flag.Bool("SimNpc", false, "Set to true if the SimEndpoint is the npc-chat-api endpoint")
	fromId := flag.Int("FromID", 2, "Entity sending messages to the npc-chat-api. Ignored when autonomous, which uses random FromID")
	toId := flag.Int("ToID", 1, "Entity receiving messages on the npc-chat-api (the NPC's ID)")
	concurrentPlayers := flag.Int("ConcurrentPlayers", 1, "Number of concurrent players.")

	flag.Parse()
	if ep := os.Getenv("PORT"); ep != "" {
		port = &ep
	}
	if sc := os.Getenv("SIM_CONTEXT"); sc != "" {
		simContext = &sc
	}
	if ss := os.Getenv("STOP_PHRASE"); ss != "" {
		stopPhrase = &ss
	}
	if gac := os.Getenv("GEN_AI_CONTEXT"); gac != "" {
		genAiContext = &gac
	}
	if p := os.Getenv("PROMPT"); p != "" {
		prompt = &p
	}
	if se := os.Getenv("SIM_ENDPOINT"); se != "" {
		simEndpoint = &se
	}
	if gae := os.Getenv("GEN_AI_ENDPOINT"); gae != "" {
		genAiEndpoint = &gae
	}
	if nc := os.Getenv("NUM_CHATS"); nc != "" {
		num, err := strconv.Atoi(nc)
		if err != nil {
			log.Fatalf("Could not parse NumChats: %v", err)
		}
		numChats = &num
	}
	if gan := os.Getenv("GEN_AI_NPC"); gan != "" {
		gnpc, err := strconv.ParseBool(gan)
		if err != nil {
			log.Fatalf("Could parse GenAiNpc: %v", err)
		}
		genAiNpc = &gnpc
	}
	if sn := os.Getenv("SIM_NPC"); sn != "" {
		snpc, err := strconv.ParseBool(sn)
		if err != nil {
			log.Fatalf("Could parse GenAiNpc: %v", err)
		}
		simNpc = &snpc
	}
	if fid := os.Getenv("FROM_ID"); fid != "" {
		num, err := strconv.Atoi(fid)
		if err != nil {
			log.Fatalf("Could not parse FromId: %v", err)
		}
		fromId = &num
	}
	if tid := os.Getenv("TO_ID"); tid != "" {
		num, err := strconv.Atoi(tid)
		if err != nil {
			log.Fatalf("Could not parse ToId: %v", err)
		}
		toId = &num
	}
	if cp := os.Getenv("CONCURRENT_PLAYERS"); cp != "" {
		num, err := strconv.Atoi(cp)
		if err != nil {
			log.Fatalf("Could not parse ToID: %v", err)
		}
		concurrentPlayers = &num
	}

	log.Print("Creating SDK instance")
	s, err := sdk.NewSDK()
	if err != nil {
		log.Fatalf("Could not connect to sdk: %v", err)
	}

	log.Print("Starting Health Ping")
	go doHealth(s, sigCtx)

	log.Print("Marking this server as ready")
	if err := s.Ready(); err != nil {
		log.Fatalf("Could not send ready message")
	}

	if *genAiEndpoint == "" {
		log.Fatalf("GenAiEndpoint must be specified")
	}

	// Start up TCP listener so the user can interact with the GenAI endpoint manually
	if *simEndpoint == "" {
		log.Printf("Creating GenAI Client at endpoint %s (from_id=%d, to_id=%d)", *genAiEndpoint, *fromId, *toId)
		genAiConn := initClient(*genAiEndpoint, *genAiContext, "GenAI", *genAiNpc, *fromId, *toId)
		go tcpListener(*port, genAiConn)
		<-sigCtx.Done()
	} else {
		var wg sync.WaitGroup

		for slot := 0; slot < *concurrentPlayers; slot++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					// Create a random from_id and name
					fid := int(rand.Int31())
					name := fmt.Sprintf("Sim%08x", fid)
					log.Printf("=== New player %s (id %d) ===", name, fid)

					log.Printf("Creating GenAI Client at endpoint %s (from_id=%d, to_id=%d)", *genAiEndpoint, fid, *toId)
					genAiConn := initClient(*genAiEndpoint, *genAiContext, "GenAI", *genAiNpc, fid, *toId)

					log.Printf("%s: Creating client at endpoint %s, sending prompt: %s", name, *simEndpoint, *prompt)
					simConn := initClient(*simEndpoint, *simContext, name, *simNpc, *toId, *toId)

					chatHistory := []Message{{Author: simConn.name, Content: *prompt}}
					autonomousChat(*prompt, genAiConn, simConn, *numChats, *stopPhrase, chatHistory)
				}
			}()
		}
		wg.Wait()
	}

	log.Printf("Shutting down the Game Server.")
	shutdownErr := s.Shutdown()
	if shutdownErr != nil {
		log.Printf("Could not shutdown")
	}
	os.Exit(0)
}

func initClient(endpoint string, context string, name string, npc bool, fromID int, toID int) *connection {
	// TODO: create option for a client certificate
	client := &http.Client{}
	return &connection{client: client, endpoint: endpoint, context: context, name: name, npc: npc, fromId: fromID, toId: toID}
}

type connection struct {
	client   *http.Client
	endpoint string // Full base URL for API requests
	context  string
	name     string // Human readable name for the connection
	npc      bool   // True if the endpoint is the NPC API
	fromId   int    // For use with NPC API, sender ID
	toId     int    // For use with NPC API, receiver ID
	// TODO: create options for routes off the base URL
}

// For use with Vertex APIs
type GenAIRequest struct {
	Context     string    `json:"context,omitempty"` // Optional
	Prompt      string    `json:"prompt,omitempty"`
	ChatHistory []Message `json:"messages,omitempty"` // Optional, stores chat history for use with Vertex Chat API
}

// For use with NPC API
type NPCRequest struct {
	Msg    string `json:"message,omitempty"`
	FromId int    `json:"from_id,omitempty"`
	ToId   int    `json:"to_id,omitempty"`
}

// Expected format for the NPC endpoint response
type NPCResponse struct {
	Response string `json:"response"`
}

// Conversation history provided to the model in a structured alternate-author form.
// https://cloud.google.com/vertex-ai/docs/generative-ai/model-reference/text-chat
type Message struct {
	Author  string `json:"author"`
	Content string `json:"content"`
}

func handleGenAIRequest(prompt string, clientConn *connection, chatHistory []Message) (string, error) {
	var jsonStr []byte
	var err error
	// If the endpoint is the NPC API, use the json request format specific to that API
	if clientConn.npc {
		npcRequest := NPCRequest{
			Msg:    prompt,
			FromId: clientConn.fromId,
			ToId:   clientConn.toId,
		}
		jsonStr, err = json.Marshal(npcRequest)
	} else {
		// Vertex expects the author to be "user" for user generated messages and "bot" for messages it previously sent.
		// Translate the chat history we have using the connection names.
		//
		// You can think of `prompt` as the message that "user" is sending to "bot", meaning chatHistory should always
		// end with "bot".
		var ch []Message
		for _, chat := range chatHistory {
			newChat := Message{Content: chat.Content}
			if chat.Author == clientConn.name {
				newChat.Author = "user"
			} else {
				newChat.Author = "bot"
			}
			ch = append(ch, newChat)
		}
		if len(ch) > 0 && ch[len(ch)-1].Author != "bot" {
			log.Fatalf("Chat history does not end in 'bot': %#v", ch)
		}

		genAIRequest := GenAIRequest{
			Context:     clientConn.context,
			Prompt:      prompt,
			ChatHistory: ch,
		}
		jsonStr, err = json.Marshal(genAIRequest)
	}
	if err != nil {
		return "", fmt.Errorf("unable to marshal json request: %v", err)
	}

	req, err := http.NewRequest("POST", clientConn.endpoint, bytes.NewBuffer(jsonStr))
	if err != nil {
		return "", fmt.Errorf("unable create http POST request: %v", err)
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := clientConn.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("unable to post request: %v", err)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("unable to read response body: %v", err)
	}
	defer resp.Body.Close()
	body := string(responseBody)

	if resp.StatusCode != 200 {
		err = fmt.Errorf("Status: %s, Body: %s", resp.Status, body)
	}
	return string(responseBody) + "\n", err
}

// Two AIs (connection endpoints) talking to each other
func autonomousChat(prompt string, conn1 *connection, conn2 *connection, numChats int, stopPhase string, chatHistory []Message) {
	if numChats <= 0 {
		return
	}

	startTime := time.Now()
	response, err := handleGenAIRequest(prompt, conn1, chatHistory)
	latency := time.Now().Sub(startTime)
	if err != nil {
		log.Printf("ERROR: Could not send request (stopping this chat): %v", err)
		return
	}
	// If we sent the request to the NPC endpoint we need to parse the json response {response: "response"}
	if conn1.npc {
		npcResponse := NPCResponse{}
		err = json.Unmarshal([]byte(response), &npcResponse)
		if err != nil {
			log.Fatalf("FATAL ERROR: Unable to unmarshal NPC endpoint response: %v", err)
		}
		response = npcResponse.Response
	}
	log.Printf("%s->%s [%d turns left]: %s\n", conn1.name, conn2.name, numChats, response)
	log.Printf("%s PREDICTION RATE: %0.2f b/s", conn1.name, float64(len(response))/latency.Seconds())

	chat := Message{Author: conn1.name, Content: response}
	chatHistory = append(chatHistory, chat)

	numChats -= 1

	if strings.Contains(response, stopPhase) {
		if numChats > 1 {
			numChats = 1
		}
		log.Printf("%s stop received, final turn\n", conn1.name)
	}

	// Flip between the connection that the response is sent to.
	autonomousChat(response, conn2, conn1, numChats, stopPhase, chatHistory)
}

// Manually interact via TCP with the GenAI endpoint
func tcpListener(port string, genAiConn *connection) {
	log.Printf("Starting TCP server, listening on port %s", port)
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Could not start TCP server: %v", err)
	}
	defer ln.Close() // nolint: errcheck

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatalf("Unable to accept incoming TCP connection: %v", err)
		}
		go tcpHandleConnection(conn, genAiConn)
	}
}

// handleConnection services a single tcp connection to the GenAI endpoint
func tcpHandleConnection(conn net.Conn, genAiConn *connection) {
	log.Printf("TCP Client %s connected", conn.RemoteAddr().String())

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		txt := scanner.Text()
		log.Printf("TCP txt: %v", txt)

		// TODO: update with chathistroy
		response, err := handleGenAIRequest(txt, genAiConn, nil)
		if err != nil {
			response = "ERROR: " + err.Error() + "\n"
		}

		if _, err := conn.Write([]byte(response)); err != nil {
			log.Fatalf("Could not write to TCP stream: %v", err)
		}
	}

	log.Printf("TCP Client %s disconnected", conn.RemoteAddr().String())
}

// doHealth sends the regular Health Pings
func doHealth(sdk *sdk.SDK, ctx context.Context) {
	tick := time.Tick(2 * time.Second)
	for {
		err := sdk.Health()
		if err != nil {
			log.Fatalf("Could not send health ping, %v", err)
		}
		select {
		case <-ctx.Done():
			log.Print("Stopped health pings")
			return
		case <-tick:
		}
	}
}
