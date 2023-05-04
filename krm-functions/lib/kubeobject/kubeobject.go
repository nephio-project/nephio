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
	"reflect"

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
func NewFromGoStruct[T1 any](x T1) (*KubeObjectExt[T1], error) {
	o, err := fn.NewFromTypedObject(x)
	if err != nil {
		return nil, err
	}
	return NewFromKubeObject[T1](o)
}

// SetSpec sets the `spec` field of a KubeObjectExt to the value of `newSpec`,
// while trying to keep as much formatting as possible
func (o *KubeObjectExt[T1]) SetSpec(newSpec interface{}) error {
	return setNestedFieldKeepFormatting(&(o.KubeObject), newSpec, "spec")
}

// SetStatus sets the `status` field of a KubeObjectExt to the value of `newStatus`,
// while trying to keep as much formatting as possible
func (o *KubeObjectExt[T1]) SetStatus(newStatus interface{}) error {
	return setNestedFieldKeepFormatting(&o.KubeObject, newStatus, "status")
}

// SetSpec sets the `spec` field of a KubeObjectExt to the value of `newSpec`,
// while trying to keep as much formatting as possible
func (o *KubeObjectExt[T1]) SafeSetSpec(value T1) error {
	newSpec, err := o.getField(value, "Spec")
	if err != nil {
		// TODO: consider panicking here
		return err
	}
	return setNestedFieldKeepFormatting(&(o.KubeObject), newSpec, "spec")
}

// SetStatus sets the `status` field of a KubeObjectExt to the value of `newStatus`,
// while trying to keep as much formatting as possible
func (o *KubeObjectExt[T1]) SafeSetStatus(value T1) error {
	newStatus, err := o.getField(value, "Status")
	if err != nil {
		// TODO: consider panicking here
		return err
	}
	return setNestedFieldKeepFormatting(&o.KubeObject, newStatus, "status")
}

// setNestedFieldKeepFormatting is similar to KubeObject.SetNestedField(), but keeps the
// comments and the order of fields in the YAML wherever it is possible.
//
// NOTE: This functionality should be solved in the upstream SDK.
// Merging the code below to the upstream SDK is in progress and tracked in this issue:
// https://github.com/GoogleContainerTools/kpt/issues/3923
func setNestedFieldKeepFormatting(obj *fn.KubeObject, value interface{}, fields ...string) error {
	oldNode := yamlNodeOf(&obj.SubObject)
	err := obj.SetNestedField(value, fields...)
	if err != nil {
		return err
	}
	newNode := yamlNodeOf(&obj.SubObject)

	deepCopyFormatting(oldNode, newNode)

	return setYamlNodeOf(obj, newNode)
}

///////////////// internals

func (o *KubeObjectExt[T1]) getField(value T1, fieldName string) (interface{}, error) {
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("type %q is not a struct, so it doesn't have a %q field", v.Type().Name(), fieldName)
	}
	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return nil, fmt.Errorf("type %q doesn't have a %q field", v.Type().Name(), fieldName)
	}
	return field.Interface(), nil
}

// shallowCopyComments copies comments from `src` to `dst` non-recursively
func shallowCopyComments(src, dst *yaml.Node) {
	dst.HeadComment = src.HeadComment
	dst.LineComment = src.LineComment
	dst.FootComment = src.FootComment
}

// deepCopyFormatting copies formatting (comments and order of fields) from `src` to `dst` recursively
func deepCopyFormatting(src, dst *yaml.Node) {
	if src.Kind != dst.Kind {
		return
	}

	switch dst.Kind {
	case yaml.MappingNode:
		copyMapFormatting(src, dst)
	case yaml.SequenceNode:
		copyListFormatting(src, dst)
	case yaml.DocumentNode:
		if len(src.Content) == 1 && len(dst.Content) == 1 {
			shallowCopyComments(src, dst)
			deepCopyFormatting(src.Content[0], dst.Content[0])
		} else {
			// this shouldn't really happen with YAML nodes in KubeObjects
			copyListFormatting(src, dst)
		}
	default:
		shallowCopyComments(src, dst)
	}
}

// copyMapFormatting copies formatting between MappingNodes recursively
func copyMapFormatting(src, dst *yaml.Node) {
	if (len(src.Content)%2 != 0) || (len(dst.Content)%2 != 0) {
		panic("unexpected number of children for a YAML MappingNode")
	}

	// keep comments
	shallowCopyComments(src, dst)

	// copy formatting of `src` fields to corresponding `dst` fields
	unorderedPartOfDst := dst.Content // the slice of dst.Content that hasn't been reformatted yet
	for i := 0; i < len(src.Content); i += 2 {
		key, ok := asString(src.Content[i])
		if !ok {
			continue
		}
		j, found := findKey(unorderedPartOfDst, key)
		if !found {
			continue
		}

		// keep ordering: swap key & value to the beginning of the unordered part
		if j != 0 {
			unorderedPartOfDst[j], unorderedPartOfDst[0] = unorderedPartOfDst[0], unorderedPartOfDst[j]
			unorderedPartOfDst[j+1], unorderedPartOfDst[1] = unorderedPartOfDst[1], unorderedPartOfDst[j+1]
		}
		// keep comments
		shallowCopyComments(src.Content[i], unorderedPartOfDst[0])
		deepCopyFormatting(src.Content[i+1], unorderedPartOfDst[1])
		unorderedPartOfDst = unorderedPartOfDst[2:]
	}
}

// copyListFormatting copies formatting between SequenceNodes recursively
func copyListFormatting(src, dst *yaml.Node) {
	// keep comments
	shallowCopyComments(src, dst)

	// copy formatting of `src` items to corresponding `dst` fields
	for _, srcItem := range src.Content {
		j, found := findMatchingItemForFormattingCopy(srcItem, dst.Content)
		if !found {
			continue
		}

		// NOTE: the order of list items isn't restored,
		// since the change in order might be significant and deliberate

		deepCopyFormatting(srcItem, dst.Content[j])
	}
}

func asString(node *yaml.Node) (string, bool) {
	if node.Kind == yaml.ScalarNode && (node.Tag == "!!str" || node.Tag == "") {
		return node.Value, true
	}
	return "", false
}

// findKey finds `key` in the Content list of a YAML MappingNode passed as `mapContents`
// Returns with the index of the key node as an int and whether the search was succesful as a bool
func findKey(mapContents []*yaml.Node, key string) (int, bool) {
	for i := 0; i < len(mapContents); i += 2 {
		keyNode := mapContents[i]
		k, ok := asString(keyNode)
		if ok && k == key {
			return i, true
		}
	}
	return 0, false
}

// findMatchingItemForFormattingCopy finds the node in `dstList` that matches with `srcItem` in the sense that
// formatting should be copied from `srcItem` to the matching item in `dstList`
// Returns with the index of the matching item as an int and whether the search was succesful as a bool
func findMatchingItemForFormattingCopy(srcItem *yaml.Node, dstList []*yaml.Node) (int, bool) {
	for i, dstItem := range dstList {
		if shouldCopyFormatting(srcItem, dstItem) {
			return i, true
		}
	}
	return 0, false
}

// shouldCopyFormatting retrurns whether `src` and `dst` nodes are matching in the sense that
// formatting should be copied from `src` to `dst`
func shouldCopyFormatting(src, dst *yaml.Node) bool {
	if src.Kind != dst.Kind {
		return false
	}
	switch src.Kind {
	case yaml.ScalarNode:
		return src.Value == dst.Value
	case yaml.MappingNode:
		if (len(src.Content)%2 != 0) || (len(dst.Content)%2 != 0) {
			panic("unexpected number of children for YAML map")
		}
		// If all `src` fields are present in `dst` with the same value, the two are considered equal
		// In other words, adding new fields to a map isn't considered as a difference for our purposes
		for i := 0; i < len(src.Content); i += 2 {
			key, ok := asString(src.Content[i])
			if !ok {
				return false
			}
			j, found := findKey(dst.Content, key)
			if !found {
				return false
			}
			if !shouldCopyFormatting(src.Content[i+1], dst.Content[j+1]) {
				return false
			}
		}
		return true
	case yaml.SequenceNode:
		// Any change in embedded lists isn't considered as a difference for our purposes,
		// or in other words: only map fields are compared recursively, but list items are ignored.
		// In the extreme case of list of lists this can lead to inapropriate formatting,
		// but I find this liberal approach to be more practical and efficient in real-life cases.
		return true
	case yaml.AliasNode, yaml.DocumentNode:
		// TODO: check AliasNode properly?
		return true
	}
	panic(fmt.Sprintf("unexpected YAML node type: %v", src.Kind))
}

// yamlNodeOf returns with unexposed yaml.Node inside `obj` without using unsafe
func yamlNodeOf(obj *fn.SubObject) *yaml.Node {
	// NOTE: the round-trip YAML marshalling is only needed to get the internal YAML node from inside of `obj` without using unsafe
	var node *yaml.Node
	yamlBytes := []byte(obj.String())     // marshal to YAML
	node, err := parseFirstObj(yamlBytes) // unmarshal from YAML
	if err != nil {
		panic(fmt.Sprintf("round-trip YAML serialization failed (ParseFirstObj): %v", err))
	}
	if node.Kind == yaml.DocumentNode {
		if len(node.Content) != 1 {
			panic(fmt.Sprintf("unexpected YAML DocumentNode after round-trip YAML serialization: Contents has %v items", len(node.Content)))
		}
		node = node.Content[0]
	}
	return node
}

// setYamlNodeOf puts `newNode` inside `obj` without using unsafe
func setYamlNodeOf(obj *fn.KubeObject, newNode *yaml.Node) error {
	b, err := toYAML(newNode) // marshal to YAML
	if err != nil {
		return fmt.Errorf("unexpected error during round-trip YAML parsing (ToYAML): %v", err)
	}
	obj2, err := fn.ParseKubeObject(b) // unmarshal from YAML
	if err != nil {
		return fmt.Errorf("unexpected error during round-trip YAML parsing (ParseKubeObject): %v", err)
	}
	*obj = *obj2
	return nil
}

// unmarshal YAML text (bytes) to a yaml.Node
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

// marshal yaml.Node to YAML text (bytes)
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
