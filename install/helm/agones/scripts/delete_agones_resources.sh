#!/bin/sh

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

namespaces=$@

for ns in $namespaces; do
  # Building the list of pods we need to ensure are deleted.
  gs=$(kubectl -n $ns get gs -o jsonpath='{.items[*].metadata.name}')

  for g in $gs; do
    pod=$(kubectl -n $ns get po -l agones.dev/gameserver=$g -o jsonpath='{.items[*].metadata.name}')
    pods="$pods $pod"
  done

  # Delete Agones resources to kickstart pod deletion.
  kubectl -n $ns delete fleetautoscalers --all
  kubectl -n $ns delete fleets --all
  kubectl -n $ns delete gameserversets --all
  kubectl -n $ns delete gameservers --all
  kubectl -n $ns delete gameserverallocationpolicies --all

  # Since we don't have the nifty kubectl wait yet, hack one in the meantime
  # Wait for GS underlying pods to be deleted
  for p in $pods; do
    get_po=$(kubectl -n $ns get po $p -o jsonpath='{.metadata.name}')
    while [ "$get_po" = "$p" ]; do
      sleep 0.1
      get_po=$(kubectl -n $ns get po $p -o jsonpath='{.metadata.name}')
    done
  done
done
