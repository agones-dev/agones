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

package crd

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	extfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
)

func TestWaitForEstablishedCRD(t *testing.T) {
	t.Parallel()
	crd := &apiextv1.CustomResourceDefinition{
		Status: apiextv1.CustomResourceDefinitionStatus{
			Conditions: []apiextv1.CustomResourceDefinitionCondition{{
				Type:   apiextv1.Established,
				Status: apiextv1.ConditionTrue,
			}},
		},
	}

	t.Run("CRD already established", func(t *testing.T) {
		extClient := &extfake.Clientset{}
		extClient.AddReactor("get", "customresourcedefinitions", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			return true, crd, nil
		})

		err := WaitForEstablishedCRD(context.Background(), extClient.ApiextensionsV1().CustomResourceDefinitions(), "test", logrus.WithField("test", "already-established"))
		assert.Nil(t, err)
	})

	t.Run("CRD takes a second to become established", func(t *testing.T) {
		extClient := &extfake.Clientset{}
		m := sync.RWMutex{}
		established := false

		extClient.AddReactor("get", "customresourcedefinitions", func(_ k8stesting.Action) (bool, runtime.Object, error) {
			m.RLock()
			defer m.RUnlock()
			if established {
				return true, crd, nil
			}
			return false, nil, nil
		})

		go func() {
			time.Sleep(3 * time.Second)
			m.Lock()
			defer m.Unlock()
			established = true
		}()

		err := WaitForEstablishedCRD(context.Background(), extClient.ApiextensionsV1().CustomResourceDefinitions(), "test", logrus.WithField("test", "already-established"))
		assert.Nil(t, err)
	})
}
