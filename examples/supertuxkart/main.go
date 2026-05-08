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
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"

	sdk "agones.dev/agones/sdks/go"
	"github.com/nxadm/tail"
)

// logLocation is the path to the location of the SuperTuxKart log file
const logLocation = "/.config/supertuxkart/config-0.10/server_config.log"

// main intercepts the log file of the SuperTuxKart gameserver and uses it
// to determine if the game server is ready or not.
func main() {
	log.SetPrefix("[wrapper] ")
	input := flag.String("i", "", "the command and arguments to execute the server binary")

	// Since player tracking is not on by default, it is behind this flag.
	// If it is off, still log messages about players, but don't actually call the player tracking functions.
	enablePlayerTracking := flag.Bool("player-tracking", false, "If true, player tracking will be enabled.")
	flag.Parse()

	log.Println("Connecting to Agones with the SDK")
	s, err := sdk.NewSDK()
	if err != nil {
		log.Fatalf("could not connect to SDK: %v", err)
	}

	if *enablePlayerTracking {
		if err = s.Alpha().SetPlayerCapacity(8); err != nil {
			log.Fatalf("could not set play count: %v", err)
		}
	}

	log.Println("Starting health checking")
	go doHealth(s)

	log.Println("Starting wrapper for SuperTuxKart")
	log.Printf("Command being run for SuperTuxKart server: %s \n", *input)

	cmdString := strings.Split(*input, " ")
	command, args := cmdString[0], cmdString[1:]

	cmd := exec.Command(command, args...) // #nosec
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Start(); err != nil {
		log.Fatalf("error starting cmd: %v", err)
	}

	// SuperTuxKart refuses to output to foreground, so we're going to
	// poll the server log.
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("could not get home dir: %v", err)
	}

	t := &tail.Tail{}
	// Loop to make sure the log has been created. Let's give it at least a minute, because it can be slow.
	for i := 0; i < 60; i++ {
		time.Sleep(time.Second)

		absoluteLogPath := path.Join(home, logLocation)
		log.Printf("Checking file %s (attempt %d)", absoluteLogPath, i)
		fileInfo, err := os.Stat(absoluteLogPath)
		if os.IsNotExist(err) {
			log.Printf("File %s does not exist yet, retrying...", absoluteLogPath)
			continue
		}

		if fileInfo.Size() == 0 {
			log.Printf("File %s is empty, retrying...", absoluteLogPath)
			continue
		}

		t, err = tail.TailFile(absoluteLogPath, tail.Config{Follow: true})
		if err != nil {
			log.Printf("Error tailing file: %s", err)
			continue
		} else {
			break
		}
	}
	defer t.Cleanup()
	for line := range t.Lines {
		// Don't use the logger here. This would add multiple prefixes to the logs. We just want
		// to show the supertuxkart logs as they are, and layer the wrapper logs in with them.
		fmt.Println(line.Text)
		action, player := handleLogLine(line.Text)
		switch action {
		case "READY":
			if err := s.Ready(); err != nil {
				log.Fatal("failed to mark server ready")
			}
		case "PLAYERJOIN":
			if player == nil {
				log.Print("could not determine player")
				break
			}
			if *enablePlayerTracking {
				result, err := s.Alpha().PlayerConnect(*player)
				if err != nil {
					log.Print(err)
				} else {
					log.Print(result)
				}
			}
		case "PLAYERLEAVE":
			if player == nil {
				log.Print("could not determine player")
				break
			}
			if *enablePlayerTracking {
				result, err := s.Alpha().PlayerDisconnect(*player)
				if err != nil {
					log.Print(err)
				} else {
					log.Print(result)
				}
			}
		case "SHUTDOWN":
			if err := s.Shutdown(); err != nil {
				log.Fatal(err)
			}
			os.Exit(0)
		}
	}
	log.Fatal("tail ended")
}

// doHealth sends the regular Health Pings
func doHealth(sdk *sdk.SDK) {
	tick := time.Tick(2 * time.Second)
	for {
		if err := sdk.Health(); err != nil {
			log.Fatalf("could not send health ping: %v", err)
		}
		<-tick
	}
}

// handleLogLine compares the log line to a series of regexes to determine if any action should be taken.
// TODO: This could probably be handled better with a custom type rather than just (string, *string)
func handleLogLine(line string) (string, *string) {
	// The various regexes that match server lines
	playerJoin := regexp.MustCompile(`ServerLobby: New player (.+) with online id [0-9][0-9]?`)
	playerLeave := regexp.MustCompile(`ServerLobby: (.+) disconnected$`)
	noMorePlayers := regexp.MustCompile(`STKHost.+There are now 0 peers\.$`)
	serverStart := regexp.MustCompile(`Listening has been started`)

	// Start the server
	if serverStart.MatchString(line) {
		log.Print("server ready")
		return "READY", nil
	}

	// Player tracking
	if playerJoin.MatchString(line) {
		matches := playerJoin.FindSubmatch([]byte(line))
		player := string(matches[1])
		log.Printf("Player %s joined\n", player)
		return "PLAYERJOIN", &player
	}
	if playerLeave.MatchString(line) {
		matches := playerLeave.FindSubmatch([]byte(line))
		player := string(matches[1])
		log.Printf("Player %s disconnected", player)
		return "PLAYERLEAVE", &player
	}

	// All the players left, send a shutdown
	if noMorePlayers.MatchString(line) {
		log.Print("server has no more players. shutting down")
		return "SHUTDOWN", nil
	}
	return "", nil
}
