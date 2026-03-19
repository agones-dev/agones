// Copyright 2019 Google LLC All Rights Reserved.
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

// Package apiserver manages kubernetes api extension apis
package apiserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"agones.dev/agones/pkg/util/https"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/go-openapi/spec"
	"github.com/munnerz/goautoneg"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/kube-openapi/pkg/handler3"
)

var (
	// Reference:
	// https://github.com/agones-dev/agones/blob/main/vendor/k8s.io/apiextensions-apiserver/pkg/apiserver/apiserver.go
	// These are public as they may be needed by CRDHandler implementations (usually for returning Status values)

	// Scheme scheme for unversioned types - such as APIResourceList, and Status
	Scheme = k8sruntime.NewScheme()
	// Codecs for unversioned types - such as APIResourceList, and Status
	Codecs = serializer.NewCodecFactory(Scheme)

	unversionedVersion = schema.GroupVersion{Version: "v1"}
	unversionedTypes   = []k8sruntime.Object{
		&metav1.Status{},
		&metav1.APIResourceList{},
	}
)

const (
	// ContentTypeHeader = "Content-Type"
	ContentTypeHeader = "Content-Type"
	// AcceptHeader = "Accept"
	AcceptHeader = "Accept"
)

func init() {
	Scheme.AddUnversionedTypes(unversionedVersion, unversionedTypes...)
}

// CRDHandler is a http handler, that gets passed the Namespace it's working
// on, and returns an error if a server error occurs
type CRDHandler func(http.ResponseWriter, *http.Request, string) error

// APIServer is a lightweight library for registering, and providing handlers
// for Kubernetes APIServer extensions.
type APIServer struct {
	logger             *logrus.Entry
	mux                *http.ServeMux
	resourceList       map[string]*metav1.APIResourceList
	openapiv2          *spec.Swagger
	openapiv3Discovery *handler3.OpenAPIV3Discovery
	delegates          map[string]CRDHandler
}

// NewAPIServer returns a new API Server from the given Mux.
// creates a empty Swagger definition and sets up the endpoint.
func NewAPIServer(mux *http.ServeMux) *APIServer {
	s := &APIServer{
		mux:                mux,
		resourceList:       map[string]*metav1.APIResourceList{},
		openapiv2:          &spec.Swagger{SwaggerProps: spec.SwaggerProps{}},
		openapiv3Discovery: &handler3.OpenAPIV3Discovery{Paths: map[string]handler3.OpenAPIV3DiscoveryGroupVersion{}},
		delegates:          map[string]CRDHandler{},
	}
	s.logger = runtime.NewLoggerWithType(s)
	s.logger.Debug("API Server Started")

	// We don't *have* to have a v3 openapi api, so just do an empty one for now, and we can expand as needed.
	// If we implement /v3/ we can likely omit the /v2/ handler since only one is needed.
	// This at least stops the K8s api pinging us for the spec all the time.
	mux.HandleFunc("/openapi/v3", https.ErrorHTTPHandler(s.logger, func(w http.ResponseWriter, _ *http.Request) error {
		w.Header().Set(ContentTypeHeader, k8sruntime.ContentTypeJSON)
		err := json.NewEncoder(w).Encode(s.openapiv3Discovery)
		if err != nil {
			return errors.Wrap(err, "error encoding openapi/v3")
		}
		return nil
	}))

	// We don't *have* to have a v2 openapi api, so just do an empty one for now, and we can expand as needed.
	// kube-openapi could be a potential library to look at for future if we want to be more specific.
	// This at least stops the K8s api pinging us for every iteration of a api descriptor that may exist
	s.openapiv2.SwaggerProps.Info = &spec.Info{InfoProps: spec.InfoProps{Title: "allocation.agones.dev"}}
	mux.HandleFunc("/openapi/v2", https.ErrorHTTPHandler(s.logger, func(w http.ResponseWriter, _ *http.Request) error {
		w.Header().Set(ContentTypeHeader, k8sruntime.ContentTypeJSON)
		err := json.NewEncoder(w).Encode(s.openapiv2)
		if err != nil {
			return errors.Wrap(err, "error encoding openapi/v2")
		}
		return nil
	}))

	// We don't currently support a root /apis, but since Aggregate Discovery expects
	// a 406, let's give it what it wants, otherwise namespaces don't successfully terminate on <= 1.27.2
	mux.HandleFunc("/apis", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotAcceptable)
		w.Header().Set(ContentTypeHeader, k8sruntime.ContentTypeJSON)
	})

	return s
}

// AddAPIResource stores the APIResource under the given groupVersion string, and returns it
// in the appropriate place for the K8s discovery service
// e.g. http://localhost:8001/apis/scheduling.k8s.io/v1
// as well as registering a CRDHandler that all http requests for the given APIResource are routed to
func (as *APIServer) AddAPIResource(groupVersion string, resource metav1.APIResource, handler CRDHandler) {
	_, ok := as.resourceList[groupVersion]
	if !ok {
		// discovery handler
		list := &metav1.APIResourceList{GroupVersion: groupVersion, APIResources: []metav1.APIResource{}}
		as.resourceList[groupVersion] = list
		pattern := fmt.Sprintf("/apis/%s", groupVersion)
		as.addSerializedHandler(pattern, list)
		as.logger.WithField("groupversion", groupVersion).WithField("pattern", pattern).Debug("Adding Discovery Handler")

		// e.g.  /apis/agones.dev/v1/namespaces/default/gameservers
		// CRD handler
		pattern = fmt.Sprintf("/apis/%s/namespaces/", groupVersion)
		as.mux.HandleFunc(pattern, https.ErrorHTTPHandler(as.logger, as.resourceHandler(groupVersion)))
		as.logger.WithField("groupversion", groupVersion).WithField("pattern", pattern).Debug("Adding Resource Handler")
	}

	// discovery resource
	as.resourceList[groupVersion].APIResources = append(as.resourceList[groupVersion].APIResources, resource)

	// add specific crd resource handler
	key := fmt.Sprintf("%s/%s", groupVersion, resource.Name)
	as.delegates[key] = handler

	as.logger.WithField("groupversion", groupVersion).WithField("apiresource", resource).Debug("Adding APIResource")
}

// resourceHandler handles namespaced resource calls, and sends them to the appropriate CRDHandler delegate
func (as *APIServer) resourceHandler(gv string) https.ErrorHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		namespace, resource, err := splitNameSpaceResource(r.URL.Path)
		if err != nil {
			https.FourZeroFour(as.logger.WithError(err), w, r)
			return nil
		}

		delegate, ok := as.delegates[fmt.Sprintf("%s/%s", gv, resource)]
		if !ok {
			https.FourZeroFour(as.logger, w, r)
			return nil
		}

		if err := delegate(w, r, namespace); err != nil {
			return err
		}

		return nil
	}
}

// addSerializedHandler sets up a handler than will send the serialised content
// to the specified path.
func (as *APIServer) addSerializedHandler(pattern string, m k8sruntime.Object) {
	as.mux.HandleFunc(pattern, https.ErrorHTTPHandler(as.logger, func(w http.ResponseWriter, r *http.Request) error {
		if r.Method == http.MethodGet {
			info, err := AcceptedSerializer(r, Codecs)
			if err != nil {
				return err
			}

			shallowCopy := shallowCopyObjectForTargetKind(m)
			w.Header().Set(ContentTypeHeader, info.MediaType)
			err = Codecs.EncoderForVersion(info.Serializer, unversionedVersion).Encode(shallowCopy, w)
			if err != nil {
				return errors.New("error marshalling")
			}
		} else {
			https.FourZeroFour(as.logger, w, r)
		}

		return nil
	}))
}

// shallowCopyObjectForTargetKind ensures obj is unique by performing a shallow copy
// of the struct Object points to (all Object must be a pointer to a struct in a scheme).
// Copied from https://github.com/kubernetes/kubernetes/pull/101123 until the referenced PR is merged
func shallowCopyObjectForTargetKind(obj k8sruntime.Object) k8sruntime.Object {
	v := reflect.ValueOf(obj).Elem()
	copied := reflect.New(v.Type())
	copied.Elem().Set(v)
	return copied.Interface().(k8sruntime.Object)
}

// AcceptedSerializer takes the request, and returns a serialiser (if it exists)
// for the given codec factory and
// for the Accepted media types.  If not found, returns error
func AcceptedSerializer(r *http.Request, codecs serializer.CodecFactory) (k8sruntime.SerializerInfo, error) {
	// this is so we know what we can accept
	mediaTypes := codecs.SupportedMediaTypes()
	alternatives := make([]string, len(mediaTypes))
	for i, media := range mediaTypes {
		alternatives[i] = media.MediaType
	}
	header := r.Header.Get(AcceptHeader)
	accept := goautoneg.Negotiate(header, alternatives)
	if accept == "" {
		accept = k8sruntime.ContentTypeJSON
	}
	info, ok := k8sruntime.SerializerInfoForMediaType(mediaTypes, accept)
	if !ok {
		return info, errors.Errorf("Could not find serializer for Accept: %s", header)
	}

	return info, nil
}

// splitNameSpaceResource returns the namespace and the type of resource
func splitNameSpaceResource(path string) (namespace, resource string, err error) {
	list := strings.Split(strings.Trim(path, "/"), "/")
	if len(list) < 3 {
		return namespace, resource, errors.Errorf("could not find namespace and resource in path: %s", path)
	}
	last := list[len(list)-3:]

	if last[0] != "namespaces" {
		return namespace, resource, errors.Errorf("wrong format in path: %s", path)
	}

	return last[1], last[2], err
}
