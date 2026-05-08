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

// Package main is controller for running the Agones in-place upgrades tests
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/client/informers/externalversions"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

const (
	// HelmCmd is the command line invocation of helm
	HelmCmd = "helm"
	// KubectlCmd is the command line invocation of kubectl
	KubectlCmd = "kubectl"
	// ImagePullPolicy sets the Agones Helm configuration to always pull the image
	ImagePullPolicy = "Always"
	// SidecarPullPolicy sets the Agones Helm configuration to always pull the SDK image
	SidecarPullPolicy = "true"
	// LogLevel sets the Agones Helm configuration log level
	LogLevel = "debug"
	// Timeout sets the amount of time to wait for resources to become ready. Should be more than the
	// time for an Autopilot cluster to scale up.
	Timeout = 10 * time.Minute
	// HelmChart is the helm chart for the public Agones releases
	HelmChart = "agones/agones"
	// TestChart is the registry for Agones Helm chart development builds
	TestChart = "./install/helm"
	// AgonesRegistry is the public registry for Agones releases
	AgonesRegistry = "us-docker.pkg.dev/agones-images/release"
	// TestRegistry is the registry for Agones development builds
	TestRegistry = "us-docker.pkg.dev/agones-images/ci"
	// ContainerRegistry is the registry for upgrade test container files
	ContainerRegistry = "us-docker.pkg.dev/agones-images/ci/sdk-client-test"
	// Retries is the number of times to retry creating a game server.
	Retries = 8
)

var (
	// DevVersion is the current development version of Agones
	DevVersion = os.Getenv("DevVersion")
	// ReleaseVersion is the latest released version of Agones (DEV - 1).
	ReleaseVersion = os.Getenv("ReleaseVersion")
	// PodName the name of the pod this container is running in
	PodName = os.Getenv("PodName")
	// PodNamespace the name of the pod namespace this container is running in
	PodNamespace = os.Getenv("PodNamespace")
	// VersionMappings are the valid Kubernetes, Agones, and Feature Gate version configurations
	VersionMappings = os.Getenv("version-mappings.json")
)

func main() {
	if err := run(); err != nil {
		// Write the final error to the termination log before exiting.
		msg := err.Error()
		if writeErr := os.WriteFile("/dev/termination-log", []byte(msg), 0644); writeErr != nil {
			log.Printf("Error writing to termination log: %v", writeErr)
		}
		log.Fatal(msg)
	}
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	// Defer the primary cancel and cleanup to ensure they always run at the very end.
	defer cancel()
	defer cleanUpResources()

	// Creates an errgroup. gCtx will be canceled when the first goroutine returns a non-nil error.
	g, gCtx := errgroup.WithContext(ctx)

	cfg, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("could not create in cluster config: %w", err)
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("could not create the kubernetes api clientset: %w", err)
	}
	agonesClient, err := versioned.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("could not create the agones api clientset: %w", err)
	}

	validConfigs, err := configTestSetup(gCtx, kubeClient)
	if err != nil {
		return fmt.Errorf("could not create the configuration test setup: %w", err)
	}

	// Watch Game Servers
	g.Go(func() error {
		return watchGameServers(gCtx, agonesClient, len(validConfigs)*2)
	})

	// Watch Game Server Events
	g.Go(func() error {
		return watchGameServerEvents(gCtx, kubeClient)
	})

	// Run the main test logic
	g.Go(func() error {
		if err := addAgonesRepo(); err != nil {
			return err
		}
		// Pass cancel here to stop other go routines when the runConfigWalker returns successfully.
		return runConfigWalker(gCtx, g, validConfigs, cancel)
	})

	// g.Wait() blocks until all goroutines have returned or returns the first non-nil error.
	return g.Wait()
}

type versionMappings struct {
	K8sToAgonesVersions       map[string][]string     `json:"k8sToAgonesVersions"`
	AgonesVersionFeatureGates map[string]featureGates `json:"agonesVersionFeatureGates"`
}

type featureGates struct {
	AlphaGates []string `json:"alphaGates"`
	BetaGates  []string `json:"betaGates"`
}

type configTest struct {
	agonesVersion  string
	featureGates   string
	gameServerPath string
}

// CountsAndLists can be removed from the template once CountsAndLists is GA in all tested versions
type gameServerTemplate struct {
	AgonesVersion  string
	Registry       string
	CountsAndLists bool
}

type gsLog struct {
	SdkVersion        string
	GameServerVersion string
	GameServerState   string
}

type helmStatuses []struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Revision   string `json:"revision"`
	Updated    string `json:"updated"`
	Status     string `json:"status"`
	Chart      string `json:"chart"`
	AppVersion string `json:"app_version"`
}

// Determine test scenario to run
func configTestSetup(ctx context.Context, kubeClient *kubernetes.Clientset) ([]*configTest, error) {
	versionMap := versionMappings{}

	// Find the Kubernetes version of the node that this test is running on.
	k8sVersion, err := findK8sVersion(ctx, kubeClient)
	if err != nil {
		return nil, err
	}

	// Get the mappings of valid Kubernetes, Agones, and Feature Gate versions from the configmap.
	err = json.Unmarshal([]byte(VersionMappings), &versionMap)
	if err != nil {
		return nil, fmt.Errorf("Could not Unmarshal ", err)
	}

	// Find valid Agones versions and feature gates for the current version of Kubernetes.
	configTests := []*configTest{}
	for _, agonesVersion := range versionMap.K8sToAgonesVersions[k8sVersion] {
		ct := configTest{}
		// TODO: create different valid config based off of available feature gates.
		// containsCountsAndLists will need to be updated to return true for when CountsAndLists=true.
		countsAndLists, err := containsCountsAndLists(agonesVersion)
		if err != nil {
			return nil, err
		}
		ct.agonesVersion = agonesVersion
		if agonesVersion == "Dev" {
			ct.agonesVersion = DevVersion
			// Game server container cannot be created at DEV version due to go.mod only able to access
			// published Agones versions. Use N-1 for DEV.
			ct.gameServerPath, err = createGameServerFile(ReleaseVersion, countsAndLists)
		} else {
			ct.gameServerPath, err = createGameServerFile(agonesVersion, countsAndLists)
		}
		if err != nil {
			return nil, err
		}
		configTests = append(configTests, &ct)
	}

	return configTests, nil
}

// containsCountsAndLists returns true if the agonesVersion >= 1.41.0 when the CountsAndLists
// feature entered Beta (on by default)
func containsCountsAndLists(agonesVersion string) (bool, error) {
	if agonesVersion == "Dev" {
		return true, nil
	}
	r := regexp.MustCompile(`\d+\.\d+`)
	strVersion := r.FindString(agonesVersion)
	floatVersion, err := strconv.ParseFloat(strVersion, 64)
	if err != nil {
		return false, fmt.Errorf("Could not convert agonesVersion %s to float: %s", agonesVersion, err)
	}
	if floatVersion > 1.40 {
		return true, nil
	}
	return false, nil
}

// Finds the Kubernetes version of the Kubelet on the node that the current pod is running on.
// The Kubelet version is the same version as the node.
func findK8sVersion(ctx context.Context, kubeClient *kubernetes.Clientset) (string, error) {
	// Wait to get pod and node as these may take a while to start on a new Autopilot cluster.
	var pod *v1.Pod
	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 7*time.Minute, true, func(ctx context.Context) (done bool, err error) {
		pod, err = kubeClient.CoreV1().Pods(PodNamespace).Get(ctx, PodName, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}

		return true, nil
	})
	if pod == nil || pod.Spec.NodeName == "" {
		return "", fmt.Errorf("PollUntilContextTimeout timed out. Could not get Pod: %s", err)
	}

	var node *v1.Node
	err = wait.PollUntilContextTimeout(ctx, 5*time.Second, 3*time.Minute, true, func(ctx context.Context) (done bool, err error) {
		node, err = kubeClient.CoreV1().Nodes().Get(ctx, pod.Spec.NodeName, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}

		return true, nil
	})
	if node == nil || node.Status.NodeInfo.KubeletVersion == "" {
		return "", fmt.Errorf("PollUntilContextTimeout timed out. Could not get Node: %s", err)
	}

	// Finds the major.min version. I.e. k8sVersion 1.30 from gkeVersion v1.30.2-gke.1587003
	gkeVersion := node.Status.NodeInfo.KubeletVersion
	log.Println("KubeletVersion", gkeVersion)
	r, err := regexp.Compile(`\d+\.\d+`)
	if err != nil {
		return "", err
	}

	return r.FindString(gkeVersion), nil
}

// runExecCommand executes the command with the given arguments.
func runExecCommand(cmd string, args ...string) ([]byte, error) {
	log.Println("Running command", cmd, args)
	execCommand := exec.Command(cmd, args...)

	out, err := execCommand.CombinedOutput()
	if err != nil {
		log.Printf("CombinedOutput: %s", string(out))
		log.Printf("CombinedOutput err: %s", err)
	} else {
		log.Printf("CombinedOutput: %s", string(out))
	}

	return out, err
}

// Adds public Helm Agones releases to the cluster
func addAgonesRepo() error {
	installArgs := [][]string{{"repo", "add", "agones", "https://agones.dev/chart/stable"},
		{"repo", "update"}}

	for _, args := range installArgs {
		_, err := runExecCommand(HelmCmd, args...)
		if err != nil {
			return fmt.Errorf("Could not add Agones helm repo: %s", err)
		}
	}
	return nil
}

func installAgonesRelease(version, registry, featureGates, imagePullPolicy, sidecarPullPolicy,
	logLevel, chart string) error {

	// TODO: Include feature gates. (Current issue with Helm and string formatting of the feature gates.)
	// 		"--set agones.featureGates=%s "+
	// Remove this print line which is here to prevent linter complaining about featureGates not used.
	log.Printf("Agones Version %s, FeatureGates %s", version, featureGates)

	helmString := fmt.Sprintf(
		"upgrade --install --atomic --wait --timeout=10m --namespace=agones-system --create-namespace --version %s "+
			"--set agones.image.tag=%s "+
			"--set agones.image.registry=%s "+
			"--set agones.image.allocator.pullPolicy=%s "+
			"--set agones.image.controller.pullPolicy=%s "+
			"--set agones.image.extensions.pullPolicy=%s "+
			"--set agones.image.ping.pullPolicy=%s "+
			"--set agones.image.sdk.alwaysPull=%s "+
			"--set agones.controller.logLevel=%s "+
			"agones %s",
		version, version, registry, imagePullPolicy, imagePullPolicy, imagePullPolicy, imagePullPolicy,
		sidecarPullPolicy, logLevel, chart,
	)

	helmArgs := strings.Split(helmString, " ")

	_, err := runExecCommand(HelmCmd, helmArgs...)
	if err != nil {
		return err
	}

	return err
}

func runConfigWalker(ctx context.Context, g *errgroup.Group, validConfigs []*configTest, cancel context.CancelFunc) error {
	for _, config := range validConfigs {
		registry := AgonesRegistry
		chart := HelmChart
		if config.agonesVersion == DevVersion {
			registry = TestRegistry
			chart = TestChart
		}
		// Install the specified version of Agones.
		err := installAgonesRelease(config.agonesVersion, registry, config.featureGates, ImagePullPolicy,
			SidecarPullPolicy, LogLevel, chart)
		if err != nil {
			return fmt.Errorf("installAgonesRelease err: %w", err)
		}

		// Wait for the Helm release to become fully deployed.
		var helmStatus string
		err = wait.PollUntilContextTimeout(ctx, 10*time.Second, Timeout, true, func(_ context.Context) (done bool, err error) {
			helmStatus, err = checkHelmStatus(config.agonesVersion)
			if err != nil {
				return true, err
			}
			if helmStatus == "deployed" {
				return true, nil
			}
			return false, nil
		})
		if err != nil || helmStatus != "deployed" {
			return fmt.Errorf("timed out while attempting upgrade to Agones version %s. Helm Status: %s",
				config.agonesVersion, helmStatus)
		}

		// Launch a new goroutine that is managed by the errgroup to continuously create game servers
		// for this version. If this function returns an error, the errgroup will catch it.
		gsReady := make(chan bool)
		g.Go(func() error {
			return createGameServers(ctx, config.gameServerPath, gsReady)
		})

		// Wait for the first game server to be ready before proceeding to the next step.
		select {
		case <-gsReady:
			log.Printf("First game server for Agones version %s is ready.", config.agonesVersion)
		case <-ctx.Done():
			return nil
		}
		close(gsReady)

		// Allow some soak time at this version before upgrading to the next.
		log.Printf("Soaking at version %s for 1 minute.", config.agonesVersion)
		time.Sleep(1 * time.Minute)
	}

	log.Println("All upgrade steps successfully completed.")
	cancel()
	return nil
}

// checkHelmStatus returns the status of the Helm release at a specified agonesVersion if it exists.
func checkHelmStatus(agonesVersion string) (string, error) {
	helmStatus := helmStatuses{}
	checkStatus := []string{"list", "-a", "-nagones-system", "-ojson"}
	out, err := runExecCommand(HelmCmd, checkStatus...)
	if err != nil {
		return "", fmt.Errorf("Could not run command %s %s, err: %s", KubectlCmd, checkStatus, err)
	}

	err = json.Unmarshal(out, &helmStatus)
	if err != nil {
		return "", fmt.Errorf("Could not Unmarshal", err)
	}

	// Remove the commit sha from the DevVersion i.e. from 1.46.0-dev-7168dd3 to 1.46.0-dev or 1.46.0
	// for the case of this test running during a new Agones version release PR.
	agonesRelease := ""
	if agonesVersion == DevVersion {
		r := regexp.MustCompile(`1\.\d+\.\d+-dev`)
		agonesVersion = r.FindString(DevVersion)
		r = regexp.MustCompile(`1\.\d+\.\d`)
		agonesRelease = r.FindString(agonesVersion)
	}

	for _, status := range helmStatus {
		if (status.AppVersion == agonesVersion) || (status.AppVersion == agonesRelease) {
			return status.Status, nil
		}
	}
	return "", nil
}

// Creates a gameserver yaml file from the mounted gameserver.yaml template. The name of the new
// gameserver yaml is based on the Agones version, i.e. gs1440.yaml for Agones version 1.44.0
// Note: This does not validate the created file.
func createGameServerFile(agonesVersion string, countsAndLists bool) (string, error) {
	gsTmpl := gameServerTemplate{Registry: ContainerRegistry, AgonesVersion: agonesVersion, CountsAndLists: countsAndLists}

	gsTemplate, err := template.ParseFiles("gameserver.yaml")
	if err != nil {
		return "", fmt.Errorf("Could not ParseFiles template gameserver.yaml", err)
	}

	// Must use /tmp since the container is a non-root user.
	gsPath := strings.ReplaceAll("/tmp/gs"+agonesVersion, ".", "")
	gsPath += ".yaml"
	// Check if the file already exists
	if _, err := os.Stat(gsPath); err == nil {
		return gsPath, nil
	}
	gsFile, err := os.Create(gsPath)
	if err != nil {
		return "", fmt.Errorf("Could not create file ", err)
	}

	err = gsTemplate.Execute(gsFile, gsTmpl)
	if err != nil {
		if fErr := gsFile.Close(); fErr != nil {
			log.Printf("Could not close game server file %s, err: %s", gsPath, fErr)
		}
		return "", fmt.Errorf("Could not Execute template gameserver.yaml ", err)
	}
	if err = gsFile.Close(); err != nil {
		log.Printf("Could not close game server file %s, err: %s", gsPath, err)
	}

	return gsPath, nil
}

// Create a game server every five seconds until the context is cancelled. The game server container
// is the same binary version as the game server file. The SDK version is always the same as the
// version of the Agones controller that created it. The Game Server shuts itself down after the
// tests have run as part of the `sdk-client-test` logic.
func createGameServers(ctx context.Context, gsPath string, gsReady chan bool) error {
	args := []string{"create", "-f", gsPath}
	err := checkFirstGameServerReady(ctx, gsReady, args...)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(5 * time.Second)
	retry := 0

	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return nil
		case <-ticker.C:
			_, err := runExecCommand(KubectlCmd, args...)
			// Ignore failures for ~45s at at time to account for the brief (~30s) during which the
			// controller service is unavailable during upgrade.
			if err != nil {
				if retry > Retries {
					return fmt.Errorf("Could not create Gameserver %s: %s. Too many successive errors.", gsPath, err)
				}
				log.Printf("Could not create Gameserver %s: %s. Retries left: %d.", gsPath, err, Retries-retry)
				retry++
			} else {
				retry = 0
			}
		}
	}
}

// checkFirstGameServerReady waits for the Game Server Pod to be running. This may take several
// minutes in Autopilot.
func checkFirstGameServerReady(ctx context.Context, gsReady chan bool, args ...string) error {
	// Sample output: gameserver.agones.dev/sdk-client-test-5zjdn created
	output, err := runExecCommand(KubectlCmd, args...)
	if err != nil {
		return fmt.Errorf("Could not create Gameserver: %s", err)
	}
	r := regexp.MustCompile(`sdk-client-test-\S+`)
	gsName := r.FindString(string(output))
	// Game Server has too many states, so using the pod instead as there are only two healthy states.
	// Includes the gs name to make output logs easier to read.
	getPodStatus := []string{"get", "pod", gsName, "-o=custom-columns=:.status.phase,:.metadata.name", "--no-headers"}

	// Pod is created after Game Server, wait briefly before erroring out on unable to get pod.
	retry := 0
	err = wait.PollUntilContextTimeout(ctx, 2*time.Second, Timeout, true, func(_ context.Context) (done bool, err error) {
		out, err := runExecCommand(KubectlCmd, getPodStatus...)
		if err != nil && retry > Retries {
			return true, fmt.Errorf("Could not get Gameserver %s state: %s", gsName, err)
		}
		if err != nil {
			retry++
			return false, nil
		}
		// Sample output: Running   sdk-client-test-bbvx9
		podStatus := strings.Split(string(out), " ")
		if podStatus[0] == "Running" || podStatus[0] == "Succeeded" {
			gsReady <- true
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("PollUntilContextTimeout timed out while wait for first gameserver %s to be Ready", gsName)
	}
	return nil
}

// watchGameServers watches for failed GameServers and returns an error if the threshold is exceeded.
func watchGameServers(ctx context.Context, agonesClient *versioned.Clientset, acceptedFailures int) error {
	errCh := make(chan error, 1)
	failedGs := make(map[string]gsLog)

	agonesInformerFactory := externalversions.NewSharedInformerFactory(agonesClient, 5*time.Second)
	gsInformer := agonesInformerFactory.Agones().V1().GameServers().Informer()

	_, err := gsInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(_, newObj interface{}) {
			newGs := newObj.(*agonesv1.GameServer)
			if newGs.Status.State == "Error" || newGs.Status.State == "Unhealthy" {
				gsVersion := newGs.Labels["agonesVersion"]
				sdkVersion := newGs.Annotations["agones.dev/sdk-version"]
				log.Printf("Game server %s with binary version %s, and SDK version %s in %s state\n",
					newGs.Name, gsVersion, sdkVersion, newGs.Status.State)

				// Put failed game servers into the map until it reaches capacity.
				failedGs[newGs.Name] = gsLog{GameServerVersion: gsVersion, SdkVersion: sdkVersion,
					GameServerState: string(newGs.Status.State)}
				if len(failedGs) > acceptedFailures {
					select {
					case errCh <- fmt.Errorf("Too many Game Servers in Error or Unhealthy states: %v", failedGs):
					default:
					}
				}
			}
		},
	})
	if err != nil {
		return fmt.Errorf("not able to create AddEventHandler: %w", err)
	}

	go gsInformer.Run(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), gsInformer.HasSynced) {
		return fmt.Errorf("Timed out waiting for game server informer cache to sync")
	}

	select {
	case err := <-errCh:
		return err // A fatal error occurred.
	case <-ctx.Done():
		return nil // Context was canceled, normal shutdown.
	}
}

// watchGameServerEvents watches all events on `sdk-client-test` containers for BackOff errors.
// The purpose is to catch ImagePullBackOff errors. It returns an error if a "Failed" event is detected.
func watchGameServerEvents(ctx context.Context, kubeClient *kubernetes.Clientset) error {
	// Use a buffered channel to prevent the event handler from blocking.
	errCh := make(chan error, 1)

	// Filter by Game Server `sdk-client-test` containers.
	containerPath := "spec.containers{sdk-client-test}"
	fieldSelector := fields.OneTermEqualSelector("involvedObject.fieldPath", containerPath).String()

	// First delete previous `sdk-client-test` events, otherwise there will be events from previous runs.
	_, err := runExecCommand(KubectlCmd, []string{"delete", "events", "--field-selector", fieldSelector}...)
	if err != nil {
		// This is not a fatal error, as it might just mean no events existed.
		log.Println("Could not delete `sdk-client-test` events, which may be expected:", err)
	}

	eventOptions := informers.WithTweakListOptions(func(opts *metav1.ListOptions) {
		opts.FieldSelector = fieldSelector
	})
	kubeInformerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, 5*time.Second,
		informers.WithNamespace("default"), eventOptions)
	eventInformer := kubeInformerFactory.Core().V1().Events().Informer()

	_, err = eventInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			newEvent := obj.(*v1.Event)
			gsPodName := newEvent.InvolvedObject.Name
			if newEvent.Reason == "Failed" {
				err := fmt.Errorf("%s on %s %s has failed. Latest event: message %s",
					"sdk-client-test", newEvent.Kind, gsPodName, newEvent.Message)
				// Send the error to the channel without blocking.
				select {
				case errCh <- err:
				default:
				}
			}
		},
	})
	if err != nil {
		return fmt.Errorf("not able to create AddEventHandler: %w", err)
	}

	// Run the informer in a separate goroutine. It will stop when the context is canceled.
	go eventInformer.Run(ctx.Done())

	// Wait for the informer's cache to sync with the cluster.
	if !cache.WaitForCacheSync(ctx.Done(), eventInformer.HasSynced) {
		return fmt.Errorf("timed out waiting for eventInformer cache to sync")
	}

	// Block until an error is received or the context is canceled.
	select {
	case err := <-errCh:
		// A fatal error was detected in the event handler.
		return err
	case <-ctx.Done():
		// The context was canceled, indicating a normal shutdown or a failure in another goroutine.
		return nil
	}
}

// Deletes any remaining Game Servers, Uninstalls Agones, and Deletes agones-system namespace.
func cleanUpResources() {
	args := []string{"delete", "gs", "-l", "app=sdk-client-test", "--timeout", "10m"}
	_, err := runExecCommand(KubectlCmd, args...)
	if err != nil {
		log.Println("Could not delete game servers", err)
	}

	args = []string{"uninstall", "agones", "-n", "agones-system", "--wait", "--timeout", "10m"}
	_, err = runExecCommand(HelmCmd, args...)
	if err != nil {
		log.Println("Could not Helm uninstall Agones", err)
	}

	// Apiservice v1.allocation.agones.dev, which is part of Service agones-system/agones-controller-service,
	// does not always get cleaned up on Helm uninstall, and needs to be deleted (if it exists) before
	// the agones-system namespace can be removed.
	// Ignore the error, because an "error" means Helm already uninstalled the apiservice.
	args = []string{"delete", "apiservice", "v1.allocation.agones.dev", "--timeout", "1m"}
	out, err := runExecCommand(KubectlCmd, args...)
	if err == nil {
		fmt.Println(string(out))
	}

	args = []string{"delete", "ns", "agones-system", "--timeout", "10m"}
	_, err = runExecCommand(KubectlCmd, args...)
	if err != nil {
		log.Println("Could not delete agones-system namespace", err)
	}
}
