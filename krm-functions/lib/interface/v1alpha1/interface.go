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
)

type Interface interface {
	// GetKubeObject returns the present kubeObject
	GetKubeObject() *fn.KubeObject
	// GetAttachmentType returns the attachmentType from the spec
	// if an error occurs or the attribute is not present an empty string is returned
	GetAttachmentType() string
	// GetCNIType returns the cniType from the spec
	// if an error occurs or the attribute is not present an empty string is returned
	GetCNIType() string
	// GetNetworkInstanceName returns the name of the networkInstance from the spec
	// if an error occurs or the attribute is not present an empty string is returned
	GetNetworkInstanceName() string
	// SetAttachmentType sets the attachmentType in the spec
	// returns an error if the value is an unknown type or when the
	// set fails
	SetAttachmentType(s string) error
	// SetCNIType sets the cniType in the spec
	// returns an error if the value is an unknown type or when the
	// set fails
	SetCNIType(s string) error
	// SetNetworkInstanceName sets the name of the networkInstance in the spec
	// returns an error when the set fails
	SetNetworkInstanceName(s string) error
    // SetInterfaceSpec sets the spec attributes in the kubeobject
	SetInterfaceSpec(*nephioreqv1alpha1.InterfaceSpec) error
}

// New creates a new parser interface
// It expects a raw byte slice as input representing the serialized yaml file
func New(b []byte) (Interface, error) {
	o, err := fn.ParseKubeObject(b)
	return &itfce{
		o: o,
	}, err
}

type itfce struct {
	o *fn.KubeObject
}

// GetKubeObject returns the present kubeObject
func (r *itfce) GetKubeObject() *fn.KubeObject {
	return r.o
}

func (r *itfce) GetAttachmentType() string {
	if r.o == nil {
        return ""
    }
    s, ok, err := r.o.NestedString("spec", "attachmentType")
	if err != nil {
		return ""
	}
	if !ok {
		return ""
	}
	return s
}

func (r *itfce) GetCNIType() string {
	if r.o == nil {
        return ""
    }
    s, ok, err := r.o.NestedString("spec", "cniType")
	if err != nil {
		return ""
	}
	if !ok {
		return ""
	}
	return s
}

func (r *itfce) GetNetworkInstanceName() string {
	if r.o == nil {
        return ""
    }
    s, ok, err := r.o.NestedString("spec", "networkInstance", "name")
	if err != nil {
		return ""
	}
	if !ok {
		return ""
	}
	return s
}

func (r *itfce) SetAttachmentType(s string) error {
	if r.o == nil {
        return fmt.Errorf("KubeObject not initialized")
    }
    // validation -> should be part of the interface api repo
	switch s {
	case string(nephioreqv1alpha1.AttachmentTypeNone):
	case string(nephioreqv1alpha1.AttachmentTypeVLAN):
	default:
		return fmt.Errorf("unknown attachmentType")
	}

	if err := r.o.SetNestedField(s, "spec", "attachmentType"); err != nil {
		return err
	}
	return nil
}

func (r *itfce) SetCNIType(s string) error {
	if r.o == nil {
        return fmt.Errorf("KubeObject not initialized")
    }
    // validation -> should be part of the interface api repo
	switch s {
	case string(nephioreqv1alpha1.CNITypeIPVLAN):
	case string(nephioreqv1alpha1.CNITypeSRIOV):
	case string(nephioreqv1alpha1.CNITypeMACVLAN):
	default:
		return fmt.Errorf("unknown cniType")
	}

	if err := r.o.SetNestedField(s, "spec", "cniType"); err != nil {
		return err
	}
	return nil
}

func (r *itfce) SetNetworkInstanceName(s string) error {
    if r.o == nil {
        return fmt.Errorf("KubeObject not initialized")
    }
	if err := r.o.SetNestedField(s, "spec", "networkInstance", "name"); err != nil {
		return err
	}
	return nil
}

// SetInterfaceSpec sets the spec attributes in the kubeobject
func (r *itfce) SetInterfaceSpec(spec *nephioreqv1alpha1.InterfaceSpec) error {
	if spec != nil {
		if spec.AttachmentType != "" {
			if err := r.SetAttachmentType(string(spec.AttachmentType)); err != nil {
				return err
			}
		}
		if spec.CNIType != "" {
			if err := r.SetCNIType(string(spec.CNIType)); err != nil {
				return err
			}
		}
		if spec.NetworkInstance != nil {
			if err := r.SetNetworkInstanceName(string(spec.NetworkInstance.Name)); err != nil {
				return err
			}
		}
	}
	return nil
}
