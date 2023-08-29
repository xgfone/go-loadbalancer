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

// Package loadbalancer provides some common functions.
package loadbalancer

import "errors"

// ErrNoAvailableEndpoints is used to represents no available endpoints.
var ErrNoAvailableEndpoints = errors.New("no available endpoints")

// RetryError represents a retry error.
type RetryError interface {
	Retry() bool
	error
}

// NewRetryError returns a new retry error, but returns nil instead if err is nil.
func NewRetryError(retry bool, err error) error {
	if err == nil {
		return nil
	}
	return retryError{retry: retry, err: err}
}

type retryError struct {
	retry bool
	err   error
}

func (e retryError) Retry() bool   { return e.retry }
func (e retryError) Error() string { return e.err.Error() }
func (e retryError) Unwrap() error { return e.err }
