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
	"context"
	"strings"
	"time"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"agones.dev/agones/pkg/client/clientset/versioned"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	gameServerImage      = "GAMESERVER_IMAGE"
	isHelmTest           = "IS_HELM_TEST"
	gameserversNamespace = "GAMESERVERS_NAMESPACE"

	defaultNs = "default"
)

func main() {
	viper.AllowEmptyEnv(true)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	pflag.String(gameServerImage, "", "The Address to bind the server grpcPort to. Defaults to 'localhost'")
	viper.SetDefault(gameServerImage, "")
	runtime.Must(viper.BindEnv(gameServerImage))

	pflag.Bool(isHelmTest, false,
		"Is Helm test - defines whether GameServer should be shut down at the end of the test or not. Defaults to false")
	viper.SetDefault(isHelmTest, false)
	runtime.Must(viper.BindEnv(isHelmTest))

	pflag.String(gameserversNamespace, defaultNs, "Namespace where GameServers are created. Defaults to default")
	viper.SetDefault(gameserversNamespace, defaultNs)
	runtime.Must(viper.BindEnv(gameserversNamespace))

	pflag.Parse()
	runtime.Must(viper.BindPFlags(pflag.CommandLine))

	config, err := rest.InClusterConfig()
	logger := runtime.NewLoggerWithSource("main")
	if err != nil {
		logger.WithError(err).Fatal("Could not create in cluster config")
	}

	// Access to standard Kubernetes resources through the Kubernetes Clientset
	// We don't actually need this for this example, but it's just here for
	// illustrative purposes
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.WithError(err).Fatal("Could not create the kubernetes clientset")
	}
	p := kubeClient.CoreV1().Pods("default")
	logger.Info(p)

	// Access to the Agones resources through the Agones Clientset
	// Note that we reuse the same config as we used for the Kubernetes Clientset
	agonesClient, err := versioned.NewForConfig(config)
	if err != nil {
		logger.WithError(err).Fatal("Could not create the agones api clientset")
	}

	// Create a GameServer
	gs := &agonesv1.GameServer{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "helm-test-server-",
			Namespace:    viper.GetString(gameserversNamespace),
		},
		Spec: agonesv1.GameServerSpec{
			Container: "simple-game-server",
			Ports: []agonesv1.GameServerPort{{
				ContainerPort: 7654,
				Name:          "gameport",
				PortPolicy:    agonesv1.Dynamic,
				Protocol:      corev1.ProtocolUDP,
			}},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "simple-game-server",
							Image: viper.GetString(gameServerImage),
						},
					},
				},
			},
		},
	}
	ctx := context.Background()
	newGS, err := agonesClient.AgonesV1().GameServers(gs.Namespace).Create(ctx, gs, metav1.CreateOptions{})
	if err != nil {
		logrus.Fatalf("Unable to create GameServer: %v", err)
	}
	logrus.Infof("New GameServer name is: %s", newGS.ObjectMeta.Name)

	if viper.GetBool(isHelmTest) {
		err = wait.PollImmediate(1*time.Second, 60*time.Second, func() (bool, error) {
			checkGs, err := agonesClient.AgonesV1().GameServers(gs.Namespace).Get(ctx, newGS.Name, metav1.GetOptions{})

			if err != nil {
				logrus.WithError(err).Warn("error retrieving gameserver")
				return false, nil
			}

			state := agonesv1.GameServerStateReady
			logger.WithField("gs", checkGs.ObjectMeta.Name).
				WithField("currentState", checkGs.Status.State).
				WithField("awaitingState", state).Info("Waiting for states to match")

			if checkGs.Status.State == state {
				return true, nil
			}

			return false, nil
		})
		if err != nil {
			logrus.Fatalf("Wait GameServer to become Ready failed: %v", err)
		}

		err = agonesClient.AgonesV1().GameServers(gs.Namespace).Delete(ctx, newGS.ObjectMeta.Name, metav1.DeleteOptions{})
		if err != nil {
			logrus.Fatalf("Unable to delete GameServer: %v", err)
		}
	}
}
