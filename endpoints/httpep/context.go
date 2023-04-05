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
	"net/http"
)

var (
	// GetReqRespFromCtx is used to get the http request and response from the context.
	GetReqRespFromCtx func(context.Context) (w http.ResponseWriter, r *http.Request, ok bool) = getReqRespFromCtx

	// SetReqRespIntoCtx is used to store the http request and response into the context.
	SetReqRespIntoCtx func(context.Context, http.ResponseWriter, *http.Request) context.Context = setReqRespIntoCtx
)

type reqrespkey uint8
type reqrespctx struct {
	res http.ResponseWriter
	req *http.Request
}

func setReqRespIntoCtx(ctx context.Context, w http.ResponseWriter, r *http.Request) context.Context {
	return context.WithValue(ctx, reqrespkey(255), reqrespctx{w, r})
}

func getReqRespFromCtx(ctx context.Context) (http.ResponseWriter, *http.Request, bool) {
	c, ok := ctx.Value(reqrespkey(255)).(reqrespctx)
	return c.res, c.req, ok
}
