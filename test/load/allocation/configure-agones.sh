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

#!/bin/bash

set -e


function main() {
  echo "Make sure you have kubectl pointed at the right cluster"
  tmp_dir=$(mktemp -d)

  patch_controller="${tmp_dir}/patch-controller.yaml"
  cat << EOF > "${patch_controller}"
spec:
  template:
    spec:
      affinity:
        nodeAffinity:
          \$retainKeys:
            - requiredDuringSchedulingIgnoredDuringExecution
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: agones.dev/agones-system
                operator: Exists
      containers:
      - name: agones-controller
        resources:
          limits:
            cpu: "4"
            memory: 4000Mi
          requests:
            cpu: "4"
            memory: 4000Mi
EOF

  kubectl patch deployment -n agones-system agones-controller --patch "$(cat "${patch_controller}")"

  echo "Restarting controller pods"
  kubectl get pods -n agones-system -o=name | grep "agones-controller" | xargs kubectl delete -n agones-system

  patch_allocator="${tmp_dir}/patch-allocator.yaml"
  cat << EOF > "${patch_allocator}"
spec:
  template:
    spec:
      affinity:
        nodeAffinity:
          \$retainKeys:
            - requiredDuringSchedulingIgnoredDuringExecution
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: agones.dev/agones-system
                operator: Exists
      containers:
      - name: agones-allocator
        resources:
          limits:
            cpu: "4"
            memory: 4000Mi
          requests:
            cpu: "4"
            memory: 4000Mi
EOF

  kubectl patch deployment -n agones-system agones-allocator --patch "$(cat "${patch_allocator}")"
  echo "Restarting allocator pods"
  kubectl get pods -n agones-system -o=name | grep "agones-allocator" | xargs kubectl delete -n agones-system

  patch_ping="${tmp_dir}/patch-ping.yaml"
  cat << EOF > "${patch_ping}"
spec:
  template:
    spec:
      affinity:
        nodeAffinity:
          \$retainKeys:
            - requiredDuringSchedulingIgnoredDuringExecution
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: agones.dev/agones-system
                operator: Exists
EOF

  kubectl patch deployment -n agones-system agones-ping --patch "$(cat "${patch_ping}")"
  echo "Restarting ping pods"
  kubectl get pods -n agones-system -o=name | grep "agones-ping" | xargs kubectl delete -n agones-system
}

main "$@"
