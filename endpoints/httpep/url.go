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
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/xgfone/go-generics/maps"
)

// URL is the metadata information of the http endpoint.
type URL struct {
	Method   string            `json:"method,omitempty"`   // Such as "GET"
	Scheme   string            `json:"scheme,omitempty"`   // Such as "http" or "https"
	Hostname string            `json:"hostname,omitempty"` // Such as "www.example.com"
	IP       string            `json:"ip,omitempty"`       // Such as "1.2.3.4"
	Port     uint16            `json:"port,omitempty"`     // Such as 80 or 443
	Path     string            `json:"path,omitempty"`     // Such as "/"
	Queries  map[string]string `json:"queries,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
}

func map2urlvalues(ms map[string]string) url.Values {
	if len(ms) == 0 {
		return nil
	}

	values := make(url.Values, len(ms))
	for key, value := range ms {
		values[key] = []string{value}
	}

	return values
}

func map2httpheader(ms map[string]string) http.Header {
	if len(ms) == 0 {
		return nil
	}

	header := make(http.Header, len(ms))
	for key, value := range ms {
		header.Set(key, value)
	}

	return header
}

func encodeJSONBytes(value interface{}) (data []byte, err error) {
	buf := bytes.NewBuffer(make([]byte, 0, 128))
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	err = enc.Encode(value)
	if err == nil {
		data = buf.Bytes()
		if _len := len(data); _len > 0 && data[_len-1] == '\n' {
			data = data[:_len-1]
		}
	}
	return
}

// Clone clones itself and returns a new one.
func (u URL) Clone() URL {
	u.Queries = maps.Clone(u.Queries)
	u.Headers = maps.Clone(u.Headers)
	return u
}

// ID returns the unique identity, for example,
//
//	"http://127.0.0.1/path#md5=21aca36be0bd34307f635553a460db41"
//	"http://www.example.com+127.0.0.1/path#md5=32243ff8dfc9ac922946dcd0a89cc1b9"
func (u URL) ID() string {
	var host string
	if u.Hostname == "" {
		if u.IP != "" {
			host = u.IP
		}
	} else {
		if u.IP == "" {
			host = u.Hostname
		} else {
			host = strings.Join([]string{u.Hostname, u.IP}, "+")
		}
	}

	if u.Port > 0 {
		host = net.JoinHostPort(host, fmt.Sprint(u.Port))
	}

	var fragment string
	if data, _ := encodeJSONBytes(u); len(data) > 0 {
		fragment = fmt.Sprintf("md5=%x", md5.Sum(data))
	}
	_url := url.URL{Scheme: u.Scheme, Host: host, Path: u.Path, Fragment: fragment}
	return _url.String()
}

// Equal reports whether the url is equal to other.
func (u URL) Equal(other URL) bool { return reflect.DeepEqual(u, other) }

// IsZero reports whether the url is ZERO.
func (u URL) IsZero() bool {
	return u.Port == 0 &&
		len(u.IP) == 0 &&
		len(u.Path) == 0 &&
		len(u.Method) == 0 &&
		len(u.Scheme) == 0 &&
		len(u.Hostname) == 0 &&
		len(u.Queries) == 0 &&
		len(u.Headers) == 0
}

// String returns the URL string.
func (u URL) String() string { url := u.URL(); return url.String() }

// URL returns the stdlib url.URL.
func (u URL) URL() url.URL {
	_url := url.URL{Scheme: u.Scheme, Path: u.Path}

	if u.IP != "" {
		if u.Port == 0 {
			_url.Host = u.IP
		} else {
			_url.Host = net.JoinHostPort(u.IP, fmt.Sprint(u.Port))
		}
	} else if u.Hostname != "" {
		if u.Port == 0 {
			_url.Host = u.Hostname
		} else {
			_url.Host = net.JoinHostPort(u.Hostname, fmt.Sprint(u.Port))
		}
	} else {
		panic(fmt.Errorf("no url host: %+v", u))
	}

	_url.RawQuery = map2urlvalues(u.Queries).Encode()
	return _url
}

// Request converts the URL to a http request with the GET method.
func (u URL) Request(ctx context.Context, method string) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, method, u.Default().String(), nil)
}

// Default sets the url information to the default if not set and returns the new.
//
//	Path: "/"
//	Method: "GET"
//	Scheme: "http"
func (u URL) Default() URL {
	if u.Path == "" {
		u.Path = "/"
	}
	if u.Method == "" {
		u.Method = http.MethodGet
	}
	if u.Scheme == "" {
		u.Scheme = "http"
	}
	return u
}
