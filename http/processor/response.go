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

package processor

import (
	"io"
	"net/http"
	"slices"
	"sync"

	"github.com/xgfone/go-loadbalancer"
)

// DefaultFilterededHeaders is the filtered headers.
var DefaultFilterededHeaders = []string{"Host", "Connection"}

// HandleResponse copies the response header and body to w.
//
// cb will be called after copying the response header.
func HandleResponse(w http.ResponseWriter, resp *http.Response, cb func(http.ResponseWriter)) error {
	header := w.Header()
	for k, vs := range resp.Header {
		switch {
		case len(vs) == 0:
		case len(vs) == 1 && vs[0] == "":
		case slices.Contains(DefaultFilterededHeaders, k):
		default:
			header[k] = vs
		}
	}
	if cb != nil {
		cb(w)
	}
	return HandleResponseBody(w, resp)
}

// HandleResponseBody copies the response body, not contain header, to w.
func HandleResponseBody(w http.ResponseWriter, resp *http.Response) error {
	w.WriteHeader(resp.StatusCode)
	buf := bspool.Get().(*buffer)
	_, err := io.CopyBuffer(w, resp.Body, buf.data)
	bspool.Put(buf)
	return loadbalancer.NewError(false, err)
}

var bspool = &sync.Pool{New: func() any { return &buffer{make([]byte, 1024)} }}

type buffer struct{ data []byte }
