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

package v1alpha1

import (
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/henderiw-nephio/pkg-examples/pkg/parser"
	nephioreqv1alpha1 "github.com/nephio-project/api/nf_requirements/v1alpha1"
)

var (
	networkInstanceName = []string{"spec", "networkInstance", "name"}
	pool                = []string{"spec", "pools"}
)

type DataNetwork interface {
	parser.Parser[*nephioreqv1alpha1.DataNetwork]
	// GetNetworkInstanceName returns the name of the networkInstance from the spec
	// if an error occurs or the attribute is not present an empty string is returned
	GetNetworkInstanceName() string
	// GetPools returns the pools of the dnn from the spec
	// if an error occurs or the attribute is not present an empty slice is returned
	GetPools() []*nephioreqv1alpha1.Pool
}

// NewFromKubeObject creates a new parser interface
// It expects a *fn.KubeObject as input representing the serialized yaml file
func NewFromKubeObject(o *fn.KubeObject) DataNetwork {
	return &obj{
		p: parser.NewFromKubeObject[*nephioreqv1alpha1.DataNetwork](o),
	}
}

// NewFromYAML creates a new parser interface
// It expects a raw byte slice as input representing the serialized yaml file
func NewFromYAML(b []byte) (DataNetwork, error) {
	p, err := parser.NewFromYaml[*nephioreqv1alpha1.DataNetwork](b)
	if err != nil {
		return nil, err
	}
	return &obj{
		p: p,
	}, nil
}

// NewFromGoStruct creates a new parser interface
// It expects a go struct representing the interface krm resource
func NewFromGoStruct(x *nephioreqv1alpha1.DataNetwork) (DataNetwork, error) {
	p, err := parser.NewFromGoStruct[*nephioreqv1alpha1.DataNetwork](x)
	if err != nil {
		return nil, err
	}
	return &obj{
		p: p,
	}, nil
}

type obj struct {
	p parser.Parser[*nephioreqv1alpha1.DataNetwork]
}

// GetKubeObject returns the present kubeObject
func (r *obj) GetKubeObject() *fn.KubeObject {
	return r.p.GetKubeObject()
}

// GetGoStruct returns a go struct representing the present KRM resource
func (r *obj) GetGoStruct() (*nephioreqv1alpha1.DataNetwork, error) {
	return r.p.GetGoStruct()
}

func (r *obj) GetStringValue(fields ...string) string {
	return r.p.GetStringValue()
}

func (r *obj) GetBoolValue(fields ...string) bool {
	return r.p.GetBoolValue()
}

func (r *obj) GetIntValue(fields ...string) int {
	return r.p.GetIntValue()
}

func (r *obj) GetStringMap(fields ...string) map[string]string {
	return r.p.GetStringMap()
}

func (r *obj) GetSlice(fields ...string) fn.SliceSubObjects {
	return r.p.GetSlice()
}

func (r *obj) SetNestedString(s string, fields ...string) error {
	return r.p.SetNestedString(s, fields...)
}

func (r *obj) SetNestedInt(s int, fields ...string) error {
	return r.p.SetNestedInt(s, fields...)
}

func (r *obj) SetNestedBool(s bool, fields ...string) error {
	return r.p.SetNestedBool(s, fields...)
}

func (r *obj) SetNestedMap(s map[string]string, fields ...string) error {
	return r.p.SetNestedMap(s, fields...)
}

func (r *obj) DeleteNestedField(fields ...string) error {
	return r.p.DeleteNestedField(fields...)
}

// GetNetworkInstanceName returns the name of the networkInstance from the spec
// if an error occurs or the attribute is not present an empty string is returned
func (r *obj) GetNetworkInstanceName() string {
	return r.p.GetStringValue(networkInstanceName...)
}

func (r *obj) GetPools() []*nephioreqv1alpha1.Pool {
	pools := []*nephioreqv1alpha1.Pool{}
	x := r.p.GetSlice(pool...)
	for _, o := range x {
		pools = append(pools, &nephioreqv1alpha1.Pool{
			Name:         o.GetString("name"),
			PrefixLength: uint8(o.GetInt("prefixLength")),
		})
	}
	return pools
}
