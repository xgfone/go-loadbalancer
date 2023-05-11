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

package upstream

import (
	"context"
	"net/http"

	"github.com/xgfone/go-loadbalancer/http/endpoint"
	"github.com/xgfone/go-loadbalancer/http/processor"
)

// ForwardHTTP forwards the http request to one of the backend endpoints,
// which uses http/endpoint.Request as the request context.
func (up *Upstream) ForwardHTTP(ctx context.Context,
	srcrw http.ResponseWriter, srcreq *http.Request,
	dstreq *http.Request, respBodyHandler processor.ResponseProcessor) error {
	return up.forwarder.Serve(ctx, endpoint.Request{
		RespBodyProcessor: respBodyHandler,

		SrcRes: srcrw,
		SrcReq: srcreq,
		DstReq: dstreq,
	})
}
