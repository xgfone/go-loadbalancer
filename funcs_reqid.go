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

package loadbalancer

import (
	"context"
	"net/http"
)

// GetRequestID is used to get the unique request session id.
//
// For the default implementation, it only detects req
// and supports the types or interfaces:
//
//	*http.Request
//	interface{ RequestID() string }
//	interface{ GetRequestID() string }
//	interface{ GetRequest() *http.Request }
//	interface{ GetHTTPRequest() *http.Request }
//
// For *http.Request, it will returns the header "X-Request-Id".
//
// Return "" instead if not found.
var GetRequestID func(ctx context.Context, req interface{}) string = getRequestID

func getRequestID(ctx context.Context, req interface{}) string {
	switch r := req.(type) {
	case *http.Request:
		return r.Header.Get("X-Request-Id")

	case interface{ RequestID() string }:
		return r.RequestID()

	case interface{ GetRequestID() string }:
		return r.GetRequestID()

	case interface{ GetRequest() *http.Request }:
		return r.GetRequest().Header.Get("X-Request-Id")

	case interface{ GetHTTPRequest() *http.Request }:
		return r.GetHTTPRequest().Header.Get("X-Request-Id")

	default:
		return ""
	}
}
