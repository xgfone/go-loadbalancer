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

package loadbalancer

import (
	"context"
	"sort"
)

// Request represents a request.
type Request interface {
	// RemoteAddrString returns the address string of the remote peer,
	// that's, the sender of the current request.
	//
	// Notice: it maybe return "" to represent that it is the originator
	// and not the remote peer.
	RemoteAddrString() string

	// SessionID returns the session id of the request context, which is used
	// to bind the requests with the same session id to the same endpoint
	// to be handled, when enabling the session stick.
	//
	// Notice: it maybe return an empty string, but use RemoteAddrString instead.
	SessionID() string
}

// EndpointState represents the running state of the endpoint.
type EndpointState struct {
	TotalConnections   int64 // The total number of all the connections.
	CurrentConnections int64 // The number of all the current connections.

	// Extension is the extension data which is contained by the implementation
	// but EndpointState.
	Extension interface{}
}

// Endpoint represents a service endpoint.
type Endpoint interface {
	// ID returns the unique id of the endpoint, which may be an address or url.
	ID() string

	// Type returns the type of the endpoint, such as "http", "tcp", etc.
	Type() string

	// State returns the state of the current endpoint.
	State() EndpointState

	// MetaData returns the metadata of the endpoint.
	MetaData() map[string]interface{}

	// RoundTrip sends the request to the current endpoint.
	RoundTrip(c context.Context, req Request) (response interface{}, err error)
}

// EndpointUpdater is used to add or delete the endpoint.
type EndpointUpdater interface {
	DelEndpointByID(id string)
	DelEndpoint(Endpoint)
	AddEndpoint(Endpoint)
}

// EndpointBatchUpdater is used to add or delete the endpoints in bulk.
type EndpointBatchUpdater interface {
	AddEndpoints([]Endpoint)
	DelEndpoints([]Endpoint)
}

// Endpoints is a set of Endpoint.
type Endpoints []Endpoint

// Sort sorts the endpoints.
func (es Endpoints) Sort()              { sort.Stable(es) }
func (es Endpoints) Len() int           { return len(es) }
func (es Endpoints) Swap(i, j int)      { es[i], es[j] = es[j], es[i] }
func (es Endpoints) Less(i, j int) bool { return es[i].ID() < es[j].ID() }

// Contains reports whether the endpoints contains the endpoint.
func (es Endpoints) Contains(endpoint Endpoint) bool {
	return binarySearchEndpoints(es, endpoint.ID()) != -1
}

// NotContains reports whether the endpoints does not contain the endpoint.
func (es Endpoints) NotContains(endpoint Endpoint) bool {
	return binarySearchEndpoints(es, endpoint.ID()) == -1
}

func binarySearchEndpoints(eps Endpoints, id string) int {
	for low, high := 0, len(eps)-1; low <= high; {
		mid := (low + high)
		if _id := eps[mid].ID(); _id == id {
			return mid
		} else if id < _id {
			high = mid - 1
		} else {
			low = mid + 1
		}
	}

	return -1
}

// WeightEndpoint represents an endpoint with the weight.
type WeightEndpoint interface {
	Endpoint

	// Weight returns the weight of the endpoint, which may be equal to 0,
	// but not the negative.
	//
	// The larger the weight, the higher the weight.
	Weight() int
}

// NewWeightEndpoint returns a WeightEndpoint with the weight and the endpoint.
func NewWeightEndpoint(endpoint Endpoint, weight int) WeightEndpoint {
	return NewDynamicWeightEndpoint(endpoint, func(Endpoint) int { return weight })
}

// NewDynamicWeightEndpoint returns a new WeightEndpoint with the endpoint and
// the weigthFunc that returns the weight of the endpoint.
func NewDynamicWeightEndpoint(endpoint Endpoint, weightFunc func(Endpoint) int) WeightEndpoint {
	return weightEndpoint{Endpoint: endpoint, weight: weightFunc}
}

type weightEndpoint struct {
	Endpoint
	weight func(Endpoint) int
}

func (we weightEndpoint) Weight() int              { return we.weight(we.Endpoint) }
func (we weightEndpoint) UnwrapEndpoint() Endpoint { return we.Endpoint }
func (we weightEndpoint) MetaData() map[string]interface{} {
	md := we.Endpoint.MetaData()
	md["weight"] = we.weight(we.Endpoint)
	return md
}

// EndpointUnwrap is used to unwrap the inner endpoint.
type EndpointUnwrap interface {
	// Unwrap unwraps the inner endpoint, or nil instead if no inner endpoint.
	UnwrapEndpoint() Endpoint
}

// UnwrapEndpoint unwraps the endpoint until it has not implemented
// the interface EndpointUnwrap.
func UnwrapEndpoint(endpoint Endpoint) Endpoint {
	for {
		if eu, ok := endpoint.(EndpointUnwrap); ok {
			if ep := eu.UnwrapEndpoint(); ep != nil {
				endpoint = ep
			} else {
				break
			}
		} else {
			break
		}
	}
	return endpoint
}
