// Copyright 2026 xgfone
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

// Package selector provides a selector interface and builder, which is used to
// select one of the backend endpoints by the specific policy to handle the request.
package selector

import (
	"context"

	"github.com/xgfone/go-loadbalancer"
)

// Selector is used to select one of the backend endpoints.
type Selector interface {
	Select(ctx context.Context, req any, eps *loadbalancer.Static) (loadbalancer.Endpoint, error)
	Policy() string
}
