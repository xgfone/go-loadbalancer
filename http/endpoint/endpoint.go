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

// Package endpoint provides a backend endpoint based on the stdlib "net/http".
package endpoint

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/xgfone/go-defaults"
	"github.com/xgfone/go-loadbalancer/endpoint"
	"github.com/xgfone/go-loadbalancer/internal/slog"
)

// Request represents the request context of the http endpoint.
type Request interface {
	Request() *http.Request
}

// NewRequest returns a new Request.
func NewRequest(req *http.Request) Request { return request{req: req} }

var _ Request = NewRequest(nil)

type request struct{ req *http.Request }

func (r request) Request() *http.Request { return r.req }

// ------------------------------------------------------------------------- //

// Config is used to configure and new a http endpoint.
type Config struct {
	// Required
	IP   string
	Port uint16

	// Default: 1
	Weight    int
	GetWeight func(endpoint.Endpoint) int

	// HandleResponse is used to wrap the http response and return a new response.
	//
	// Default: do nothing and return resp.
	HandleResponse func(ep endpoint.Endpoint, resp *http.Response) interface{}

	// If nil, use http.DefaultClient instead.
	Client *http.Client
}

func (c Config) ID() string {
	return net.JoinHostPort(c.IP, strconv.FormatUint(uint64(c.Port), 10))
}

// NewEndpoint returns a new endpoint.
//
// For the method Serve, req must be one of
//   - Request
//   - interface{ GetRequest() Request }
//
// For the method Check, req must be one of
//   - nil
//   - Request
//   - *http.Request
//   - interface{ GetRequest() *http.Request }
//   - interface{ GetHTTPRequest() *http.Request }
func (c Config) NewEndpoint() endpoint.WeightedEndpoint {
	if c.IP == "" {
		panic("HttpEndpoint: ip must not be empty")
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
	if conf := s.getConf(); conf.GetWeight != nil {
		return conf.GetWeight(s)
	} else if conf.Weight > 0 {
		return conf.Weight
	}
	return 1
}

func (s *server) Serve(ctx context.Context, req interface{}) (res interface{}, err error) {
	s.state.Inc()
	defer s.state.Dec()

	var statusCode int
	start := time.Now()

	var _req *http.Request
	switch r := req.(type) {
	case *http.Request:
		_req = r
	case Request:
		_req = r.Request()
	case interface{ GetRequest() *http.Request }:
		_req = r.GetRequest()
	case interface{ GetHTTPRequest() *http.Request }:
		_req = r.GetHTTPRequest()
	default:
		panic(fmt.Errorf("HttpEndpoint: unsupported request type %T", req))
	}

	_req.URL.Host = s.host
	resp, err := s.do(ctx, _req)
	if err == nil {
		statusCode = resp.StatusCode
		s.state.IncSuccess()
	}

	if slog.Enabled(ctx, slog.LevelDebug) {
		slog.Debug("forward the http request to the backend http endpoint",
			"epid", s.host,
			"reqid", defaults.GetRequestID(ctx, req),
			"method", _req.Method,
			"host", _req.Host,
			"addr", _req.URL.Host,
			"path", _req.URL.Path,
			"code", statusCode,
			"cost", time.Since(start).String(),
			"err", err,
		)
	}

	res = resp
	return
}

func (s *server) Check(ctx context.Context, req interface{}) (ok bool) {
	switch r := req.(type) {
	case interface{ GetHTTPRequest() *http.Request }:
		return s.checkHTTP(ctx, r.GetHTTPRequest())

	case interface{ GetRequest() *http.Request }:
		return s.checkHTTP(ctx, r.GetRequest())

	case *http.Request:
		return s.checkHTTP(ctx, r)

	case nil:
		return s.checkTCP()

	default:
		panic(fmt.Errorf("HttpEndpoint.Check: unsupported req type '%T'", req))
	}
}

func (s *server) checkHTTP(ctx context.Context, req *http.Request) (ok bool) {
	req.RequestURI = ""
	req.URL.Host = s.host
	resp, err := s.do(ctx, req)
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

func (s *server) do(ctx context.Context, req *http.Request) (*http.Response, error) {
	if _, ok := ctx.Deadline(); ok {
		req = req.WithContext(ctx)
	}

	if client := s.getConf().Client; client != nil {
		return client.Do(req)
	}
	return http.DefaultClient.Do(req)
}
