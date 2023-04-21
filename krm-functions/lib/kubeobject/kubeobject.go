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

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
)

type KubeObjectExt[T1 any] struct {
	fn.KubeObject
}

func (r *KubeObjectExt[T1]) GetGoStruct() (T1, error) {
	var x T1
	err := r.KubeObject.As(&x)
	return x, err
}

func (r *KubeObjectExt[T1]) GetNestedString(fields ...string) string {
	s, ok, err := r.NestedString(fields...)
	if err != nil {
		return ""
	}
	if !ok {
		return ""
	}
	return s
}

func (r *KubeObjectExt[T1]) GetNestedInt(fields ...string) int {
	s, ok, err := r.NestedInt(fields...)
	if err != nil {
		return 0
	}
	if !ok {
		return 0
	}
	return s
}

func (r *KubeObjectExt[T1]) GetNestedBool(fields ...string) bool {
	s, ok, err := r.NestedBool(fields...)
	if err != nil {
		return false
	}
	if !ok {
		return false
	}
	return s
}

func (r *KubeObjectExt[T1]) GetNestedStringMap(fields ...string) map[string]string {
	s, ok, err := r.NestedStringMap(fields...)
	if err != nil {
		return map[string]string{}
	}
	if !ok {
		return map[string]string{}
	}
	return s
}

// NewFromKubeObject returns a KubeObjectExt struct
// It expects a fn.KubeObject as input representing the serialized yaml file
func NewFromKubeObject[T1 any](o *fn.KubeObject) (*KubeObjectExt[T1], error) {
	if o == nil {
		return nil, fmt.Errorf("cannot initialize with a nil object")
	}
	return &KubeObjectExt[T1]{*o}, nil
}

// NewFromYaml returns a KubeObjectExt struct
// It expects raw byte slice as input representing the serialized yaml file
func NewFromYaml[T1 any](b []byte) (*KubeObjectExt[T1], error) {
	o, err := fn.ParseKubeObject(b)
	if err != nil {
		return nil, err
	}
	return NewFromKubeObject[T1](o)
}

// NewFromGoStruct returns a KubeObjectExt struct
// It expects a go struct representing the interface krm resource
func NewFromGoStruct[T1 any](x any) (*KubeObjectExt[T1], error) {
	o, err := fn.NewFromTypedObject(x)
	if err != nil {
		return nil, err
	}
	return NewFromKubeObject[T1](o)
}
