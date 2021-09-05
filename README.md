# LoadBalancer [![Build Status](https://github.com/xgfone/go-loadbalancer/actions/workflows/go.yml/badge.svg)](https://github.com/xgfone/go-loadbalancer/actions/workflows/go.yml) [![GoDoc](https://pkg.go.dev/badge/github.com/xgfone/go-loadbalancer)](https://pkg.go.dev/github.com/xgfone/go-loadbalancer) [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat-square)](https://raw.githubusercontent.com/xgfone/go-loadbalancer/master/LICENSE)

A set of the loadbalancer functions supporting `Go1.7+`.

## Installation
```shell
$ go get -u github.com/xgfone/go-loadbalancer
```

## Example

### `Client` Mode
```go
package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/xgfone/go-loadbalancer"
)

func roundTripp(lb *loadbalancer.LoadBalancer, host string) http.RoundTripper {
	return &loadbalancer.HTTPRoundTripper{
		GetRoundTripper: func(r *http.Request) (rt loadbalancer.RoundTripper) {
			if r.Host == host {
				return lb
			}
			return nil
		},
	}
}

func printResponse(resp *http.Response, err error) {
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("URL: %s\n", resp.Request.URL.String())

	buf := bytes.NewBuffer(nil)
	io.CopyN(buf, resp.Body, resp.ContentLength)
	resp.Body.Close()

	fmt.Println("StatusCode:", resp.StatusCode)
	fmt.Println("Body:", buf.String())
}

func main() {
	lb := loadbalancer.NewLoadBalancer("default", nil)
	defer lb.Close()

	hc := loadbalancer.NewHealthCheck()
	hc.AddUpdater(lb.Name(), lb)
	defer hc.Stop()

	http.DefaultClient.Transport = roundTripp(lb, "127.0.0.1:80")
	ep1, _ := loadbalancer.NewHTTPEndpoint("192.168.1.1", nil)
	ep2, _ := loadbalancer.NewHTTPEndpoint("192.168.1.2", nil)
	ep3, _ := loadbalancer.NewHTTPEndpoint("192.168.1.3", nil)
	duration := loadbalancer.EndpointCheckerDuration{Interval: time.Second * 10}
	hc.AddEndpoint(ep1, loadbalancer.NewHTTPEndpointHealthChecker(ep1.ID()), duration)
	hc.AddEndpoint(ep2, loadbalancer.NewHTTPEndpointHealthChecker(ep2.ID()), duration)
	hc.AddEndpoint(ep3, loadbalancer.NewHTTPEndpointHealthChecker(ep3.ID()), duration)

	// Wait to check the health status of all end endpoints.
	time.Sleep(time.Second)

	// 127.0.0.1:80 will be replaced with one of 192.168.1.1:80, 192.168.1.2:80, 192.168.1.3:80.
	resp, err := http.Get("http://127.0.0.1:80")
	printResponse(resp, err)

	// 127.0.0.1:8000 won't be replaced, and it will send the request to 127.0.0.1:8000 directly.
	resp, err = http.Get("http://127.0.0.1:8000")
	printResponse(resp, err)
}
```

### `Proxy` Mode
```go
package main

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/xgfone/go-loadbalancer"
)

func proxyHandler(lb *loadbalancer.LoadBalancer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: Add other headers
		resp, err := lb.RoundTrip(context.Background(), loadbalancer.NewHTTPRequest(r, r.Header.Get("SessionID")))
		if err != nil {
			w.WriteHeader(502)
			w.Write([]byte(err.Error()))
			return
		}

		hresp := resp.(*http.Response)
		for key, value := range hresp.Header {
			w.Header()[key] = value
		}
		// TODO: Add and fix the response headers

		w.WriteHeader(hresp.StatusCode)
		if hresp.ContentLength > 0 {
			io.CopyBuffer(w, hresp.Body, make([]byte, 1024))
		}
	})
}

func main() {
	lb := loadbalancer.NewLoadBalancer("default", nil)
	defer lb.Close()

	hc := loadbalancer.NewHealthCheck()
	hc.AddUpdater(lb.Name(), lb)
	defer hc.Stop()

	ep1, _ := loadbalancer.NewHTTPEndpoint("192.168.1.1", nil)
	ep2, _ := loadbalancer.NewHTTPEndpoint("192.168.1.2", nil)
	ep3, _ := loadbalancer.NewHTTPEndpoint("192.168.1.3", nil)
	duration := loadbalancer.EndpointCheckerDuration{Interval: time.Second * 10}
	hc.AddEndpoint(ep1, loadbalancer.NewHTTPEndpointHealthChecker(ep1.ID()), duration)
	hc.AddEndpoint(ep2, loadbalancer.NewHTTPEndpointHealthChecker(ep2.ID()), duration)
	hc.AddEndpoint(ep3, loadbalancer.NewHTTPEndpointHealthChecker(ep3.ID()), duration)

	http.ListenAndServe(":80", proxyHandler(lb))
}
```
