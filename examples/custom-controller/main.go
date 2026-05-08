// Copyright Contributors to Agones a Series of LF Projects, LLC.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// This custom controller is deployed in the Kubernetes cluster, specifically in the agones-system namespace,
// and it's designed to observe and react to changes in GameServer instances.

package main

import (
	"context"
	"fmt"
	"os"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// GameServerReconciler reconciles a GameServer object
type GameServerReconciler struct {
	client.Client
	Log logr.Logger
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *GameServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log.Info("Entering Reconcile method.")
	gameServer := &agonesv1.GameServer{}
	err := r.Get(ctx, req.NamespacedName, gameServer)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("GameServer resource not found and it must be deleted", "name", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		r.Log.Error(err, "unable to fetch GameServer", "name", req.NamespacedName)
		return ctrl.Result{}, err
	}

	// Log current state and labels
	r.Log.Info("Fetched GameServer", "gameServerName", gameServer.Name, "gameServerNamespace", gameServer.Namespace, "state", gameServer.Status.State, "currentLabels", gameServer.Labels)

	if gameServer.Labels == nil {
		gameServer.Labels = map[string]string{}
	}

	originalState := gameServer.Labels["state"]
	newState := fmt.Sprintf("%v", gameServer.Status.State)
	if originalState == newState {
		return ctrl.Result{}, nil
	}
	gameServer.Labels["state"] = newState

	r.Log.Info("Updating GameServer labels", "originalState", originalState, "newState", newState)

	if err := r.Update(ctx, gameServer); err != nil {
		r.Log.Error(err, "Failed to update GameServer", "name", gameServer.Name)
		return ctrl.Result{}, err
	}
	r.Log.Info("GameServer label updated successfully", "name", gameServer.Name, "updatedLabels", gameServer.Labels)
	return ctrl.Result{}, nil
}

func main() {
	var err error
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	ctrl.Log.Info("Starting controller manager.")
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		LeaderElection:             true,
		LeaderElectionResourceLock: "leases",
		LeaderElectionNamespace:    "agones-system",
		LeaderElectionID:           "custom-controller-leader",
	})
	if err != nil {
		ctrl.Log.Error(err, "Unable to start manager")
		os.Exit(1)
	}

	ctrl.Log.Info("Creating Controller.")
	err = ctrl.NewControllerManagedBy(mgr).
		For(&agonesv1.GameServer{}).
		Complete(&GameServerReconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("GameServer"),
		})
	if err != nil {
		ctrl.Log.Error(err, "Unable to create controller")
		os.Exit(1)
	}

	// Start the controller manager
	ctrl.Log.Info("Starting the Cmd.")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		ctrl.Log.Error(err, "Problem running manager")
		os.Exit(1)
	}
}
