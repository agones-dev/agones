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
// limitations under the License

// evictpod.go --pod <pod> --namespace <namespace> initiates a pod eviction
// using the k8s eviction API.
package main

import (
	"context"
	"flag"
	"path/filepath"

	policy "k8s.io/api/policy/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/homedir"

	"agones.dev/agones/pkg/util/runtime"
)

// Borrowed from https://stackoverflow.com/questions/62803041/how-to-evict-or-delete-pods-from-kubernetes-using-golang-client
func evictPod(ctx context.Context, client *kubernetes.Clientset, name, namespace string) error {
	return client.PolicyV1().Evictions(namespace).Evict(ctx, &policy.Eviction{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		}})
}

func main() {
	ctx := context.Background()

	kubeconfig := flag.String("kubeconfig", filepath.Join(homedir.HomeDir(), ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	namespace := flag.String("namespace", "default", "Namespace (defaults to `default`)")
	pod := flag.String("pod", "", "Pod name (required)")
	flag.Parse()
	logger := runtime.NewLoggerWithSource("evictpod")

	if *pod == "" {
		logger.Fatal("--pod must be non-empty")
	}

	config, err := runtime.InClusterBuildConfig(logger, *kubeconfig)
	if err != nil {
		logger.WithError(err).Fatalf("Could not build config: %v", err)
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.WithError(err).Fatalf("Could not create the kubernetes clientset: %v", err)
	}

	if err := evictPod(ctx, kubeClient, *pod, *namespace); err != nil {
		logger.WithError(err).Fatalf("Pod eviction failed: %v", err)
	}
}
