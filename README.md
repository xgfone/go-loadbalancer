# Go LoadBalancer [![Build Status](https://github.com/xgfone/go-loadbalancer/actions/workflows/go.yml/badge.svg)](https://github.com/xgfone/go-loadbalancer/actions/workflows/go.yml) [![GoDoc](https://pkg.go.dev/badge/github.com/xgfone/go-loadbalancer)](https://pkg.go.dev/github.com/xgfone/go-loadbalancer) [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat-square)](https://raw.githubusercontent.com/xgfone/go-loadbalancer/master/LICENSE)

Require Go `1.21+`.

## Install
```shell
$ go get -u github.com/xgfone/go-loadbalancer
```

## Example

### Mini API Gateway
```go
package main

import (
	"encoding/json"
	"flag"
	"net/http"

	"github.com/xgfone/go-loadbalancer/balancer"
	"github.com/xgfone/go-loadbalancer/endpoint"
	"github.com/xgfone/go-loadbalancer/forwarder"
	"github.com/xgfone/go-loadbalancer/healthcheck"
	httpep "github.com/xgfone/go-loadbalancer/http/endpoint"
)

var listenAddr = flag.String("listenaddr", ":80", "The address that api gateway listens on.")

func main() {
	flag.Parse()

	healthcheck.DefaultHealthChecker.Start()
	defer healthcheck.DefaultHealthChecker.Stop()

	http.HandleFunc("/admin/route", registerRouteHandler)
	http.ListenAndServe(*listenAddr, nil)
}

func registerRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		// Route Matcher
		Path   string `json:"path" validate:"required"`
		Method string `json:"method" validate:"required"`

		// Upstream Endpoints
		Upstream struct {
			ForwardPolicy string              `json:"forwardPolicy" default:"weight_random"`
			HealthCheck   healthcheck.Checker `json:"healthCheck"`
			Servers       []struct {
				Host   string `json:"host" validate:"host"`
				Port   uint16 `json:"port" validate:"ranger(1,65535)"`
				Weight int    `json:"weight" default:"1" validate:"min(1)"`
			} `json:"servers"`
		} `json:"upstream"`
	}

	// Notice: here we don't validate whether the values are valid.
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request route paramenter: "+err.Error(), 400)
		return
	}

	// Build the upstream backend servers.
	endpoints := make(endpoint.Endpoints, len(req.Upstream.Servers))
	for i, server := range req.Upstream.Servers {
		endpoints[i] = httpep.Config{
			Host:   server.Host,
			Port:   server.Port,
			Weight: server.Weight,
		}.NewEndpoint()
	}

	// Build the loadbalancer forwarder.
	balancer, _ := balancer.Build(req.Upstream.ForwardPolicy, nil)
	forwarder := forwarder.NewForwarder(req.Method+"@"+req.Path, balancer)

	healthcheck.DefaultHealthChecker.AddUpdater(forwarder.Name(), forwarder)
	healthcheck.DefaultHealthChecker.UpsertEndpoints(endpoints, req.Upstream.HealthCheck)

	// Register the route and forward the request to forwarder.
	http.HandleFunc(req.Path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != req.Method {
			w.WriteHeader(http.StatusMethodNotAllowed)
		} else {
			// You can use forwarder.ForwardHTTP to control the request and response.
			forwarder.ServeHTTP(w, r)
		}
	})
}
```

```shell
# Run the mini API-Gateway on the host 192.168.1.10
$ nohup go run main.go &

# Add the route
# Notice: remove the characters from // to the line end.
$ curl -XPOST http://127.0.0.1/admin/route -H 'Content-Type: application/json' -d '
{
    "path": "/path",
    "method": "GET",
    "upstream": {
        "forwardPolicy": "weight_round_robin",
        "servers": [
            {"ip": "192.168.1.11", "port": 80, "weight": 10}, // 33.3% requests
            {"ip": "192.168.1.12", "port": 80, "weight": 20}  // 66.7% requests
        ]
    }
}'

# Access the backend servers by the mini API-Gateway:
# 2/6(33.3%) requests -> 192.168.1.11
# 4/6(66.7%) requests -> 192.168.1.12
$ curl http://192.168.1.10/path
192.168.1.11/backend/path

$ curl http://192.168.1.10/path
192.168.1.12/backend/path

$ curl http://192.168.1.10/path
192.168.1.12/backend/path

$ curl http://192.168.1.10/path
192.168.1.11/backend/path

$ curl http://192.168.1.10/path
192.168.1.12/backend/path

$ curl http://192.168.1.10/path
192.168.1.12/backend/path
```
