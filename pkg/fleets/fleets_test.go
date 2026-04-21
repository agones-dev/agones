/*
 * Copyright Contributors to Agones a Series of LF Projects, LLC.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package fleets

import (
	"sort"
	"strconv"
	"testing"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	agtesting "agones.dev/agones/pkg/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
)

func TestListGameServerSetsByFleetOwner(t *testing.T) {
	t.Parallel()

	f := &agonesv1.Fleet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fleet-1",
			Namespace: "default",
			UID:       "1234",
		},
		Spec: agonesv1.FleetSpec{
			Replicas: 5,
			Template: agonesv1.GameServerTemplateSpec{},
		},
	}

	gsSet1 := f.GameServerSet()
	gsSet1.ObjectMeta.Name = "gsSet1"
	gsSet2 := f.GameServerSet()
	gsSet2.ObjectMeta.Name = "gsSet2"
	gsSet3 := f.GameServerSet()
	gsSet3.ObjectMeta.Name = "gsSet3"
	gsSet3.ObjectMeta.Labels = nil
	gsSet4 := f.GameServerSet()
	gsSet4.ObjectMeta.Name = "gsSet4"
	gsSet4.ObjectMeta.OwnerReferences = nil

	m := agtesting.NewMocks()
	m.AgonesClient.AddReactor("list", "gameserversets", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, &agonesv1.GameServerSetList{Items: []agonesv1.GameServerSet{*gsSet1, *gsSet2, *gsSet3, *gsSet4}}, nil
	})

	gameServerSets := m.AgonesInformerFactory.Agones().V1().GameServerSets()
	_, cancel := agtesting.StartInformers(m, gameServerSets.Informer().HasSynced)
	defer cancel()

	list, err := ListGameServerSetsByFleetOwner(gameServerSets.Lister().GameServerSets(f.ObjectMeta.Namespace), f)
	require.NoError(t, err)

	// sort of stable ordering
	sort.SliceStable(list, func(i, j int) bool {
		return list[i].ObjectMeta.Name < list[j].ObjectMeta.Name
	})
	assert.Equal(t, []*agonesv1.GameServerSet{gsSet1, gsSet2}, list)
}

func TestListGameServersByFleetOwner(t *testing.T) {
	t.Parallel()
	f := defaultFixture()

	gsSet := f.GameServerSet()
	gsSet.ObjectMeta.Name = "gsSet1"

	var gsList []agonesv1.GameServer
	for i := 1; i <= 3; i++ {
		gs := gsSet.GameServer()
		gs.ObjectMeta.Name = "gs" + strconv.Itoa(i)
		gsList = append(gsList, *gs)
	}

	m := agtesting.NewMocks()

	m.AgonesClient.AddReactor("list", "gameservers", func(_ k8stesting.Action) (bool, runtime.Object, error) {
		return true, &agonesv1.GameServerList{Items: gsList}, nil
	})

	informer := m.AgonesInformerFactory.Agones().V1()
	_, cancel := agtesting.StartInformers(m,
		informer.GameServers().Informer().HasSynced)
	defer cancel()

	list, err := ListGameServersByFleetOwner(informer.GameServers().Lister().GameServers(f.ObjectMeta.Namespace), f)
	require.NoError(t, err)
	assert.Len(t, list, len(gsList), "Retrieved list should be same size as original")

	sort.SliceStable(list, func(i, j int) bool {
		return list[i].ObjectMeta.Name < list[j].ObjectMeta.Name
	})

	// need to loop, because need to dereference
	for i, gs := range gsList {
		assert.Equal(t, gs, *list[i])
	}
}
