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
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/xgfone/go-generics/slices"
	"github.com/xgfone/go-loadbalancer"
	"github.com/xgfone/go-loadbalancer/endpoint"
	"github.com/xgfone/go-loadbalancer/http/processor"
	"github.com/xgfone/go-loadbalancer/internal/slog"
)

// Context represents a upstream request context.
type Context struct {
	processor.ResponseProcessor

	SrcRes http.ResponseWriter
	SrcReq *http.Request
	DstReq *http.Request

	RAddr string
	RID   string
}

// NewContext returns a new request Context.
func NewContext(srcres http.ResponseWriter, srcreq, dstreq *http.Request) Context {
	return Context{SrcRes: srcres, SrcReq: srcreq, DstReq: dstreq}
}

// WithResponseProcessor returns a new request Context with the response processor.
func (c Context) WithResponseProcessor(processor processor.ResponseProcessor) Context {
	c.ResponseProcessor = processor
	return c
}

// WithRemoteAddr returns a new request Context with the remote address.
func (c Context) WithRemoteAddr(raddr string) Context {
	c.RAddr = raddr
	return c
}

// WithRequestID returns a new request Context with the request id.
func (c Context) WithRequestID(rid string) Context {
	c.RID = rid
	return c
}

// ProcessorContext converts itself to the processor context.
func (c Context) ProcessorContext() processor.Context {
	return processor.NewContext(c.SrcRes, c.SrcReq, c.DstReq)
}

// RequestID returns the request id.
func (c Context) RequestID() string {
	switch {
	case len(c.RID) != 0:
		return c.RID
	case c.SrcReq != nil:
		return c.SrcReq.Header.Get("X-Request-Id")
	default:
		return ""
	}
}

// RemoteAddr returns the address of the client.
func (c Context) RemoteAddr() string {
	switch {
	case len(c.RAddr) != 0:
		return c.RAddr
	case c.SrcReq != nil:
		return c.SrcReq.RemoteAddr
	default:
		return ""
	}
}

// GetRequest returns the request.
func (c Context) GetRequest() *http.Request {
	return c.SrcReq
}

// ------------------------------------------------------------------------- //

// Config is used to configure and new a http endpoint.
type Config struct {
	// Required
	IP   string
	Port uint16

	// Default: 1
	Weight    int
	GetWeight func(endpoint.Endpoint) int

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
	id    string
	conf  atomic.Value
	state endpoint.State
	endpoint.StatusManager
}

func newServer(id string, conf Config) *server {
	s := new(server)
	s.setConf(conf)
	s.SetStatus(endpoint.StatusOnline)
	s.id = id
	return s
}

func (s *server) String() string { return s.id }

func (s *server) getConf() Config        { return s.conf.Load().(Config) }
func (s *server) setConf(c Config) error { s.conf.Store(c); return nil }

func (s *server) ID() string                    { return s.id }
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

func (s *server) Serve(ctx context.Context, req interface{}) (err error) {
	s.state.Inc()
	defer s.state.Dec()

	var c Context
	switch r := req.(type) {
	case Context:
		c = r
	case interface{ GetContext() Context }:
		c = r.GetContext()
	default:
		panic(fmt.Errorf("HttpEndpoint.Serve: unsupported req type '%T'", req))
	}

	c.DstReq.RequestURI = ""
	c.DstReq.URL.Host = s.id

	start := time.Now()
	var statusCode int
	resp, err := s.do(ctx, c.DstReq)
	if resp != nil {
		statusCode = resp.StatusCode
		defer resp.Body.Close()
	}

	if err != nil {
		err = loadbalancer.NewForwardError(err)
	} else if c.ResponseProcessor != nil {
		err = c.ProcessResponse(ctx, c.ProcessorContext(), resp)
	} else {
		err = HandleResponseBody(c.SrcRes, resp)
	}

	cost := time.Since(start)
	if err == nil {
		s.state.IncSuccess()
		if slog.Enabled(ctx, slog.LevelDebug) {
			var srcreq map[string]string
			if c.SrcReq != nil {
				srcreq = map[string]string{
					"raddr":  c.SrcReq.RemoteAddr,
					"method": c.SrcReq.Method,
					"host":   c.SrcReq.Host,
					"path":   c.SrcReq.URL.Path,
					"uri":    c.SrcReq.RequestURI,
				}
			}

			slog.Debug("forward the http request to the backend http endpoint",
				"epid", s.id,
				"reqid", c.RequestID(),
				"srcreq", srcreq,
				"dstreq", map[string]interface{}{
					"method": c.DstReq.Method,
					"host":   c.DstReq.Host,
					"addr":   c.DstReq.URL.Host,
					"path":   c.DstReq.URL.Path,
					"code":   statusCode,
				},
				"start", start.Unix(),
				"cost", cost.String(),
			)
		}
	} else {
		var srcreq map[string]string
		if c.SrcReq != nil {
			srcreq = map[string]string{
				"raddr":  c.SrcReq.RemoteAddr,
				"method": c.SrcReq.Method,
				"host":   c.SrcReq.Host,
				"path":   c.SrcReq.URL.Path,
				"uri":    c.SrcReq.RequestURI,
			}
		}

		slog.Error("forward the http request to the backend http endpoint",
			"epid", s.id,
			"reqid", c.RequestID(),
			"srcreq", srcreq,
			"dstreq", map[string]interface{}{
				"method": c.DstReq.Method,
				"host":   c.DstReq.Host,
				"addr":   c.DstReq.URL.Host,
				"path":   c.DstReq.URL.Path,
				"code":   statusCode,
			},
			"start", start.Unix(),
			"cost", cost.String(),
			"err", err,
		)
	}

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

	case Context:
		return s.checkHTTP(ctx, r.DstReq)

	case nil:
		return s.checkTCP()

	default:
		panic(fmt.Errorf("HttpEndpoint.Check: unsupported req type '%T'", req))
	}
}

func (s *server) checkHTTP(ctx context.Context, req *http.Request) (ok bool) {
	req.RequestURI = ""
	req.URL.Host = s.id
	resp, err := s.do(ctx, req)
	ok = err == nil && resp.StatusCode < 500
	if resp != nil {
		resp.Body.Close()
	}
	return
}

func (s *server) checkTCP() (ok bool) {
	conn, err := net.DialTimeout("tcp", s.id, time.Second)
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

// Define some header keys to be used HandleResponseBody.
var (
	// If not empty, ignore the headers.
	IgnoredRespHeaders = []string{} //{"Host", "Connection"}

	// If not empty, just allow the headers.
	AllowedRespHeaders []string
)

// HandleResponseBody copies the response header and body to writer.
func HandleResponseBody(w http.ResponseWriter, resp *http.Response) error {
	header := w.Header()
	for k, vs := range resp.Header {
		if len(IgnoredRespHeaders) > 0 && slices.Contains(IgnoredRespHeaders, k) {
			continue
		}

		if len(AllowedRespHeaders) > 0 && !slices.Contains(AllowedRespHeaders, k) {
			continue
		}

		header[k] = vs
	}

	w.WriteHeader(resp.StatusCode)
	_, err := io.CopyBuffer(w, resp.Body, make([]byte, 1024))
	return loadbalancer.NewError(false, err)
}
