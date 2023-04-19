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
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	"github.com/nephio-project/nephio/krm-functions/lib/parser"
)

const (
	// errors
	errKubeObjectNotInitialized = "KubeObject not initialized"
	CniVersion                  = "0.3.1"
	NadMode                     = "bridge"
	NadType                     = "static"
)

var (
	ConfigType = []string{"spec", "config"}
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
	parser.Parser[*nadv1.NetworkAttachmentDefinition]
	// GetKubeObject returns the present kubeObject
	GetKubeObject() *fn.KubeObject
	// GetConfigSpec returns the  spec
	// if an error occurs or the attribute is not present an empty string is returned
	GetConfigSpec() string
	// GetCNIType returns the cniType from the spec
	// if an error occurs or the attribute is not present an empty string is returned
	GetCNIType() string
	// GetVlan get the name of the Vlan in the spec
	GetVlan() int
	// GetNadMaster get the name of the NAD Master interface in the spec
	GetNadMaster() string
	// GetIpamAddress get the name of the NAD IPAM addresses in the spec
	GetIpamAddress() []Addresses
	// SetConfigSpec sets the spec attributes in the kubeobject according the go struct
	SetConfigSpec(s *nadv1.NetworkAttachmentDefinitionSpec) error
	// SetCNIType sets the cniType in the spec
	SetCNIType(s string) error
	// SetVlan sets the name of the Vlan in the spec
	SetVlan(s int) error
	// SetNadMaster sets the master interface in the spec
	SetNadMaster(s string) error
	// SetIpamAddress sets the name of the IPAM info in the spec
	SetIpamAddress(a []Addresses) error
}

// NewFromKubeObject creates a new parser interface
// It expects a *fn.KubeObject as input representing the serialized yaml file
func NewFromKubeObject(o *fn.KubeObject) Nad {
	return &nadStruct{
		p: parser.NewFromKubeObject[*nadv1.NetworkAttachmentDefinition](o),
	}
}

// NewFromYAML creates a new parser interface
// It expects a raw byte slice as input representing the serialized yaml file
func NewFromYAML(b []byte) (Nad, error) {
	p, err := parser.NewFromYaml[*nadv1.NetworkAttachmentDefinition](b)
	if err != nil {
		return nil, err
	}
	return &nadStruct{
		p: p,
	}, nil
}

// NewFromGoStruct creates a new parser interface
// It expects a go struct representing the interface krm resource
func NewFromGoStruct(x *nadv1.NetworkAttachmentDefinition) (Nad, error) {
	p, err := parser.NewFromGoStruct[*nadv1.NetworkAttachmentDefinition](x)
	if err != nil {
		return nil, err
	}
	return &nadStruct{
		p: p,
	}, nil
}

type nadStruct struct {
	p parser.Parser[*nadv1.NetworkAttachmentDefinition]
}

// GetKubeObject returns the present kubeObject
func (r *nadStruct) GetKubeObject() *fn.KubeObject {
	return r.p.GetKubeObject()
}

// GetGoStruct returns a go struct representing the present KRM resource
func (r *nadStruct) GetGoStruct() (*nadv1.NetworkAttachmentDefinition, error) {
	return r.p.GetGoStruct()
}

func (r *nadStruct) GetStringValue(fields ...string) string {
	return r.p.GetStringValue()
}

func (r *nadStruct) GetBoolValue(fields ...string) bool {
	return r.p.GetBoolValue()
}

func (r *nadStruct) GetIntValue(fields ...string) int {
	return r.p.GetIntValue()
}

func (r *nadStruct) GetStringMap(fields ...string) map[string]string {
	return r.p.GetStringMap()
}

func (r *nadStruct) SetNestedString(s string, fields ...string) error {
	return r.p.SetNestedString(s, fields...)
}

func (r *nadStruct) SetNestedInt(s int, fields ...string) error {
	return r.p.SetNestedInt(s, fields...)
}

func (r *nadStruct) SetNestedBool(s bool, fields ...string) error {
	return r.p.SetNestedBool(s, fields...)
}

func (r *nadStruct) SetNestedMap(s map[string]string, fields ...string) error {
	return r.p.SetNestedMap(s, fields...)
}

func (r *nadStruct) DeleteNestedField(fields ...string) error {
	return r.p.DeleteNestedField(fields...)
}

// GetConfigSpec gets the spec attributes in the kubeobject according the go struct
func (r *nadStruct) GetConfigSpec() string {
	return r.p.GetStringValue(ConfigType...)
}

func (r *nadStruct) GetCNIType() string {
	nadConfigStruct := NadConfig{}
	if err := json.Unmarshal([]byte(r.p.GetStringValue(ConfigType...)), &nadConfigStruct); err != nil {
		return ""
	}
	return nadConfigStruct.Plugins[0].Type
}

func (r *nadStruct) GetVlan() int {
	nadConfigStruct := NadConfig{}
	if err := json.Unmarshal([]byte(r.p.GetStringValue(ConfigType...)), &nadConfigStruct); err != nil {
		return 0
	}
	return nadConfigStruct.Vlan
}

func (r *nadStruct) GetNadMaster() string {
	nadConfigStruct := NadConfig{}
	if err := json.Unmarshal([]byte(r.p.GetStringValue(ConfigType...)), &nadConfigStruct); err != nil {
		return ""
	}
	return nadConfigStruct.Plugins[0].Master
}

func (r *nadStruct) GetIpamAddress() []Addresses {
	nadConfigStruct := NadConfig{}
	if err := json.Unmarshal([]byte(r.p.GetStringValue(ConfigType...)), &nadConfigStruct); err != nil {
		return []Addresses{}
	}
	return nadConfigStruct.Plugins[0].Ipam.Addresses
}

// SetConfigSpec sets the spec attributes in the kubeobject according the go struct
func (r *nadStruct) SetConfigSpec(spec *nadv1.NetworkAttachmentDefinitionSpec) error {
	b, err := json.Marshal(spec.Config)
	if err != nil {
		return err
	}
	return r.p.SetNestedString(string(b), ConfigType...)
}

func (r *nadStruct) SetCNIType(cnfType string) error {
	if cnfType != "" {
		nadConfigStruct := NadConfig{}
		if err := json.Unmarshal([]byte(r.p.GetStringValue(ConfigType...)), &nadConfigStruct); err != nil {
			panic(err)
		}
		nadConfigStruct.Plugins[0].Type = cnfType
		b, err := json.Marshal(nadConfigStruct)
		if err != nil {
			return err
		}
		return r.p.SetNestedString(string(b), ConfigType...)
	} else {
		return fmt.Errorf("unknown cniType")
	}
}

func (r *nadStruct) SetVlan(vlanType int) error {
	if vlanType != 0 {
		nadConfigStruct := NadConfig{}
		if err := json.Unmarshal([]byte(r.p.GetStringValue(ConfigType...)), &nadConfigStruct); err != nil {
			panic(err)
		}
		nadConfigStruct.Vlan = vlanType
		b, err := json.Marshal(nadConfigStruct)
		if err != nil {
			return err
		}
		return r.p.SetNestedString(string(b), ConfigType...)
	} else {
		return fmt.Errorf("unknown vlanType")
	}
}

func (r *nadStruct) SetNadMaster(nadMaster string) error {
	if nadMaster != "" {
		nadConfigStruct := NadConfig{}
		if err := json.Unmarshal([]byte(r.p.GetStringValue(ConfigType...)), &nadConfigStruct); err != nil {
			panic(err)
		}
		nadConfigStruct.Plugins[0].Master = nadMaster
		b, err := json.Marshal(nadConfigStruct)
		if err != nil {
			return err
		}
		return r.p.SetNestedString(string(b), ConfigType...)
	} else {
		return fmt.Errorf("unknown cniType")
	}
}

func (r *nadStruct) SetIpamAddress(ipam []Addresses) error {
	if ipam != nil {
		nadConfigStruct := NadConfig{}
		if err := json.Unmarshal([]byte(r.p.GetStringValue(ConfigType...)), &nadConfigStruct); err != nil {
			panic(err)
		}
		nadConfigStruct.Plugins[0].Ipam.Addresses = ipam
		b, err := json.Marshal(nadConfigStruct)
		if err != nil {
			return err
		}
		return r.p.SetNestedString(string(b), ConfigType...)
	} else {
		return fmt.Errorf("unknown cniType")
	}
}

func (n *NadConfig) ToString() string {
	b, err := json.Marshal(n)
	if err != nil {
		panic(err)
	}
	return string(b)
}
