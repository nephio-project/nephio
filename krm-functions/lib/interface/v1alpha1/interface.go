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
	"sigs.k8s.io/yaml"
)

const (
	// errors
	errKubeObjectNotInitialized = "KubeObject not initialized"
)

var (
	attachmentType      = []string{"spec", "attachmentType"}
	cniType             = []string{"spec", "cniType"}
	networkInstanceName = []string{"spec", "networkInstance", "name"}
)

type Interface interface {
	// GetKubeObject returns the present kubeObject
	GetKubeObject() *fn.KubeObject
	// GetGoStruct returns a go struct representing the present KRM resource
	GetGoStruct() (*nephioreqv1alpha1.Interface, error)
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
	SetAttachmentType(s string) error
	// SetCNIType sets the cniType in the spec
	SetCNIType(s string) error
	// SetNetworkInstanceName sets the name of the networkInstance in the spec
	SetNetworkInstanceName(s string) error
	// SetSpec sets the spec attributes in the kubeobject according the go struct
	SetSpec(*nephioreqv1alpha1.InterfaceSpec) error
	// DeleteAttachmentType deletes the attachmentType from the spec
	DeleteAttachmentType() error
	// DeleteAttachmentType deletes the attachmentType from the spec
	DeleteCNIType() error
}

// NewFromYAML creates a new parser interface
// It expects a raw byte slice as input representing the serialized yaml file
func NewFromYAML(b []byte) (Interface, error) {
	o, err := fn.ParseKubeObject(b)
	if err != nil {
		return nil, err
	}
	return &itfce{
		o: o,
	}, nil
}

// NewFromGoStruct creates a new parser interface
// It expects a go struct representing the interface krm resource
func NewFromGoStruct(x *nephioreqv1alpha1.Interface) (Interface, error) {
	b, err := yaml.Marshal(x)
	if err != nil {
		return nil, err
	}
	return NewFromYAML(b)
}

type itfce struct {
	o *fn.KubeObject
}

// GetKubeObject returns the present kubeObject
func (r *itfce) GetKubeObject() *fn.KubeObject {
	return r.o
}

// GetGoStruct returns a go struct representing the present KRM resource
func (r *itfce) GetGoStruct() (*nephioreqv1alpha1.Interface, error) {
	x := &nephioreqv1alpha1.Interface{}
	if err := yaml.Unmarshal([]byte(r.o.String()), x); err != nil {
		return nil, err
	}
	return x, nil
}

// GetAttachmentType returns the attachmentType from the spec
// if an error occurs or the attribute is not present an empty string is returned
func (r *itfce) GetAttachmentType() string {
	return r.getNestedField(attachmentType...)
}

// GetCNIType returns the cniType from the spec
// if an error occurs or the attribute is not present an empty string is returned
func (r *itfce) GetCNIType() string {
	return r.getNestedField(cniType...)
}

// GetNetworkInstanceName returns the name of the networkInstance from the spec
// if an error occurs or the attribute is not present an empty string is returned
func (r *itfce) GetNetworkInstanceName() string {
	return r.getNestedField(networkInstanceName...)
}

// SetAttachmentType sets the attachmentType in the spec
func (r *itfce) SetAttachmentType(s string) error {
	if !nephioreqv1alpha1.IsAttachmentTypeSupported(s) {
		return fmt.Errorf("unknown attachmentType")
	}
	return r.setNestedField(s, attachmentType...)
}

// SetCNIType sets the cniType in the spec
func (r *itfce) SetCNIType(s string) error {
	if !nephioreqv1alpha1.IsCNITypeSupported(s) {
		return fmt.Errorf("unknown cniType")
	}
	return r.setNestedField(s, cniType...)
}

// SetNetworkInstanceName sets the name of the networkInstance in the spec
func (r *itfce) SetNetworkInstanceName(s string) error {
	return r.setNestedField(s, networkInstanceName...)
}

// SetSpec sets the spec attributes in the kubeobject according the go struct
func (r *itfce) SetSpec(spec *nephioreqv1alpha1.InterfaceSpec) error {
	if spec == nil {
		return nil
	}

	// validate the spec
	if err := nephioreqv1alpha1.ValidateInterfaceSpec(spec); err != nil {
		return err
	}

	// set or delete the values in the spec based on the information
	if spec.AttachmentType != "" {
		if err := r.SetAttachmentType(string(spec.AttachmentType)); err != nil {
			return err
		}
	} else {
		if err := r.DeleteAttachmentType(); err != nil {
			return err
		}
	}
	if spec.CNIType != "" {
		if err := r.SetCNIType(string(spec.CNIType)); err != nil {
			return err
		}
	} else {
		if err := r.DeleteCNIType(); err != nil {
			return err
		}
	}
	if spec.NetworkInstance != nil {
		if err := r.SetNetworkInstanceName(string(spec.NetworkInstance.Name)); err != nil {
			return err
		}
	}
	return nil
}

// DeleteAttachmentType deletes the attachmentType from the spec
func (r *itfce) DeleteAttachmentType() error {
	return r.deleteNestedField(attachmentType...)
}

// DeleteCNIType deletes the cniType from the spec
func (r *itfce) DeleteCNIType() error {
	return r.deleteNestedField(cniType...)
}

// getStringValue is a generic utility function that returns a string from
// a string slice representing the path in the yaml doc
func (r *itfce) getStringValue(fields ...string) string {
	s, ok, err := r.o.NestedString(fields...)
	if err != nil {
		return ""
	}
	if !ok {
		return ""
	}
	return s
}

// setNestedField is a generic utility function that sets a string on
// a string slice representing the path in the yaml doc
func (r *itfce) setNestedField(s string, fields ...string) error {
	if r.o == nil {
		return fmt.Errorf(errKubeObjectNotInitialized)
	}
	if err := r.o.SetNestedField(s, fields...); err != nil {
		return err
	}
	return nil
}

// deleteNestedField is a generic utility function that deletes
// a string slice representing the path from the yaml doc
func (r *itfce) deleteNestedField(fields ...string) error {
	if r.o == nil {
		return fmt.Errorf(errKubeObjectNotInitialized)
	}
	_, err := r.o.RemoveNestedField(fields...)
	if err != nil {
		return err
	}
	return nil
}

// getNestedField is a generic utility function that gets
// a string slice representing the path from the yaml doc
func (r *itfce) getNestedField(fields ...string) string {
	if r.o == nil {
		return ""
	}
	return r.getStringValue(fields...)
}
