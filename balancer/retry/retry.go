// Copyright 2023 xgfone
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

// Package retry provides a retry balancer, which will retry the rest endpoints
// when failing to forward the request.
package retry

import (
	"context"
	"time"

	"github.com/xgfone/go-loadbalancer"
	"github.com/xgfone/go-loadbalancer/balancer"
	"github.com/xgfone/go-loadbalancer/endpoint"
)

var _ balancer.Balancer = Retry{}

// Retry is used to retry the rest endpoints when failing to forward the request
// until all the endpoints are tried or it reaches the maximum retry number.
//
// Notice: It will retry the same backend endpoints for the sourceip
// or consistent hash balancer.
type Retry struct {
	MaxNumber int // If 0, ignore it
	Interval  time.Duration
	balancer.Balancer
}

// NewRetry returns a new retry balancer.
func NewRetry(balancer balancer.Balancer, interval time.Duration, maxNum int) Retry {
	return Retry{Balancer: balancer, Interval: interval, MaxNumber: maxNum}
}

// Forward overrides the Forward method.
func (b Retry) Forward(c context.Context, r interface{}, sd endpoint.Discovery) (err error) {
	eps := sd.OnEndpoints()
	_len := len(eps)
	switch _len {
	case 0:
		return loadbalancer.ErrNoAvailableEndpoints
	case 1:
		return eps[0].Serve(c, r)
	}

	if b.MaxNumber > 0 && b.MaxNumber < _len {
		_len = b.MaxNumber
	}

	for ; _len > 0; _len-- {
		select {
		case <-c.Done():
			return c.Err()
		default:
		}

		err = b.Balancer.Forward(c, r, sd)
		switch e := err.(type) {
		case nil:
			return
		case loadbalancer.RetryError:
			if !e.Retry() {
				return
			}
		}

		if b.Interval > 0 {
			timer := time.NewTimer(b.Interval)
			select {
			case <-timer.C:
			case <-c.Done():
				timer.Stop()
				select {
				case <-timer.C:
				default:
				}
				return c.Err()
			}
		}
	}

	return
}
