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
	"time"

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
	HandleRequest  func(*http.Request) (*http.Response, error)
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
	ep := &httpEndpoint{}
	err := ep.Update(c)
	return ep, err
}

type config struct {
	Config

	id   string
	addr string
	host string

	headers  http.Header
	queries  url.Values
	rawQuery string

	getWeight      func(loadbalancer.Endpoint) int
	getStatus      func(loadbalancer.Endpoint) loadbalancer.EndpointStatus
	handleRequest  func(*http.Request) (*http.Response, error)
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

func getStaticStatus(s loadbalancer.Endpoint) loadbalancer.EndpointStatus {
	return loadbalancer.EndpointStatusOnline
}

func handleRequest(req *http.Request) (*http.Response, error) {
	return http.DefaultClient.Do(req)
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
	conf  atomic.Value // config
	epid  string
}

func (ep *httpEndpoint) loadConf() config { return ep.conf.Load().(config) }
func (ep *httpEndpoint) Update(info interface{}) (err error) {
	conf, err := newConfig(info.(Config))
	if err == nil {
		if ep.epid == "" {
			ep.epid = conf.id
		} else if ep.epid != conf.id {
			return fmt.Errorf("the endpoint id is inconsistent: old=%s, new=%s", ep.epid, conf.id)
		}
		ep.conf.Store(conf)
	}
	return
}

func (ep *httpEndpoint) ID() string                          { return ep.epid }
func (ep *httpEndpoint) Type() string                        { return "http" }
func (ep *httpEndpoint) Info() interface{}                   { return ep.loadConf().Config }
func (ep *httpEndpoint) Weight() int                         { return ep.loadConf().getWeight(ep) }
func (ep *httpEndpoint) Status() loadbalancer.EndpointStatus { return ep.loadConf().getStatus(ep) }
func (ep *httpEndpoint) State() loadbalancer.EndpointState   { return ep.state.Clone() }

func (ep *httpEndpoint) Check(ctx context.Context, r interface{}) (ok bool) {
	conf := ep.loadConf()
	req, err := conf.CheckURL.Request(ctx, http.MethodGet)
	if err != nil {
		slog.Error("fail to new the check request", "epid", ep.ID(), "err", err)
		return false
	}

	resp, err := conf.handleRequest(req)
	if resp != nil {
		io.CopyBuffer(io.Discard, resp.Body, make([]byte, 256))
		resp.Body.Close()
	}
	if err != nil {
		slog.Error("fail to check the endpoint", "epid", ep.ID(), "err", err)
		return false
	}
	return resp.StatusCode < 400
}

func (ep *httpEndpoint) Serve(ctx context.Context, _req interface{}) (err error) {
	start := time.Now()

	w, req, ok := GetReqRespFromCtx(ctx)
	if !ok {
		w, req, _ = GetReqRespFromCtx(_req.(*http.Request).Context())
	}

	ep.state.Inc()
	defer func() {
		ep.state.Dec()
		if err == nil {
			ep.state.IncSuccess()
		}
	}()

	conf := ep.loadConf()

	r := req.Clone(ctx)
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

	resp, err := conf.handleRequest(r)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err == nil {
		err = conf.handleResponse(w, resp)
	}

	cost := time.Since(start)
	if err != nil {
		slog.Error("forward the http request to the backend http endpoint",
			"epid", ep.ID(),
			"reqid", defaults.GetRequestID(ctx, req),
			"srcreq", map[string]interface{}{
				"raddr":  req.RemoteAddr,
				"method": req.Method,
				"host":   req.Host,
				"uri":    req.RequestURI,
			},
			"dstreq", map[string]interface{}{
				"method": r.Method,
				"host":   r.Host,
				"uri":    r.URL.String(),
			},
			"start", start.Unix(),
			"cost", cost.String(),
			"err", err,
		)
	} else if slog.Enabled(ctx, slog.LevelDebug) {
		slog.Debug("forward the http request to the backend http endpoint",
			"epid", ep.ID(),
			"reqid", defaults.GetRequestID(ctx, req),
			"srcreq", map[string]interface{}{
				"raddr":  req.RemoteAddr,
				"method": req.Method,
				"host":   req.Host,
				"uri":    req.RequestURI,
			},
			"dstreq", map[string]interface{}{
				"method": r.Method,
				"host":   r.Host,
				"uri":    r.URL.String(),
			},
			"start", start.Unix(),
			"cost", cost.String(),
		)
	}

	return err
}
