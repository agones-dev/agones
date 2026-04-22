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

package main

import (
	"bytes"
	"context"
	"math"
	"net"
	"os"
	"sync"
	"time"

	"agones.dev/agones/pkg/util/runtime"

	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/clock"
)

// udpServer is a rate limited udp server that echos
// udp packets back to senders
type udpServer struct {
	logger      *logrus.Entry
	conn        net.PacketConn
	rateLimit   rate.Limit
	rateBurst   int
	clock       clock.WithTickerAndDelayedExecution
	limitsMutex sync.Mutex
	limits      map[string]*visitor
	healthMutex sync.RWMutex
	health      bool
}

// visitor tracks when a visitor last sent
// a packet, and it's rate limit
type visitor struct {
	stamp time.Time
	limit *rate.Limiter
}

// newUDPServer returns a new udpServer implementation
// withe the rate limit
func newUDPServer(rateLimit rate.Limit) *udpServer {
	udpSrv := &udpServer{
		rateLimit:   rateLimit,
		rateBurst:   int(math.Floor(float64(rateLimit))),
		clock:       clock.RealClock{},
		limitsMutex: sync.Mutex{},
		limits:      map[string]*visitor{},
	}
	udpSrv.logger = runtime.NewLoggerWithType(udpSrv)
	return udpSrv
}

// run runs the udp server. Non blocking operation
func (u *udpServer) run(ctx context.Context) {
	u.healthy()

	logger.Info("Starting UDP server")
	var err error
	u.conn, err = net.ListenPacket("udp", ":8080")
	if err != nil {
		logger.WithError(err).Fatal("Could not start udp server")
	}

	go func() {
		defer u.unhealthy()
		wait.Until(u.cleanUp, time.Minute, ctx.Done())
	}()

	u.readWriteLoop(ctx)
}

// cleans up visitors, if they are more than a
// minute without being touched
func (u *udpServer) cleanUp() {
	u.limitsMutex.Lock()
	defer u.limitsMutex.Unlock()
	for k, v := range u.limits {
		if u.clock.Now().Sub(v.stamp) > time.Minute {
			delete(u.limits, k)
		}
	}
}

// readWriteLoop reads the UDP packet in, and then echos the data back
// in a rate limited way
func (u *udpServer) readWriteLoop(ctx context.Context) {
	go func() {
		defer u.unhealthy()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				b := make([]byte, 1024)
				_, sender, err := u.conn.ReadFrom(b)
				if err != nil {
					if ctx.Err() != nil && err == os.ErrClosed {
						return
					}
					u.logger.WithError(err).Error("Error reading udp packet")
					continue
				}
				go u.rateLimitedEchoResponse(b, sender)
			}
		}
	}()
}

// rateLimitedEchoResponse echos the udp request, but is ignored if
// it is past its rate limit
func (u *udpServer) rateLimitedEchoResponse(b []byte, sender net.Addr) {
	var vis *visitor
	u.limitsMutex.Lock()
	key := sender.String()
	vis, ok := u.limits[key]
	if !ok {
		vis = &visitor{limit: rate.NewLimiter(u.rateLimit, u.rateBurst)}
		u.limits[key] = vis
	}
	vis.stamp = u.clock.Now()
	u.limitsMutex.Unlock()

	if vis.limit.Allow() {
		b = bytes.TrimRight(b, "\x00")
		if _, err := u.conn.WriteTo(b, sender); err != nil {
			u.logger.WithError(err).Error("Error sending returning udp packet")
		}
	} else {
		logger.WithField("addr", sender.String()).Warn("Rate limited. No response sent")
	}
}

// close closes and shutdown the udp server
func (u *udpServer) close() {
	if err := u.conn.Close(); err != nil {
		logger.WithError(err).Error("Error closing udp connection")
	}
}

// healthy marks this udpServer as healthy
func (u *udpServer) healthy() {
	u.healthMutex.Lock()
	defer u.healthMutex.Unlock()
	u.health = true
}

// unhealthy marks this udpServer as unhealthy
func (u *udpServer) unhealthy() {
	u.healthMutex.Lock()
	defer u.healthMutex.Unlock()
	u.health = false
}

// Health returns the health of the UDP Server.
// true is healthy, false is not
func (u *udpServer) Health() error {
	u.healthMutex.RLock()
	defer u.healthMutex.RUnlock()
	if !u.health {
		return errors.New("UDP Server is unhealthy")
	}
	return nil
}
