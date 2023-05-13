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

// ForwardError represents a forward error.
type ForwardError struct{ Err error }

// NewForwardError returns a new forward error.
func NewForwardError(err error) ForwardError { return ForwardError{err} }

// Error implements the error interface.
func (e ForwardError) Error() string { return e.Error() }

// Unwrap returns the inner wrapped error.
func (e ForwardError) Unwrap() error { return e.Err }

// ErrIsForward reports whether the error is a forward error.
func ErrIsForward(err error) bool {
	_, ok := err.(ForwardError)
	return ok
}

// RetryError represents a no-retry error.
type RetryError interface {
	Retry() bool
	error
}

// NewRetryError returns a new retry error.
func NewRetryError(retry bool, err error) RetryError {
	return retryError{retry: retry, err: err}
}

// NewError is the same as NewRetryError, but returns nil instead if err is nil.
func NewError(retry bool, err error) error {
	if err != nil {
		err = NewRetryError(retry, err)
	}
	return err
}

type retryError struct {
	retry bool
	err   error
}

func (e retryError) Unwrap() error { return e.err }
func (e retryError) Retry() bool   { return e.retry }
func (e retryError) Error() string {
	if e.err == nil {
		return "<nil>"
	}
	return e.err.Error()
}
