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
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"net/netip"

	"github.com/xgfone/go-loadbalancer"
)

var random = rand.Intn

// GetSourceIP is the default function to get the source ip of the request.
var GetSourceIP func(ctx context.Context, req any) (netip.Addr, error)

func getsourceip(_ context.Context, req any) (addr netip.Addr) {
	switch v := req.(type) {
	case interface{ RemoteAddr() netip.Addr }:
		return v.RemoteAddr()

	case interface{ RemoteAddr() string }:
		return parseaddr(v.RemoteAddr())

	case *http.Request:
		return parseaddr(v.RemoteAddr)

	default:
		panic(fmt.Errorf("unknown request %T", req))
	}
}

func parseaddr(s string) (addr netip.Addr) {
	host, _, err := net.SplitHostPort(s)
	if err != nil {
		slog.Error("fail to split hostport", "addr", s, "err", err)
		return
	}

	addr, err = netip.ParseAddr(host)
	if err != nil {
		slog.Error("fail to parse ip", "ip", host, "err", err)
	}

	return
}

// Balancer implements the balancer based on the source-ip hash.
type Balancer struct {
	// GetSourceAddr is used to get the source address.
	//
	// If nil, use GetSourceIP or defaults.GetClientIP insead.
	GetSourceIP func(ctx context.Context, req any) (netip.Addr, error)

	policy string
}

// NewBalancer returns a new balancer based on the source-ip hash
// with the policy name.
//
// If policy is empty, use "sourceip_hash" instead.
func NewBalancer(policy string) *Balancer {
	if policy == "" {
		policy = "sourceip_hash"
	}
	return &Balancer{policy: policy}
}

// Policy returns the policy of the balancer.
func (b *Balancer) Policy() string { return b.policy }

// Forward forwards the request to one of the backend endpoints.
func (b *Balancer) Forward(c context.Context, r any, eps *loadbalancer.Static) (any, error) {
	_len := len(eps.Endpoints)
	switch _len {
	case 0:
		return nil, loadbalancer.ErrNoAvailableEndpoints
	case 1:
		return eps.Endpoints[0].Serve(c, r)
	}

	var err error
	var sip netip.Addr
	switch {
	case b.GetSourceIP != nil:
		sip, err = b.GetSourceIP(c, r)
	case GetSourceIP != nil:
		sip, err = GetSourceIP(c, r)
	default:
		sip = getsourceip(c, r)
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
		value = uint64(random(_len))
	}

	return eps.Endpoints[value%uint64(_len)].Serve(c, r)
}
