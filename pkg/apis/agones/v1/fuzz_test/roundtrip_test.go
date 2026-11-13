package install

import (
	"math/rand"
	"testing"

	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	autoscalingv1 "agones.dev/agones/pkg/apis/autoscaling/v1"
	multiclusterv1 "agones.dev/agones/pkg/apis/multicluster/v1"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	"k8s.io/apimachinery/pkg/api/apitesting/roundtrip"
	genericfuzzer "k8s.io/apimachinery/pkg/apis/meta/fuzzer"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestRoundTripTypes(t *testing.T) {
	scheme := runtime.NewScheme()
	codecs := serializer.NewCodecFactory(scheme)

	localSchemeBuilder := runtime.SchemeBuilder{
		agonesv1.AddToScheme,
		allocationv1.AddToScheme,
		autoscalingv1.AddToScheme,
		multiclusterv1.AddToScheme,
	}
	seed := rand.Int63()
	localFuzzer := fuzzer.FuzzerFor(genericfuzzer.Funcs, rand.NewSource(seed), codecs)
	require.NoError(t, localSchemeBuilder.AddToScheme(scheme))

	var globalNonRoundTrippableTypes = sets.NewString(
		"ExportOptions",
		"GetOptions",
		"WatchEvent",
		"ListOptions",
		"DeleteOptions",
	)
	kinds := scheme.AllKnownTypes()
	for gvk := range kinds {
		if gvk.Version == runtime.APIVersionInternal || globalNonRoundTrippableTypes.Has(gvk.Kind) {
			continue
		}
		t.Run(gvk.Group+"."+gvk.Version+"."+gvk.Kind, func(t *testing.T) {
			roundtrip.RoundTripSpecificKindWithoutProtobuf(t, gvk, scheme, codecs, localFuzzer, nil)
		})
	}
}
