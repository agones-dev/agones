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

// Package main runs allocation load-test scenarios against the Agones allocator
// service, as described by a scenarios file.
package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	pb "agones.dev/agones/pkg/allocation/go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type allocErrorCode string

const (
	noerror                   allocErrorCode = "Noerror"
	unknown                   allocErrorCode = "Unknown"
	objectHasBeenModified     allocErrorCode = "ObjectHasBeenModified"
	tooManyConcurrentRequests allocErrorCode = "TooManyConcurrentRequests"
	noAvailableGameServer     allocErrorCode = "NoAvailableGameServer"
	storageError              allocErrorCode = "StorageError"
	deadLineExceeded          allocErrorCode = "DeadLineExceeded"
	connectionTimedOut        allocErrorCode = "ConnectionTimedOut"
	connectionRefused         allocErrorCode = "ConnectionRefused"
	errReadingFromServer      allocErrorCode = "ErrReadingFromServer"
)

var (
	logger    = log.New(os.Stdout, "", 0)
	errorMsgs = map[allocErrorCode]string{
		objectHasBeenModified:     "the object has been modified",
		tooManyConcurrentRequests: "too many concurrent requests",
		noAvailableGameServer:     "no available GameServer to allocate",
		storageError:              "storage error",
		deadLineExceeded:          "context deadline exceeded",
		connectionTimedOut:        "connection timed out",
		connectionRefused:         "connection refused",
		errReadingFromServer:      "allocator seems crashed and restarted",
	}
)

var (
	externalIP                 = flag.String("ip", "missing external IP", "the external IP for allocator server")
	port                       = flag.String("port", "443", "the port for allocator server")
	certFile                   = flag.String("cert", "missing cert", "the public key file for the client certificate in PEM format")
	keyFile                    = flag.String("key", "missing key", "the private key file for the client certificate in PEM format")
	cacertFile                 = flag.String("cacert", "missing cacert", "the CA cert file for server signing certificate in PEM format")
	scenariosFile              = flag.String("scenariosFile", "", "Scenario File that have duration of each allocations with different number of clients and allocations")
	namespace                  = flag.String("namespace", "default", "the game server kubernetes namespace")
	multicluster               = flag.Bool("multicluster", false, "set to true to enable the multi-cluster allocation")
	displaySuccessfulResponses = flag.Bool("displaySuccessfulResponses", false, "Display successful responses")
	displayFailedResponses     = flag.Bool("displayFailedResponses", false, "Display failed responses")
)

type scenario struct {
	duration            time.Duration
	numOfClients        int
	intervalMillisecond int
}

type result struct {
	end     time.Time
	latency time.Duration
	code    allocErrorCode
}

// summary aggregates the results of one or more scenarios.
type summary struct {
	sent              uint64
	succeeded         uint64
	failed            uint64
	succeededInWindow uint64
	window            time.Duration
	elapsed           time.Duration
	latencies         []time.Duration
	errors            map[allocErrorCode]uint64
}

func main() {
	flag.Parse()

	endpoint := *externalIP + ":" + *port
	dialOpts, err := dialOptions(*certFile, *keyFile, *cacertFile)
	if err != nil {
		logger.Fatalf("%v", err)
	}

	scenarios := readScenarios(*scenariosFile)
	total := summary{errors: allocErrorCodeCntMap()}
	for i, sc := range *scenarios {
		logger.Printf("\n\n%v :Running Scenario %v with %v clients submitting requests every %vms for %v\n===================\n", time.Now(), i+1, sc.numOfClients, sc.intervalMillisecond, sc.duration)

		s := runScenario(endpoint, dialOpts, sc)
		logger.Printf("\nFinished Scenario %v\n", i+1)
		s.print(fmt.Sprintf("Scenario %v", i+1))
		total.add(s)
	}

	logger.Print("\nFinal Totals\n")
	total.print("Final Total")
}

func runScenario(endpoint string, dialOpts grpc.DialOption, sc scenario) summary {
	var (
		mu      sync.Mutex
		results []result
		sent    atomic.Uint64
		wg      sync.WaitGroup
	)
	start := time.Now()
	windowEnd := start.Add(sc.duration)
	durCtx, cancel := context.WithDeadline(context.Background(), windowEnd)
	defer cancel()

	for k := 0; k < sc.numOfClients; k++ {
		wg.Go(func() {
			conn, err := grpc.NewClient(endpoint, dialOpts)
			if err != nil {
				logger.Printf("Failed to dial for client %v: %v", k, err)
				return
			}
			defer func() { _ = conn.Close() }() // Ignore error handling because the connection will be closed when the main func exits anyway.
			client := pb.NewAllocationServiceClient(conn)
			ticker := time.NewTicker(time.Duration(sc.intervalMillisecond) * time.Millisecond)
			defer ticker.Stop()

			var wgc sync.WaitGroup
			defer wgc.Wait()
			for durCtx.Err() == nil {
				sent.Add(1)
				wgc.Go(func() {
					reqStart := time.Now()
					code := allocate(client)
					end := time.Now()
					mu.Lock()
					results = append(results, result{end: end, latency: end.Sub(reqStart), code: code})
					mu.Unlock()
				})
				select {
				case <-durCtx.Done():
				case <-ticker.C:
				}
			}
		})
	}
	wg.Wait()

	s := summary{sent: sent.Load(), window: sc.duration, errors: allocErrorCodeCntMap()}
	lastEnd := windowEnd
	for _, r := range results {
		if r.end.After(lastEnd) {
			lastEnd = r.end
		}
		if r.code != noerror {
			s.failed++
			s.errors[r.code]++
			continue
		}
		s.succeeded++
		s.latencies = append(s.latencies, r.latency)
		if !r.end.After(windowEnd) {
			s.succeededInWindow++
		}
	}
	s.elapsed = lastEnd.Sub(start)
	return s
}

func (s *summary) add(o summary) {
	s.sent += o.sent
	s.succeeded += o.succeeded
	s.failed += o.failed
	s.succeededInWindow += o.succeededInWindow
	s.window += o.window
	s.elapsed += o.elapsed
	s.latencies = append(s.latencies, o.latencies...)
	for k, v := range o.errors {
		s.errors[k] += v
	}
}

func (s *summary) print(label string) {
	for k, v := range s.errors {
		if k != noerror {
			logger.Printf("Count: %v\t\tError: %v", v, k)
		}
	}
	completed := s.succeeded + s.failed
	logger.Printf("\n\n%v\n%v Requests Sent: %v, Offered rate: %.1f/s", time.Now(), label, s.sent, rate(s.sent, s.window))
	logger.Printf("%v Allocation Count: %v, Failure Count: %v, Failure rate: %.4f", label, s.succeeded, s.failed, float64(s.failed)/float64(completed))
	logger.Printf("%v Throughput: %.1f allocations/s (%v allocations in %v, from first request to last response)", label, rate(s.succeeded, s.elapsed), s.succeeded, s.elapsed.Round(time.Millisecond))
	logger.Printf("%v Throughput within window: %.1f allocations/s (%v allocations completed within %v)", label, rate(s.succeededInWindow, s.window), s.succeededInWindow, s.window)
	if len(s.latencies) == 0 {
		return
	}
	slices.Sort(s.latencies)
	logger.Printf("%v Allocation latency: min %v, p50 %v, p90 %v, p95 %v, p99 %v, max %v", label,
		s.latencies[0].Round(time.Millisecond), percentile(s.latencies, 50), percentile(s.latencies, 90),
		percentile(s.latencies, 95), percentile(s.latencies, 99), s.latencies[len(s.latencies)-1].Round(time.Millisecond))
}

func rate(count uint64, d time.Duration) float64 {
	return float64(count) / d.Seconds()
}

// percentile returns the nearest-rank percentile p of the sorted durations.
func percentile(sorted []time.Duration, p float64) time.Duration {
	idx := int(math.Ceil(p/100*float64(len(sorted)))) - 1
	return sorted[max(idx, 0)].Round(time.Millisecond)
}

func dialOptions(certFile, keyFile, cacertFile string) (grpc.DialOption, error) {
	clientCert, err := os.ReadFile(certFile)
	if err != nil {
		return nil, err
	}
	clientKey, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}
	cacert, err := os.ReadFile(cacertFile)
	if err != nil {
		return nil, err
	}

	// Load client cert
	cert, err := tls.X509KeyPair(clientCert, clientKey)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
	if len(cacert) != 0 {
		// Load CA cert, if provided and trust the server certificate.
		// This is required for self-signed certs.
		tlsConfig.RootCAs = x509.NewCertPool()
		if !tlsConfig.RootCAs.AppendCertsFromPEM(cacert) {
			return nil, errors.New("only PEM format is accepted for server CA")
		}
	}

	return grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)), nil
}

func allocate(client pb.AllocationServiceClient) allocErrorCode {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	resp, err := client.Allocate(ctx, &pb.AllocationRequest{
		Namespace:           *namespace,
		MultiClusterSetting: &pb.MultiClusterSetting{Enabled: *multicluster},
	})
	if err != nil {
		alErrType := errorType(err)
		if *displayFailedResponses || alErrType == unknown {
			logger.Printf("Error received: %v", err)
		}
		return alErrType
	}
	if *displaySuccessfulResponses {
		gsAddress := resp.GetAddress()
		gsName := resp.GetGameServerName()
		portName := resp.GetPorts()[0].GetName()
		port := resp.GetPorts()[0].GetPort()
		logger.Println("Allocate Response")
		logger.Printf("  Received GS:\n    Addreess: %s\n    Name: %s\n    Port Name: %s\n    Port: %v", gsAddress, gsName, portName, port)
	}
	return noerror
}

func readScenarios(file string) *[]scenario {
	fp, err := os.Open(file)
	if err != nil {
		logger.Fatalf("Failed opening the scenario  file %s: %v", file, err)
	}
	defer func() { _ = fp.Close() }()

	var scenarios []scenario

	scanner := bufio.NewScanner(fp)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		lineParts := strings.Split(line, ",")
		if len(lineParts) != 3 {
			logger.Fatalf("There should be 3 parts for each scenario but there is %d for %q", len(lineParts), line)
		}
		duration, err := time.ParseDuration(lineParts[0])
		if err != nil {
			logger.Fatalf("Failed parsing duration %s: %v", lineParts[0], err)
		}
		numClients, err := strconv.Atoi(lineParts[1])
		if err != nil {
			logger.Fatalf("Failed parsing number of clients %s: %v", lineParts[1], err)
		}
		intervalMillisecond, err := strconv.Atoi(lineParts[2])
		if err != nil {
			logger.Fatalf("Failed parsing intervalMillisecond %s: %v", lineParts[2], err)
		}
		scenarios = append(scenarios, scenario{duration, numClients, intervalMillisecond})
	}

	return &scenarios
}

func errorType(err error) allocErrorCode {
	for ec, em := range errorMsgs {
		if strings.Contains(err.Error(), em) {
			return ec
		}
	}
	return unknown
}

func allocErrorCodeCntMap() map[allocErrorCode]uint64 {
	return map[allocErrorCode]uint64{
		noerror:                   0,
		unknown:                   0,
		objectHasBeenModified:     0,
		tooManyConcurrentRequests: 0,
		noAvailableGameServer:     0,
		storageError:              0,
		deadLineExceeded:          0,
		connectionTimedOut:        0,
		connectionRefused:         0,
		errReadingFromServer:      0,
	}
}
