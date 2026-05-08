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
// limitations under the License

//nolint:typecheck
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	pb "agones.dev/agones/pkg/allocation/go"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
	keyFile := flag.String("key", "missing key", "the private key file for the client certificate in PEM format")
	certFile := flag.String("cert", "missing cert", "the public key file for the client certificate in PEM format")
	cacertFile := flag.String("cacert", "missing cacert", "the CA cert file for server signing certificate in PEM format")
	externalIP := flag.String("ip", "missing external IP", "the external IP for allocator server")
	port := flag.String("port", "443", "the port for allocator server")
	namespace := flag.String("namespace", "default", "the game server kubernetes namespace")
	multicluster := flag.Bool("multicluster", false, "set to true to enable the multi-cluster allocation")
	numOfClients := flag.Int("numberofclients", 1, "number of clients to do allocations in parallel")
	perClientAllocs := flag.Int("perclientallocations", 1, "number of allocations to be done per client")

	flag.Parse()

	endpoint := *externalIP + ":" + *port
	cert, err := os.ReadFile(*certFile)
	if err != nil {
		panic(err)
	}
	key, err := os.ReadFile(*keyFile)
	if err != nil {
		panic(err)
	}
	cacert, err := os.ReadFile(*cacertFile)
	if err != nil {
		panic(err)
	}

	request := &pb.AllocationRequest{
		Namespace: *namespace,
		MultiClusterSetting: &pb.MultiClusterSetting{
			Enabled: *multicluster,
		},
	}

	dialOpts, err := createRemoteClusterDialOption(cert, key, cacert)
	if err != nil {
		panic(err)
	}

	fmt.Printf("started: %v\n", time.Now())

	var wg sync.WaitGroup

	// Allocate GS by numOfClients in parallel
	for k := 0; k < *numOfClients; k++ {
		wg.Add(1)

		go func(clientID int) {
			defer wg.Done()
			conn, err := grpc.NewClient(endpoint, dialOpts)
			if err != nil {
				fmt.Printf("(failed(client=%v) to get connection: %v\n", clientID, err)
				return
			}
			grpcClient := pb.NewAllocationServiceClient(conn)

			// Create a WaitGroup for the allocations within this client
			var allocWg sync.WaitGroup

			for i := 0; i < *perClientAllocs; i++ {
				allocWg.Add(1)

				// Launch each allocation in its own goroutine
				go func(allocID int) {
					defer allocWg.Done()
					_, err := grpcClient.Allocate(context.Background(), request)
					if err != nil {
						fmt.Printf("(failed(client=%v,allocation=%v): %v\n", clientID, allocID+1, err)
					}
				}(i)
			}

			// Wait for all allocations from this client to complete
			allocWg.Wait()
			_ = conn.Close()
		}(k)
	}
	wg.Wait()
	fmt.Printf("finished: %v\n", time.Now())
}

// createRemoteClusterDialOption creates a grpc client dial option with TLS configuration.
func createRemoteClusterDialOption(clientCert, clientKey, caCert []byte) (grpc.DialOption, error) {
	// Load client cert
	cert, err := tls.X509KeyPair(clientCert, clientKey)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
	if len(caCert) != 0 {
		// Load CA cert, if provided and trust the server certificate.
		// This is required for self-signed certs.
		tlsConfig.RootCAs = x509.NewCertPool()
		if !tlsConfig.RootCAs.AppendCertsFromPEM(caCert) {
			return nil, errors.New("only PEM format is accepted for server CA")
		}
	}

	return grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)), nil
}
