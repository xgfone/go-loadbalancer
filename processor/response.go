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
)

func errDefaultHandler(w http.ResponseWriter, err error) (e error) {
	w.WriteHeader(500)
	if h, ok := err.(http.Handler); ok {
		h.ServeHTTP(w, nil)
	} else {
		_, e = io.WriteString(w, err.Error())
	}
	return
}

// DefaultResponseBody returns a response processor to handle the response body,
// which just redirects the response body stream to the response writer.
//
// handleError may be nil, which is equal to send the response with code 500.
func DefaultResponseBody(rewriteContentType bool, handleError func(http.ResponseWriter, error) error) ResponseProcessor {
	if handleError == nil {
		handleError = errDefaultHandler
	}

	return SimpleResponseProcessor(func(w http.ResponseWriter, r *http.Response, err error) error {
		if err != nil {
			err = handleError(w, err)
			return err
		}

		if rewriteContentType {
			w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
		}

		w.WriteHeader(r.StatusCode)
		_, err = io.CopyBuffer(w, r.Body, make([]byte, 1024))
		return err
	})
}
