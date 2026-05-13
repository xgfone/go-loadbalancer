// Copyright 2026 xgfone
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

package balancer

import (
	"context"
	"slices"
	"time"

	"github.com/xgfone/go-loadbalancer"
	"github.com/xgfone/go-loadbalancer/selector"
)

var _ Balancer = Retry{}

// Retry is used to retry the rest endpoints when failing to forward the request
// until all the endpoints are tried or it reaches the maximum retry number.
//
// Notice: It will retry the same backend endpoints for the sourceip
// or consistent hash selector.
type Retry struct {
	MaxNumber int // If 0, ignore it
	Interval  time.Duration
	selector.Selector
}

// NewRetry returns a new retry balancer.
func NewRetry(selector selector.Selector, interval time.Duration, maxNum int) Retry {
	return Retry{Selector: selector, Interval: interval, MaxNumber: maxNum}
}

// Forward forwards the request to one of the endpoints by the selector.
func (b Retry) Forward(c context.Context, r any, eps *loadbalancer.Static) (resp any, err error) {
	_len := len(eps.Endpoints)
	switch _len {
	case 0:
		return nil, loadbalancer.ErrNoAvailableEndpoints

	case 1:
		return eps.Endpoints[0].Serve(c, r)
	}

	if b.MaxNumber > 0 && b.MaxNumber < _len {
		_len = b.MaxNumber
	}

	_eps := eps
	var ep loadbalancer.Endpoint

	for ; _len > 0 && len(_eps.Endpoints) > 0; _len-- {
		select {
		case <-c.Done():
			if err == nil {
				err = c.Err()
			}
			return
		default:
		}

		ep, err = b.Select(c, r, _eps)
		if err != nil {
			return nil, err
		}

		resp, err = ep.Serve(c, r)
		if err == nil {
			return
		}

		if e, ok := err.(loadbalancer.RetryError); ok && !e.Retry() {
			return
		}

		if _eps.Len() <= 1 {
			return nil, err
		}

		if _eps == eps {
			_eps = &loadbalancer.Static{Endpoints: slices.Clone(eps.Endpoints)}
		}

		if index := _eps.Index(ep.ID()); index > -1 {
			_eps.Endpoints = slices.Delete(_eps.Endpoints, index, index+1)
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
				return
			}
		}
	}

	return
}
