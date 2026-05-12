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
	"net/url"
	"strconv"
	"sync"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	// FeatureGateFlag is a name of a command line flag, which turns on specific tests for FeatureGates
	FeatureGateFlag = "feature-gates"

	////////////////
	// Stable features

	// FeatureDisableResyncOnSDKServer is a feature flag to enable/disable resync on SDK server.
	FeatureDisableResyncOnSDKServer Feature = "DisableResyncOnSDKServer"

	// FeatureAutopilotPassthroughPort is a feature flag that enables/disables Passthrough Port Policy.
	FeatureAutopilotPassthroughPort Feature = "AutopilotPassthroughPort"

	////////////////
	// Beta features

	// FeatureCountsAndLists is a feature flag that enables counts and lists feature
	// (a generic implenetation of the player tracking feature).
	FeatureCountsAndLists Feature = "CountsAndLists"

	// FeatureGKEAutopilotExtendedDurationPods enables the use of Extended Duration pods
	// when Agones is running on Autopilot. Available on 1.28+ only.
	FeatureGKEAutopilotExtendedDurationPods = "GKEAutopilotExtendedDurationPods"

	// FeaturePortPolicyNone is a feature flag to allow setting Port Policy to None.
	FeaturePortPolicyNone Feature = "PortPolicyNone"

	// FeaturePortRanges is a feature flag to enable/disable specific port ranges.
	FeaturePortRanges Feature = "PortRanges"

	// FeatureRollingUpdateFix is a feature flag to enable/disable fleet controller fixes.
	FeatureRollingUpdateFix Feature = "RollingUpdateFix"

	// FeatureScheduledAutoscaler is a feature flag to enable/disable scheduled fleet autoscaling.
	FeatureScheduledAutoscaler Feature = "ScheduledAutoscaler"

	// FeatureSidecarContainers is a feature flag to enable/disable k8s sidecar containers for the sdkserver
	FeatureSidecarContainers = "SidecarContainers"

	// FeatureFleetAutoscaleRequestMetaData is a feature flag that enables/disables fleet metadata on webhook autoscaler request.
	FeatureFleetAutoscaleRequestMetaData Feature = "FleetAutoscaleRequestMetaData"

	////////////////
	// Alpha features

	// FeaturePlayerAllocationFilter is a feature flag that enables the ability for Allocations to filter based on
	// player capacity.
	FeaturePlayerAllocationFilter Feature = "PlayerAllocationFilter"

	// FeaturePlayerTracking is a feature flag to enable/disable player tracking features.
	FeaturePlayerTracking Feature = "PlayerTracking"

	// FeatureWasmAutoscaler is a feature flag to enable/disable the script based autoscaler.
	FeatureWasmAutoscaler Feature = "WasmAutoscaler"

	// FeatureGameServerScheduledRestart is a feature flag to enable/disable the script based RestartPolicy.
	FeatureGameServerScheduledRestart Feature = "GameServerScheduledRestart"

	////////////////
	// Dev features

	// FeatureProcessorAllocator is a feature flag to enable/disable the processor allocator feature.
	FeatureProcessorAllocator = "ProcessorAllocator"

	////////////////
	// Example feature

	// FeatureExample is an example feature gate flag, used for testing and demonstrative purposes
	FeatureExample Feature = "Example"
)

var (
	// featureDefaults is a map of all Feature Gates that are
	// operational in Agones, and what their default configuration is.
	// dev & alpha features are disabled by default; beta features are enabled.
	//
	// To add a new dev feature (an in progress feature, not tested in CI and not publicly documented):
	// * add a const above
	// * add it to `featureDefaults`
	// * add it to install/helm/agones/defaultfeaturegates.yaml
	// * note: you can add a new feature as an alpha feature if you're ready to test it in CI
	//
	// To promote a feature from dev->alpha:
	// * add it to `ALPHA_FEATURE_GATES` in build/Makefile
	// * add the inverse to the e2e-runner config in cloudbuild.yaml
	// * add it to site/content/en/docs/Guides/feature-stages.md
	// * add it to test/upgrade/versionMap.yaml
	// * Ensure that the features in each file are organized categorically and alphabetically.
	//
	// To promote a feature from alpha->beta:
	// * move from `false` to `true` in `featureDefaults`.
	// * move from `false` to `true` in install/helm/agones/defaultfeaturegates.yaml
	// * remove from `ALPHA_FEATURE_GATES` in build/Makefile
	// * add to `BETA_FEATURE_GATES` in build/Makefile
	// * invert in the e2e-runner config in cloudbuild.yaml
	// * change the value in site/content/en/docs/Guides/feature-stages.md.
	// * add it to test/upgrade/versionMap.yaml
	// * Ensure that the features in each file are organized categorically and alphabetically.
	//
	// Feature Promotion: alpha->beta for SDK Functions
	// * Move methods from alpha->beta files:
	// 		- From proto/sdk/alpha/alpha.proto to proto/sdk/beta/beta.proto
	//		- For each language-specific SDK (e.g., Go, C#, Rust):
	//			- Move implementation files (e.g., alpha.go to beta.go)
	//			- Move test files (e.g., alpha_test.go to beta_test.go)
	//			- Note: Delete references to 'alpha' in the moved alpha methods.
	// * Change all code and documentation references of alpha->beta:
	//		- Proto Files: proto/sdk/sdk.proto `[Stage:Alpha]->[Stage:Beta]`
	//		- SDK Implementations: Update in language-specific SDKs (e.g., sdks/go/sdk.go, sdks/csharp/sdk/AgonesSDK.cs).
	//		- Examples & Tests: Adjust in files like examples/simple-game-server/main.go and language-specific test files.
	// * Modify automation scripts in the build/build-sdk-images directory to support beta file generation.
	// * Run `make gen-all-sdk-grpc` to generate the required files. If there are changes to the `proto/allocation/allocation.proto` run `make gen-allocation-grpc`.
	// * Afterwards, execute the `make run-sdk-conformance-test-go` command and address any issues that arise.
	// * NOTE: DO NOT EDIT any autogenerated code. `make gen-all-sdk-grpc` will take care of it.
	//
	// To promote a feature from beta->GA:
	// * Remove all places consuming the feature gate and fold logic to true.
	// * Consider cleanup - often folding a gate to true allows refactoring.
	// * Remove from the e2e-runner config in cloudbuild.yaml.
	// * Update site/content/en/docs/Guides/feature-stages.md to indicate the feature is Stable.
	// * Remove from `BETA_FEATURE_GATES` in build/Makefile.
	// * Move from 'Beta features' in pkg/util/runtime/features.go and
	//   install/helm/agones/defaultfeaturegates.yaml to 'Stable features'.
	//
	// In each of these, keep the feature sorted by descending maturity then alphabetical
	featureDefaults = map[Feature]bool{
		// Stable features
		// Note that stable features cannot be set to "false", and are here so that upgrades from a
		// previous version with the feature flag do not fail on parsing an unknown flag.
		FeatureDisableResyncOnSDKServer: true,
		FeatureAutopilotPassthroughPort: true,

		// Beta features
		FeatureCountsAndLists:                   true,
		FeatureGKEAutopilotExtendedDurationPods: true,
		FeaturePortPolicyNone:                   true,
		FeaturePortRanges:                       true,
		FeatureRollingUpdateFix:                 true,
		FeatureScheduledAutoscaler:              true,
		FeatureFleetAutoscaleRequestMetaData:    true,
		FeatureSidecarContainers:                true,

		// Alpha features
		FeaturePlayerAllocationFilter:     false,
		FeaturePlayerTracking:             false,
		FeatureWasmAutoscaler:             false,
		FeatureGameServerScheduledRestart: false,

		// Dev features
		FeatureProcessorAllocator: false,

		// Example feature
		FeatureExample: false,
	}

	// featureGates is the storage of what features are enabled
	// or disabled.
	featureGates map[Feature]bool

	// featureMutex ensures that updates to featureGates don't happen at the same time as reads.
	// this is mostly to protect tests which can change gates in parallel.
	featureMutex = sync.RWMutex{}

	// FeatureTestMutex is a mutex to be shared between tests to ensure that a test that involves changing featureGates
	// cannot accidentally run at the same time as another test that also changing feature flags.
	FeatureTestMutex sync.Mutex
)

// Feature is a type for defining feature gates.
type Feature string

// FeaturesBindFlags does the Viper arguments configuration. Call before running pflag.Parse()
func FeaturesBindFlags() {
	viper.SetDefault(FeatureGateFlag, "")
	pflag.String(FeatureGateFlag, viper.GetString(FeatureGateFlag), "Flag to pass in the url query list of feature flags to enable or disable")
}

// FeaturesBindEnv binds the environment variables, based on the flags provided.
// call after viper.SetEnvKeyReplacer(...) if it is being set.
func FeaturesBindEnv() error {
	return viper.BindEnv(FeatureGateFlag)
}

// ParseFeaturesFromEnv will parse the feature flags from the Viper args
// configured by FeaturesBindFlags() and FeaturesBindEnv()
func ParseFeaturesFromEnv() error {
	return ParseFeatures(viper.GetString(FeatureGateFlag))
}

// ParseFeatures parses the url encoded query string of features and stores the value
// for later retrieval
func ParseFeatures(queryString string) error {
	featureMutex.Lock()
	defer featureMutex.Unlock()

	features := map[Feature]bool{}
	// copy the defaults into this map
	for k, v := range featureDefaults {
		features[k] = v
	}

	values, err := url.ParseQuery(queryString)
	if err != nil {
		return errors.Wrap(err, "error parsing query string for feature gates")
	}

	for k := range values {
		f := Feature(k)

		if _, ok := featureDefaults[f]; !ok {
			return errors.Errorf("Feature Gate %q is not a valid Feature Gate", f)
		}

		b, err := strconv.ParseBool(values.Get(k))
		if err != nil {
			return errors.Wrapf(err, "error parsing bool value from flag %s ", k)
		}
		features[f] = b
	}

	featureGates = features
	return nil
}

// EnableAllFeatures turns on all feature flags.
// This is useful for libraries/processes/tests that want to
// enable all Alpha/Beta features without having to track all
// the current feature flags.
func EnableAllFeatures() {
	featureMutex.Lock()
	defer featureMutex.Unlock()

	features := map[Feature]bool{}
	// copy the defaults into this map
	for k := range featureDefaults {
		features[k] = true
	}

	featureGates = features
}

// FeatureEnabled returns if a Feature is enabled or not
func FeatureEnabled(feature Feature) bool {
	featureMutex.RLock()
	defer featureMutex.RUnlock()
	return featureGates[feature]
}

// EncodeFeatures returns the feature set as a URL encoded query string
func EncodeFeatures() string {
	values := url.Values{}
	featureMutex.RLock()
	defer featureMutex.RUnlock()

	for k, v := range featureGates {
		values.Add(string(k), strconv.FormatBool(v))
	}
	return values.Encode()
}
