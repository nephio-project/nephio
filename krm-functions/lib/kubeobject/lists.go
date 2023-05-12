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

package kubeobject

import (
	"fmt"
	"reflect"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// TheScheme holds the mapping between Go types and schema.GroupVersionKinds.
// Types have to be registered in it before they are used with the generic functions below.
// The typical way of registering is by using the AddToScheme function of the package holding the API types.
// E.g.:
//
//	_ = nephioreqv1alpha1.AddToScheme(kubeobject.TheScheme)
var TheScheme *runtime.Scheme = runtime.NewScheme()

// Type constraint for checking if *T implements the runtime.Object interface
type PtrIsRuntimeObject[T any] interface {
	runtime.Object
	*T
}

// GetKindOrPanic returns with the Kind of a Kubernetes API resource type `T`.
// Panics if `T` is not registered in `TheScheme`.
func GetGVKOrPanic[T any, PT PtrIsRuntimeObject[T]]() schema.GroupVersionKind {
	var pt PT = new(T)
	gvks, _, err := TheScheme.ObjectKinds(pt)
	if err != nil || len(gvks) == 0 {
		panic(err)
	}
	return gvks[0]
}

// FilterByType returns the objects in `objs` whose Group-Version-Kind matches with the Go type `T`.
// Panics if `T` is not registered in `TheScheme`.
// FilterByType returns with
//   - the list of matching KubeObjects coneverted to `*T`.
//   - the rest of the `objs` list (KubeObjects that don't match)
//   - a potential error
func FilterByType[T any, PT PtrIsRuntimeObject[T]](objs fn.KubeObjects) ([]*T, fn.KubeObjects, error) {
	result := make([]*T, 0, len(objs))
	var rest fn.KubeObjects
	for _, o := range objs {
		if o.GroupVersionKind() == GetGVKOrPanic[T, PT]() {
			var x T
			err := o.As(&x)
			if err != nil {
				return nil, nil, err
			}
			result = append(result, &x)
		} else {
			rest = append(rest, o)
		}
	}
	return result, rest, nil
}

// GetSingleton returns with the one-and-only resource in `objs` whose Go type is `T`, or an error
// if there is not exactly 1 instance of type `T` is present in `objs`.
// Panics if `T` is not registered in `TheScheme`.
func GetSingleton[T any, PT PtrIsRuntimeObject[T]](objs fn.KubeObjects) (*T, error) {
	typedObjs, _, err := FilterByType[T, PT](objs)
	if err != nil {
		return nil, err
	}
	if len(typedObjs) != 1 {
		var x T
		return nil, fmt.Errorf("expected exactly 1 instance of %v in the kpt package, but got %v", reflect.TypeOf(x).Name(), len(typedObjs))
	}
	return typedObjs[0], nil
}
