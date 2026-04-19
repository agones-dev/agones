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

// Package webhooks manages and receives Kubernetes Webhooks
package webhooks

import (
	"encoding/json"
	"net/http"

	"agones.dev/agones/pkg/util/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// WebHook manage Kubernetes webhooks
type WebHook struct {
	logger   *logrus.Entry
	mux      *http.ServeMux
	handlers map[string][]operationHandler
}

// operationHandler stores the data for a handler to match against
type operationHandler struct {
	handler   Handler
	groupKind schema.GroupKind
	operation admissionv1.Operation
}

// Handler handles a webhook's AdmissionReview coming in, and will return the
// AdmissionReview that will be the return value of the webhook
type Handler func(review admissionv1.AdmissionReview) (admissionv1.AdmissionReview, error)

// NewWebHook returns a Kubernetes webhook manager
func NewWebHook(mux *http.ServeMux) *WebHook {
	wh := &WebHook{
		mux:      mux,
		handlers: map[string][]operationHandler{},
	}

	wh.logger = runtime.NewLoggerWithType(wh)
	return wh
}

// AddHandler adds a handler for a given path, group and kind, and operation
func (wh *WebHook) AddHandler(path string, gk schema.GroupKind, op admissionv1.Operation, h Handler) {
	if len(wh.handlers[path]) == 0 {
		wh.mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			err := wh.handle(path, w, r)
			if err != nil {
				runtime.HandleError(wh.logger.WithField("url", r.URL), err)
				w.WriteHeader(http.StatusInternalServerError)
			}
		})
	}
	wh.logger.WithField("path", path).WithField("groupKind", gk).WithField("op", op).Debug("Added webhook handler")
	wh.handlers[path] = append(wh.handlers[path], operationHandler{groupKind: gk, operation: op, handler: h})
}

// handle Handles http requests for webhooks
func (wh *WebHook) handle(path string, w http.ResponseWriter, r *http.Request) error { // nolint: interfacer
	wh.logger.WithField("path", path).Debug("running webhook")

	var review admissionv1.AdmissionReview
	err := json.NewDecoder(r.Body).Decode(&review)
	if err != nil {
		return errors.Wrapf(err, "error decoding decoding json for path %v", path)
	}

	// set it to true, in case there are no handlers
	if review.Response == nil {
		review.Response = &admissionv1.AdmissionResponse{Allowed: true}
	}
	review.Response.UID = review.Request.UID
	wh.logger.WithField("name", review.Request.Name).WithField("path", path).WithField("kind", review.Request.Kind.Kind).WithField("group", review.Request.Kind.Group).Debug("handling webhook request")

	for _, oh := range wh.handlers[path] {
		if oh.operation == review.Request.Operation &&
			oh.groupKind.Kind == review.Request.Kind.Kind &&
			review.Request.Kind.Group == oh.groupKind.Group {

			review, err = oh.handler(review)
			if err != nil {
				review.Response.Allowed = false
				details := metav1.StatusDetails{
					Name:  review.Request.Name,
					Group: review.Request.Kind.Group,
					Kind:  review.Request.Kind.Kind,
					Causes: []metav1.StatusCause{{
						Type:    metav1.CauseType("InternalError"),
						Message: err.Error(),
					}},
				}
				review.Response.Result = &metav1.Status{
					Status:  metav1.StatusFailure,
					Message: err.Error(),
					Reason:  metav1.StatusReasonInternalError,
					Details: &details,
				}
			}
		}
	}
	err = json.NewEncoder(w).Encode(review)
	if err != nil {
		return errors.Wrapf(err, "error decoding encoding json for path %v", path)
	}

	return nil
}
