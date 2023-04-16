/*
Copyright 2023 Nephio.

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

package v1

import (
	"encoding/json"
	"fmt"
	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	"reflect"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const (
	// errors
	errKubeObjectNotInitialized = "KubeObject not initialized"
	CniVersion                  = "0.3.1"
	NadMode                     = "bridge"
	NadType                     = "static"
)

// NetworkAttachmentDefinition serves as model for NAD yaml
type NetworkAttachmentDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec NetworkAttachmentDefinitionSpec `json:"spec"`
}

// NetworkAttachmentDefinitionSpec serves as model for NAD yaml
// NetworkAttachmentDefinitionSpec is defined as config string on orginal lib
type NetworkAttachmentDefinitionSpec struct {
	Config NadConfig `json:"config"`
}

var (
	ConfigType = []string{"spec", "config"}
	NadSpec    = []string{"spec"}
)

type NadConfig struct {
	CniVersion string          `json:"cniVersion"`
	Vlan       int             `json:"vlan"`
	Plugins    []PluginCniType `json:"plugins"`
}

type PluginCniType struct {
	Type         string       `json:"type"`
	Capabilities Capabilities `json:"capabilities"`
	Master       string       `json:"master"`
	Mode         string       `json:"mode"`
	Ipam         Ipam         `json:"ipam"`
}

type Capabilities struct {
	Ips bool `json:"ips"`
	Mac bool `json:"mac"`
}

type Ipam struct {
	Type      string      `json:"type"`
	Addresses []Addresses `json:"addresses"`
}

type Addresses struct {
	Address string `json:"address"`
	Gateway string `json:"gateway"`
}

type Nad interface {
	// GetKubeObject returns the present kubeObject
	GetKubeObject() *fn.KubeObject
	// GetGoStruct returns a go struct representing the present KRM resource
	GetGoStruct() (*NetworkAttachmentDefinition, error)
	// GetSpec returns the  spec
	// if an error occurs or the attribute is not present an empty string is returned
	GetSpec() *NetworkAttachmentDefinitionSpec
	// GetCNIType returns the cniType from the spec
	// if an error occurs or the attribute is not present an empty string is returned
	GetCNIType() string
	// GetVlan get the name of the Vlan in the spec
	GetVlan() int
	// GetNadMaster get the name of the NAD Master interface in the spec
	GetNadMaster() string
	// GetIpamAddress get the name of the NAD IPAM addresses in the spec
	GetIpamAddress() []Addresses
	// SetSpec sets the spec attributes in the kubeobject according the go struct
	SetSpec(s *NetworkAttachmentDefinitionSpec) error
	// SetCNIType sets the cniType in the spec
	SetCNIType(s string) error
	// SetVlanType sets the name of the Vlan in the spec
	SetVlanType(s int) error
}

// NewFromYAML creates a new parser interface
// It expects a raw byte slice as input representing the serialized yaml file
func NewFromYAML(b []byte) (Nad, error) {
	o, err := fn.ParseKubeObject(b)
	if err != nil {
		return nil, err
	}
	return &nadStruct{
		o: o,
	}, nil
}

// NewFromGoStruct creates a new parser interface
// It expects a go struct representing the interface krm resource
func NewFromGoStruct(x *NetworkAttachmentDefinition) (Nad, error) {
	b, err := yaml.Marshal(x)
	if err != nil {
		return nil, err
	}
	return NewFromYAML(b)
}

type nadStruct struct {
	o *fn.KubeObject
}

// GetKubeObject returns the present kubeObject
func (r *nadStruct) GetKubeObject() *fn.KubeObject {
	return r.o
}

// GetGoStruct returns a go struct representing the present KRM resource
func (r *nadStruct) GetGoStruct() (*NetworkAttachmentDefinition, error) {
	x := &NetworkAttachmentDefinition{}
	if err := yaml.Unmarshal([]byte(r.o.String()), x); err != nil {
		return nil, err
	}
	return x, nil
}

// GetSpec gets the spec attributes in the kubeobject according the go struct
func (r *nadStruct) GetSpec() *NetworkAttachmentDefinitionSpec {
	nadConfigStruct := NetworkAttachmentDefinitionSpec{}
	if err := json.Unmarshal([]byte(r.getStringValue(NadSpec...)), &nadConfigStruct); err != nil {
		panic(err)
	}
	return &nadConfigStruct
}

func (r *nadStruct) GetCNIType() string {
	nadConfigStruct := NadConfig{}
	if err := json.Unmarshal([]byte(r.getStringValue(ConfigType...)), &nadConfigStruct); err != nil {
		panic(err)
	}
	return nadConfigStruct.Plugins[0].Type
}

func (r *nadStruct) GetVlan() int {
	nadConfigStruct := NadConfig{}
	if err := json.Unmarshal([]byte(r.getStringValue(ConfigType...)), &nadConfigStruct); err != nil {
		panic(err)
	}
	return nadConfigStruct.Vlan
}

func (r *nadStruct) GetNadMaster() string {
	nadConfigStruct := NadConfig{}
	if err := json.Unmarshal([]byte(r.getStringValue(ConfigType...)), &nadConfigStruct); err != nil {
		panic(err)
	}
	return nadConfigStruct.Plugins[0].Master
}

func (r *nadStruct) GetIpamAddress() []Addresses {
	nadConfigStruct := NadConfig{}
	if err := json.Unmarshal([]byte(r.getStringValue(ConfigType...)), &nadConfigStruct); err != nil {
		panic(err)
	}
	return nadConfigStruct.Plugins[0].Ipam.Addresses
}

// SetSpec sets the spec attributes in the kubeobject according the go struct
func (r *nadStruct) SetSpec(spec *NetworkAttachmentDefinitionSpec) error {
	return r.o.SetNestedField(spec.Config, ConfigType...)
}

func (r *nadStruct) SetCNIType(cnfType string) error {
	if cnfType != "" {
		nadConfigStruct := NadConfig{}
		if err := json.Unmarshal([]byte(r.getStringValue(ConfigType...)), &nadConfigStruct); err != nil {
			panic(err)
		}
		nadConfigStruct.Plugins[0].Type = cnfType
		return r.o.SetNestedField(nadConfigStruct, ConfigType...)
	} else {
		return fmt.Errorf("unknown cniType")
	}
}

func (r *nadStruct) SetVlanType(vlanType int) error {
	if vlanType != 0 {
		nadConfigStruct := NadConfig{}
		if err := json.Unmarshal([]byte(r.getStringValue(ConfigType...)), &nadConfigStruct); err != nil {
			panic(err)
		}
		nadConfigStruct.Vlan = vlanType
		return r.o.SetNestedField(nadConfigStruct, ConfigType...)
	} else {
		return fmt.Errorf("unknown vlanType")
	}
}

func (r *nadStruct) SetNadMaster(nadMaster string) error {
	if nadMaster != "" {
		nadConfigStruct := NadConfig{}
		if err := json.Unmarshal([]byte(r.getStringValue(ConfigType...)), &nadConfigStruct); err != nil {
			panic(err)
		}
		nadConfigStruct.Plugins[0].Master = nadMaster
		return r.o.SetNestedField(nadConfigStruct, ConfigType...)
	} else {
		return fmt.Errorf("unknown cniType")
	}
}

func (r *nadStruct) SetNadIpam(ipam []Addresses) error {
	if ipam != nil {
		nadConfigStruct := NadConfig{}
		if err := json.Unmarshal([]byte(r.getStringValue(ConfigType...)), &nadConfigStruct); err != nil {
			panic(err)
		}
		nadConfigStruct.Plugins[0].Ipam.Addresses = ipam
		return r.o.SetNestedField(nadConfigStruct, ConfigType...)
	} else {
		return fmt.Errorf("unknown cniType")
	}
}

// getStringValue is a generic utility function that returns a string from
// a string slice representing the path in the yaml doc
func (r *nadStruct) getStringValue(fields ...string) string {
	if r.o == nil {
		return ""
	}
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
func (r *nadStruct) setNestedField(s string, fields ...string) error {
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
func (r *nadStruct) deleteNestedField(fields ...string) error {
	if r.o == nil {
		return fmt.Errorf(errKubeObjectNotInitialized)
	}
	_, err := r.o.RemoveNestedField(fields...)
	if err != nil {
		return err
	}
	return nil
}

type NAD interface {
	ParseKubeObject() (*fn.KubeObject, error)
}

// NewGenerator creates a new generator for the nad
// It expects a raw byte slice as input representing the serialized yaml file
func NewGenerator(meta metav1.ObjectMeta, spec NetworkAttachmentDefinitionSpec) NAD {
	return &nad{
		meta: meta,
		spec: spec,
	}
}

type nad struct {
	meta metav1.ObjectMeta
	spec NetworkAttachmentDefinitionSpec
}

func (r *nad) ParseKubeObject() (*fn.KubeObject, error) {
	ipa := &NetworkAttachmentDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: nadv1.SchemeGroupVersion.Identifier(),
			Kind:       reflect.TypeOf(NetworkAttachmentDefinition{}).Name(),
		},
		ObjectMeta: r.meta,
		Spec:       r.spec,
	}
	b, err := yaml.Marshal(ipa)
	if err != nil {
		return nil, err
	}
	return fn.ParseKubeObject(b)
}
