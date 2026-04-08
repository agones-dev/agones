# Copyright 2019 Google LLC All Rights Reserved.
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

MINIKUBE_DRIVER ?= docker
MINIKUBE_NODES ?= 1

MINIKUBE_DEBUG_SDK_NAMESPACE ?= default
MINIKUBE_DEBUG_CONTROLLER_PORT ?= 2346
MINIKUBE_DEBUG_EXTENSIONS_PORT ?= 2347
MINIKUBE_DEBUG_PING_PORT ?= 2348
MINIKUBE_DEBUG_ALLOCATOR_PORT ?= 2349
MINIKUBE_DEBUG_PROCESSOR_PORT ?= 2350
MINIKUBE_DEBUG_SDK_PORT ?= 2351

# minikube shell mount for certificates
minikube_cert_mount := ~/.minikube:$(HOME)/.minikube

#   __  __ _       _ _          _
#  |  \/  (_)_ __ (_) | ___   _| |__   ___
#  | |\/| | | '_ \| | |/ / | | | '_ \ / _ \
#  | |  | | | | | | |   <| |_| | |_) |  __/
#  |_|  |_|_|_| |_|_|_|\_\\__,_|_.__/ \___|
#

# Switches to an "agones" profile, and starts a kubernetes cluster
# of the right version.
minikube-test-cluster: DOCKER_RUN_ARGS+=--network=host -v $(minikube_cert_mount)
minikube-test-cluster: $(ensure-build-image)
	$(MINIKUBE) start --kubernetes-version v1.34.6 -p $(MINIKUBE_PROFILE) --driver $(MINIKUBE_DRIVER) --nodes $(MINIKUBE_NODES)
	$(MAKE) minikube-metallb-helm-install
	$(MAKE) minikube-metallb-configure

# minikube-metallb-helm-install installs metallb via helm
minikube-metallb-helm-install:
	helm repo add metallb https://metallb.github.io/metallb
	helm repo update
	helm upgrade metallb metallb/metallb --install --namespace metallb-system --create-namespace --version 0.15.2 --wait --timeout 5m

# minikube-metallb-configure configures metallb with an ip address range based on the minikube ip
minikube-metallb-configure:
	MINIKUBE_IP=$$($(MINIKUBE) ip -p $(MINIKUBE_PROFILE)); \
	NETWORK_PREFIX=$$(echo "$$MINIKUBE_IP" | cut -d '.' -f 1-3); \
	METALLB_IP_RANGE="$$NETWORK_PREFIX.50-$$NETWORK_PREFIX.250"; \
	sed "s|__RANGE__|$${METALLB_IP_RANGE}|g" $(build_path)/metallb-config.yaml.tpl | kubectl apply -f -

# Connecting to minikube requires so enhanced permissions, so use this target
# instead of `make shell` to start an interactive shell for development on minikube.
minikube-shell: $(ensure-build-image)
	$(MAKE) shell DOCKER_RUN_ARGS="--network=host -v $(minikube_cert_mount) $(DOCKER_RUN_ARGS)"

# Push the local Agones Docker images that have already been built
# via `make build` or `make build-images` into the "agones" minikube instance.
minikube-push:
	$(MINIKUBE) image load $(sidecar_linux_amd64_tag) -p $(MINIKUBE_PROFILE)
	$(MINIKUBE) image load $(controller_amd64_tag) -p $(MINIKUBE_PROFILE)
	$(MINIKUBE) image load $(ping_amd64_tag) -p $(MINIKUBE_PROFILE)
	$(MINIKUBE) image load $(allocator_amd64_tag) -p $(MINIKUBE_PROFILE)
	$(MINIKUBE) image load $(extensions_amd64_tag) -p $(MINIKUBE_PROFILE)
	$(MINIKUBE) image load $(processor_amd64_tag) -p $(MINIKUBE_PROFILE)

	$(MINIKUBE) image tag $(sidecar_linux_amd64_tag) $(sidecar_tag) -p $(MINIKUBE_PROFILE)
	$(MINIKUBE) image tag $(controller_amd64_tag) $(controller_tag) -p $(MINIKUBE_PROFILE)
	$(MINIKUBE) image tag $(ping_amd64_tag) $(ping_tag) -p $(MINIKUBE_PROFILE)
	$(MINIKUBE) image tag $(allocator_amd64_tag) $(allocator_tag) -p $(MINIKUBE_PROFILE)
	$(MINIKUBE) image tag $(extensions_amd64_tag) $(extensions_tag) -p $(MINIKUBE_PROFILE)
	$(MINIKUBE) image tag $(processor_amd64_tag) $(processor_tag) -p $(MINIKUBE_PROFILE)

# Installs the current development version of Agones into the Kubernetes cluster.
# Use this instead of `make install`, as it disables PullAlways on the install.yaml
minikube-install: PING_SERVICE_TYPE := LoadBalancer
minikube-install: ALLOCATOR_SERVICE_TYPE := LoadBalancer
minikube-install: 
	$(MAKE) install DOCKER_RUN_ARGS="--network=host -v $(minikube_cert_mount)" ALWAYS_PULL_SIDECAR=false \
		IMAGE_PULL_POLICY=IfNotPresent PING_SERVICE_TYPE=$(PING_SERVICE_TYPE) \
		ALLOCATOR_SERVICE_TYPE=$(ALLOCATOR_SERVICE_TYPE)

minikube-uninstall: $(ensure-build-image)
	$(MAKE) uninstall DOCKER_RUN_ARGS="--network=host -v $(minikube_cert_mount)"

# Runs e2e tests against our minikube
minikube-test-e2e: DOCKER_RUN_ARGS=--network=host -v $(minikube_cert_mount)
minikube-test-e2e: test-e2e

# Runs stress tests against our minikube
minikube-stress-test-e2e: DOCKER_RUN_ARGS=--network=host -v $(minikube_cert_mount)
minikube-stress-test-e2e: stress-test-e2e

# prometheus on minkube
# we have to disable PVC as it's not supported on minkube.
minikube-setup-prometheus:
	$(MAKE) setup-prometheus \
		DOCKER_RUN_ARGS="--network=host -v $(minikube_cert_mount)" \
		PVC=false HELM_ARGS="--set server.resources.requests.cpu=0,server.resources.requests.memory=0"

# grafana on minkube with dashboards and prometheus datasource installed.
# we have to disable PVC as it's not supported on minkube.
minikube-setup-grafana:
	$(MAKE) setup-grafana \
		DOCKER_RUN_ARGS="--network=host -v $(minikube_cert_mount)"

# prometheus-stack on minkube
# we have to disable PVC as it's not supported on minkube.
minikube-setup-prometheus-stack:
	$(MAKE) setup-prometheus-stack \
		DOCKER_RUN_ARGS="--network=host -v $(minikube_cert_mount)" \
		PVC=false HELM_ARGS="--set prometheus.server.resources.requests.cpu=0,prometheus.server.resources.requests.memory=0"

# minikube port forwarding
minikube-controller-portforward:
	$(MAKE) controller-portforward DOCKER_RUN_ARGS="--network=host -v $(minikube_cert_mount)"

# minikube port forwarding grafana
minikube-grafana-portforward:
	$(MAKE) grafana-portforward \
		DOCKER_RUN_ARGS="--network=host -v $(minikube_cert_mount)"

# minikube port forwarding for prometheus web ui
minikube-prometheus-portforward:
	$(MAKE) prometheus-portforward \
		DOCKER_RUN_ARGS="--network=host -v $(minikube_cert_mount)"

# Install Agones with debug images / single replicas
minikube-install-debug:
	$(MAKE) minikube-install HELM_ARGS="\
		--set agones.controller.replicas=1 \
		--set agones.extensions.replicas=1 \
		--set agones.allocator.replicas=1 \
		--set agones.allocator.processor.replicas=1 \
		--set agones.ping.replicas=1"

# Port forward debug ports for all Agones services to localhost
# Each service gets its own port with configurable defaults:
# - Controller: 2346 (MINIKUBE_DEBUG_CONTROLLER_PORT)
# - Extensions: 2347 (MINIKUBE_DEBUG_EXTENSIONS_PORT)  
# - Ping: 2348 (MINIKUBE_DEBUG_PING_PORT)
# - Allocator: 2349 (MINIKUBE_DEBUG_ALLOCATOR_PORT)
# - Processor: 2350 (MINIKUBE_DEBUG_PROCESSOR_PORT)
# E.g. make minikube-debug-portforward MINIKUBE_DEBUG_CONTROLLER_PORT=2345
minikube-debug-portforward:
	@echo "Starting port forwarding for all Agones services..."
	@echo "Controller: localhost:$(MINIKUBE_DEBUG_CONTROLLER_PORT) -> agones-controller:2346"
	@echo "Extensions: localhost:$(MINIKUBE_DEBUG_EXTENSIONS_PORT) -> agones-extensions:2346" 
	@echo "Ping: localhost:$(MINIKUBE_DEBUG_PING_PORT) -> agones-ping:2346"
	@echo "Allocator: localhost:$(MINIKUBE_DEBUG_ALLOCATOR_PORT) -> agones-allocator:2346"
	@echo "Processor: localhost:$(MINIKUBE_DEBUG_PROCESSOR_PORT) -> agones-processor:2346"
	@echo "Use Ctrl+C to stop all port forwards"
	@trap 'echo "Stopping all port forwards..."; kill $$(jobs -p) 2>/dev/null || true; exit 0' EXIT SIGINT SIGTERM; \
	(kubectl port-forward deployment/agones-controller $(MINIKUBE_DEBUG_CONTROLLER_PORT):2346 -n agones-system 2>/dev/null || echo "Warning: Controller port-forward failed") & \
	(kubectl port-forward deployment/agones-extensions $(MINIKUBE_DEBUG_EXTENSIONS_PORT):2346 -n agones-system 2>/dev/null || echo "Warning: Extensions port-forward failed") & \
	(kubectl port-forward deployment/agones-ping $(MINIKUBE_DEBUG_PING_PORT):2346 -n agones-system 2>/dev/null || echo "Warning: Ping port-forward failed") & \
	(kubectl port-forward deployment/agones-allocator $(MINIKUBE_DEBUG_ALLOCATOR_PORT):2346 -n agones-system 2>/dev/null || echo "Warning: Allocator port-forward failed") & \
	(kubectl port-forward deployment/agones-processor $(MINIKUBE_DEBUG_PROCESSOR_PORT):2346 -n agones-system 2>/dev/null || echo "Warning: Processor port-forward failed") & \
	wait

# Port forward debug port for Agones SDK to localhost
# By default, it looks for pods in the "default" namespace with the
# "agones-gameserver-sidecar" container. You can specify a pod name directly
# using the MINIKUBE_DEBUG_POD_NAME variable, or change the target namespace
# using MINIKUBE_DEBUG_SDK_NAMESPACE (defaults to "default").
# The local port can be set with MINIKUBE_DEBUG_SDK_PORT (default 2351)
minikube-debug-sdk-portforward:
ifdef MINIKUBE_DEBUG_POD_NAME
	kubectl port-forward pod/$(MINIKUBE_DEBUG_POD_NAME) $(MINIKUBE_DEBUG_SDK_PORT):2346 -n $(MINIKUBE_DEBUG_SDK_NAMESPACE)
else
	@echo "Searching for pods with agones-gameserver-sidecar container..."
	@PODS=$$(kubectl get pods -n $(MINIKUBE_DEBUG_SDK_NAMESPACE) -o json | jq -r '.items[] | select(.spec.containers[]?.name == "agones-gameserver-sidecar" or .spec.initContainers[]?.name == "agones-gameserver-sidecar") | .metadata.name' 2>/dev/null || true); \
	if [ -z "$$PODS" ]; then \
		echo "No pods with agones-gameserver-sidecar container found in namespace $(MINIKUBE_DEBUG_SDK_NAMESPACE)"; \
		exit 1; \
	fi; \
	echo "Found pods with agones-gameserver-sidecar container:"; \
	echo "$$PODS" | nl -v 1; \
	echo -n "Select pod number (1-$$(echo "$$PODS" | wc -l)): "; \
	read choice; \
	SELECTED_POD=$$(echo "$$PODS" | sed -n "$${choice}p"); \
	if [ -z "$$SELECTED_POD" ]; then \
		echo "Invalid selection"; \
		exit 1; \
	fi; \
	echo "Port forwarding to pod: $$SELECTED_POD"; \
	kubectl port-forward pod/$$SELECTED_POD $(MINIKUBE_DEBUG_SDK_PORT):2346 -n $(MINIKUBE_DEBUG_SDK_NAMESPACE)
endif
