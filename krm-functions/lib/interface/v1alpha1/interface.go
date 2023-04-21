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

var (
	attachmentType      = []string{"spec", "attachmentType"}
	cniType             = []string{"spec", "cniType"}
	networkInstanceName = []string{"spec", "networkInstance", "name"}
)

type Interface struct {
	kubeobject.KubeObjectExt[*nephioreqv1alpha1.Interface]
}

// NewFromKubeObject creates a new parser interface
// It expects a *fn.KubeObject as input representing the serialized yaml file
func NewFromKubeObject(o *fn.KubeObject) (*Interface, error) {
	r, err := kubeobject.NewFromKubeObject[*nephioreqv1alpha1.Interface](o)
	if err != nil {
		return nil, err
	}
	return &Interface{*r}, nil
}

// NewFromYAML creates a new parser interface
// It expects a raw byte slice as input representing the serialized yaml file
func NewFromYAML(b []byte) (*Interface, error) {
	r, err := kubeobject.NewFromYaml[*nephioreqv1alpha1.Interface](b)
	if err != nil {
		return nil, err
	}
	return &Interface{*r}, nil
}

// NewFromGoStruct creates a new parser interface
// It expects a go struct representing the interface krm resource
func NewFromGoStruct(x *nephioreqv1alpha1.Interface) (*Interface, error) {
	r, err := kubeobject.NewFromGoStruct[*nephioreqv1alpha1.Interface](x)
	if err != nil {
		return nil, err
	}
	return &Interface{*r}, nil
}

func (r *Interface) GetNestedString(fields ...string) string {
	s, ok, err := r.NestedString(fields...)
	if err != nil {
		return ""
	}
	if !ok {
		return ""
	}
	return s
}


// GetAttachmentType returns the attachmentType from the spec
// if an error occurs or the attribute is not present an empty string is returned
func (r *Interface) GetAttachmentType() string {
	return r.GetNestedString(attachmentType...)
}

// GetCNIType returns the cniType from the spec
// if an error occurs or the attribute is not present an empty string is returned
func (r *Interface) GetCNIType() string {
	return r.GetNestedString(cniType...)
}

// GetNetworkInstanceName returns the name of the networkInstance from the spec
// if an error occurs or the attribute is not present an empty string is returned
func (r *Interface) GetNetworkInstanceName() string {
	return r.GetNestedString(networkInstanceName...)
}

// SetAttachmentType sets the attachmentType in the spec
func (r *Interface) SetAttachmentType(s nephioreqv1alpha1.AttachmentType) error {
	return r.SetNestedString(string(s), attachmentType...)
}

// SetCNIType sets the cniType in the spec
func (r *Interface) SetCNIType(s nephioreqv1alpha1.CNIType) error {
	return r.SetNestedString(string(s), cniType...)
}

// SetNetworkInstanceName sets the name of the networkInstance in the spec
func (r *Interface) SetNetworkInstanceName(s string) error {
	return r.SetNestedString(s, networkInstanceName...)
}

// SetSpec sets the spec attributes in the kubeobject according the go struct
func (r *Interface) SetSpec(spec *nephioreqv1alpha1.InterfaceSpec) error {
	if spec == nil {
		return nil
	}
	if spec.AttachmentType != "" {
		if err := r.SetAttachmentType(spec.AttachmentType); err != nil {
			return err
		}
	} else {
		if _, err := r.DeleteAttachmentType(); err != nil {
			return err
		}
	}
	if spec.CNIType != "" {
		if err := r.SetCNIType(spec.CNIType); err != nil {
			return err
		}
	} else {
		if _, err := r.DeleteCNIType(); err != nil {
			return err
		}
	}
	if spec.NetworkInstance != nil {
		if err := r.SetNetworkInstanceName(string(spec.NetworkInstance.Name)); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("networkInstance is required")
	}
	return nil
}

// DeleteAttachmentType deletes the attachmentType from the spec
func (r *Interface) DeleteAttachmentType() (bool, error) {
	return r.RemoveNestedField(attachmentType...)
}

// DeleteAttachmentType deletes the attachmentType from the spec
func (r *Interface) DeleteCNIType() (bool, error) {
	return r.RemoveNestedField(cniType...)
}