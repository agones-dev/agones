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

// Package main is a very simple server with UDP (default), TCP, or both
package main

import (
	"bufio"
	"context"
	"flag"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	coresdk "agones.dev/agones/pkg/sdk"
	"agones.dev/agones/pkg/util/signals"
	sdk "agones.dev/agones/sdks/go"
)

// main starts a UDP or TCP server
func main() {
	sigCtx, _ := signals.NewSigKillContext()

	port := flag.String("port", "7654", "The port to listen to traffic on")
	passthrough := flag.Bool("passthrough", false, "Get listening port from the SDK, rather than use the 'port' value")
	readyOnStart := flag.Bool("ready", true, "Mark this GameServer as Ready on startup")
	shutdownDelayMin := flag.Int("automaticShutdownDelayMin", 0, "[Deprecated] If greater than zero, automatically shut down the server this many minutes after the server becomes allocated (please use automaticShutdownDelaySec instead)")
	shutdownDelaySec := flag.Int("automaticShutdownDelaySec", 0, "If greater than zero, automatically shut down the server this many seconds after the server becomes allocated (cannot be used if automaticShutdownDelayMin is set)")
	readyDelaySec := flag.Int("readyDelaySec", 0, "If greater than zero, wait this many seconds each time before marking the game server as ready")
	readyIterations := flag.Int("readyIterations", 0, "If greater than zero, return to a ready state this number of times before shutting down")
	gracefulTerminationDelaySec := flag.Int("gracefulTerminationDelaySec", 0, "Delay after we've been asked to terminate (by SIGKILL or automaticShutdownDelaySec)")
	udp := flag.Bool("udp", true, "Server will listen on UDP")
	tcp := flag.Bool("tcp", false, "Server will listen on TCP")

	flag.Parse()
	if ep := os.Getenv("PORT"); ep != "" {
		port = &ep
	}
	if epass := os.Getenv("PASSTHROUGH"); epass != "" {
		p := strings.ToUpper(epass) == "TRUE"
		passthrough = &p
	}
	if eready := os.Getenv("READY"); eready != "" {
		r := strings.ToUpper(eready) == "TRUE"
		readyOnStart = &r
	}
	if eudp := os.Getenv("UDP"); eudp != "" {
		u := strings.ToUpper(eudp) == "TRUE"
		udp = &u
	}
	if etcp := os.Getenv("TCP"); etcp != "" {
		t := strings.ToUpper(etcp) == "TRUE"
		tcp = &t
	}

	// Check for incompatible flags.
	if *shutdownDelayMin > 0 && *shutdownDelaySec > 0 {
		log.Fatalf("Cannot set both --automaticShutdownDelayMin and --automaticShutdownDelaySec")
	}
	if *readyIterations > 0 && *shutdownDelayMin <= 0 && *shutdownDelaySec <= 0 {
		log.Fatalf("Must set a shutdown delay if using ready iterations")
	}

	log.Print("Creating SDK instance")
	s, err := sdk.NewSDK()
	if err != nil {
		log.Fatalf("Could not connect to sdk: %v", err)
	}

	log.Print("Starting Health Ping")
	ctx, cancel := context.WithCancel(context.Background())
	go doHealth(s, ctx)

	if *passthrough {
		var gs *coresdk.GameServer
		gs, err = s.GameServer()
		if err != nil {
			log.Fatalf("Could not get gameserver port details: %s", err)
		}

		p := strconv.FormatInt(int64(gs.Status.Ports[0].Port), 10)
		port = &p
	}

	if *tcp {
		go tcpListener(port, s, cancel)
	}

	if *udp {
		go udpListener(port, s, cancel)
	}

	if *shutdownDelaySec > 0 {
		shutdownAfterNAllocations(s, *readyIterations, *shutdownDelaySec)
	} else if *shutdownDelayMin > 0 {
		shutdownAfterNAllocations(s, *readyIterations, *shutdownDelayMin*60)
	}

	if *readyOnStart {
		if *readyDelaySec > 0 {
			log.Printf("Waiting %d seconds before moving to ready", *readyDelaySec)
			time.Sleep(time.Duration(*readyDelaySec) * time.Second)
		}
		log.Print("Marking this server as ready")
		err := s.Ready()
		if err != nil {
			log.Fatalf("Could not send ready message")
		}
	}

	<-sigCtx.Done()
	log.Printf("Waiting %d seconds before exiting", *gracefulTerminationDelaySec)
	time.Sleep(time.Duration(*gracefulTerminationDelaySec) * time.Second)
	os.Exit(0)
}

// shutdownAfterNAllocations creates a callback to automatically shut down
// the server a specified number of seconds after the server becomes
// allocated the Nth time.
//
// The algorithm is:
//
//  1. Move the game server back to ready N times after it is allocated
//  2. Shutdown the game server after the Nth time is becomes allocated
//
// This follows the integration pattern documented on the website at
// https://agones.dev/site/docs/integration-patterns/reusing-gameservers/
func shutdownAfterNAllocations(s *sdk.SDK, readyIterations, shutdownDelaySec int) {
	gs, err := s.GameServer()
	if err != nil {
		log.Fatalf("Could not get game server: %v", err)
	}
	log.Printf("Initial game Server state = %s", gs.Status.State)

	m := sync.Mutex{} // protects the following two variables
	lastAllocated := gs.ObjectMeta.Annotations["agones.dev/last-allocated"]
	remainingIterations := readyIterations

	if err := s.WatchGameServer(func(gs *coresdk.GameServer) {
		m.Lock()
		defer m.Unlock()
		la := gs.ObjectMeta.Annotations["agones.dev/last-allocated"]
		log.Printf("Watch Game Server callback fired. State = %s, Last Allocated = %q", gs.Status.State, la)
		if lastAllocated != la {
			log.Println("Game Server Allocated")
			lastAllocated = la
			remainingIterations--
			// Run asynchronously
			go func(iterations int) {
				time.Sleep(time.Duration(shutdownDelaySec) * time.Second)

				if iterations > 0 {
					log.Println("Moving Game Server back to Ready")
					readyErr := s.Ready()
					if readyErr != nil {
						log.Fatalf("Could not set game server to ready: %v", readyErr)
					}
					log.Println("Game Server is Ready")
					return
				}

				log.Println("Moving Game Server to Shutdown")
				if shutdownErr := s.Shutdown(); shutdownErr != nil {
					log.Fatalf("Could not shutdown game server: %v", shutdownErr)
				}
				// The process will exit when Agones removes the pod and the
				// container receives the SIGTERM signal
				return
			}(remainingIterations)
		}
	}); err != nil {
		log.Fatalf("Could not watch Game Server events, %v", err)
	}
}

func udpListener(port *string, s *sdk.SDK, cancel context.CancelFunc) {
	log.Printf("Starting UDP server, listening on port %s", *port)
	conn, err := net.ListenPacket("udp", ":"+*port)
	if err != nil {
		log.Fatalf("Could not start UDP server: %v", err)
	}
	defer conn.Close() // nolint: errcheck
	udpReadWriteLoop(conn, cancel, s)
}

func udpReadWriteLoop(conn net.PacketConn, cancel context.CancelFunc, s *sdk.SDK) {
	b := make([]byte, 1024)
	for {
		sender, txt := readPacket(conn, b)

		log.Printf("Received UDP: %v", txt)

		response, addACK, err := handleResponse(txt, s, cancel)
		if err != nil {
			response = "ERROR: " + response + "\n"
		} else if addACK {
			response = "ACK: " + response + "\n"
		}

		udpRespond(conn, sender, response)

		if txt == "EXIT" {
			exit(s)
		}
	}
}

// respond responds to a given sender.
func udpRespond(conn net.PacketConn, sender net.Addr, txt string) {
	if _, err := conn.WriteTo([]byte(txt), sender); err != nil {
		log.Fatalf("Could not write to udp stream: %v", err)
	}
}

func tcpListener(port *string, s *sdk.SDK, cancel context.CancelFunc) {
	log.Printf("Starting TCP server, listening on port %s", *port)
	ln, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		log.Fatalf("Could not start TCP server: %v", err)
	}
	defer ln.Close() // nolint: errcheck

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Unable to accept incoming TCP connection: %v", err)
		}
		go tcpHandleConnection(conn, s, cancel)
	}
}

// handleConnection services a single tcp connection to the server
func tcpHandleConnection(conn net.Conn, s *sdk.SDK, cancel context.CancelFunc) {
	log.Printf("TCP Client %s connected", conn.RemoteAddr().String())
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		tcpHandleCommand(conn, scanner.Text(), s, cancel)
	}
	log.Printf("TCP Client %s disconnected", conn.RemoteAddr().String())
}

func tcpHandleCommand(conn net.Conn, txt string, s *sdk.SDK, cancel context.CancelFunc) {
	log.Printf("TCP txt: %v", txt)

	response, addACK, err := handleResponse(txt, s, cancel)
	if err != nil {
		response = "ERROR: " + response + "\n"
	} else if addACK {
		response = "ACK TCP: " + response + "\n"
	}

	tcpRespond(conn, response)

	if response == "EXIT" {
		exit(s)
	}
}

// respond responds to a given sender.
func tcpRespond(conn net.Conn, txt string) {
	log.Printf("Responding to TCP with %q", txt)
	if _, err := conn.Write([]byte(txt + "\n")); err != nil {
		log.Fatalf("Could not write to TCP stream: %v", err)
	}
}

// readPacket reads a string from the connection
func readPacket(conn net.PacketConn, b []byte) (net.Addr, string) {
	n, sender, err := conn.ReadFrom(b)
	if err != nil {
		log.Fatalf("Could not read from udp stream: %v", err)
	}
	txt := strings.TrimSpace(string(b[:n]))
	log.Printf("Received packet from %v: %v", sender.String(), txt)
	return sender, txt
}

// doHealth sends the regular Health Pings
func doHealth(sdk *sdk.SDK, ctx context.Context) {
	tick := time.Tick(2 * time.Second)
	for {
		log.Printf("Health Ping")
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
