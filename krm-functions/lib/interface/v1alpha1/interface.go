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
	nephioreqv1alpha1 "github.com/nephio-project/api/nf_requirements/v1alpha1"
	"github.com/nephio-project/nephio/krm-functions/lib/kubeobject"
)

type Interface struct {
	kubeobject.KubeObjectExt[nephioreqv1alpha1.Interface]
}

// NewFromKubeObject creates a new parser interface
// It expects a *fn.KubeObject as input representing the serialized yaml file
func NewFromKubeObject(o *fn.KubeObject) (*Interface, error) {
	r, err := kubeobject.NewFromKubeObject[nephioreqv1alpha1.Interface](o)
	if err != nil {
		return nil, err
	}
	return &Interface{*r}, nil
}

// NewFromYAML creates a new parser interface
// It expects a raw byte slice as input representing the serialized yaml file
func NewFromYAML(b []byte) (*Interface, error) {
	r, err := kubeobject.NewFromYaml[nephioreqv1alpha1.Interface](b)
	if err != nil {
		return nil, err
	}
	return &Interface{*r}, nil
}

// NewFromGoStruct creates a new parser interface
// It expects a go struct representing the interface krm resource
func NewFromGoStruct(x *nephioreqv1alpha1.Interface) (*Interface, error) {
	if x == nil {
		return nil, fmt.Errorf("cannot initialize with nil pointer")
	}
	r, err := kubeobject.NewFromGoStruct(*x)
	if err != nil {
		return nil, err
	}
	return &Interface{*r}, nil
}

func (r *Interface) SetSpec(spec nephioreqv1alpha1.InterfaceSpec) error {
	return r.KubeObjectExt.UnsafeSetSpec(spec)
}

func (r *Interface) SetStatus(spec nephioreqv1alpha1.InterfaceStatus) error {
	return r.KubeObjectExt.UnsafeSetStatus(spec)
}
