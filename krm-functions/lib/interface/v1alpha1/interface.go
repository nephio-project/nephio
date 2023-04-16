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
	"github.com/nephio-project/nephio/krm-functions/lib/parser"
)

var (
	attachmentType      = []string{"spec", "attachmentType"}
	cniType             = []string{"spec", "cniType"}
	networkInstanceName = []string{"spec", "networkInstance", "name"}
)

type Interface interface {
	parser.Parser[*nephioreqv1alpha1.Interface]
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
	SetAttachmentType(s nephioreqv1alpha1.AttachmentType) error
	// SetCNIType sets the cniType in the spec
	SetCNIType(s nephioreqv1alpha1.CNIType) error
	// SetNetworkInstanceName sets the name of the networkInstance in the spec
	SetNetworkInstanceName(s string) error
	// SetSpec sets the spec attributes in the kubeobject according the go struct
	SetSpec(*nephioreqv1alpha1.InterfaceSpec) error
	// DeleteAttachmentType deletes the attachmentType from the spec
	DeleteAttachmentType() error
	// DeleteAttachmentType deletes the attachmentType from the spec
	DeleteCNIType() error
}

// NewFromKubeObject creates a new parser interface
// It expects a *fn.KubeObject as input representing the serialized yaml file
func NewFromKubeObject(o *fn.KubeObject) Interface {
	return &obj{
		p: parser.NewFromKubeObject[*nephioreqv1alpha1.Interface](o),
	}
}

// NewFromYAML creates a new parser interface
// It expects a raw byte slice as input representing the serialized yaml file
func NewFromYAML(b []byte) (Interface, error) {
	p, err := parser.NewFromYaml[*nephioreqv1alpha1.Interface](b)
	if err != nil {
		return nil, err
	}
	return &obj{
		p: p,
	}, nil
}

// NewFromGoStruct creates a new parser interface
// It expects a go struct representing the interface krm resource
func NewFromGoStruct(x *nephioreqv1alpha1.Interface) (Interface, error) {
	p, err := parser.NewFromGoStruct[*nephioreqv1alpha1.Interface](x)
	if err != nil {
		return nil, err
	}
	return &obj{
		p: p,
	}, nil
}

type obj struct {
	p parser.Parser[*nephioreqv1alpha1.Interface]
}

// GetKubeObject returns the present kubeObject
func (r *obj) GetKubeObject() *fn.KubeObject {
	return r.p.GetKubeObject()
}

// GetGoStruct returns a go struct representing the present KRM resource
func (r *obj) GetGoStruct() (*nephioreqv1alpha1.Interface, error) {
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

// GetAttachmentType returns the attachmentType from the spec
// if an error occurs or the attribute is not present an empty string is returned
func (r *obj) GetAttachmentType() string {
	return r.p.GetStringValue(attachmentType...)
}

// GetCNIType returns the cniType from the spec
// if an error occurs or the attribute is not present an empty string is returned
func (r *obj) GetCNIType() string {
	return r.p.GetStringValue(cniType...)
}

// GetNetworkInstanceName returns the name of the networkInstance from the spec
// if an error occurs or the attribute is not present an empty string is returned
func (r *obj) GetNetworkInstanceName() string {
	return r.p.GetStringValue(networkInstanceName...)
}

// SetAttachmentType sets the attachmentType in the spec
func (r *obj) SetAttachmentType(s nephioreqv1alpha1.AttachmentType) error {
	return r.p.SetNestedString(string(s), attachmentType...)
}

// SetCNIType sets the cniType in the spec
func (r *obj) SetCNIType(s nephioreqv1alpha1.CNIType) error {
	return r.p.SetNestedString(string(s), cniType...)
}

// SetNetworkInstanceName sets the name of the networkInstance in the spec
func (r *obj) SetNetworkInstanceName(s string) error {
	return r.p.SetNestedString(s, networkInstanceName...)
}

// SetSpec sets the spec attributes in the kubeobject according the go struct
func (r *obj) SetSpec(spec *nephioreqv1alpha1.InterfaceSpec) error {
	if spec == nil {
		return nil
	}
	if spec.AttachmentType != "" {
		if err := r.SetAttachmentType(spec.AttachmentType); err != nil {
			return err
		}
	} else {
		if err := r.DeleteAttachmentType(); err != nil {
			return err
		}
	}
	if spec.CNIType != "" {
		if err := r.SetCNIType(spec.CNIType); err != nil {
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
	} else {
		return fmt.Errorf("networkInstance is required")
	}
	return nil
}

// DeleteAttachmentType deletes the attachmentType from the spec
func (r *obj) DeleteAttachmentType() error {
	return r.p.DeleteNestedField(attachmentType...)
}

// DeleteAttachmentType deletes the attachmentType from the spec
func (r *obj) DeleteCNIType() error {
	return r.p.DeleteNestedField(cniType...)
}
