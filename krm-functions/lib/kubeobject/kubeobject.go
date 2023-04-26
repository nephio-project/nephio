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
	"unsafe"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type KubeObjectExt[T1 any] struct {
	fn.KubeObject
}

func (r *KubeObjectExt[T1]) GetGoStruct() (T1, error) {
	var x T1
	err := r.KubeObject.As(&x)
	return x, err
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
	if x == nil {
		return nil, fmt.Errorf("cannot initialize with a nil object")
	}
	o, err := fn.NewFromTypedObject(x)
	if err != nil {
		return nil, err
	}
	return NewFromKubeObject[T1](o)
}

// SetSpec sets the `spec` field of a KubeObjectExt to the value of `newSpec`,
// while trying to keep as much formatting as possible
func (o *KubeObjectExt[T1]) SetSpec(newSpec interface{}) error {
	return SetSpec(&o.KubeObject, newSpec)
}

// SetStatus sets the `status` field of a KubeObjectExt to the value of `newStatus`,
// while trying to keep as much formatting as possible
func (o *KubeObjectExt[T1]) SetStatus(newStatus interface{}) error {
	return SetStatus(&o.KubeObject, newStatus)
}

// SetNestedFieldKeepFormatting is similar to KubeObject.SetNestedField(), but keeps the
// comments and the order of fields in the YAML wherever it is possible.
//
// NOTE: This functionality should be solved in the upstream SDK.
// Merging the code below to the upstream SDK is in progress and tracked in this issue:
// https://github.com/GoogleContainerTools/kpt/issues/3923
func (o *KubeObjectExt[T1]) SetNestedFieldKeepFormatting(value interface{}, field string) error {
	return SetNestedFieldKeepFormatting(&o.KubeObject.SubObject, value, field)
}

// NOTE: the following functions are considered as "methods" of KubeObject,
// and thus nill checking of `obj` was omitted intentionally:
// the caller is responsible for ensuring that `obj` is not nil`

// ToStruct converts the KubeObject to a go struct with type `T`
func ToStruct[T any](obj *fn.KubeObject) (T, error) {
	var x T
	err := obj.As(&x)
	return x, err
}

// GetSpec returns with the `spec` field of the KubeObject as a go struct with type `T`
// NOTE: consider using ToStruct() instead
func GetSpec[T any](obj *fn.KubeObject) (T, error) {
	var spec T
	err := obj.UpsertMap("spec").As(&spec)
	return spec, err

}

// GetStatus returns with the `status` field of the KubeObject as a go struct with type `T`
// NOTE: consider using ToStruct() instead
func GetStatus[T any](obj *fn.KubeObject) (T, error) {
	var status T
	err := obj.UpsertMap("status").As(&status)
	return status, err

}

// SetSpec sets the `spec` field of a KubeObject to the value of `newSpec`,
// while trying to keep as much formatting as possible
func SetSpec(obj *fn.KubeObject, newSpec interface{}) error {
	return SetNestedFieldKeepFormatting(&obj.SubObject, newSpec, "spec")
}

// SetStatus sets the `status` field of a KubeObject to the value of `newStatus`,
// while trying to keep as much formatting as possible
func SetStatus(obj *fn.KubeObject, newStatus interface{}) error {
	return SetNestedFieldKeepFormatting(&obj.SubObject, newStatus, "status")
}

// SetNestedFieldKeepFormatting is similar to KubeObject.SetNestedField(), but keeps the
// comments and the order of fields in the YAML wherever it is possible.
//
// NOTE: This functionality should be solved in the upstream SDK.
// Merging the code below to the upstream SDK is in progress and tracked in this issue:
// https://github.com/GoogleContainerTools/kpt/issues/3923
func SetNestedFieldKeepFormatting(obj *fn.SubObject, value interface{}, field string) error {
	oldNode := yamlNodeOf(obj.UpsertMap(field))
	err := obj.SetNestedField(value, field)
	if err != nil {
		return err
	}
	newNode := yamlNodeOf(obj.GetMap(field))

	restoreFieldOrder(oldNode, newNode)
	deepCopyComments(oldNode, newNode)
	return nil
}

///////////////// internals

func shallowCopyComments(src, dst *yaml.Node) {
	dst.HeadComment = src.HeadComment
	dst.LineComment = src.LineComment
	dst.FootComment = src.FootComment
}

func deepCopyComments(src, dst *yaml.Node) {
	if src.Kind != dst.Kind {
		return
	}
	shallowCopyComments(src, dst)
	if dst.Kind == yaml.MappingNode {
		if (len(src.Content)%2 != 0) || (len(dst.Content)%2 != 0) {
			panic("unexpected number of children for YAML map")
		}
		for i := 0; i < len(dst.Content); i += 2 {
			dstKeyNode := dst.Content[i]
			key, ok := asString(dstKeyNode)
			if !ok {
				continue
			}

			j, ok := findKey(src, key)
			if !ok {
				continue
			}
			srcKeyNode, srcValueNode := src.Content[j], src.Content[j+1]
			dstValueNode := dst.Content[i+1]
			shallowCopyComments(srcKeyNode, dstKeyNode)
			deepCopyComments(srcValueNode, dstValueNode)
		}
	}
}

func restoreFieldOrder(src, dst *yaml.Node) {
	if (src.Kind != dst.Kind) || (dst.Kind != yaml.MappingNode) {
		return
	}
	if (len(src.Content)%2 != 0) || (len(dst.Content)%2 != 0) {
		panic("unexpected number of children for YAML map")
	}

	nextInDst := 0
	for i := 0; i < len(src.Content); i += 2 {
		key, ok := asString(src.Content[i])
		if !ok {
			continue
		}

		j, ok := findKey(dst, key)
		if !ok {
			continue
		}
		if j != nextInDst {
			dst.Content[j], dst.Content[nextInDst] = dst.Content[nextInDst], dst.Content[j]
			dst.Content[j+1], dst.Content[nextInDst+1] = dst.Content[nextInDst+1], dst.Content[j+1]
		}
		nextInDst += 2

		srcValueNode := src.Content[i+1]
		dstValueNode := dst.Content[nextInDst-1]
		restoreFieldOrder(srcValueNode, dstValueNode)
	}
}

func asString(node *yaml.Node) (string, bool) {
	if node.Kind == yaml.ScalarNode && (node.Tag == "!!str" || node.Tag == "") {
		return node.Value, true
	}
	return "", false
}

func findKey(m *yaml.Node, key string) (int, bool) {
	children := m.Content
	if len(children)%2 != 0 {
		panic("unexpected number of children for YAML map")
	}
	for i := 0; i < len(children); i += 2 {
		keyNode := children[i]
		k, ok := asString(keyNode)
		if ok && k == key {
			return i, true
		}
	}
	return 0, false
}

// This is a temporary workaround until SetNestedFieldKeppFormatting functionality is merged into the upstream SDK
// The merge process has already started and tracked in this issue: https://github.com/GoogleContainerTools/kpt/issues/3923
func yamlNodeOf(obj *fn.SubObject) *yaml.Node {
	internalObj := reflect.ValueOf(*obj).FieldByName("obj")
	nodePtr := internalObj.Elem().FieldByName("node")
	nodePtr = reflect.NewAt(nodePtr.Type(), unsafe.Pointer(nodePtr.UnsafeAddr())).Elem()
	return nodePtr.Interface().(*yaml.Node)
}
