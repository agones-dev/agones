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

package testing

import (
	"context"
	"net/http"
	gotesting "testing"
	"time"

	agonesfake "agones.dev/agones/pkg/client/clientset/versioned/fake"
	"agones.dev/agones/pkg/client/informers/externalversions"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	extfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	"k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

// Handy tools for testing controllers

// Mocks is a holder for all my fakes and Mocks
type Mocks struct {
	KubeClient            *kubefake.Clientset
	KubeInformerFactory   informers.SharedInformerFactory
	ExtClient             *extfake.Clientset
	AgonesClient          *agonesfake.Clientset
	AgonesInformerFactory externalversions.SharedInformerFactory
	FakeRecorder          *record.FakeRecorder
	Mux                   *http.ServeMux
}

// NewMocks creates a new set of fakes and mocks.
func NewMocks() Mocks {
	kubeClient := &kubefake.Clientset{}
	agonesClient := &agonesfake.Clientset{}

	m := Mocks{
		KubeClient:            kubeClient,
		KubeInformerFactory:   informers.NewSharedInformerFactory(kubeClient, 30*time.Second),
		ExtClient:             &extfake.Clientset{},
		AgonesClient:          agonesClient,
		AgonesInformerFactory: externalversions.NewSharedInformerFactory(agonesClient, 30*time.Second),
		FakeRecorder:          record.NewFakeRecorder(100),
	}
	return m
}

// StartInformers starts new fake informers
func StartInformers(mocks Mocks, sync ...cache.InformerSynced) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	mocks.KubeInformerFactory.Start(ctx.Done())
	mocks.AgonesInformerFactory.Start(ctx.Done())

	logrus.Info("Wait for cache sync")
	if !cache.WaitForCacheSync(ctx.Done(), sync...) {
		panic("Cache never synced")
	}

	return ctx, cancel
}

// NewEstablishedCRD fakes CRD installation success.
func NewEstablishedCRD() *apiextv1.CustomResourceDefinition {
	return &apiextv1.CustomResourceDefinition{
		Status: apiextv1.CustomResourceDefinitionStatus{
			Conditions: []apiextv1.CustomResourceDefinitionCondition{{
				Type:   apiextv1.Established,
				Status: apiextv1.ConditionTrue,
			}},
		},
	}
}

// AssertEventContains asserts that a k8s event stream contains a
// value, and assert.FailNow() if it does not
func AssertEventContains(t *gotesting.T, events <-chan string, contains string) {
	select {
	case e := <-events:
		assert.Contains(t, e, contains)
	case <-time.After(3 * time.Second):
		assert.FailNow(t, "Did not receive "+contains+" event")
	}
}

// AssertNoEvent asserts that the event stream does not
// have a value in it (at least in the next second)
func AssertNoEvent(t *gotesting.T, events <-chan string) {
	select {
	case e := <-events:
		assert.Fail(t, "should not have an event", e)
	case <-time.After(1 * time.Second):
	}
}
