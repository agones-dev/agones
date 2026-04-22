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

package cloudproduct

import (
	"context"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/informers/externalversions"
	"agones.dev/agones/pkg/cloudproduct/generic"
	"agones.dev/agones/pkg/cloudproduct/gke"
	"agones.dev/agones/pkg/portallocator"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

// The Cloud Product abstraction currently consists of disjoint interfaces:
// * ControllerHooks() is an interface available to the pkg/controller binaries
// * api hooks are separated into api group and registered directly with the API
//
// Users should bind flags with Bind{Flags,Env}, then call Initialize with a
// k8s client to initialize.

// ControllerHooksInterface provides a generic interface that abstracts cloud product
// specific functionality for pkg/controller packages.
type ControllerHooksInterface interface {
	agonesv1.APIHooks

	// SyncPodPortsToGameServer runs after a Pod has been assigned to a Node and before we sync
	// Pod host ports to the GameServer status.
	SyncPodPortsToGameServer(*agonesv1.GameServer, *corev1.Pod) error

	// NewPortAllocator creates a PortAllocator. See gameservers.NewPortAllocator for parameters.
	NewPortAllocator(map[string]portallocator.PortRange, informers.SharedInformerFactory, externalversions.SharedInformerFactory) portallocator.Interface

	// WaitOnFreePorts
	WaitOnFreePorts() bool
}

const (
	cloudProductFlag = "cloud-product"

	// If --cloud-product=auto, auto-detect
	autoDetectString = "auto"

	genericProduct      = "generic"
	gkeAutopilotProduct = "gke-autopilot"
)

var (
	logger           = runtime.NewLoggerWithSource("cloudproduct")
	productDetectors = []func(context.Context, *kubernetes.Clientset) string{gke.Detect}
)

// BindFlags binds the --cloud-product flag.
func BindFlags() {
	viper.SetDefault(cloudProductFlag, autoDetectString)
	pflag.String(cloudProductFlag, viper.GetString(cloudProductFlag), "Cloud product. Set to 'auto' to auto-detect, set to 'generic' to force generic behavior, set to 'gke-autopilot' for GKE Autopilot. Can also use CLOUD_PRODUCT env variable.")
}

// BindEnv binds the CLOUD_PRODUCT env variable.
func BindEnv() error {
	return viper.BindEnv(cloudProductFlag)
}

// NewFromFlag instantiates a new CloudProduct interface from the flags in BindFlags/BindEnv.
func NewFromFlag(ctx context.Context, kc *kubernetes.Clientset) (ControllerHooksInterface, error) {
	product := autoDetect(ctx, viper.GetString(cloudProductFlag), kc)

	switch product {
	case gkeAutopilotProduct:
		return gke.Autopilot(), nil
	case genericProduct:
		return generic.New(), nil
	}
	return nil, errors.Errorf("unknown cloud product: %q", product)
}

func autoDetect(ctx context.Context, product string, kc *kubernetes.Clientset) string {
	if product != autoDetectString {
		logger.Infof("Cloud product forced to %q, skipping auto-detection", product)
		return product
	}
	for _, detect := range productDetectors {
		product = detect(ctx, kc)
		if product != "" {
			logger.Infof("Cloud product detected as %q", product)
			return product
		}
	}
	logger.Infof("Cloud product defaulted to %q", genericProduct)
	return genericProduct
}
