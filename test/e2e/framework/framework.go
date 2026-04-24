// Copyright 2018 Google LLC All Rights Reserved.
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

// Package framework is a package helping setting up end-to-end testing across a
// Kubernetes cluster.
package framework

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"

	// required to use gcloud login see: https://github.com/kubernetes/client-go/issues/242
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	autoscaling "agones.dev/agones/pkg/apis/autoscaling/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/util/runtime"
)

// special labels that can be put on pods to trigger automatic cleanup.
const (
	AutoCleanupLabelKey   = "agones.dev/e2e-test-auto-cleanup"
	AutoCleanupLabelValue = "true"
)

// NamespaceLabel is the label that is put on all namespaces that are created
// for e2e tests.
var NamespaceLabel = map[string]string{"owner": "e2e-test"}

// Framework is a testing framework
type Framework struct {
	KubeClient      kubernetes.Interface
	AgonesClient    versioned.Interface
	GameServerImage string
	PullSecret      string
	StressTestLevel int
	PerfOutputDir   string
	Version         string
	Namespace       string
	CloudProduct    string
	WaitForState    time.Duration // default time to wait for state changes, may change based on cloud product.
}

func newFramework(kubeconfig string, qps float32, burst int) (*Framework, error) {
	logger := runtime.NewLoggerWithSource("framework")
	config, err := runtime.InClusterBuildConfig(logger, kubeconfig)
	if err != nil {
		return nil, errors.Wrap(err, "build config from flags failed")
	}

	if qps > 0 {
		config.QPS = qps
	}
	if burst > 0 {
		config.Burst = burst
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "creating new kube-client failed")
	}

	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "creating new agones-client failed")
	}

	return &Framework{
		KubeClient:   kubeClient,
		AgonesClient: agonesClient,
	}, nil
}

const (
	kubeconfigFlag      = "kubeconfig"
	gsimageFlag         = "gameserver-image"
	pullSecretFlag      = "pullsecret"
	stressTestLevelFlag = "stress"
	perfOutputDirFlag   = "perf-output"
	versionFlag         = "version"
	namespaceFlag       = "namespace"
	cloudProductFlag    = "cloud-product"
)

// ParseTestFlags Parses go test flags separately because pflag package ignores flags with '-test.' prefix
// Related issues:
// https://github.com/spf13/pflag/issues/63
// https://github.com/spf13/pflag/issues/238
func ParseTestFlags() error {
	// if we have a "___" in the arguments path, then this is IntelliJ running the test, so ignore this, as otherwise
	// it breaks.
	if strings.Contains(os.Args[0], "___") {
		logrus.Info("Running test via Intellij. Skipping Test Flag Parsing")
		return nil
	}

	var testFlags []string
	for _, f := range os.Args[1:] {
		if strings.HasPrefix(f, "-test.") {
			testFlags = append(testFlags, f)
		}
	}
	return flag.CommandLine.Parse(testFlags)
}

// NewFromFlags sets up the testing framework with the standard command line flags.
func NewFromFlags() (*Framework, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}

	viper.SetDefault(kubeconfigFlag, filepath.Join(usr.HomeDir, ".kube", "config"))
	viper.SetDefault(gsimageFlag, "us-docker.pkg.dev/agones-images/examples/simple-game-server:0.42")
	viper.SetDefault(pullSecretFlag, "")
	viper.SetDefault(stressTestLevelFlag, 0)
	viper.SetDefault(perfOutputDirFlag, "")
	viper.SetDefault(versionFlag, "")
	viper.SetDefault(runtime.FeatureGateFlag, "")
	viper.SetDefault(namespaceFlag, "")
	viper.SetDefault(cloudProductFlag, "generic")

	pflag.String(kubeconfigFlag, viper.GetString(kubeconfigFlag), "kube config path, e.g. $HOME/.kube/config")
	pflag.String(gsimageFlag, viper.GetString(gsimageFlag), "gameserver image to use for those tests")
	pflag.String(pullSecretFlag, viper.GetString(pullSecretFlag), "optional secret to be used for pulling the gameserver and/or Agones SDK sidecar images")
	pflag.Int(stressTestLevelFlag, viper.GetInt(stressTestLevelFlag), "enable stress test at given level 0-100")
	pflag.String(perfOutputDirFlag, viper.GetString(perfOutputDirFlag), "write performance statistics to the specified directory")
	pflag.String(versionFlag, viper.GetString(versionFlag), "agones controller version to be tested, consists of release version plus a short hash of the latest commit")
	pflag.String(namespaceFlag, viper.GetString(namespaceFlag), "namespace is used to isolate test runs to their own namespaces")
	pflag.String(cloudProductFlag, viper.GetString(cloudProductFlag), "cloud product of cluster references by kubeconfig; defaults to 'generic'; options are 'generic', 'gke-autopilot'")
	runtime.FeaturesBindFlags()
	pflag.Parse()

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	runtime.Must(viper.BindEnv(kubeconfigFlag))
	runtime.Must(viper.BindEnv(gsimageFlag))
	runtime.Must(viper.BindEnv(pullSecretFlag))
	runtime.Must(viper.BindEnv(stressTestLevelFlag))
	runtime.Must(viper.BindEnv(perfOutputDirFlag))
	runtime.Must(viper.BindEnv(versionFlag))
	runtime.Must(viper.BindEnv(namespaceFlag))
	runtime.Must(viper.BindEnv(cloudProductFlag))
	runtime.Must(viper.BindPFlags(pflag.CommandLine))
	runtime.Must(runtime.FeaturesBindEnv())
	runtime.Must(runtime.ParseFeaturesFromEnv())

	framework, err := newFramework(viper.GetString(kubeconfigFlag), 0, 0)
	if err != nil {
		return framework, err
	}
	framework.GameServerImage = viper.GetString(gsimageFlag)
	framework.PullSecret = viper.GetString(pullSecretFlag)
	framework.StressTestLevel = viper.GetInt(stressTestLevelFlag)
	framework.PerfOutputDir = viper.GetString(perfOutputDirFlag)
	framework.Version = viper.GetString(versionFlag)
	framework.Namespace = viper.GetString(namespaceFlag)
	framework.CloudProduct = viper.GetString(cloudProductFlag)
	framework.WaitForState = 5 * time.Minute
	if framework.CloudProduct == "gke-autopilot" {
		// Autopilot can take a little while due to autoscaling, be a little liberal.
		// Keeping it under 10m so we don't get stack track dumps at 10m as unit tests can't be extended past 10m.
		framework.WaitForState = 8 * time.Minute
	}

	logrus.WithField("gameServerImage", framework.GameServerImage).
		WithField("pullSecret", framework.PullSecret).
		WithField("stressTestLevel", framework.StressTestLevel).
		WithField("perfOutputDir", framework.PerfOutputDir).
		WithField("version", framework.Version).
		WithField("namespace", framework.Namespace).
		WithField("cloudProduct", framework.CloudProduct).
		WithField("featureGates", runtime.EncodeFeatures()).
		Info("Starting e2e test(s)")

	return framework, nil
}

// CreateGameServerAndWaitUntilReady Creates a GameServer and wait for its state to become ready.
func (f *Framework) CreateGameServerAndWaitUntilReady(t *testing.T, ns string, gs *agonesv1.GameServer) (*agonesv1.GameServer, error) {
	t.Helper()
	log := TestLogger(t)
	newGs, err := f.AgonesClient.AgonesV1().GameServers(ns).Create(context.Background(), gs, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating %v GameServer instances failed (%v): %v", gs.Spec, gs.Name, err)
	}

	log.WithField("gs", newGs.ObjectMeta.Name).Info("GameServer created, waiting for Ready")

	readyGs, err := f.WaitForGameServerState(t, newGs, agonesv1.GameServerStateReady, f.WaitForState)

	if err != nil {
		return readyGs, fmt.Errorf("waiting for %v GameServer instance readiness timed out (%v): %v",
			gs.Spec, gs.Name, err)
	}

	expectedPortCount := len(gs.Spec.Ports)
	if expectedPortCount > 0 {
		for _, port := range gs.Spec.Ports {
			if port.Protocol == agonesv1.ProtocolTCPUDP {
				expectedPortCount++
			}
		}
	}

	if len(readyGs.Status.Ports) != expectedPortCount {
		return readyGs, fmt.Errorf("ready GameServer instance has %d port(s), want %d", len(readyGs.Status.Ports), expectedPortCount)
	}

	logrus.WithField("gs", newGs.ObjectMeta.Name).Info("GameServer Ready")

	return readyGs, nil
}

// WaitForGameServerState Waits untils the gameserver reach a given state before the timeout expires (with a default logger)
func (f *Framework) WaitForGameServerState(t *testing.T, gs *agonesv1.GameServer, state agonesv1.GameServerState,
	timeout time.Duration) (*agonesv1.GameServer, error) {
	t.Helper()
	log := TestLogger(t)

	var checkGs *agonesv1.GameServer

	err := wait.PollUntilContextTimeout(context.Background(), 1*time.Second, timeout, true, func(_ context.Context) (bool, error) {
		var err error
		checkGs, err = f.AgonesClient.AgonesV1().GameServers(gs.Namespace).Get(context.Background(), gs.Name, metav1.GetOptions{})

		if err != nil {
			log.WithError(err).Warn("error retrieving GameServer")
			return false, nil
		}

		checkState := checkGs.Status.State
		if checkState == state {
			log.WithField("gs", checkGs.ObjectMeta.Name).
				WithField("currentState", checkState).
				WithField("awaitingState", state).Info("GameServer states match")
			return true, nil
		}
		if agonesv1.TerminalGameServerStates[checkState] {
			log.WithField("gs", checkGs.ObjectMeta.Name).
				WithField("currentState", checkState).
				WithField("awaitingState", state).Error("GameServer reached terminal state")
			return false, errors.Errorf("GameServer reached terminal state %s", checkState)
		}
		log.WithField("gs", checkGs.ObjectMeta.Name).
			WithField("currentState", checkState).
			WithField("awaitingState", state).Info("Waiting for states to match")

		return false, nil
	})

	return checkGs, errors.Wrapf(err, "waiting for GameServer %v/%v to be %v",
		gs.Namespace, gs.Name, state)
}

// CycleAllocations repeatedly Allocates a GameServer in the Fleet (if one is available), once every specified period.
// Each Allocated GameServer gets deleted allocDuration after it was Allocated.
// GameServers will continue to be Allocated until a message is passed to the done channel.
func (f *Framework) CycleAllocations(ctx context.Context, t *testing.T, flt *agonesv1.Fleet, period time.Duration, allocDuration time.Duration) {
	err := wait.PollUntilContextCancel(ctx, period, true, func(_ context.Context) (bool, error) {
		gsa := GetAllocation(flt)
		gsa, err := f.AgonesClient.AllocationV1().GameServerAllocations(flt.Namespace).Create(context.Background(), gsa, metav1.CreateOptions{})
		if err != nil || gsa.Status.State != allocationv1.GameServerAllocationAllocated {
			// Ignore error. Could be that the buffer was empty, will try again next cycle.
			return false, nil
		}

		// Deallocate after allocDuration.
		go func(gsa *allocationv1.GameServerAllocation) {
			time.Sleep(allocDuration)
			err := f.AgonesClient.AgonesV1().GameServers(gsa.Namespace).Delete(context.Background(), gsa.Status.GameServerName, metav1.DeleteOptions{})
			require.NoError(t, err)
		}(gsa)

		return false, nil
	})
	// Ignore wait timeout error, will always be returned when the context is cancelled at the end of the test.
	if !wait.Interrupted(err) {
		require.NoError(t, err)
	}
}

// ScaleFleet will scale a Fleet with retries to a specified replica size.
func (f *Framework) ScaleFleet(t *testing.T, log *logrus.Entry, flt *agonesv1.Fleet, replicas int32) {
	fleets := f.AgonesClient.AgonesV1().Fleets(f.Namespace)
	ctx := context.Background()

	require.Eventuallyf(t, func() bool {
		flt, err := fleets.Get(ctx, flt.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			log.WithError(err).Info("Could not get Fleet")
			return false
		}

		fltCopy := flt.DeepCopy()
		fltCopy.Spec.Replicas = replicas
		_, err = fleets.Update(ctx, fltCopy, metav1.UpdateOptions{})
		if err != nil {
			log.WithError(err).Info("Could not scale Fleet")
			return false
		}

		return true
	}, 5*time.Minute, time.Second, "Could not scale Fleet %s", flt.ObjectMeta.Name)
}

// AssertFleetCondition waits for the Fleet to be in a specific condition or fails the test if the condition can't be met in 5 minutes.
func (f *Framework) AssertFleetCondition(t *testing.T, flt *agonesv1.Fleet, condition func(*logrus.Entry, *agonesv1.Fleet) bool) {
	err := f.WaitForFleetCondition(t, flt, condition)
	require.NoError(t, err, "error waiting for fleet condition on fleet: %v", flt.Name)
}

// WaitForFleetCondition waits for the Fleet to be in a specific condition or returns an error if the condition can't be met in 5 minutes.
func (f *Framework) WaitForFleetCondition(t *testing.T, flt *agonesv1.Fleet, condition func(*logrus.Entry, *agonesv1.Fleet) bool) error {
	log := TestLogger(t).WithField("fleet", flt.Name)
	log.Info("waiting for fleet condition")
	err := wait.PollUntilContextTimeout(context.Background(), 2*time.Second, f.WaitForState, true, func(_ context.Context) (bool, error) {
		fleet, err := f.AgonesClient.AgonesV1().Fleets(flt.ObjectMeta.Namespace).Get(context.Background(), flt.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}

		return condition(log, fleet), nil
	})
	if err != nil {
		// save this to be returned later.
		resultErr := err
		log.WithField("fleetStatus", fmt.Sprintf("%+v", flt.Status)).WithError(err).
			Info("error waiting for fleet condition, dumping Fleet and Gameserver data")

		f.LogEvents(t, log, flt.ObjectMeta.Namespace, flt)

		gsList, err := f.ListGameServersFromFleet(flt)
		require.NoError(t, err)

		for i := range gsList {
			gs := gsList[i]
			log = log.WithField("gs", gs.ObjectMeta.Name)
			log.WithField("status", fmt.Sprintf("%+v", gs.Status)).Info("GameServer state dump:")
			f.LogEvents(t, log, gs.ObjectMeta.Namespace, &gs)
		}

		return resultErr
	}
	return nil
}

// WaitForFleetAutoScalerCondition waits for the FleetAutoscaler to be in a specific condition or fails the test if the condition can't be met in 2 minutes.
// nolint: dupl
func (f *Framework) WaitForFleetAutoScalerCondition(t *testing.T, fas *autoscaling.FleetAutoscaler, condition func(log *logrus.Entry, fas *autoscaling.FleetAutoscaler) bool) {
	log := TestLogger(t).WithField("fleetautoscaler", fas.Name)
	log.Info("waiting for fleetautoscaler condition")
	err := wait.PollUntilContextTimeout(context.Background(), 2*time.Second, 2*time.Minute, true, func(_ context.Context) (bool, error) {
		fleetautoscaler, err := f.AgonesClient.AutoscalingV1().FleetAutoscalers(fas.ObjectMeta.Namespace).Get(context.Background(), fas.ObjectMeta.Name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}

		return condition(log, fleetautoscaler), nil
	})
	require.NoError(t, err, "error waiting for fleetautoscaler condition on fleetautoscaler %v", fas.Name)
}

// ListGameServersFromFleet lists GameServers from a particular fleet
func (f *Framework) ListGameServersFromFleet(flt *agonesv1.Fleet) ([]agonesv1.GameServer, error) {
	var results []agonesv1.GameServer

	opts := metav1.ListOptions{LabelSelector: labels.Set{agonesv1.FleetNameLabel: flt.ObjectMeta.Name}.String()}
	gsSetList, err := f.AgonesClient.AgonesV1().GameServerSets(flt.ObjectMeta.Namespace).List(context.Background(), opts)
	if err != nil {
		return results, err
	}

	for i := range gsSetList.Items {
		gsSet := &gsSetList.Items[i]
		opts := metav1.ListOptions{LabelSelector: labels.Set{agonesv1.GameServerSetGameServerLabel: gsSet.ObjectMeta.Name}.String()}
		gsList, err := f.AgonesClient.AgonesV1().GameServers(flt.ObjectMeta.Namespace).List(context.Background(), opts)
		if err != nil {
			return results, err
		}

		results = append(results, gsList.Items...)
	}

	return results, nil
}

// FleetReadyCount returns the ready count in a fleet
func FleetReadyCount(amount int32) func(*logrus.Entry, *agonesv1.Fleet) bool {
	return func(log *logrus.Entry, fleet *agonesv1.Fleet) bool {
		log.WithField("fleetStatus", fmt.Sprintf("%+v", fleet.Status)).WithField("fleet", fleet.ObjectMeta.Name).WithField("expected", amount).Info("Checking Fleet Ready replicas")
		return fleet.Status.ReadyReplicas == amount
	}
}

// WaitForFleetGameServersCondition waits for all GameServers for a given fleet to match
// a condition specified by a callback.
func (f *Framework) WaitForFleetGameServersCondition(flt *agonesv1.Fleet,
	cond func(server *agonesv1.GameServer) bool) error {
	return f.WaitForFleetGameServerListCondition(flt,
		func(servers []agonesv1.GameServer) bool {
			for i := range servers {
				gs := &servers[i]
				if !cond(gs) {
					return false
				}
			}
			return true
		})
}

// WaitForFleetGameServerListCondition waits for the list of GameServers to match a condition
// specified by a callback and the size of GameServers to match fleet's Spec.Replicas.
func (f *Framework) WaitForFleetGameServerListCondition(flt *agonesv1.Fleet,
	cond func(servers []agonesv1.GameServer) bool) error {
	return wait.PollUntilContextTimeout(context.Background(), 2*time.Second, f.WaitForState, true, func(_ context.Context) (done bool, err error) {
		gsList, err := f.ListGameServersFromFleet(flt)
		if err != nil {
			return false, err
		}
		if int32(len(gsList)) != flt.Spec.Replicas {
			return false, nil
		}
		return cond(gsList), nil
	})
}

// NewStatsCollector returns new instance of statistics collector,
// which can be used to emit performance statistics for load tests and stress tests.
func (f *Framework) NewStatsCollector(name, version string) *StatsCollector {
	if f.StressTestLevel > 0 {
		name = fmt.Sprintf("stress_%v_%v", f.StressTestLevel, name)
	}
	return &StatsCollector{name: name, outputDir: f.PerfOutputDir, version: version}
}

// CleanUp Delete all Agones resources in a given namespace.
func (f *Framework) CleanUp(ns string) error {
	logrus.Info("Cleaning up now.")
	defer logrus.Info("Finished cleanup.")
	agonesV1 := f.AgonesClient.AgonesV1()
	deleteOptions := metav1.DeleteOptions{}
	listOptions := metav1.ListOptions{}

	// find and delete pods created by tests and labeled with our special label
	pods := f.KubeClient.CoreV1().Pods(ns)
	ctx := context.Background()
	podList, err := pods.List(ctx, metav1.ListOptions{
		LabelSelector: AutoCleanupLabelKey + "=" + AutoCleanupLabelValue,
	})
	if err != nil {
		return err
	}

	for i := range podList.Items {
		p := &podList.Items[i]
		if err := pods.Delete(ctx, p.ObjectMeta.Name, deleteOptions); err != nil {
			return err
		}
	}

	err = agonesV1.Fleets(ns).DeleteCollection(ctx, deleteOptions, listOptions)
	if err != nil {
		return err
	}

	err = f.AgonesClient.AutoscalingV1().FleetAutoscalers(ns).DeleteCollection(ctx, deleteOptions, listOptions)
	if err != nil {
		return err
	}

	return agonesV1.GameServers(ns).
		DeleteCollection(ctx, deleteOptions, listOptions)
}

// CreateAndApplyAllocation creates and applies an Allocation to a Fleet
func (f *Framework) CreateAndApplyAllocation(t *testing.T, flt *agonesv1.Fleet) *allocationv1.GameServerAllocation {
	gsa := GetAllocation(flt)
	gsa, err := f.AgonesClient.AllocationV1().GameServerAllocations(flt.ObjectMeta.Namespace).Create(context.Background(), gsa, metav1.CreateOptions{})
	require.NoError(t, err)
	require.Equal(t, string(allocationv1.GameServerAllocationAllocated), string(gsa.Status.State))
	return gsa
}

// SendGameServerUDP sends a message to a gameserver and returns its reply
// finds the first udp port from the spec to send the message to,
// returns error if no Ports were allocated
func (f *Framework) SendGameServerUDP(t *testing.T, gs *agonesv1.GameServer, msg string) (string, error) {
	if len(gs.Status.Ports) == 0 {
		return "", errors.New("Empty Ports array")
	}

	// use first udp port
	for _, p := range gs.Spec.Ports {
		if p.Protocol == corev1.ProtocolUDP {
			return f.SendGameServerUDPToPort(t, gs, p.Name, msg)
		}
	}
	return "", errors.New("No UDP ports")
}

// SendGameServerUDPToPort sends a message to a gameserver at the named port and returns its reply
// returns error if no Ports were allocated or a port of the specified name doesn't exist
func (f *Framework) SendGameServerUDPToPort(t *testing.T, gs *agonesv1.GameServer, portName string, msg string) (string, error) {
	log := TestLogger(t)
	if len(gs.Status.Ports) == 0 {
		return "", errors.New("Empty Ports array")
	}
	var port agonesv1.GameServerStatusPort
	for _, p := range gs.Status.Ports {
		if p.Name == portName {
			port = p
		}
	}
	address := fmt.Sprintf("%s:%d", gs.Status.Address, port.Port)
	reply, err := f.SendUDP(t, address, msg)

	if err != nil {
		log.WithField("gs", gs.ObjectMeta.Name).WithField("status", fmt.Sprintf("%+v", gs.Status)).Info("Failed to send UDP packet to GameServer. Dumping Events!")
		f.LogEvents(t, log, gs.ObjectMeta.Namespace, gs)
	}

	return reply, err
}

// SendUDP sends a message to an address, and returns its reply if
// it returns one in 10 seconds. Will retry 5 times, in case UDP packets drop.
func (f *Framework) SendUDP(t *testing.T, address, msg string) (string, error) {
	log := TestLogger(t).WithField("address", address)
	b := make([]byte, 1024)
	var n int
	// sometimes we get I/O timeout, so let's do a retry
	err := wait.PollUntilContextTimeout(context.Background(), 2*time.Second, time.Minute, true, func(_ context.Context) (bool, error) {
		conn, err := net.Dial("udp", address)
		if err != nil {
			log.WithError(err).Info("could not dial address")
			return false, nil
		}

		defer func() {
			err = conn.Close()
		}()

		_, err = conn.Write([]byte(msg))
		if err != nil {
			log.WithError(err).Info("could not write message to address")
			return false, nil
		}

		err = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		if err != nil {
			log.WithError(err).Info("Could not set read deadline")
			return false, nil
		}

		n, err = conn.Read(b)
		if err != nil {
			log.WithError(err).Info("Could not read from address")
		}

		return err == nil, nil
	})

	if err != nil {
		return "", errors.Wrap(err, "timed out attempting to send UDP packet to address")
	}

	return string(b[:n]), nil
}

// SendGameServerTCP sends a message to a gameserver and returns its reply
// finds the first tcp port from the spec to send the message to,
// returns error if no Ports were allocated
func SendGameServerTCP(gs *agonesv1.GameServer, msg string) (string, error) {
	if len(gs.Status.Ports) == 0 {
		return "", errors.New("Empty Ports array")
	}

	// use first tcp port
	for _, p := range gs.Spec.Ports {
		if p.Protocol == corev1.ProtocolTCP {
			return SendGameServerTCPToPort(gs, p.Name, msg)
		}
	}
	return "", errors.New("No TCP ports")
}

// SendGameServerTCPToPort sends a message to a gameserver at the named port and returns its reply
// returns error if no Ports were allocated or a port of the specified name doesn't exist
func SendGameServerTCPToPort(gs *agonesv1.GameServer, portName string, msg string) (string, error) {
	if len(gs.Status.Ports) == 0 {
		return "", errors.New("Empty Ports array")
	}
	var port agonesv1.GameServerStatusPort
	for _, p := range gs.Status.Ports {
		if p.Name == portName {
			port = p
		}
	}
	address := fmt.Sprintf("%s:%d", gs.Status.Address, port.Port)
	return SendTCP(address, msg)
}

// SendTCP sends a message to an address, and returns its reply if
// it returns one in 30 seconds
func SendTCP(address, msg string) (string, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return "", err
	}

	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return "", err
	}

	defer func() {
		if err := conn.Close(); err != nil {
			logrus.Warn("Could not close TCP connection")
		}
	}()

	// writes to the tcp connection
	_, err = fmt.Fprintln(conn, msg)
	if err != nil {
		return "", err
	}

	response, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return "", err
	}

	return response, nil
}

// GetAllocation returns a GameServerAllocation that is looking for a Ready
// GameServer from this fleet.
func GetAllocation(f *agonesv1.Fleet) *allocationv1.GameServerAllocation {
	// get an allocation
	return &allocationv1.GameServerAllocation{
		Spec: allocationv1.GameServerAllocationSpec{
			Selectors: []allocationv1.GameServerSelector{
				{LabelSelector: metav1.LabelSelector{MatchLabels: map[string]string{agonesv1.FleetNameLabel: f.ObjectMeta.Name}}},
			},
		}}
}

// CreateNamespace creates a namespace and a service account in the test cluster
func (f *Framework) CreateNamespace(namespace string) error {
	kubeCore := f.KubeClient.CoreV1()
	kubeRbac := f.KubeClient.RbacV1()
	ctx := context.Background()
	options := metav1.CreateOptions{}
	saName := "agones-sdk"
	roleName := "agones-gameserver-access"

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   namespace,
			Labels: NamespaceLabel,
		},
	}
	if _, err := kubeCore.Namespaces().Create(ctx, ns, options); err != nil {
		return errors.Errorf("creating namespace %s failed: %s", namespace, err.Error())
	}
	logrus.Infof("Namespace %s is created", namespace)

	if _, err := kubeCore.ServiceAccounts(namespace).Create(ctx, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: namespace,
			Labels:    map[string]string{"app": "agones"},
		},
	}, options); err != nil {
		err = errors.Errorf("creating ServiceAccount %s in namespace %s failed: %s", saName, namespace, err.Error())
		_ = f.DeleteNamespace(namespace) // Use _ to ignore derr since we return err anyway
		return err
	}
	logrus.Infof("ServiceAccount %s/%s is created", namespace, saName)

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: namespace,
			Labels:    map[string]string{"app": "agones"},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"agones.dev"},
				Resources: []string{"gameservers"},
				Verbs:     []string{"get", "patch", "update"},
			},
		},
	}
	if _, err := kubeRbac.Roles(namespace).Create(ctx, role, options); err != nil {
		err = errors.Errorf("creating Role %s in namespace %s failed: %s", roleName, namespace, err.Error())
		_ = f.DeleteNamespace(namespace)
		return err
	}
	logrus.Infof("Role %s/%s is created", namespace, roleName)

	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "agones-sdk-access",
			Namespace: namespace,
			Labels:    map[string]string{"app": "agones"},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     roleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      saName,
				Namespace: namespace,
			},
		},
	}
	if _, err := kubeRbac.RoleBindings(namespace).Create(ctx, rb, options); err != nil {
		err = errors.Errorf("creating RoleBinding for service account %q in namespace %q failed: %s", saName, namespace, err.Error())
		_ = f.DeleteNamespace(namespace)
		return err
	}
	logrus.Infof("RoleBinding %s/%s is created", namespace, rb.Name)

	rbCluster := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "agones-sdk-cluster-access",
			Namespace: namespace,
			Labels:    map[string]string{"app": "agones"},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "agones-sdk",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      saName,
				Namespace: namespace,
			},
		},
	}
	if _, err := kubeRbac.RoleBindings(namespace).Create(ctx, rbCluster, options); err != nil {
		logrus.Warnf("Failed to create ClusterRoleBinding for agones-sdk, relying on local Role: %s", err.Error())
	}

	return nil
}

// DeleteNamespace deletes a namespace from the test cluster
func (f *Framework) DeleteNamespace(namespace string) error {
	kubeCore := f.KubeClient.CoreV1()
	ctx := context.Background()

	// Remove finalizers
	pods, err := kubeCore.Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return errors.Errorf("listing pods in namespace %s failed: %s", namespace, err)
	}
	for i := range pods.Items {
		pod := &pods.Items[i]
		if len(pod.Finalizers) > 0 {
			pod.Finalizers = nil
			payload := []patchRemoveNoValue{{
				Op:   "remove",
				Path: "/metadata/finalizers",
			}}
			payloadBytes, _ := json.Marshal(payload)
			if _, err := kubeCore.Pods(namespace).Patch(ctx, pod.Name, types.JSONPatchType, payloadBytes, metav1.PatchOptions{}); err != nil {
				return errors.Wrapf(err, "updating pod %s failed", pod.GetName())
			}
		}
	}

	if err := kubeCore.Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{}); err != nil {
		return errors.Wrapf(err, "deleting namespace %s failed", namespace)
	}
	logrus.Infof("Namespace %s is deleted", namespace)
	return nil
}

type patchRemoveNoValue struct {
	Op   string `json:"op"`
	Path string `json:"path"`
}

// DefaultGameServer provides a default GameServer fixture, based on parameters
// passed to the Test Framework.
func (f *Framework) DefaultGameServer(namespace string) *agonesv1.GameServer {
	gs := &agonesv1.GameServer{ObjectMeta: metav1.ObjectMeta{GenerateName: "game-server", Namespace: namespace},
		Spec: agonesv1.GameServerSpec{
			Container: "game-server",
			Ports: []agonesv1.GameServerPort{{
				ContainerPort: 7654,
				Name:          "udp-port",
				PortPolicy:    agonesv1.Dynamic,
				Protocol:      corev1.ProtocolUDP,
			}},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:            "game-server",
						Image:           f.GameServerImage,
						ImagePullPolicy: corev1.PullIfNotPresent,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("30m"),
								corev1.ResourceMemory: resource.MustParse("32Mi"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("30m"),
								corev1.ResourceMemory: resource.MustParse("32Mi"),
							},
						},
					}},
				},
			},
		},
	}

	if f.PullSecret != "" {
		gs.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{
			Name: f.PullSecret}}
	}

	return gs
}

// LogEvents logs all the events for a given Kubernetes objects. Useful for debugging why something
// went wrong.
func (f *Framework) LogEvents(t *testing.T, log *logrus.Entry, namespace string, objOrRef k8sruntime.Object) {
	log.WithField("kind", objOrRef.GetObjectKind().GroupVersionKind().Kind).Info("Dumping Events:")
	events, err := f.KubeClient.CoreV1().Events(namespace).SearchWithContext(context.Background(), scheme.Scheme, objOrRef)
	require.NoError(t, err, "error searching for events")
	for i := range events.Items {
		event := events.Items[i]
		log.WithField("lastTimestamp", event.LastTimestamp).WithField("type", event.Type).WithField("reason", event.Reason).WithField("message", event.Message).Info("Event!")
	}
}

// LogPodContainers takes a Pod as an argument and attempts to output the current and previous logs from each container
// in that Pod It uses the framework's KubeClient to retrieve the logs and outputs them using the provided logger.
func (f *Framework) LogPodContainers(t *testing.T, pod *corev1.Pod) {
	log := TestLogger(t)
	log.WithField("pod", pod.Name).WithField("namespace", pod.Namespace).Info("Logs for Pod:")

	// sub-function so defer will fire on each printLogs, rather than at the end.
	printLogs := func(container corev1.Container, previous bool) {
		logOptions := &corev1.PodLogOptions{
			Container: container.Name,
			Follow:    false,
			Previous:  previous,
		}

		req := f.KubeClient.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, logOptions)
		podLogs, err := req.Stream(context.Background())
		log = log.WithField("options", logOptions)

		if err != nil {
			log.WithError(err).Warn("Error opening log stream for container")
			return
		}
		defer podLogs.Close() // nolint:errcheck,staticcheck

		logBytes, err := io.ReadAll(podLogs)
		if err != nil {
			log.WithError(err).WithField("options", logOptions).Warn("Error reading logs for container")
			return
		}

		log.Info("---Logs for container---")
		lines := strings.Split(string(logBytes), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			log.Info(line)
		}
		log.Info("---End of container logs---")
	}

	// run through the container list twice, so we group current vs previous logs nicely.
	for _, container := range pod.Spec.Containers {
		printLogs(container, false)
	}

	for _, container := range pod.Spec.Containers {
		printLogs(container, true)
	}

}

// SkipOnCloudProduct skips the test if the e2e was invoked with --cloud-product=<product>.
func (f *Framework) SkipOnCloudProduct(t *testing.T, product, reason string) {
	if f.CloudProduct == product {
		t.Skipf("skipping test on cloud product %s: %s", product, reason)
	}
}

// TestLogger returns the standard logger for helper functions.
func TestLogger(t *testing.T) *logrus.Entry {
	return logrus.WithField("test", t.Name())
}
