#!/usr/bin/env bash

# Copyright Contributors to Agones a Series of LF Projects, LLC.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

echo "Listing Helm releases in agones-system namespace..."
helm ls -n agones-system

echo "Removing Agones deployment from agones-system namespace..."
helm uninstall -n agones-system agones || echo "Failed to uninstall. Consider deleting the current cluster and setting up a new one. Refer to https://agones.dev/site/docs/installation/creating-cluster/gke/#create-a-standard-mode-cluster-for-agones"

echo "Listing pods in the agones-system namespace..."
kubectl get pods -n agones-system

echo "Deleting agones-system namespace..."
kubectl delete ns agones-system
echo "Agones system namespace deleted."

echo "Helm repo update to fetch the latest version of Agones..."
helm repo update

echo "Verifying the new version..."
helm search repo agones --versions --devel

echo "Installing Agones in agones-system namespace..."
helm install --create-namespace --namespace=agones-system agones agones/agones

echo "Listing all pods in agones-system namespace..."
kubectl get pods --namespace agones-system

echo "Execute any of the test functions from test/e2e directory to complete the smoke test."
