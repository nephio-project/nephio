/*
 Copyright 2023 The Nephio Authors.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package resource

import (
	"github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

// An ErrorIs function returns true if an error satisfies a particular condition.
type ErrorIs func(err error) bool

// Ignore any errors that satisfy the supplied ErrorIs function by returning
// nil. Errors that do not satisfy the supplied function are returned unmodified.
func Ignore(is ErrorIs, err error) error {
	if is(err) {
		return nil
	}
	return err
}

// IgnoreAny ignores errors that satisfy any of the supplied ErrorIs functions
// by returning nil. Errors that do not satisfy any of the supplied functions
// are returned unmodified.
func IgnoreAny(err error, is ...ErrorIs) error {
	for _, f := range is {
		if f(err) {
			return nil
		}
	}
	return err
}

// IgnoreNotFound returns the supplied error, or nil if the error indicates a
// Kubernetes resource was not found.
func IgnoreNotFound(err error) error {
	return Ignore(kerrors.IsNotFound, err)
}

// IsAPIError returns true if the given error's type is of Kubernetes API error.
func IsAPIError(err error) bool {
	_, ok := err.(kerrors.APIStatus)
	return ok
}

// IsAPIErrorWrapped returns true if err is a K8s API error, or recursively wraps a K8s API error
func IsAPIErrorWrapped(err error) bool {
	return IsAPIError(errors.Cause(err))
}
