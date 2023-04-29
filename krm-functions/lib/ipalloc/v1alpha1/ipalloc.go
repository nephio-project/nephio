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
	"github.com/nephio-project/nephio/krm-functions/lib/kubeobject"
	ipamv1alpha1 "github.com/nokia/k8s-ipam/apis/alloc/ipam/v1alpha1"
)

type IPAllocation struct {
	kubeobject.KubeObjectExt[*ipamv1alpha1.IPAllocation]
}

// NewFromKubeObject creates a new KubeObjectExt
// It expects a *fn.KubeObject as input representing the serialized yaml file
func NewFromKubeObject(o *fn.KubeObject) (*IPAllocation, error) {
	r, err := kubeobject.NewFromKubeObject[*ipamv1alpha1.IPAllocation](o)
	if err != nil {
		return nil, err
	}
	return &IPAllocation{*r}, nil
}

// NewFromYAML creates a new KubeObjectExt
// It expects a raw byte slice as input representing the serialized yaml file
func NewFromYAML(b []byte) (*IPAllocation, error) {
	r, err := kubeobject.NewFromYaml[*ipamv1alpha1.IPAllocation](b)
	if err != nil {
		return nil, err
	}
	return &IPAllocation{*r}, nil
}

// NewFromGoStruct creates a new KubeObjectExt
// It expects a go struct representing the KRM resource
func NewFromGoStruct(x *ipamv1alpha1.IPAllocation) (*IPAllocation, error) {
	r, err := kubeobject.NewFromGoStruct[*ipamv1alpha1.IPAllocation](x)
	if err != nil {
		return nil, err
	}
	return &IPAllocation{*r}, nil
}

func (r *IPAllocation) SetSpec(spec ipamv1alpha1.IPAllocationSpec) error {
	return r.KubeObjectExt.SetSpec(spec)
}

func (r *IPAllocation) SetStatus(spec ipamv1alpha1.IPAllocationStatus) error {
	return r.KubeObjectExt.SetStatus(spec)
}