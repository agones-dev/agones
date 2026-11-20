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

package v1

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"agones.dev/agones/pkg/util/errors"
	"agones.dev/agones/pkg/util/runtime"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// URLConfiguration is a generic configuration for any integration with an external URL or
// cluster provided service. For example, it can be used to configure a webhook or Wasm module
type URLConfiguration admissionregistrationv1.WebhookClientConfig

// ValidateURLConfiguration validates the URLConfiguration settings to make sure either a URL is set or
// a Service is set, but not both. Also checks that the CABundle is valid if HTTPS is used.
func (w *URLConfiguration) ValidateURLConfiguration(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	if w == nil {
		return append(allErrs, field.Required(fldPath, "url configuration is missing"))
	}
	if w.Service == nil && w.URL == nil {
		allErrs = append(allErrs, field.Required(fldPath, "url should be provided"))
	}
	if w.Service != nil && w.URL != nil {
		allErrs = append(allErrs, field.Duplicate(fldPath.Child("url"), "service and url cannot be used simultaneously"))
	}
	if w.CABundle != nil {
		rootCAs := x509.NewCertPool()
		// Check that CABundle provided is correctly encoded certificate
		if ok := rootCAs.AppendCertsFromPEM(w.CABundle); !ok {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("caBundle"), w.CABundle, "CA Bundle is not valid"))
		}
	}
	if w.URL != nil {
		_, err := url.Parse(*w.URL)
		if err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("url"), *w.URL, "url is not valid"))
		}
	}
	return allErrs
}

// BuildURL returns the URL that this URLConfiguration points to, whether that is a direct URL
// or a Service in the cluster.
func (w *URLConfiguration) BuildURL() (*url.URL, error) {
	errs := errors.FromStruct(w)
	if w.URL != nil && w.Service != nil {
		return nil, errs.New("service and URL cannot be used simultaneously")
	}

	if w.URL != nil {
		if *w.URL == "" {
			return nil, errs.New("URL was not provided")
		}

		return url.ParseRequestURI(*w.URL)
	}

	if w.Service == nil {
		return nil, errs.New("service was not provided, either URL or Service must be provided")
	}

	if w.Service.Name == "" {
		return nil, errs.New("service name was not provided")
	}

	scheme := "http"
	if w.CABundle != nil {
		scheme = "https"
	}

	path := ""
	if w.Service.Path != nil {
		path = *w.Service.Path
	}

	namespace := w.Service.Namespace
	if namespace == "" {
		namespace = "default"
	}

	return createURL(scheme, w.Service.Name, namespace, path, w.Service.Port), nil
}

// HTTPClient returns a http.Client to connect to this URLConfiguration, that trusts the
// CABundle, if one is provided.
func (w *URLConfiguration) HTTPClient(timeout time.Duration) (*http.Client, error) {
	config := &tls.Config{}
	if w.CABundle != nil {
		rootCAs := x509.NewCertPool()
		if ok := rootCAs.AppendCertsFromPEM(w.CABundle); !ok {
			return nil, errors.FromStruct(w).New("no certs were appended from caBundle")
		}
		config.RootCAs = rootCAs
	}

	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: config,
		},
	}, nil
}

// moved to a separate method to cover it with unit tests and check that URL corresponds to a proper pattern
func createURL(scheme, name, namespace, path string, port *int32) *url.URL {
	var hostPort int32 = 8000
	if port != nil {
		hostPort = *port
	}

	return &url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s.%s.svc:%d", name, namespace, hostPort),
		Path:   path,
	}
}

// WasmFrom defines the source of the Wasm module
type WasmFrom struct {
	// URL is the URL of the Wasm module.
	URL *URLConfiguration `json:"url"`
}

// WasmPolicy contains the configuration for a WebAssembly module that runs as a plugin to Agones.
type WasmPolicy struct {
	// Function is the exported function to call in the wasm module. The default depends on where the plugin is used.
	// +optional
	Function string `json:"function,omitempty"`
	// Config values to pass to the wasm program on startup
	// +optional
	Config map[string]string `json:"config,omitempty"`
	// WasmFrom defines the source of the Wasm module
	From WasmFrom `json:"from"`
	// Hash of the Wasm module, used to verify the integrity of the module
	// +optional
	Hash string `json:"hash,omitempty"`
}

// ValidateWasmPolicy validates the Wasm policy settings. The gate is the feature gate that must
// be enabled for the Wasm policy to be used.
func (w *WasmPolicy) ValidateWasmPolicy(fldPath *field.Path, gate runtime.Feature) field.ErrorList {
	var allErrs field.ErrorList
	if w == nil {
		return append(allErrs, field.Required(fldPath, "wasm policy config params are missing"))
	}

	if !runtime.FeatureEnabled(gate) {
		return append(allErrs, field.Forbidden(fldPath, fmt.Sprintf("feature %s must be enabled", gate)))
	}

	fldPath = fldPath.Child("from")
	if w.From.URL == nil {
		return append(allErrs, field.Required(fldPath, "wasm from configuration is missing"))
	}
	allErrs = w.From.URL.ValidateURLConfiguration(fldPath.Child("url"))

	return allErrs
}
