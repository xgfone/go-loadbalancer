// Copyright 2021 xgfone
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

// +build !go1.13

package loadbalancer

import (
	"context"
	"io"
	"net/http"
)

// newHTTPRequestWithContext is the compatibility of http.NewRequestWithContext.
func newHTTPRequestWithContext(ctx context.Context, method, url string,
	body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err == nil {
		req = req.WithContext(ctx)
	}
	return req, err
}
