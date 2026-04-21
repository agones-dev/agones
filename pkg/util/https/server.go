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

package https

import (
	"context"
	cryptotls "crypto/tls"
	"net/http"
	"sync"
	"time"

	"agones.dev/agones/pkg/util/fswatch"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	tlsDir = "/certs/"
)

// tls is a http server interface to enable easier testing
type tls interface {
	Shutdown(context.Context) error
	ListenAndServeTLS(certFile, keyFile string) error
}

// certServer holds the Server certificate
type certServer struct {
	certs  *cryptotls.Certificate
	certMu sync.Mutex
}

// Server is a HTTPs server that conforms to the runner interface
// we use in /cmd/controller, and has a public Mux that can be updated
// has a default 404 handler, to make discovery of k8s services a bit easier.
type Server struct {
	certServer certServer
	logger     *logrus.Entry
	Mux        *http.ServeMux
	tls        tls
	certFile   string
	keyFile    string
	port       string
}

// NewServer returns a Server instance.
func NewServer(certFile, keyFile string, port string) *Server {
	mux := http.NewServeMux()

	wh := &Server{
		Mux:      mux,
		certFile: certFile,
		keyFile:  keyFile,
		port:     port,
	}
	wh.logger = runtime.NewLoggerWithType(wh)
	wh.setupServer()
	wh.Mux.HandleFunc("/", wh.defaultHandler)

	return wh
}

func (s *Server) setupServer() {
	s.tls = &http.Server{
		Addr:    ":" + s.port,
		Handler: s.Mux,
		TLSConfig: &cryptotls.Config{
			GetCertificate: s.getCertificate,
		},
	}

	tlsCert, err := cryptotls.LoadX509KeyPair(tlsDir+"server.crt", tlsDir+"server.key")
	if err != nil {
		s.logger.WithError(err).Error("could not load Initial TLS certs; keeping old one")
		return
	}

	s.certServer.certMu.Lock()
	defer s.certServer.certMu.Unlock()
	s.certServer.certs = &tlsCert
}

// getCertificate returns the current TLS certificate
func (s *Server) getCertificate(_ *cryptotls.ClientHelloInfo) (*cryptotls.Certificate, error) {
	s.certServer.certMu.Lock()
	defer s.certServer.certMu.Unlock()
	return s.certServer.certs, nil
}

// WatchForCertificateChanges watches for changes in the certificate files
func (s *Server) WatchForCertificateChanges() (func(), error) {

	cancelTLS, err := fswatch.Watch(s.logger, tlsDir, time.Second, func() {
		// Load the new TLS certificate
		s.logger.Info("TLS certs changed, reloading")
		tlsCert, err := cryptotls.LoadX509KeyPair(tlsDir+"server.crt", tlsDir+"server.key")
		if err != nil {
			s.logger.WithError(err).Error("could not load TLS certs; keeping old one")
			return
		}
		s.certServer.certMu.Lock()
		defer s.certServer.certMu.Unlock()
		s.certServer.certs = &tlsCert
		s.logger.Info("TLS certs updated")
	})
	if err != nil {
		s.logger.WithError(err).Fatal("could not create watcher for TLS certs")
		return nil, err
	}
	return cancelTLS, nil
}

// Run runs the webhook server, starting a https listener.
// Will close the http server on stop channel close.
func (s *Server) Run(ctx context.Context, _ int) error {
	go func() {
		<-ctx.Done()
		_ = s.tls.Shutdown(context.Background())
	}()

	s.logger.WithField("server", s).Infof("https server started on port :%s", s.port)

	err := s.tls.ListenAndServeTLS(s.certFile, s.keyFile)
	if err == http.ErrServerClosed {
		s.logger.WithError(err).Info("https server closed")
		return nil
	}

	return errors.Wrap(err, "Could not listen on :"+s.port)
}

// defaultHandler Handles all the HTTP requests
// useful for debugging requests
func (s *Server) defaultHandler(w http.ResponseWriter, r *http.Request) {
	// "/" is the default health check used by APIServers
	if r.URL.Path == "/" {
		w.WriteHeader(http.StatusOK)
		return
	}

	FourZeroFour(s.logger, w, r)
}
