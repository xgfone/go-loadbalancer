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

package httpep

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func ExampleURL_ID() {
	var url URL

	url = URL{Scheme: "http", IP: "127.0.0.1", Path: "/path"}
	fmt.Println(url.ID())

	url = URL{Scheme: "http", IP: "127.0.0.1", Port: 80, Path: "/path"}
	fmt.Println(url.ID())

	url = URL{Scheme: "http", Hostname: "www.example.com", Path: "/path"}
	fmt.Println(url.ID())

	url = URL{Scheme: "http", Hostname: "www.example.com", Port: 80, Path: "/path"}
	fmt.Println(url.ID())

	url = URL{Scheme: "http", Hostname: "www.example.com", IP: "127.0.0.1", Path: "/path"}
	fmt.Println(url.ID())

	url = URL{Scheme: "http", Hostname: "www.example.com", IP: "127.0.0.1", Port: 80, Path: "/path"}
	fmt.Println(url.ID())

	// Output:
	// http://127.0.0.1/path#md5=21aca36be0bd34307f635553a460db41
	// http://127.0.0.1:80/path#md5=3da30ab9783aad141993ce4e2940608a
	// http://www.example.com/path#md5=8aa32ab56942b28249eaf6e06ecb3d08
	// http://www.example.com:80/path#md5=1c622fa8baecdf9570ecb95e89249f02
	// http://www.example.com+127.0.0.1/path#md5=32243ff8dfc9ac922946dcd0a89cc1b9
	// http://www.example.com+127.0.0.1:80/path#md5=b4729cc202e4b573fd33563c4c496adc
}

func TestURL_Request(t *testing.T) {
	url := URL{IP: "127.0.0.1", Port: 8200}
	req, err := url.Request(context.Background(), http.MethodGet)
	if err != nil {
		t.Fatal(err)
	}

	handler := http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(201)
	})

	go http.ListenAndServe("127.0.0.1:8200", handler)
	time.Sleep(time.Millisecond * 100)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 201 {
		t.Errorf("expect status code '%d', but got '%d'", 201, resp.StatusCode)
	}
}
