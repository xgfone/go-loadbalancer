# Go LoadBalancer [![Build Status](https://github.com/xgfone/go-loadbalancer/actions/workflows/go.yml/badge.svg)](https://github.com/xgfone/go-loadbalancer/actions/workflows/go.yml) [![GoDoc](https://pkg.go.dev/badge/github.com/xgfone/go-loadbalancer)](https://pkg.go.dev/github.com/xgfone/go-loadbalancer) [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat-square)](https://raw.githubusercontent.com/xgfone/go-loadbalancer/master/LICENSE)


## Install
```shell
$ go get -u github.com/xgfone/go-loadbalancer
```

## Example

### Mini API Gateway
```go
package main

import (
	"flag"
	"net/http"

	"github.com/xgfone/go-binder"
	"github.com/xgfone/go-loadbalancer"
	"github.com/xgfone/go-loadbalancer/balancer"
	"github.com/xgfone/go-loadbalancer/endpoints/httpep"
	"github.com/xgfone/go-loadbalancer/forwarder"
	"github.com/xgfone/go-loadbalancer/healthcheck"
)

var listenAddr = flag.String("listen-addr", ":80", "The address that api gateway listens on.")

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
		Path     string `json:"path" validate:"required"`
		Method   string `json:"method" validate:"required"`
		Upstream struct {
			ForwardPolicy string                  `json:"forwardPolicy" default:"weight_random"`
			ForwardURL    httpep.URL              `json:"forwardUrl"`
			HealthCheck   healthcheck.CheckConfig `json:"healthCheck"`

			Servers []struct {
				IP     string `json:"ip" validate:"ip"`
				Port   uint16 `json:"port" validate:"ranger(1,65535)"`
				Weight int    `json:"weight" default:"1" validate:"min(1)"`
			} `json:"servers"`
		} `json:"upstream"`
	}

	if err := binder.BodyDecoder.Decode(&req, r.Body); err != nil {
		http.Error(w, "invalid request route paramenter: "+err.Error(), 400)
		return
	}

	// Build the upstream backend servers.
	endpoints := make(loadbalancer.Endpoints, len(req.Upstream.Servers))
	for i, server := range req.Upstream.Servers {
		config := httpep.Config{URL: req.Upstream.ForwardURL}
		config.StaticWeight = server.Weight
		config.URL.Port = server.Port
		config.URL.IP = server.IP

		endpoint, err := config.NewEndpoint()
		if err != nil {
			http.Error(w, "fail to build the upstream server: "+err.Error(), 400)
			return
		}

		endpoints[i] = endpoint
	}

	// Build the loadbalancer forwarder.
	balancer, _ := balancer.Build(req.Upstream.ForwardPolicy, nil)
	forwarder := forwarder.NewForwarder(req.Method+""+req.Path, balancer)

	healthcheck.DefaultHealthChecker.AddUpdater(forwarder.Name(), forwarder)
	healthcheck.DefaultHealthChecker.UpsertEndpoints(endpoints, req.Upstream.HealthCheck)

	// Register the route and forward the request to forwarder.
	http.HandleFunc(req.Path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != req.Method {
			w.WriteHeader(http.StatusMethodNotAllowed)
		} else {
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
    "rule": "Method(`GET`) && Path(`/path`)",
    "upstream": {
        "forwardPolicy": "weight_round_robin",
        "forwardUrl" : {"path": "/backend/path"},
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
