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
	"bytes"
	"fmt"
	"io"

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
	return safeSetNestedFieldKeepFormatting(&(o.KubeObject), newSpec, "spec")
}

// SetStatus sets the `status` field of a KubeObjectExt to the value of `newStatus`,
// while trying to keep as much formatting as possible
func (o *KubeObjectExt[T1]) SetStatus(newStatus interface{}) error {
	return safeSetNestedFieldKeepFormatting(&o.KubeObject, newStatus, "status")
}

// NOTE: the following functions are considered as "methods" of KubeObject,
// and thus nill checking of `obj` was omitted intentionally:
// the caller is responsible for ensuring that `obj` is not nil`

// setNestedFieldKeepFormatting is similar to KubeObject.SetNestedField(), but keeps the
// comments and the order of fields in the YAML wherever it is possible.
//
// NOTE: This functionality should be solved in the upstream SDK.
// Merging the code below to the upstream SDK is in progress and tracked in this issue:
// https://github.com/GoogleContainerTools/kpt/issues/3923
func safeSetNestedFieldKeepFormatting(obj *fn.KubeObject, value interface{}, fields ...string) error {
	oldNode := yamlNodeOf2(&obj.SubObject)
	err := obj.SetNestedField(value, fields...)
	if err != nil {
		return err
	}
	newNode := yamlNodeOf2(&obj.SubObject)

	if oldNode.Kind != yaml.DocumentNode || len(oldNode.Content) == 0 ||
		newNode.Kind != yaml.DocumentNode || len(newNode.Content) == 0 {
		panic("unexpected YAML node type after parsing SubObject")
	}
	restoreFieldOrder(oldNode.Content[0], newNode.Content[0])
	deepCopyComments(oldNode.Content[0], newNode.Content[0])

	b, err := toYAML(newNode)
	if err != nil {
		return fmt.Errorf("unexpected error during round-trip YAML parsing (ToYAML): %v", err)
	}

	obj2, err := fn.ParseKubeObject(b)
	if err != nil {
		return fmt.Errorf("unexpected error during round-trip YAML parsing (ParseKubeObject): %v", err)
	}
	*obj = *obj2
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

func yamlNodeOf2(obj *fn.SubObject) *yaml.Node {
	var node *yaml.Node
	yamlBytes := []byte(obj.String())
	node, err := parseFirstObj(yamlBytes)
	if err != nil {
		panic(fmt.Sprintf("round-trip YAML serialization failed (ParseFirstObj): %v", err))
	}
	return node
}

func parseFirstObj(b []byte) (*yaml.Node, error) {
	br := bytes.NewReader(b)
	decoder := yaml.NewDecoder(br)
	node := &yaml.Node{}
	if err := decoder.Decode(node); err != nil {
		if err != io.EOF {
			return nil, err
		}
	}
	return node, nil
}

func toYAML(node *yaml.Node) ([]byte, error) {
	var w bytes.Buffer
	encoder := yaml.NewEncoder(&w)
	if node.Kind == yaml.DocumentNode {
		if len(node.Content) == 0 {
			// These cause errors when we try to write them
			return nil, fmt.Errorf("ToYAML: invalid DocumentNode")
		}
	}
	if err := encoder.Encode(node); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}
