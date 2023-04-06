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

package httpep

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"

	"github.com/xgfone/go-defaults"
	"github.com/xgfone/go-loadbalancer"
	"github.com/xgfone/go-loadbalancer/internal/slog"
)

// Config is used to configure the http backend endpoint.
type Config struct {
	// Hostname or IP is required, and others are optional.
	URL URL

	// CheckURL is used to check the http backend endpoint is healthy.
	//
	// Optional
	CheckURL URL

	// Optional
	//
	// Default: URL.ID()
	ID string

	// The extra information about the http backend endpoint.
	//
	// Optional
	Info interface{}

	// Handle the request or response.
	//
	// Optional
	GetHTTPClient  func(*http.Request) *http.Client // Default: use http.DefaultClient
	HandleRequest  func(*http.Client, *http.Request) (*http.Response, error)
	HandleResponse func(http.ResponseWriter, *http.Response) error

	// If DynamicWeight is configured, use it. Or use StaticWeight.
	//
	// Optional
	DynamicWeight func(loadbalancer.Endpoint) int
	StaticWeight  int // Default: 1

	// GetEndpointStatus is used to check the endpoint status dynamically.
	//
	// If nil, use EndpointStatusOnline instead.
	GetEndpointStatus func(loadbalancer.Endpoint) loadbalancer.EndpointStatus
}

// NewEndpoint returns a new loadbalancer.WeightedEndpoint based on HTTP.
func (c Config) NewEndpoint() (loadbalancer.WeightedEndpoint, error) {
	s := &httpEndpoint{}
	err := s.Update(c)
	return s, err
}

type config struct {
	Config

	id   string
	addr string
	host string

	headers  http.Header
	queries  url.Values
	rawQuery string

	getClient      func(*http.Request) *http.Client
	getWeight      func(loadbalancer.Endpoint) int
	getStatus      func(loadbalancer.Endpoint) loadbalancer.EndpointStatus
	handleRequest  func(*http.Client, *http.Request) (*http.Response, error)
	handleResponse func(http.ResponseWriter, *http.Response) error
}

func newConfig(c Config) (conf config, err error) {
	conf.Config = c
	err = conf.init()
	return
}

func (c *config) init() error {
	c.URL = c.URL.Clone()
	c.CheckURL = c.CheckURL.Clone()

	if c.URL.Hostname == "" && c.URL.IP == "" {
		return fmt.Errorf("missing the hostname and ip")
	}
	if c.URL.Scheme == "" {
		c.URL.Scheme = "http"
	}
	if c.StaticWeight <= 0 {
		c.StaticWeight = 1
	}

	if c.URL.Port > 0 {
		if c.URL.IP != "" {
			c.addr = net.JoinHostPort(c.URL.IP, fmt.Sprint(c.URL.Port))
		} else {
			c.addr = net.JoinHostPort(c.URL.Hostname, fmt.Sprint(c.URL.Port))
		}
	} else {
		if c.URL.IP != "" {
			c.addr = c.URL.IP
		} else {
			c.addr = c.URL.Hostname
		}
	}

	c.id = c.ID
	if c.id == "" {
		c.id = c.URL.ID()
	}

	c.host = c.URL.Hostname
	if c.host != "" && c.URL.Port > 0 {
		c.host = net.JoinHostPort(c.URL.Hostname, strconv.FormatInt(int64(c.URL.Port), 10))
	}

	c.headers = map2httpheader(c.URL.Headers)
	c.queries = map2urlvalues(c.URL.Queries)
	c.rawQuery = c.queries.Encode()

	c.getWeight = c.DynamicWeight
	c.getStatus = c.GetEndpointStatus
	c.handleRequest = c.HandleRequest
	c.handleResponse = c.HandleResponse
	if c.getWeight == nil {
		c.getWeight = c.getStaticWeight
	}
	if c.getStatus == nil {
		c.getStatus = getStaticStatus
	}
	if c.handleRequest == nil {
		c.handleRequest = handleRequest
	}
	if c.handleResponse == nil {
		c.handleResponse = handleResponse
	}
	if c.GetHTTPClient == nil {
		c.getClient = getHTTPClient
	} else {
		c.getClient = c.GetHTTPClient
	}

	if c.CheckURL.Method == "" {
		c.CheckURL.Method = http.MethodGet
	}
	if c.CheckURL.Scheme == "" {
		c.CheckURL.Scheme = c.URL.Scheme
	}
	if c.CheckURL.Hostname == "" {
		c.CheckURL.Hostname = c.URL.Hostname
	}
	c.CheckURL.IP = c.URL.IP
	if c.CheckURL.Port == 0 {
		c.CheckURL.Port = c.URL.Port
	}

	return nil
}

func (c config) getStaticWeight(loadbalancer.Endpoint) int {
	return c.StaticWeight
}

func getHTTPClient(*http.Request) *http.Client {
	return http.DefaultClient
}

func getStaticStatus(s loadbalancer.Endpoint) loadbalancer.EndpointStatus {
	return loadbalancer.EndpointStatusOnline
}

func handleRequest(client *http.Client, req *http.Request) (*http.Response, error) {
	resp, err := client.Do(req)
	if slog.Enabled(req.Context(), slog.LevelTrace) {
		slog.Trace("forward the http request",
			"requestid", defaults.GetRequestID(req.Context(), req),
			"method", req.Method, "host", req.Host, "url", req.URL.String(),
			"err", err)
	}
	return resp, err
}

func handleResponse(w http.ResponseWriter, resp *http.Response) error {
	respHeader := w.Header()
	for k, vs := range resp.Header {
		respHeader[k] = vs
	}

	w.WriteHeader(resp.StatusCode)
	_, err := io.CopyBuffer(w, resp.Body, make([]byte, 1024))
	return err
}

type httpEndpoint struct {
	state loadbalancer.EndpointState
	conf  atomic.Value
}

func (s *httpEndpoint) loadConf() config { return s.conf.Load().(config) }
func (s *httpEndpoint) Update(info interface{}) (err error) {
	conf, err := newConfig(info.(Config))
	if err == nil {
		s.conf.Store(conf)
	}
	return
}

func (s *httpEndpoint) ID() string                          { return s.loadConf().id }
func (s *httpEndpoint) Type() string                        { return "http" }
func (s *httpEndpoint) Info() interface{}                   { return s.loadConf().Config }
func (s *httpEndpoint) Weight() int                         { return s.loadConf().getWeight(s) }
func (s *httpEndpoint) Status() loadbalancer.EndpointStatus { return s.loadConf().getStatus(s) }
func (s *httpEndpoint) State() loadbalancer.EndpointState   { return s.state.Clone() }
func (s *httpEndpoint) Check(ctx context.Context) (ok bool) {
	conf := s.loadConf()
	req, err := conf.CheckURL.Request(ctx, http.MethodGet)
	if err != nil {
		slog.Error("fail to new the check request", "epid", s.ID(), "err", err)
		return false
	}

	resp, err := conf.getClient(req).Do(req)
	if resp != nil {
		io.CopyBuffer(io.Discard, resp.Body, make([]byte, 256))
		resp.Body.Close()
	}
	if err != nil {
		slog.Error("fail to check the endpoint", "epid", s.ID(), "err", err)
		return false
	}
	return resp.StatusCode < 400
}

func (s *httpEndpoint) Serve(ctx context.Context, req interface{}) (err error) {
	w, r, ok := GetReqRespFromCtx(ctx)
	if !ok {
		w, r, _ = GetReqRespFromCtx(req.(*http.Request).Context())
	}

	s.state.Inc()
	defer func() {
		s.state.Dec()
		if err == nil {
			s.state.IncSuccess()
		}
	}()

	conf := s.loadConf()

	r = r.Clone(ctx)
	r.RequestURI = ""      // Pretend to be a client request.
	r.URL.Host = conf.addr // Dial to the backend http endpoint.
	r.URL.Scheme = conf.URL.Scheme

	if conf.URL.Method != "" {
		r.Method = conf.URL.Method
	}

	if conf.host != "" {
		r.Host = conf.host // Override the header "Host"
	}

	if conf.URL.Path != "" {
		r.URL.RawPath = ""
		r.URL.Path = conf.URL.Path
	}

	if len(conf.queries) > 0 {
		if r.URL.RawQuery == "" {
			r.URL.RawQuery = conf.rawQuery
		} else if values := r.URL.Query(); len(values) == 0 {
			r.URL.RawQuery = conf.rawQuery
		} else {
			for k, vs := range conf.queries {
				values[k] = vs
			}
			r.URL.RawQuery = values.Encode()
		}
	}

	if len(conf.headers) > 0 {
		if len(r.Header) == 0 {
			r.Header = conf.headers.Clone()
		} else {
			for k, vs := range conf.headers {
				r.Header[k] = vs
			}
		}
	}

	resp, err := conf.handleRequest(conf.getClient(r), r)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err == nil {
		err = conf.handleResponse(w, resp)
	}
	return err
}
