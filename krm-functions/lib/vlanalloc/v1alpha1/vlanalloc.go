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
	"fmt"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/nephio-project/nephio/krm-functions/lib/kubeobject"
	vlanv1alpha1 "github.com/nokia/k8s-ipam/apis/alloc/vlan/v1alpha1"
)

type VLANAllocation struct {
	kubeobject.KubeObjectExt[vlanv1alpha1.VLANAllocation]
}

// NewFromKubeObject creates a new KubeObjectExt
// It expects a *fn.KubeObject as input representing the serialized yaml file
func NewFromKubeObject(o *fn.KubeObject) (*VLANAllocation, error) {
	r, err := kubeobject.NewFromKubeObject[vlanv1alpha1.VLANAllocation](o)
	if err != nil {
		return nil, err
	}
	return &VLANAllocation{*r}, nil
}

// NewFromYAML creates a new KubeObjectExt
// It expects a raw byte slice as input representing the serialized yaml file
func NewFromYAML(b []byte) (*VLANAllocation, error) {
	r, err := kubeobject.NewFromYaml[vlanv1alpha1.VLANAllocation](b)
	if err != nil {
		return nil, err
	}
	return &VLANAllocation{*r}, nil
}

// NewFromGoStruct creates a new KubeObjectExt
// It expects a go struct representing the KRM resource
func NewFromGoStruct(x *vlanv1alpha1.VLANAllocation) (*VLANAllocation, error) {
	if x == nil {
		return nil, fmt.Errorf("cannot initialize with nil pointer")
	}
	r, err := kubeobject.NewFromGoStruct(*x)
	if err != nil {
		return nil, err
	}
	return &VLANAllocation{*r}, nil
}

func (r *VLANAllocation) SetSpec(spec vlanv1alpha1.VLANAllocationSpec) error {
	return r.KubeObjectExt.UnsafeSetSpec(spec)
}

func (r *VLANAllocation) SetStatus(spec vlanv1alpha1.VLANAllocationStatus) error {
	return r.KubeObjectExt.UnsafeSetStatus(spec)
}
