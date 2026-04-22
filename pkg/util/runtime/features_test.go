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

package runtime

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestFeatures(t *testing.T) {
	t.Parallel()
	FeatureTestMutex.Lock()
	defer FeatureTestMutex.Unlock()

	orig := featureDefaults
	// stable feature flag state
	featureDefaults = map[Feature]bool{
		FeatureExample:  true,
		Feature("Test"): false,
	}

	t.Run("invalid Feature gate", func(t *testing.T) {
		err := ParseFeatures("Foo")
		assert.EqualError(t, err, "Feature Gate \"Foo\" is not a valid Feature Gate")
	})

	t.Run("Empty query string", func(t *testing.T) {
		err := ParseFeatures("")
		assert.NoError(t, err)
		assert.True(t, FeatureEnabled(FeatureExample))
		assert.Equal(t, "Example=true&Test=false", EncodeFeatures())
	})

	t.Run("query string", func(t *testing.T) {
		err := ParseFeatures("Example=0&Test=1")
		assert.NoError(t, err)
		assert.False(t, FeatureEnabled(FeatureExample))
		assert.True(t, FeatureEnabled(Feature("Test")))
		assert.Equal(t, "Example=false&Test=true", EncodeFeatures())
		err = ParseFeatures(EncodeFeatures())
		assert.NoError(t, err)
		assert.False(t, FeatureEnabled(FeatureExample))
		assert.True(t, FeatureEnabled(Feature("Test")))
	})

	t.Run("Error on query parsing", func(t *testing.T) {
		err := ParseFeatures("Example=foobar")
		assert.Error(t, err)
	})

	t.Run("parse env vars", func(t *testing.T) {
		assert.NoError(t, os.Setenv("FEATURE_GATES", "Test=true"))

		FeaturesBindFlags()
		pflag.Parse()
		viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
		assert.NoError(t, FeaturesBindEnv())
		assert.NoError(t, ParseFeaturesFromEnv())

		assert.True(t, FeatureEnabled(Feature("Test")))
		assert.True(t, FeatureEnabled(FeatureExample))
	})

	// reset things back the way they were
	featureDefaults = orig
}

func TestEnableAllFeatures(t *testing.T) {
	t.Parallel()
	FeatureTestMutex.Lock()
	defer FeatureTestMutex.Unlock()

	orig := featureDefaults
	// stable feature flag state
	featureDefaults = map[Feature]bool{
		FeatureExample:  true,
		Feature("Test"): false,
	}

	err := ParseFeatures("Example=0&Test=0")
	assert.NoError(t, err)
	assert.False(t, FeatureEnabled("Example"))
	assert.False(t, FeatureEnabled("Test"))

	EnableAllFeatures()

	assert.True(t, FeatureEnabled("Example"))
	assert.True(t, FeatureEnabled("Test"))

	// reset things back the way they were
	featureDefaults = orig
}
