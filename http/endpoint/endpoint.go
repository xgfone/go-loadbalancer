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
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/xgfone/go-loadbalancer"
	"github.com/xgfone/go-loadbalancer/endpoint"
	"github.com/xgfone/go-loadbalancer/http/processor"
)

// Request represents a upstream request context.
type Request struct {
	RespBodyProcessor processor.ResponseProcessor

	SrcRes http.ResponseWriter
	SrcReq *http.Request
	DstReq *http.Request
}

// NewRequest returns a new Request.
func NewRequest(srcres http.ResponseWriter, srcreq, dstreq *http.Request) Request {
	return Request{SrcRes: srcres, SrcReq: srcreq, DstReq: dstreq}
}

func (r Request) WithRespBodyProcessor(processor processor.ResponseProcessor) Request {
	r.RespBodyProcessor = processor
	return r
}

// RequestID returns the request id.
func (r Request) RequestID() string { return r.SrcReq.Header.Get("X-Request-Id") }

// RemoteAddr returns the address of the client.
func (r Request) RemoteAddr() string { return r.SrcReq.RemoteAddr }

// GetRequest returns the request.
func (r Request) GetRequest() *http.Request { return r.SrcReq }

// ------------------------------------------------------------------------- //

// Config is used to configure and new a http endpoint.
type Config struct {
	IP   string
	Port uint16

	Weight    int
	GetWeight func(loadbalancer.Endpoint) int

	Client *http.Client
}

func (c Config) ID() string {
	return net.JoinHostPort(c.IP, strconv.FormatUint(uint64(c.Port), 10))
}

// NewEndpoint returns a new endpoint.
//
// For the method Serve, req must be Request.
// For the method Check, req must be one of *http.Request, Request, or nil.
func (c Config) NewEndpoint() loadbalancer.WeightedEndpoint {
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
	id    string
	conf  atomic.Value
	state loadbalancer.EndpointState
	endpoint.StatusManager
}

func newServer(id string, conf Config) *server {
	s := new(server)
	s.setConf(conf)
	s.SetStatus(loadbalancer.EndpointStatusOnline)
	s.id = id
	return s
}

func (s *server) String() string { return s.id }

func (s *server) getConf() Config        { return s.conf.Load().(Config) }
func (s *server) setConf(c Config) error { s.conf.Store(c); return nil }

func (s *server) ID() string                        { return s.id }
func (s *server) Type() string                      { return "http" }
func (s *server) Info() interface{}                 { return s.getConf() }
func (s *server) Update(info interface{}) error     { return s.setConf(info.(Config)) }
func (s *server) State() loadbalancer.EndpointState { return s.state.Clone() }

func (s *server) Weight() int {
	if conf := s.getConf(); conf.GetWeight != nil {
		return conf.GetWeight(s)
	} else if conf.Weight > 0 {
		return conf.Weight
	}
	return 1
}

func (s *server) Serve(ctx context.Context, req interface{}) (err error) {
	s.state.Inc()
	defer s.state.Dec()

	r := req.(Request)
	r.DstReq.RequestURI = ""
	r.DstReq.URL.Host = s.id

	resp, err := s.do(r.DstReq)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err == nil {
		if r.RespBodyProcessor != nil {
			err = r.RespBodyProcessor.ProcessResponse(ctx, r.SrcRes, r.SrcReq, r.DstReq, resp, nil)
		} else {
			err = s.handleResponse(r.SrcRes, resp)
		}
	}

	if err == nil {
		s.state.IncSuccess()
		// TODO: log
	} else {
		// TODO: log
	}

	log.Printf("forward request to %s, %d", s.id, resp.StatusCode)

	return
}

func (s *server) Check(ctx context.Context, req interface{}) (ok bool) {
	switch r := req.(type) {
	case *http.Request:
		// HTTP
		r.RequestURI = ""
		r.URL.Host = s.id
		resp, err := s.do(r)
		ok = err == nil
		if resp != nil {
			resp.Body.Close()
		}

	case Request:
		// HTTP
		r.DstReq.RequestURI = ""
		r.DstReq.URL.Host = s.id
		resp, err := s.do(r.DstReq)
		ok = err == nil
		if resp != nil {
			resp.Body.Close()
		}

	case nil:
		// TCP
		conn, err := net.DialTimeout("tcp", s.id, time.Second)
		ok = err == nil
		if conn != nil {
			conn.Close()
		}

	default:
		panic(fmt.Errorf("HttpEndpoint.Check: unsupported req type '%s'", req))
	}

	return
}

func (s *server) do(req *http.Request) (*http.Response, error) {
	if client := s.getConf().Client; client != nil {
		return client.Do(req)
	}
	return http.DefaultClient.Do(req)
}

func (s *server) handleResponse(w http.ResponseWriter, resp *http.Response) error {
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	_, err := io.CopyBuffer(w, resp.Body, make([]byte, 1024))
	return loadbalancer.NewError(false, err)
}
