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

// Package sourceiphash provides a balancer based on the source-ip hash.
package sourceiphash

import (
	"context"
	"encoding/binary"
	"net/netip"

	"github.com/xgfone/go-defaults"
	"github.com/xgfone/go-loadbalancer"
	"github.com/xgfone/go-loadbalancer/endpoint"
	"github.com/xgfone/go-loadbalancer/internal/random"
)

// Balancer implements the balancer based on the source-ip hash.
type Balancer struct {
	// GetSourceAddr is used to get the source address.
	//
	// If nil, use defaults.GetRemoteAddr instead.
	GetSourceAddr func(ctx context.Context, req interface{}) (netip.Addr, error)

	policy string
	random func(int) int
}

// NewBalancer returns a new balancer based on the source-ip hash
// with the policy name.
//
// If policy is empty, use "source_ip_hash" instead.
func NewBalancer(policy string) *Balancer {
	if policy == "" {
		policy = "source_ip_hash"
	}
	return &Balancer{policy: policy, random: random.NewRandom()}
}

// Policy returns the policy of the balancer.
func (b *Balancer) Policy() string { return b.policy }

// Forward forwards the request to one of the backend endpoints.
func (b *Balancer) Forward(c context.Context, r interface{}, sd endpoint.Discovery) (interface{}, error) {
	eps := sd.OnEndpoints()
	_len := len(eps)
	switch _len {
	case 0:
		return nil, loadbalancer.ErrNoAvailableEndpoints
	case 1:
		return eps[0].Serve(c, r)
	}

	var err error
	var sip netip.Addr
	if b.GetSourceAddr == nil {
		sip, err = defaults.GetRemoteAddr(c, r)
	} else {
		sip, err = b.GetSourceAddr(c, r)
	}

	if err != nil {
		return nil, err
	}

	var value uint64
	switch sip.BitLen() {
	case 32:
		b4 := sip.As4()
		value = uint64(binary.BigEndian.Uint32(b4[:]))

	case 128:
		b16 := sip.As16()
		value = binary.BigEndian.Uint64(b16[8:16])

	default:
		value = uint64(b.random(_len))
	}

	return eps[value%uint64(_len)].Serve(c, r)
}
