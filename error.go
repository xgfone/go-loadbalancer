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

package loadbalancer

// RetryError represents a no-retry error.
type RetryError interface {
	Retry() bool
	error
}

// NewRetryError returns a new retry error.
//
// err may be nil, which is just used to indicate whether to retry.
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
