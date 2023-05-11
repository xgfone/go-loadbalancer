// Copyright 2021~2023 xgfone
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

package forwarder

import (
	"context"
	"net/http"

	"github.com/xgfone/go-defaults"
	"github.com/xgfone/go-loadbalancer"
	"github.com/xgfone/go-loadbalancer/http/endpoint"
	"github.com/xgfone/go-loadbalancer/internal/nets"
	"github.com/xgfone/go-loadbalancer/internal/slog"
)

// ServeHTTP implements the interface http.Handler,
// which will use http/endpoint.SetReqRespIntoCtx to store w and r
// into the context, then call the Serve method.
//
// NOTICE: YOU SHOULD IMPLEMENT YOURSELF ServeHTTP.
func (f *Forwarder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := endpoint.SetReqRespIntoCtx(r.Context(), w, r)
	f.handleHTTPError(ctx, w, r, f.Serve(ctx, endpoint.NewRequest(w, nil, r)))
}

func (f *Forwarder) handleHTTPError(ctx context.Context,
	w http.ResponseWriter, r *http.Request, err error) {
	switch err {
	case nil:
		if slog.Enabled(r.Context(), slog.LevelDebug) {
			slog.Debug("forward the http request",
				"reqid", defaults.GetRequestID(r.Context(), r),
				"forwarder", f.name,
				"balancer", f.GetBalancer().Policy(),
				"raddr", r.RemoteAddr,
				"method", r.Method,
				"host", r.Host,
				"uri", r.RequestURI)
		}
		return

	case loadbalancer.ErrNoAvailableEndpoints:
		w.WriteHeader(503) // Service Unavailable

	default:
		if nets.IsTimeout(err) {
			w.WriteHeader(504) // Gateway Timeout
		} else {
			w.WriteHeader(502) // Bad Gateway
		}
	}

	slog.Error("fail to forward the http request",
		"reqid", defaults.GetRequestID(r.Context(), r),
		"forwarder", f.name,
		"balancer", f.GetBalancer().Policy(),
		"raddr", r.RemoteAddr,
		"method", r.Method,
		"host", r.Host,
		"uri", r.RequestURI,
		"err", err)
}
