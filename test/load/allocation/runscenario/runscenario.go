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
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"log"
	"os"
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

func main() {
	flag.Parse()

	endpoint := *externalIP + ":" + *port
	dialOpts, err := dialOptions(*certFile, *keyFile, *cacertFile)
	if err != nil {
		logger.Fatalf("%v", err)
	}

	scenarios := readScenarios(*scenariosFile)
	var totalAllocCnt uint64
	var totalFailureCnt uint64
	var totalDuration float64

	totalFailureDtls := allocErrorCodeCntMap()
	for i, sc := range *scenarios {
		logger.Printf("\n\n%v :Running Scenario %v with %v clients submitting requests every %vms for %v\n===================\n", time.Now(), i+1, sc.numOfClients, sc.intervalMillisecond, sc.duration)

		var wg sync.WaitGroup
		failureCnts := make([]uint64, sc.numOfClients)
		allocCnts := make([]uint64, sc.numOfClients)
		failureDtls := make([]map[allocErrorCode]uint64, sc.numOfClients)
		for k := 0; k < sc.numOfClients; k++ {
			wg.Add(1)
			go func(clientID int) {
				defer wg.Done()
				failureDtls[clientID] = allocErrorCodeCntMap()
				durCtx, cancel := context.WithTimeout(context.Background(), sc.duration)
				defer cancel()
				conn, err := grpc.NewClient(endpoint, dialOpts)
				if err != nil {
					logger.Printf("Failed to dial for client %v: %v", clientID, err)
					return
				}
				client := pb.NewAllocationServiceClient(conn)
				var wgc sync.WaitGroup
				for durCtx.Err() == nil {
					wgc.Add(1)
					go func() {
						defer wgc.Done()
						if err := allocate(client); err != noerror {
							tmpVar := failureDtls[clientID][err]
							atomic.AddUint64(&tmpVar, 1)
							atomic.AddUint64(&failureCnts[clientID], 1)
							failureDtls[clientID][err] = tmpVar
						}
					}()
					atomic.AddUint64(&allocCnts[clientID], 1)
					time.Sleep(time.Duration(sc.intervalMillisecond) * time.Millisecond)
				}
				wgc.Wait()
				_ = conn.Close() // Ignore error handling because the connection will be closed when the main func exits anyway.
			}(k)
		}
		wg.Wait()
		logger.Printf("\nFinished Scenario %v\n", i+1)

		var scnFailureCnt uint64
		var scnAllocCnt uint64
		scnErrDtls := allocErrorCodeCntMap()
		for j := 0; j < sc.numOfClients; j++ {
			scnAllocCnt += atomic.LoadUint64(&allocCnts[j])
			scnFailureCnt += atomic.LoadUint64(&failureCnts[j])
			for k, v := range failureDtls[j] {
				totalFailureDtls[k] += v
				scnErrDtls[k] += v
			}
		}
		totalAllocCnt += scnAllocCnt
		totalFailureCnt += scnFailureCnt
		totalDuration += sc.duration.Seconds()
		for k, v := range scnErrDtls {
			if k != noerror {
				logger.Printf("Count: %v\t\tError: %v", v, k)
			}
		}
		logger.Printf("\n\n%v\nScenario Failure Count: %v, Allocation Count: %v, Failure rate: %v, allocation rate: %v", time.Now(), scnFailureCnt, scnAllocCnt, float64(scnFailureCnt)/float64(scnAllocCnt), float64(scnAllocCnt-scnFailureCnt)/sc.duration.Seconds())
	}

	logger.Print("\nFinal Error Totals\n")
	for k, v := range totalFailureDtls {
		if k != noerror {
			logger.Printf("Count: %v\t\tError: %v", v, k)
		}
	}
	logger.Printf("\n\n%v\nFinal Total Failure Count: %v, Total Allocation Count: %v, Failure rate: %v, allocation rate: %v", time.Now(), totalFailureCnt, totalAllocCnt, float64(totalFailureCnt)/float64(totalAllocCnt), float64(totalAllocCnt-totalFailureCnt)/totalDuration)
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
