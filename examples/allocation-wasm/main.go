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

// Package main is an example allocation plugin compiled to WebAssembly, which allocates a single
// session on a multi-session GameServer, by taking it from a List of free sessions.
package main

import (
	"fmt"

	"github.com/extism/go-pdk"
)

const (
	defaultAnnotation = "session"
	// renamed is the value AllocateIdentity attempts to change the identity of the GameServer to.
	renamed = "renamed"
)

// Allocate takes the first value from the List named in the "list" config value, and stores it in
// an annotation on the GameServer, so it is returned as part of the allocation. The annotation
// is "session" by default, and can be changed with the "annotation" config value.
//
//go:wasmexport allocate
func Allocate() int32 {
	var gs GameServer
	if err := pdk.InputJSON(&gs); err != nil {
		pdk.SetErrorString(fmt.Sprintf("Failed to decode GameServer: %v", err))
		return 1
	}

	name, ok := pdk.GetConfig("list")
	if !ok {
		pdk.SetErrorString("the 'list' config value is required")
		return 1
	}
	annotation, ok := pdk.GetConfig("annotation")
	if !ok {
		annotation = defaultAnnotation
	}

	list, ok := gs.Status.Lists[name]
	if !ok || len(list.Values) == 0 {
		pdk.SetErrorString(fmt.Sprintf("list %q has no values to allocate", name))
		return 1
	}

	session := list.Values[0]
	list.Values = list.Values[1:]
	gs.Status.Lists[name] = list

	if gs.Metadata.Annotations == nil {
		gs.Metadata.Annotations = map[string]string{}
	}
	gs.Metadata.Annotations[annotation] = session

	return output(gs)
}

// AllocateNone returns the GameServer without changing it, demonstrating multiple export functions.
//
//go:wasmexport allocateNone
func AllocateNone() int32 {
	var gs GameServer
	if err := pdk.InputJSON(&gs); err != nil {
		pdk.SetErrorString(fmt.Sprintf("Failed to decode GameServer: %v", err))
		return 1
	}

	return output(gs)
}

// AllocateIdentity is used for testing purposes only. It attempts to change the identity and state
// of the GameServer, which Agones does not allow.
//
//go:wasmexport allocateIdentity
func AllocateIdentity() int32 {
	var gs GameServer
	if err := pdk.InputJSON(&gs); err != nil {
		pdk.SetErrorString(fmt.Sprintf("Failed to decode GameServer: %v", err))
		return 1
	}

	gs.Metadata.Name = renamed
	gs.Metadata.Namespace = renamed
	gs.Metadata.UID = renamed
	gs.Metadata.ResourceVersion = renamed
	if gs.Metadata.Labels == nil {
		gs.Metadata.Labels = map[string]string{}
	}
	gs.Metadata.Labels["changed"] = "true"
	gs.Status.State = "Ready"

	return output(gs)
}

// AllocateInvalid is used for testing purposes only. It returns a value that is not a GameServer.
//
//go:wasmexport allocateInvalid
func AllocateInvalid() int32 {
	pdk.OutputString("not a GameServer")
	return 0
}

// output returns the GameServer to Agones.
func output(gs GameServer) int32 {
	if err := pdk.OutputJSON(&gs); err != nil {
		pdk.SetErrorString(fmt.Sprintf("Failed to encode GameServer: %v", err))
		return 1
	}

	return 0
}

func main() {}
