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

package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

const thisPkg = "agones.dev/agones/pkg/util/errors"

// pkgErrs exercises the package level variable pattern this package is designed
// for, where the calling frame is the package initialiser rather than a function.
var pkgErrs = FromPackage()

type testStruct struct{}

func TestFromPackage(t *testing.T) {
	errs := FromPackage()

	assert.Equal(t, thisPkg, errs.pkg)
	assert.Empty(t, errs.structName)
}

func TestFromPackageAtVarInit(t *testing.T) {
	assert.Equal(t, thisPkg, pkgErrs.pkg)
	assert.Empty(t, pkgErrs.structName)
}

func TestFromStruct(t *testing.T) {
	// the argument is only read for its type, so a typed nil resolves identically
	fixtures := map[string]*Errors{
		"pointer":   FromStruct(&testStruct{}),
		"typed nil": FromStruct((*testStruct)(nil)),
	}

	for name, errs := range fixtures {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, thisPkg, errs.pkg)
			assert.Equal(t, "testStruct", errs.structName)
		})
	}
}

// TestFromStructDegradation covers the unnamed types that still reach the
// fallback. Taking *T makes the accidental cases - FromStruct(nil),
// FromStruct(42), FromStruct(testStruct{}) - compile errors instead, so these
// are only reachable deliberately.
func TestFromStructDegradation(t *testing.T) {
	builtin := 42
	anon := struct{ A int }{}

	fixtures := map[string]*Errors{
		"pointer to builtin":          FromStruct(&builtin),
		"pointer to anonymous struct": FromStruct(&anon),
	}

	for name, errs := range fixtures {
		t.Run(name, func(t *testing.T) {
			// falls back to the calling package, with no struct name
			assert.Equal(t, thisPkg, errs.pkg)
			assert.Empty(t, errs.structName)
		})
	}
}

func TestPackageFromFunc(t *testing.T) {
	fixtures := map[string]string{
		"agones.dev/agones/pkg/gameservers.(*Controller).Run": "agones.dev/agones/pkg/gameservers",
		"agones.dev/agones/pkg/gameservers.init":              "agones.dev/agones/pkg/gameservers",
		"agones.dev/agones/pkg/gameservers.(*Foo[...]).Bar":   "agones.dev/agones/pkg/gameservers",
		"agones.dev/agones.Foo":                               "agones.dev/agones",
		"main.main":                                           "main",
		"main":                                                "main",
	}

	for name, expected := range fixtures {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, expected, packageFromFunc(name))
		})
	}
}

func TestNew(t *testing.T) {
	pkg := FromPackage()
	assert.EqualError(t, pkg.New("no capacity"), thisPkg+": no capacity")

	str := FromStruct(&testStruct{})
	assert.EqualError(t, str.New("no capacity"), thisPkg+".testStruct: no capacity")

	var zero Errors
	assert.EqualError(t, zero.New("no capacity"), "no capacity")

	// the constructors return *Errors, so the result is usable inline
	assert.EqualError(t, FromPackage().New("no capacity"), thisPkg+": no capacity")
	assert.EqualError(t, FromStruct(&testStruct{}).New("no capacity"), thisPkg+".testStruct: no capacity")
}

func TestErrorf(t *testing.T) {
	pkg := FromPackage()
	assert.EqualError(t, pkg.Errorf("no capacity on %s", "node-1"), thisPkg+": no capacity on node-1")

	str := FromStruct(&testStruct{})
	assert.EqualError(t, str.Errorf("no capacity on %s", "node-1"), thisPkg+".testStruct: no capacity on node-1")

	// a caller supplied %w still wraps
	sentinel := errors.New("boom")
	err := pkg.Errorf("no capacity: %w", sentinel)
	assert.EqualError(t, err, thisPkg+": no capacity: boom")
	assert.ErrorIs(t, err, sentinel)
}

func TestWrap(t *testing.T) {
	sentinel := errors.New("boom")

	pkg := FromPackage()
	err := pkg.Wrap(sentinel, "no capacity")
	assert.EqualError(t, err, thisPkg+": no capacity: boom")
	assert.ErrorIs(t, err, sentinel)

	str := FromStruct(&testStruct{})
	err = str.Wrap(sentinel, "no capacity")
	assert.EqualError(t, err, thisPkg+".testStruct: no capacity: boom")
	assert.ErrorIs(t, err, sentinel)

	// matches github.com/pkg/errors.Wrap: nil in, nil out
	assert.NoError(t, pkg.Wrap(nil, "no capacity"))
}
