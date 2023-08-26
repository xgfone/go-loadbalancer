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
// NOTICE: THIS IS ONLY A SIMPLE EXAMPLE, AND YOU SHOULD IMPLEMENT YOURSELF.
package endpoint

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

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

// NewEndpoint returns a new endpoint, which will get the http request
// by the function defaults.GetHTTPRequest.
func (c Config) NewEndpoint() endpoint.WeightedEndpoint {
	if c.Host == "" {
		panic("HttpEndpoint: host must not be empty")
	}
	if c.Port == 0 {
		panic("HttpEndpoint: port must not be 0")
	}
	return newServer(c.ID(), c)
}

// ------------------------------------------------------------------------- //

type server struct {
	host  string
	conf  atomic.Value
	state endpoint.State
	endpoint.StatusManager
}

func newServer(host string, conf Config) *server {
	s := new(server)
	s.setConf(conf)
	s.SetStatus(endpoint.StatusOnline)
	s.host = host
	return s
}

func (s *server) String() string { return s.host }

func (s *server) getConf() Config        { return s.conf.Load().(Config) }
func (s *server) setConf(c Config) error { s.conf.Store(c); return nil }

func (s *server) ID() string                    { return s.host }
func (s *server) Type() string                  { return "http" }
func (s *server) Info() interface{}             { return s.getConf() }
func (s *server) Update(info interface{}) error { return s.setConf(info.(Config)) }
func (s *server) State() endpoint.State         { return s.state.Clone() }

func (s *server) Weight() int {
	if conf := s.getConf(); conf.Weight > 0 {
		return conf.Weight
	}
	return 1
}

func (s *server) Serve(ctx context.Context, req interface{}) (interface{}, error) {
	s.state.Inc()
	defer s.state.Dec()

	_req := req.(*http.Request)
	_req.URL.Host = s.host
	resp, err := http.DefaultClient.Do(_req)
	if err == nil {
		s.state.IncSuccess()
	}
	return resp, err
}

func (s *server) Check(ctx context.Context, req interface{}) (ok bool) {
	if req == nil {
		return s.checkTCP()
	}
	return s.checkHTTP(ctx, req.(*http.Request))
}

func (s *server) checkHTTP(ctx context.Context, req *http.Request) (ok bool) {
	req.RequestURI = ""
	req.URL.Host = s.host
	resp, err := http.DefaultClient.Do(req)
	ok = err == nil && resp.StatusCode < 500
	if resp != nil {
		resp.Body.Close()
	}
	return
}

func (s *server) checkTCP() (ok bool) {
	conn, err := net.DialTimeout("tcp", s.host, time.Second)
	ok = err == nil
	if conn != nil {
		conn.Close()
	}
	return
}
