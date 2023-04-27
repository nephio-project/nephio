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

package parser

import (
	"fmt"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"sigs.k8s.io/yaml"
)

const (
	// errors
	errKubeObjectNotInitialized = "KubeObject not initialized"
)

type Parser[T1 any] interface {
	// GetKubeObject returns the present kubeObject
	GetKubeObject() *fn.KubeObject
	// GetGoStruct returns a go struct representing the present KRM resource
	GetGoStruct() (T1, error)
	// GetStringValue is a generic utility function that returns a string from
	// a string slice representing the path in the yaml doc
	GetStringValue(fields ...string) string
	// GetIntValue is a generic utility function that returns a int from
	// a string slice representing the path in the yaml doc
	GetIntValue(fields ...string) int
	// GetBoolValue is a generic utility function that returns a bool from
	// a string slice representing the path in the yaml doc
	GetBoolValue(fields ...string) bool
	// GetStringMap is a generic utility function that returns a map[string]string from
	// a string slice representing the path in the yaml doc
	GetStringMap(fields ...string) map[string]string
	// GetSlice is a generic utility function that returns a fn.SliceSubObjects from
	// a string slice representing the path in the yaml doc
	GetSlice(fields ...string) fn.SliceSubObjects
	// SetNestedString is a generic utility function that sets a string on
	// a string slice representing the path in the yaml doc
	SetNestedString(s string, fields ...string) error
	// SetNestedInt is a generic utility function that sets a int on
	// a string slice representing the path in the yaml doc
	SetNestedInt(s int, fields ...string) error
	// SetNestedBool is a generic utility function that sets a bool on
	// a string slice representing the path in the yaml doc
	SetNestedBool(s bool, fields ...string) error
	// SetNestedMap is a generic utility function that sets a map[string]string on
	// a string slice representing the path in the yaml doc
	SetNestedMap(s map[string]string, fields ...string) error
	// DeleteNestedField is a generic utility function that deletes
	// a string slice representing the path from the yaml doc
	DeleteNestedField(fields ...string) error
}

// NewFromKubeObject creates a new parser interface
// It expects a *fn.KubeObject as input representing the serialized yaml file
func NewFromKubeObject[T1 any](o *fn.KubeObject) Parser[T1] {
	return &obj[T1]{
		o: o,
	}
}

// NewFromYaml creates a new parser interface
// It expects raw byte slice as input representing the serialized yaml file
func NewFromYaml[T1 any](b []byte) (Parser[T1], error) {
	o, err := fn.ParseKubeObject(b)
	if err != nil {
		return nil, err
	}
	return NewFromKubeObject[T1](o), nil
}

// NewFromGoStruct creates a new parser interface
// It expects a go struct representing the interface krm resource
func NewFromGoStruct[T1 any](x any) (Parser[T1], error) {
	b, err := yaml.Marshal(x)
	if err != nil {
		return nil, err
	}
	return NewFromYaml[T1](b)
}

type obj[T1 any] struct {
	o *fn.KubeObject
}

// GetKubeObject returns the present kubeObject
func (r *obj[T1]) GetKubeObject() *fn.KubeObject {
	return r.o
}

// GetGoStruct returns a go struct representing the present KRM resource
func (r *obj[T1]) GetGoStruct() (T1, error) {
	var x T1
	if err := yaml.Unmarshal([]byte(r.o.String()), &x); err != nil {
		return x, err
	}
	return x, nil
}

// GetStringValue is a generic utility function that returns a string from
// a string slice representing the path in the yaml doc
func (r *obj[T1]) GetStringValue(fields ...string) string {
	if r.o == nil {
		return ""
	}
	s, ok, err := r.o.NestedString(fields...)
	if err != nil {
		return ""
	}
	if !ok {
		return ""
	}
	return s
}

// GetIntValue is a generic utility function that returns a int from
// a string slice representing the path in the yaml doc
func (r *obj[T1]) GetIntValue(fields ...string) int {
	if r.o == nil {
		return 0
	}
	i, ok, err := r.o.NestedInt(fields...)
	if err != nil {
		return 0
	}
	if !ok {
		return 0
	}
	return i
}

// GetBoolValue is a generic utility function that returns a bool from
// a string slice representing the path in the yaml doc
func (r *obj[T1]) GetBoolValue(fields ...string) bool {
	if r.o == nil {
		return false
	}
	b, ok, err := r.o.NestedBool(fields...)
	if err != nil {
		return false
	}
	if !ok {
		return false
	}
	return b
}

// GetStringMap is a generic utility function that returns a map[string]string from
// a string slice representing the path in the yaml doc
func (r *obj[T1]) GetStringMap(fields ...string) map[string]string {
	if r.o == nil {
		return map[string]string{}
	}
	m, ok, err := r.o.NestedStringMap(fields...)
	if err != nil {
		return map[string]string{}
	}
	if !ok {
		return map[string]string{}
	}
	return m
}

// GetSlice is a generic utility function that returns a fn.SliceSubObjects from
// a string slice representing the path in the yaml doc
func (r *obj[T1]) GetSlice(fields ...string) fn.SliceSubObjects {
	if r.o == nil {
		return fn.SliceSubObjects{}
	}
	m, ok, err := r.o.NestedSlice(fields...)
	if err != nil {
		return fn.SliceSubObjects{}
	}
	if !ok {
		return fn.SliceSubObjects{}
	}
	return m
}

// SetNestedField is a generic utility function that sets a string on
// a string slice representing the path in the yaml doc
func (r *obj[T1]) SetNestedString(s string, fields ...string) error {
	if r.o == nil {
		return fmt.Errorf(errKubeObjectNotInitialized)
	}
	if err := r.o.SetNestedField(s, fields...); err != nil {
		return err
	}
	return nil
}

// SetNestedInt is a generic utility function that sets a int on
// a string slice representing the path in the yaml doc
func (r *obj[T1]) SetNestedInt(s int, fields ...string) error {
	if r.o == nil {
		return fmt.Errorf(errKubeObjectNotInitialized)
	}
	if err := r.o.SetNestedInt(s, fields...); err != nil {
		return err
	}
	return nil
}

// SetNestedBool is a generic utility function that sets a bool on
// a string slice representing the path in the yaml doc
func (r *obj[T1]) SetNestedBool(s bool, fields ...string) error {
	if r.o == nil {
		return fmt.Errorf(errKubeObjectNotInitialized)
	}
	if err := r.o.SetNestedBool(s, fields...); err != nil {
		return err
	}
	return nil
}

// SetNestedMap is a generic utility function that sets a map[string]string on
// a string slice representing the path in the yaml doc
func (r *obj[T1]) SetNestedMap(s map[string]string, fields ...string) error {
	if r.o == nil {
		return fmt.Errorf(errKubeObjectNotInitialized)
	}
	if err := r.o.SetNestedStringMap(s, fields...); err != nil {
		return err
	}
	return nil
}

// DeleteNestedField is a generic utility function that deletes
// a string slice representing the path from the yaml doc
func (r *obj[T1]) DeleteNestedField(fields ...string) error {
	if r.o == nil {
		return fmt.Errorf(errKubeObjectNotInitialized)
	}
	_, err := r.o.RemoveNestedField(fields...)
	if err != nil {
		return err
	}
	return nil
}
