// Copyright Contributors to Agones a Series of LF Projects, LLC.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	var configPath string
	switch len(os.Args) {
	case 1:
		configPath = "vanity.yaml"
	case 2:
		configPath = os.Args[1]
	default:
		log.Fatal("usage: govanityurls [CONFIG]")
	}
	log.Printf("loading: %s", configPath)

	vanity, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatal(err)
	}
	h, err := newHandler(vanity)
	if err != nil {
		log.Fatal(err)
	}
	http.Handle("/", h)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func defaultHost(r *http.Request) string {
	return r.Host
}
