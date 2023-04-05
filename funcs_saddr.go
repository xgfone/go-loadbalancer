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

package loadbalancer

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
)

// GetRemoteAddr is used to get the remote address.
//
// For the default implementation, it only detects req
// and supports the types or interfaces:
//
//	*http.Request
//	interface{ GetRequest() *http.Request }
//	interface{ GetHTTPRequest() *http.Request }
//	interface{ RemoteAddr() string }
//	interface{ RemoteAddr() net.IP }
//	interface{ RemoteAddr() net.Addr }
//	interface{ RemoteAddr() netip.Addr }
var GetRemoteAddr func(req interface{}) (netip.Addr, error) = getRemoteAddr

func getRemoteAddr(req interface{}) (addr netip.Addr, err error) {
	switch v := req.(type) {
	case *http.Request:
		return netip.ParseAddr(v.RemoteAddr)

	case interface{ GetRequest() *http.Request }:
		return netip.ParseAddr(v.GetRequest().RemoteAddr)

	case interface{ GetHTTPRequest() *http.Request }:
		return netip.ParseAddr(v.GetHTTPRequest().RemoteAddr)

	case interface{ RemoteAddr() string }:
		return netip.ParseAddr(v.RemoteAddr())

	case interface{ RemoteAddr() net.IP }:
		return ip2addr(v.RemoteAddr()), nil

	case interface{ RemoteAddr() net.Addr }:
		return netip.ParseAddr(v.RemoteAddr().String())

	case interface{ RemoteAddr() netip.Addr }:
		return v.RemoteAddr(), nil

	default:
		panic(fmt.Errorf("GetSourceAddr: unknown type %T", req))
	}
}

func ip2addr(ip net.IP) (addr netip.Addr) {
	switch len(ip) {
	case net.IPv4len:
		var b [4]byte
		copy(b[:], ip)
		addr = netip.AddrFrom4(b)

	case net.IPv6len:
		var b [16]byte
		copy(b[:], ip)
		addr = netip.AddrFrom16(b)
	}
	return
}
