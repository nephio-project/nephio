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
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// A ClientApplicator may be used to build a single 'client' that satisfies both
// client.Client and Applicator.
type ClientApplicator struct {
	client.Client
	Applicator
}

// An ApplyFn is a function that satisfies the Applicator interface.
type ApplyFn func(context.Context, client.Object, ...ApplyOption) error

// Apply changes to the supplied object.
func (fn ApplyFn) Apply(ctx context.Context, o client.Object, ao ...ApplyOption) error {
	return fn(ctx, o, ao...)
}

// An Applicator applies changes to an object.
type Applicator interface {
	Apply(context.Context, client.Object, ...ApplyOption) error
}

// An ApplyOption is called before patching the current object to match the
// desired object. ApplyOptions are not called if no current object exists.
type ApplyOption func(ctx context.Context, current, desired runtime.Object) error

// UpdateFn returns an ApplyOption that is used to modify the current object to
// match fields of the desired.
func UpdateFn(fn func(current, desired runtime.Object)) ApplyOption {
	return func(_ context.Context, c, d runtime.Object) error {
		fn(c, d)
		return nil
	}
}
