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

// Package endpoint provides a backend endpoint based on the stdlib
// "net/http".
//
// NOTICE: THIS IS ONLY A SIMPLE EXAMPLE, AND YOU MAYBE IMPLEMENT YOURSELF.
package endpoint

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/xgfone/go-loadbalancer/endpoint"
)

// Config is used to configure and new a http endpoint.
type Config struct {
	Host   string // Required
	Port   uint16 // Required
	Weight int    // Optional, Default: 1
}

func (c Config) ID() string {
	return net.JoinHostPort(c.Host, strconv.FormatUint(uint64(c.Port), 10))
}

// NewEndpoint returns a new endpoint.
//
// For the argument request, it may be one of types:
//
//	*http.Request
//	interface{ Request() *http.Request }
func (c Config) NewEndpoint() *endpoint.Endpoint {
	if c.Host == "" {
		panic("HttpEndpoint: host must not be empty")
	}
	if c.Port == 0 {
		panic("HttpEndpoint: port must not be 0")
	}

	host := c.ID()
	return endpoint.New(host, c.Weight, func(c context.Context, i interface{}) (interface{}, error) {
		var req *http.Request
		switch r := i.(type) {
		case *http.Request:
			req = r
		case interface{ Request() *http.Request }:
			req = r.Request()
		default:
			panic(fmt.Errorf("HttpEndpoint: unknown request type %T", i))
		}

		req.URL.Host = host
		resp, err := http.DefaultClient.Do(req)
		if err != nil && resp != nil {
			resp.Body.Close() // For status code 3xx
		}
		return resp, err
	})
}
