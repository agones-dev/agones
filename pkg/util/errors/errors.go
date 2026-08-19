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

// Package errors provides a set of utilities for working with errors.
//
// Errors created through this package are prefixed with where they came from,
// so that an error message alone is enough to locate its source:
//
//	var errs = errors.FromPackage()
//
//	func Run() error {
//		return errs.New("failed to wait for caches to sync")
//		// "agones.dev/agones/pkg/gameservers: failed to wait for caches to sync"
//	}
//
// or, scoped to a struct:
//
//	type Controller struct {
//		errs *errors.Errors
//	}
//
//	func NewController() *Controller {
//		c := &Controller{}
//		c.errs = errors.FromStruct(c)
//		return c
//	}
//
//	func (c *Controller) Run() error {
//		return c.errs.New("failed to wait for caches to sync")
//		// "agones.dev/agones/pkg/gameservers.Controller: failed to wait for caches to sync"
//	}
package errors

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
)

// Errors creates errors prefixed with their origin, in the form "package: msg",
// or "package.Struct: msg" when created from a struct. Create one with
// FromPackage or FromStruct.
//
// The zero value is usable, and produces errors without a prefix.
type Errors struct {
	// pkg is the full import path, e.g. "agones.dev/agones/pkg/gameservers".
	pkg string
	// structName is the struct name, e.g. "Controller". Empty when created via FromPackage.
	structName string
}

// FromPackage returns an Errors for the package it is called from, determined
// by inspecting the calling function at runtime. It is intended to be assigned
// to a package level variable:
//
//	var errs = errors.FromPackage()
func FromPackage() *Errors {
	return &Errors{pkg: callerPackage(2)}
}

// FromStruct returns an Errors for the type T, recording both the struct name
// and the package it is declared in. It takes a pointer so that T is inferred
// from the call site:
//
//	c := &Controller{}
//	c.errs = errors.FromStruct(c)
//
// Only the argument's type is used, never its value, so a typed nil pointer
// works as well as a populated one.
//
// If T is an unnamed type, such as a pointer to a builtin or to an anonymous
// struct, this falls back to the package FromStruct was called from, with no
// struct name.
func FromStruct[T any](_ *T) *Errors {
	// T gives the type directly, so there is no nil value to guard against
	t := reflect.TypeFor[T]()
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	// Unnamed types (builtins, anonymous structs) have no import path to report,
	// so fall back to the caller's package.
	if t.PkgPath() == "" {
		return &Errors{pkg: callerPackage(2)}
	}

	return &Errors{pkg: t.PkgPath(), structName: t.Name()}
}

// New returns an error with msg, prefixed as described on Errors.
func (e *Errors) New(msg string) error {
	return errors.New(e.prefix(msg))
}

// Errorf returns an error formatted per fmt.Errorf, prefixed as described on
// Errors. A %w verb in format wraps as usual.
func (e *Errors) Errorf(format string, a ...any) error {
	return fmt.Errorf(e.prefix(format), a...)
}

// Wrap returns an error annotating err with msg, prefixed as described on
// Errors. The returned error wraps err, so errors.Is and errors.As traverse it.
//
// Wrap returns nil if err is nil.
func (e *Errors) Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", e.prefix(msg), err)
}

// Wrapf returns an error annotating err with a formatted msg, prefixed as described on
// Errors. The returned error wraps err, so errors.Is and errors.As traverse it.
//
// Wrapf returns nil if err is nil.
func (e *Errors) Wrapf(err error, format string, a ...any) error {
	return e.Wrap(err, fmt.Sprintf(format, a...))
}

// prefix returns msg prefixed with the package and struct name, omitting either
// segment if it is empty.
func (e *Errors) prefix(msg string) string {
	switch {
	case e.pkg == "" && e.structName == "":
		return msg
	case e.structName == "":
		return e.pkg + ": " + msg
	case e.pkg == "":
		return e.structName + ": " + msg
	default:
		return e.pkg + "." + e.structName + ": " + msg
	}
}

// callerPackage returns the import path of the package skip frames up the
// stack, where skip is counted as by runtime.Caller. Returns an empty string if
// the caller cannot be determined.
func callerPackage(skip int) string {
	pc, _, _, ok := runtime.Caller(skip)
	if !ok {
		return ""
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return ""
	}
	return packageFromFunc(fn.Name())
}

// packageFromFunc extracts the import path from a fully qualified function name
// as reported by runtime.Func.Name, e.g.
// "agones.dev/agones/pkg/gameservers.(*Controller).Run" yields
// "agones.dev/agones/pkg/gameservers".
//
// Path segments may contain dots (as in "agones.dev"), so only the final
// segment is searched: the import path ends at the first dot within it.
func packageFromFunc(name string) string {
	lastSlash := strings.LastIndex(name, "/")
	dot := strings.Index(name[lastSlash+1:], ".")
	if dot < 0 {
		return name
	}
	return name[:lastSlash+1+dot]
}
