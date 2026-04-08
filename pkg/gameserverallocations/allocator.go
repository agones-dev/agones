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

package gameserverallocations

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	goErrors "errors"
	"fmt"
	"strings"
	"time"

	"agones.dev/agones/pkg/allocation/converters"
	pb "agones.dev/agones/pkg/allocation/go"
	"agones.dev/agones/pkg/apis"
	agonesv1 "agones.dev/agones/pkg/apis/agones/v1"
	allocationv1 "agones.dev/agones/pkg/apis/allocation/v1"
	multiclusterv1 "agones.dev/agones/pkg/apis/multicluster/v1"
	getterv1 "agones.dev/agones/pkg/client/clientset/versioned/typed/agones/v1"
	multiclusterinformerv1 "agones.dev/agones/pkg/client/informers/externalversions/multicluster/v1"
	multiclusterlisterv1 "agones.dev/agones/pkg/client/listers/multicluster/v1"
	"agones.dev/agones/pkg/util/apiserver"
	"agones.dev/agones/pkg/util/logfields"
	"agones.dev/agones/pkg/util/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/tag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	runtimeschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	informercorev1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corev1lister "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
)

var (
	// ErrNoGameServer is returned when there are no Allocatable GameServers
	// available
	ErrNoGameServer = errors.New("Could not find an Allocatable GameServer")
	// ErrConflictInGameServerSelection is returned when the candidate gameserver already allocated
	ErrConflictInGameServerSelection = errors.New("The Gameserver was already allocated")
	// ErrTotalTimeoutExceeded is used to signal that total retry timeout has been exceeded and no additional retries should be made
	ErrTotalTimeoutExceeded = status.Errorf(codes.DeadlineExceeded, "remote allocation total timeout exceeded")
	// ErrGameServerUpdateConflict is returned when the game server selected for applying the allocation cannot be updated
	ErrGameServerUpdateConflict = errors.New("could not update the selected GameServer")
)

const (
	// LastAllocatedAnnotationKey is a GameServer annotation containing an RFC 3339 formatted
	// timestamp of the most recent allocation.
	LastAllocatedAnnotationKey = "agones.dev/last-allocated"

	secretClientCertName  = "tls.crt"
	secretClientKeyName   = "tls.key"
	secretCACertName      = "ca.crt"
	allocatorPort         = "443"
	maxBatchQueue         = 100
	maxBatchBeforeRefresh = 100
	localAllocationSource = "local"
)

var allocationRetry = wait.Backoff{
	Steps:    5,
	Duration: 10 * time.Millisecond,
	Factor:   1.0,
	Jitter:   0.1,
}

var remoteAllocationRetry = wait.Backoff{
	Steps:    7,
	Duration: 100 * time.Millisecond,
	Factor:   2.0,
}

// Allocator handles game server allocation
type Allocator struct {
	baseLogger                   *logrus.Entry
	allocationPolicyLister       multiclusterlisterv1.GameServerAllocationPolicyLister
	allocationPolicySynced       cache.InformerSynced
	secretLister                 corev1lister.SecretLister
	secretSynced                 cache.InformerSynced
	gameServerGetter             getterv1.GameServersGetter
	recorder                     record.EventRecorder
	pendingRequests              chan request
	allocationCache              *AllocationCache
	remoteAllocationCallback     func(context.Context, string, grpc.DialOption, *pb.AllocationRequest) (*pb.AllocationResponse, error)
	remoteAllocationTimeout      time.Duration
	totalRemoteAllocationTimeout time.Duration
	batchWaitTime                time.Duration
}

// request is an async request for allocation
type request struct {
	gsa      *allocationv1.GameServerAllocation
	response chan response
	ctx      context.Context
}

// response is an async response for a matching request
type response struct {
	request request
	gs      *agonesv1.GameServer
	err     error
}

// NewAllocator creates an instance of Allocator
func NewAllocator(policyInformer multiclusterinformerv1.GameServerAllocationPolicyInformer, secretInformer informercorev1.SecretInformer, gameServerGetter getterv1.GameServersGetter,
	kubeClient kubernetes.Interface, allocationCache *AllocationCache, remoteAllocationTimeout time.Duration, totalRemoteAllocationTimeout time.Duration, batchWaitTime time.Duration) *Allocator {
	ah := &Allocator{
		pendingRequests:              make(chan request, maxBatchQueue),
		allocationPolicyLister:       policyInformer.Lister(),
		allocationPolicySynced:       policyInformer.Informer().HasSynced,
		secretLister:                 secretInformer.Lister(),
		secretSynced:                 secretInformer.Informer().HasSynced,
		gameServerGetter:             gameServerGetter,
		allocationCache:              allocationCache,
		batchWaitTime:                batchWaitTime,
		remoteAllocationTimeout:      remoteAllocationTimeout,
		totalRemoteAllocationTimeout: totalRemoteAllocationTimeout,
		remoteAllocationCallback: func(ctx context.Context, endpoint string, dialOpts grpc.DialOption, request *pb.AllocationRequest) (*pb.AllocationResponse, error) {
			conn, err := grpc.NewClient(endpoint, dialOpts)
			if err != nil {
				return nil, err
			}
			defer conn.Close() // nolint: errcheck

			allocationCtx, cancel := context.WithTimeout(ctx, remoteAllocationTimeout)
			defer cancel() // nolint: errcheck
			grpcClient := pb.NewAllocationServiceClient(conn)
			return grpcClient.Allocate(allocationCtx, request)
		},
	}

	ah.baseLogger = runtime.NewLoggerWithType(ah)
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(ah.baseLogger.Debugf)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	ah.recorder = eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "GameServerAllocation-Allocator"})

	return ah
}

// Run initiates the listeners.
func (c *Allocator) Run(ctx context.Context) error {
	if err := c.Sync(ctx); err != nil {
		return err
	}

	if err := c.allocationCache.Run(ctx); err != nil {
		return err
	}

	// workers and logic for batching allocations
	go c.ListenAndAllocate(ctx, maxBatchQueue)

	return nil
}

// Sync waits for cache to sync
func (c *Allocator) Sync(ctx context.Context) error {
	c.baseLogger.Debug("Wait for Allocator cache sync")
	if !cache.WaitForCacheSync(ctx.Done(), c.secretSynced, c.allocationPolicySynced) {
		return errors.New("failed to wait for caches to sync")
	}
	return nil
}

// Allocate CRDHandler for allocating a gameserver.
func (c *Allocator) Allocate(ctx context.Context, gsa *allocationv1.GameServerAllocation) (out k8sruntime.Object, err error) {
	latency := c.newMetrics(ctx)
	defer func() {
		if err != nil {
			latency.setError()
		}
		latency.record()
	}()
	latency.setRequest(gsa)

	// server side validation
	if errs := gsa.Validate(); len(errs) > 0 {
		kind := runtimeschema.GroupKind{
			Group: allocationv1.SchemeGroupVersion.Group,
			Kind:  "GameServerAllocation",
		}
		statusErr := k8serrors.NewInvalid(kind, gsa.Name, errs)
		s := &statusErr.ErrStatus
		var gvks []schema.GroupVersionKind
		gvks, _, err := apiserver.Scheme.ObjectKinds(s)
		if err != nil {
			return nil, errors.Wrap(err, "could not find objectkinds for status")
		}

		c.loggerForGameServerAllocation(gsa).Debug("GameServerAllocation is invalid")
		s.TypeMeta = metav1.TypeMeta{Kind: gvks[0].Kind, APIVersion: gvks[0].Version}
		return s, nil
	}

	// Convert gsa required and preferred fields to selectors field
	gsa.Converter()

	// If multi-cluster setting is enabled, allocate base on the multicluster allocation policy.
	if gsa.Spec.MultiClusterSetting.Enabled {
		out, err = c.applyMultiClusterAllocation(ctx, gsa)
	} else {
		out, err = c.allocateFromLocalCluster(ctx, gsa)
	}

	if err != nil {
		c.loggerForGameServerAllocation(gsa).WithError(err).Error("allocation failed")
		return nil, err
	}
	latency.setResponse(out)

	return out, nil
}

func (c *Allocator) loggerForGameServerAllocationKey(key string) *logrus.Entry {
	return logfields.AugmentLogEntry(c.baseLogger, logfields.GameServerAllocationKey, key)
}

func (c *Allocator) loggerForGameServerAllocation(gsa *allocationv1.GameServerAllocation) *logrus.Entry {
	gsaName := "NilGameServerAllocation"
	if gsa != nil {
		gsaName = gsa.Namespace + "/" + gsa.Name
	}
	return c.loggerForGameServerAllocationKey(gsaName).WithField("gsa", gsa)
}

// allocateFromLocalCluster allocates gameservers from the local cluster.
// Registers number of times we retried before getting a success allocation
func (c *Allocator) allocateFromLocalCluster(ctx context.Context, gsa *allocationv1.GameServerAllocation) (*allocationv1.GameServerAllocation, error) {
	var gs *agonesv1.GameServer
	retry := c.newMetrics(ctx)
	retryCount := 0
	err := Retry(allocationRetry, func() error {
		var err error
		gs, err = c.allocate(ctx, gsa)
		retryCount++

		if err != nil {
			c.loggerForGameServerAllocation(gsa).WithError(err).Warn("Failed to Allocated. Retrying...")
		} else {
			retry.recordAllocationRetrySuccess(ctx, retryCount)
		}
		return err
	})

	if err != nil && err != ErrNoGameServer && err != ErrConflictInGameServerSelection {
		c.allocationCache.Resync()
		return nil, err
	}

	switch err {
	case ErrNoGameServer, ErrGameServerUpdateConflict:
		gsa.Status.State = allocationv1.GameServerAllocationUnAllocated
	case ErrConflictInGameServerSelection:
		gsa.Status.State = allocationv1.GameServerAllocationContention
	default:
		gsa.ObjectMeta.Name = gs.ObjectMeta.Name
		gsa.Status.State = allocationv1.GameServerAllocationAllocated
		gsa.Status.GameServerName = gs.ObjectMeta.Name
		gsa.Status.Ports = gs.Status.Ports
		gsa.Status.Address = gs.Status.Address
		gsa.Status.Addresses = append(gsa.Status.Addresses, gs.Status.Addresses...)
		gsa.Status.NodeName = gs.Status.NodeName
		gsa.Status.Source = localAllocationSource
		gsa.Status.Metadata = &allocationv1.GameServerMetadata{
			Labels:      gs.ObjectMeta.Labels,
			Annotations: gs.ObjectMeta.Annotations,
		}
		if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
			gsa.Status.Counters = gs.Status.Counters
			gsa.Status.Lists = gs.Status.Lists
		}
	}

	c.loggerForGameServerAllocation(gsa).Debug("Game server allocation")
	return gsa, nil
}

// applyMultiClusterAllocation retrieves allocation policies and iterate on policies.
// Then allocate gameservers from local or remote cluster accordingly.
func (c *Allocator) applyMultiClusterAllocation(ctx context.Context, gsa *allocationv1.GameServerAllocation) (result *allocationv1.GameServerAllocation, err error) {
	selector := labels.Everything()
	if len(gsa.Spec.MultiClusterSetting.PolicySelector.MatchLabels)+len(gsa.Spec.MultiClusterSetting.PolicySelector.MatchExpressions) != 0 {
		selector, err = metav1.LabelSelectorAsSelector(&gsa.Spec.MultiClusterSetting.PolicySelector)
		if err != nil {
			return nil, err
		}
	}

	policies, err := c.allocationPolicyLister.GameServerAllocationPolicies(gsa.ObjectMeta.Namespace).List(selector)
	if err != nil {
		return nil, err
	} else if len(policies) == 0 {
		return nil, errors.New("no multi-cluster allocation policy is specified")
	}

	it := multiclusterv1.NewConnectionInfoIterator(policies)
	for {
		connectionInfo := it.Next()
		if connectionInfo == nil {
			break
		}
		if len(connectionInfo.AllocationEndpoints) == 0 {
			// Change the namespace to the policy namespace and allocate locally
			gsaCopy := gsa
			if gsa.Namespace != connectionInfo.Namespace {
				gsaCopy = gsa.DeepCopy()
				gsaCopy.Namespace = connectionInfo.Namespace
			}
			result, err = c.allocateFromLocalCluster(ctx, gsaCopy)
			if err != nil {
				c.loggerForGameServerAllocation(gsaCopy).WithError(err).Error("self-allocation failed")
			}
		} else {
			result, err = c.allocateFromRemoteCluster(gsa, connectionInfo, gsa.ObjectMeta.Namespace)
			if err != nil {
				c.loggerForGameServerAllocation(gsa).WithField("allocConnInfo", connectionInfo).WithError(err).Error("remote-allocation failed")
			}
		}
		if result != nil && result.Status.State == allocationv1.GameServerAllocationAllocated {
			return result, nil
		}
	}
	return result, err
}

// allocateFromRemoteCluster allocates gameservers from a remote cluster by making
// an http call to allocation service in that cluster.
func (c *Allocator) allocateFromRemoteCluster(gsa *allocationv1.GameServerAllocation, connectionInfo *multiclusterv1.ClusterConnectionInfo, namespace string) (*allocationv1.GameServerAllocation, error) {
	var allocationResponse *pb.AllocationResponse

	// TODO: cache the client
	dialOpts, err := c.createRemoteClusterDialOption(namespace, connectionInfo)
	if err != nil {
		return nil, err
	}

	// Forward the game server allocation request to another cluster,
	// and disable multicluster settings to avoid the target cluster
	// forward the allocation request again.
	request := converters.ConvertGSAToAllocationRequest(gsa)
	request.MultiClusterSetting.Enabled = false
	request.Namespace = connectionInfo.Namespace

	ctx, cancel := context.WithTimeout(context.Background(), c.totalRemoteAllocationTimeout)
	defer cancel() // nolint: errcheck
	// Retry on remote call failures.
	var endpoint string
	err = Retry(remoteAllocationRetry, func() error {
		for i, ip := range connectionInfo.AllocationEndpoints {
			select {
			case <-ctx.Done():
				return ErrTotalTimeoutExceeded
			default:
			}
			endpoint = addPort(ip)
			c.loggerForGameServerAllocationKey("remote-allocation").WithField("request", request).WithField("endpoint", endpoint).Debug("forwarding allocation request")
			allocationResponse, err = c.remoteAllocationCallback(ctx, endpoint, dialOpts, request)
			if err != nil {
				c.baseLogger.WithError(err).Error("remote allocation failed")
				// If there are multiple endpoints for the allocator connection and the current one is
				// failing, try the next endpoint. Otherwise, return the error response.
				if (i + 1) < len(connectionInfo.AllocationEndpoints) {
					// If there is a server error try a different endpoint
					c.loggerForGameServerAllocationKey("remote-allocation").WithField("request", request).WithError(err).WithField("endpoint", endpoint).Warn("The request failed. Trying next endpoint")
					continue
				}
				return err
			}
			break
		}

		return nil
	})

	return converters.ConvertAllocationResponseToGSA(allocationResponse, endpoint), err
}

// createRemoteClusterDialOption creates a grpc client dial option with proper certs to make a remote call.
func (c *Allocator) createRemoteClusterDialOption(namespace string, connectionInfo *multiclusterv1.ClusterConnectionInfo) (grpc.DialOption, error) {
	// TODO: disableMTLS works for a single cluster; still need to address how the flag interacts with multi-cluster authentication.
	clientCert, clientKey, caCert, err := c.getClientCertificates(namespace, connectionInfo.SecretName)
	if err != nil {
		return nil, err
	}
	if clientCert == nil || clientKey == nil {
		return nil, fmt.Errorf("missing client certificate key pair in secret %s", connectionInfo.SecretName)
	}

	// Load client cert
	cert, err := tls.X509KeyPair(clientCert, clientKey)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{cert}}
	if len(connectionInfo.ServerCA) != 0 || len(caCert) != 0 {
		// Load CA cert, if provided and trust the server certificate.
		// This is required for self-signed certs.
		tlsConfig.RootCAs = x509.NewCertPool()
		if len(connectionInfo.ServerCA) != 0 && !tlsConfig.RootCAs.AppendCertsFromPEM(connectionInfo.ServerCA) {
			return nil, errors.New("only PEM format is accepted for server CA")
		}
		// Add client CA cert, which can be used instead of / as well as the specified ServerCA cert
		if len(caCert) != 0 {
			_ = tlsConfig.RootCAs.AppendCertsFromPEM(caCert)
		}
	}

	return grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)), nil
}

// getClientCertificates returns the client certificates and CA cert for remote allocation cluster call
func (c *Allocator) getClientCertificates(namespace, secretName string) (clientCert, clientKey, caCert []byte, err error) {
	secret, err := c.secretLister.Secrets(namespace).Get(secretName)
	if err != nil {
		return nil, nil, nil, err
	}
	if secret == nil || len(secret.Data) == 0 {
		return nil, nil, nil, fmt.Errorf("secret %s does not have data", secretName)
	}

	// Create http client using cert
	clientCert = secret.Data[secretClientCertName]
	clientKey = secret.Data[secretClientKeyName]
	caCert = secret.Data[secretCACertName]
	return clientCert, clientKey, caCert, nil
}

// allocate allocated a GameServer from a given GameServerAllocation
// this sets up allocation through a batch process.
func (c *Allocator) allocate(ctx context.Context, gsa *allocationv1.GameServerAllocation) (*agonesv1.GameServer, error) {
	// creates an allocation request. This contains the requested GameServerAllocation, as well as the
	// channel we expect the return values to come back for this GameServerAllocation
	req := request{gsa: gsa, response: make(chan response, 1), ctx: ctx}

	// Avoid pushing request to the queue if the context is already done to prevent
	// the pendingRequests reaching the max buffer size and blocking further allocations
	if ctx.Err() != nil {
		return nil, ErrTotalTimeoutExceeded
	}

	// this pushes the request into the batching process
	c.pendingRequests <- req

	select {
	case res := <-req.response: // wait for the batch to be completed
		return res.gs, res.err
	case <-ctx.Done():
		return nil, ErrTotalTimeoutExceeded
	}
}

// ListenAndAllocate is a blocking function that runs in a loop
// looking at c.requestBatches for batches of requests that are coming through.
func (c *Allocator) ListenAndAllocate(ctx context.Context, updateWorkerCount int) {
	// setup workers for allocation updates. Push response values into
	// this queue for concurrent updating of GameServers to Allocated
	updateQueue := c.allocationUpdateWorkers(ctx, updateWorkerCount)

	// Batch processing strategy:
	// We constantly loop around the below for loop. If nothing is found in c.pendingRequests, we move to
	// default: which will wait for half a second, to allow for some requests to backup in c.pendingRequests,
	// providing us with a batch of Allocation requests in that channel

	// Once we have 1 or more requests in c.pendingRequests (which is buffered to 100), we can start the batch process.

	// Assuming this is the first run (either entirely, or for a while), list will be nil, and therefore the first
	// thing that will be done is retrieving the Ready GameServers and sorting them for this batch via
	// c.listSortedReadyGameServers(). This list is maintained as we flow through the batch.

	// We then use findGameServerForAllocation to loop around the sorted list of Ready GameServers to look for matches
	// against the preferred and required selectors of the GameServerAllocation. If there is an error, we immediately
	// pass that straight back to the response channel for this GameServerAllocation.

	// Assuming we find a matching GameServer to our GameServerAllocation, we remove it from the list and the backing
	// Ready GameServer cache.

	// We then pass the found GameServers into the updateQueue, where there are updateWorkerCount number of goroutines
	// waiting to concurrently attempt to move the GameServer into an Allocated state, and return the result to
	// GameServerAllocation request's response channel

	// Then we get the next item off the batch (c.pendingRequests), and do this all over again, but this time, we have
	// an already sorted list of GameServers, so we only need to find one that matches our GameServerAllocation
	// selectors, and put it into updateQueue

	// The tracking of requestCount >= maxBatchBeforeRefresh is necessary, because without it, at high enough load
	// the list of GameServers that we are using to allocate would never get refreshed (list = nil) with an updated
	// list of Ready GameServers, and you would eventually never be able to Allocate anything as long as the load
	// continued.

	var list []*agonesv1.GameServer
	var sortKey uint64
	requestCount := 0

	for {
		select {
		case req := <-c.pendingRequests:
			// Ensure the request context is still active before processing to avoid allocating unnecessarily
			if req.ctx.Err() != nil {
				c.tryRespondWithError(req, ErrTotalTimeoutExceeded)
				continue
			}

			// refresh the list after every 100 allocations made in a single batch
			if requestCount >= maxBatchBeforeRefresh {
				list = nil
				requestCount = 0
			}

			if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
				// SortKey returns the sorting values (list of Priorities) as a determinstic key.
				// In case gsa.Spec.Priorities is nil this will still return a sortKey.
				// In case of error this will return 0 for the sortKey.
				newSortKey, err := req.gsa.SortKey()
				if err != nil {
					c.baseLogger.WithError(err).Warn("error getting sortKey for GameServerAllocationSpec", err)
				}
				// Set sortKey if this is the first request, or the previous request errored on creating a sortKey.
				if sortKey == uint64(0) {
					sortKey = newSortKey
				}

				if newSortKey != sortKey {
					sortKey = newSortKey
					list = nil
					requestCount = 0
				}
			}

			requestCount++

			if list == nil {
				if !runtime.FeatureEnabled(runtime.FeatureCountsAndLists) || req.gsa.Spec.Scheduling == apis.Packed {
					list = c.allocationCache.ListSortedGameServers(req.gsa)
				} else {
					// If FeatureCountsAndLists and Scheduling == Distributed, sort game servers by Priorities
					list = c.allocationCache.ListSortedGameServersPriorities(req.gsa)
				}
			}

			gs, index, err := findGameServerForAllocation(req.gsa, list)
			if err != nil {
				c.tryRespondWithError(req, err)
				continue
			}
			// remove the game server that has been allocated
			list = append(list[:index], list[index+1:]...)

			if err := c.allocationCache.RemoveGameServer(gs); err != nil {
				// this seems unlikely, but lets handle it just in case
				c.tryRespondWithError(req, err)
				continue
			}

			updateQueue <- response{request: req, gs: gs.DeepCopy(), err: nil}

		case <-ctx.Done():
			return
		default:
			list = nil
			requestCount = 0
			// slow down cpu churn, and allow items to batch
			time.Sleep(c.batchWaitTime)
		}
	}
}

// allocationUpdateWorkers runs workerCount number of goroutines as workers to
// process each GameServer passed into the returned updateQueue
// Each worker will concurrently attempt to move the GameServer to an Allocated
// state and then respond to the initial request's response channel with the
// details of that update
func (c *Allocator) allocationUpdateWorkers(ctx context.Context, workerCount int) chan<- response {
	updateQueue := make(chan response)

	for i := 0; i < workerCount; i++ {
		go func() {
			for {
				select {
				case res := <-updateQueue:
					gs, err := c.applyAllocationToGameServer(ctx, res.request.gsa.Spec.MetaPatch, res.gs, res.request.gsa)
					if err != nil {
						if !k8serrors.IsConflict(errors.Cause(err)) {
							// since we could not allocate, we should put it back
							// but not if it's a conflict, as the cache is no longer up to date, and
							// we should wait for it to get updated with fresh info.
							c.allocationCache.AddGameServer(gs)
						}
						res.err = ErrGameServerUpdateConflict
					} else {
						// put the GameServer back into the cache, so it's immediately around for re-allocation
						c.allocationCache.AddGameServer(gs)
						res.gs = gs
					}

					res.request.response <- res
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	return updateQueue
}

// tryRespondWithError attempts to send an error response without blocking
// This prevents deadlocks when the receiver may have exited due to context cancellation
func (c *Allocator) tryRespondWithError(req request, err error) {
	select {
	case req.response <- response{request: req, gs: nil, err: err}:
	default:
	}
}

// applyAllocationToGameServer patches the inputted GameServer with the allocation metadata changes, and updates it to the Allocated State.
// Returns the updated GameServer.
func (c *Allocator) applyAllocationToGameServer(ctx context.Context, mp allocationv1.MetaPatch, gs *agonesv1.GameServer, gsa *allocationv1.GameServerAllocation) (*agonesv1.GameServer, error) {
	// patch ObjectMeta labels
	if mp.Labels != nil {
		if gs.ObjectMeta.Labels == nil {
			gs.ObjectMeta.Labels = make(map[string]string, len(mp.Labels))
		}
		for key, value := range mp.Labels {
			gs.ObjectMeta.Labels[key] = value
		}
	}

	if gs.ObjectMeta.Annotations == nil {
		gs.ObjectMeta.Annotations = make(map[string]string, len(mp.Annotations))
	}
	// apply annotations patch
	for key, value := range mp.Annotations {
		gs.ObjectMeta.Annotations[key] = value
	}

	// add last allocated, so it always gets updated, even if it is already Allocated
	ts, err := time.Now().MarshalText()
	if err != nil {
		return nil, err
	}
	gs.ObjectMeta.Annotations[LastAllocatedAnnotationKey] = string(ts)
	gs.Status.State = agonesv1.GameServerStateAllocated

	// perfom any Counter or List actions
	var counterErrors error
	var listErrors error
	if runtime.FeatureEnabled(runtime.FeatureCountsAndLists) {
		if gsa.Spec.Counters != nil {
			for counter, ca := range gsa.Spec.Counters {
				counterErrors = goErrors.Join(counterErrors, ca.CounterActions(counter, gs))
			}
		}
		if gsa.Spec.Lists != nil {
			for list, la := range gsa.Spec.Lists {
				listErrors = goErrors.Join(listErrors, la.ListActions(list, gs))
			}
		}
	}

	gsUpdate, updateErr := c.gameServerGetter.GameServers(gs.ObjectMeta.Namespace).Update(ctx, gs, metav1.UpdateOptions{})
	if updateErr != nil {
		return gsUpdate, updateErr
	}

	// If successful Update record any Counter or List action errors as a warning
	if counterErrors != nil {
		c.recorder.Event(gsUpdate, corev1.EventTypeWarning, "CounterActionError", counterErrors.Error())
	}
	if listErrors != nil {
		c.recorder.Event(gsUpdate, corev1.EventTypeWarning, "ListActionError", listErrors.Error())
	}
	c.recorder.Event(gsUpdate, corev1.EventTypeNormal, string(gsUpdate.Status.State), "Allocated")

	return gsUpdate, updateErr
}

// Retry retries fn based on backoff provided.
func Retry(backoff wait.Backoff, fn func() error) error {
	var lastConflictErr error
	err := wait.ExponentialBackoff(backoff, func() (bool, error) {
		err := fn()

		st, ok := status.FromError(err)
		if ok {
			if st.Code() == codes.ResourceExhausted {
				return true, err
			}
		}

		switch {
		case err == nil:
			return true, nil
		case err == ErrNoGameServer:
			return true, err
		case err == ErrTotalTimeoutExceeded:
			return true, err
		default:
			lastConflictErr = err
			return false, nil
		}
	})
	if wait.Interrupted(err) {
		err = lastConflictErr
	}
	return err
}

// newMetrics creates a new gsa latency recorder.
func (c *Allocator) newMetrics(ctx context.Context) *metrics {
	ctx, err := tag.New(ctx, latencyTags...)
	if err != nil {
		c.baseLogger.WithError(err).Warn("failed to tag latency recorder.")
	}
	return &metrics{
		ctx:              ctx,
		gameServerLister: c.allocationCache.gameServerLister,
		logger:           c.baseLogger,
		start:            time.Now(),
	}
}

func addPort(ip string) string {
	if strings.Contains(ip, ":") {
		return ip
	}
	return fmt.Sprintf("%s:%s", ip, allocatorPort)
}
