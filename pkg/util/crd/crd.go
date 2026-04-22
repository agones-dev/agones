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

// Package crd contains utilities for working with
// CustomResourceDefinitions.
package crd

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextclientv1 "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// WaitForEstablishedCRD blocks until CRD comes to an Established state.
// Has a deadline of 60 seconds for this to occur.
func WaitForEstablishedCRD(_ context.Context, crdGetter apiextclientv1.CustomResourceDefinitionInterface, name string, logger *logrus.Entry) error {
	return wait.PollUntilContextTimeout(context.Background(), time.Second, 60*time.Second, true, func(ctx context.Context) (done bool, err error) {
		crd, err := crdGetter.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		for _, cond := range crd.Status.Conditions {
			if cond.Type == apiextv1.Established {
				if cond.Status == apiextv1.ConditionTrue {
					logger.WithField("crd", crd.ObjectMeta.Name).Debug("custom resource definition established")
					return true, err
				}
			}
		}

		return false, nil
	})
}
