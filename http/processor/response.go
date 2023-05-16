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

	"github.com/xgfone/go-generics/slices"
	"github.com/xgfone/go-loadbalancer"
)

// DefaultFilterededHeaders is the filtered headers.
var DefaultFilterededHeaders = []string{"Host", "Connection"}

// HandleResponse copies the response header and body to w.
//
// If filteredHeaders is nil, use DefaultFilterededHeaders instead.
// If filteredHeaders is empty, that's []string{}, ignore all the headers.
// Or, only copy the header not in it.
func HandleResponse(w http.ResponseWriter, resp *http.Response, filteredHeaders ...string) error {
	if filteredHeaders == nil {
		filteredHeaders = DefaultFilterededHeaders
	}
	if len(filteredHeaders) > 0 {
		header := w.Header()
		for k, vs := range resp.Header {
			switch {
			case len(vs) == 0:
			case len(vs) == 1 && vs[0] == "":
			case slices.Contains(filteredHeaders, k):
			default:
				header[k] = vs
			}
		}
	}
	return HandleResponseBody(w, resp)
}

// HandleResponseBody copies the response body, not contain header, to w.
func HandleResponseBody(w http.ResponseWriter, resp *http.Response) error {
	w.WriteHeader(resp.StatusCode)
	_, err := io.CopyBuffer(w, resp.Body, make([]byte, 1024))
	return loadbalancer.NewError(false, err)
}
