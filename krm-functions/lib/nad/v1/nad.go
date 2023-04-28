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
	"github.com/nephio-project/nephio/krm-functions/lib/kubeobject"
)

const (
	errKubeObjectNotInitialized = "KubeObject not initialized"
	CniVersion                  = "0.3.1"
	NadMode                     = "bridge"
	NadType                     = "static"
)

var (
	ConfigType = []string{"spec", "config"}
)

type NadConfig struct {
	CniVersion string          `json:"cniVersion,omitempty"`
	Vlan       int             `json:"vlan,omitempty"`
	Plugins    []PluginCniType `json:"plugins,omitempty"`
}

type PluginCniType struct {
	Type         string       `json:"type,omitempty"`
	Capabilities Capabilities `json:"capabilities,omitempty"`
	Master       string       `json:"master,omitempty"`
	Mode         string       `json:"mode,omitempty"`
	Ipam         Ipam         `json:"ipam,omitempty"`
}

type Capabilities struct {
	Ips bool `json:"ips,omitempty"`
	Mac bool `json:"mac,omitempty"`
}

type Ipam struct {
	Type      string      `json:"type,omitempty"`
	Addresses []Addresses `json:"addresses,omitempty"`
}

type Addresses struct {
	Address string `json:"address,omitempty"`
	Gateway string `json:"gateway,omitempty"`
}

type Nad interface {
	// GetKubeObject returns the present kubeObject
	GetKubeObject() *fn.KubeObject
	// GetGoStruct returns the goStruct kubeObject
	GetGoStruct() (*nadv1.NetworkAttachmentDefinition, error)
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

// NewFromYAML creates a new parser interface
// It expects a raw byte slice as input representing the serialized yaml file
func NewFromYAML(b []byte) (Nad, error) {
	p, err := kubeobject.NewFromYaml[*nadv1.NetworkAttachmentDefinition](b)
	if err != nil {
		return nil, err
	}
	return &nadStruct{
		p: *p,
	}, nil
}

// NewFromGoStruct creates a new parser interface
// It expects a go struct representing the interface krm resource
func NewFromGoStruct(x *nadv1.NetworkAttachmentDefinition) (Nad, error) {
	p, err := kubeobject.NewFromGoStruct[*nadv1.NetworkAttachmentDefinition](x)
	if err != nil {
		return nil, err
	}
	return &nadStruct{
		p: *p,
	}, nil
}

type nadStruct struct {
	p kubeobject.KubeObjectExt[*nadv1.NetworkAttachmentDefinition]
}

func (r *nadStruct) GetKubeObject() *fn.KubeObject {
	return &r.p.KubeObject
}

func (r *nadStruct) GetGoStruct() (*nadv1.NetworkAttachmentDefinition, error) {
	return r.p.GetGoStruct()
}

func (r *nadStruct) GetStringValue(fields ...string) string {
	if r == nil {
		return ""
	}
	s, ok, err := r.p.NestedString(fields...)
	if err != nil {
		return ""
	}
	if !ok {
		return ""
	}
	return s
}

func (r *nadStruct) GetConfigSpec() string {
	if r == nil {
		return ""
	}
	s, ok, err := r.p.NestedString(ConfigType...)
	if err != nil {
		return ""
	}
	if !ok {
		return ""
	}
	return s
}

func (r *nadStruct) GetCNIType() string {
	nadConfigStruct := NadConfig{}
	if err := json.Unmarshal([]byte(r.GetStringValue(ConfigType...)), &nadConfigStruct); err != nil {
		return ""
	}
	return nadConfigStruct.Plugins[0].Type
}

func (r *nadStruct) GetVlan() int {
	nadConfigStruct := NadConfig{}
	if err := json.Unmarshal([]byte(r.GetStringValue(ConfigType...)), &nadConfigStruct); err != nil {
		return 0
	}
	return nadConfigStruct.Vlan
}

func (r *nadStruct) GetNadMaster() string {
	nadConfigStruct := NadConfig{}
	if err := json.Unmarshal([]byte(r.GetStringValue(ConfigType...)), &nadConfigStruct); err != nil {
		return ""
	}
	return nadConfigStruct.Plugins[0].Master
}

func (r *nadStruct) GetIpamAddress() []Addresses {
	nadConfigStruct := NadConfig{}
	if err := json.Unmarshal([]byte(r.GetStringValue(ConfigType...)), &nadConfigStruct); err != nil {
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

func (r *nadStruct) GetNadConfig() NadConfig {
	nadConfigStruct := NadConfig{}
	configSpec := r.GetStringValue(ConfigType...)
	if configSpec == "" {
		configSpec = "{}"
	}
	if err := json.Unmarshal([]byte(configSpec), &nadConfigStruct); err != nil {
		panic(err)
	}
	if nadConfigStruct.Plugins == nil || len(nadConfigStruct.Plugins) == 0 {
		nadConfigStruct.Plugins = []PluginCniType{
			{
				Capabilities: Capabilities{
					Ips: true,
				},
				Mode: NadMode,
				Ipam: Ipam{
					Type: NadType,
					Addresses: []Addresses{
						{},
					},
				},
			},
		}
	}
	nadConfigStruct.CniVersion = CniVersion
	return nadConfigStruct
}

func (r *nadStruct) SetCNIType(cniType string) error {
	if cniType != "" {
		nadConfigStruct := r.GetNadConfig()
		nadConfigStruct.Plugins[0].Type = cniType
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
		nadConfigStruct := r.GetNadConfig()
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
		nadConfigStruct := r.GetNadConfig()
		nadConfigStruct.Plugins[0].Master = nadMaster
		b, err := json.Marshal(nadConfigStruct)
		if err != nil {
			return err
		}
		return r.p.SetNestedString(string(b), ConfigType...)
	} else {
		return fmt.Errorf("unknown nad master interface")
	}
}

func (r *nadStruct) SetIpamAddress(ipam []Addresses) error {
	if ipam != nil {
		nadConfigStruct := r.GetNadConfig()
		nadConfigStruct.Plugins[0].Ipam.Addresses = ipam
		b, err := json.Marshal(nadConfigStruct)
		if err != nil {
			return err
		}
		return r.p.SetNestedString(string(b), ConfigType...)
	} else {
		return fmt.Errorf("unknown IPAM address")
	}
}

func (n *NadConfig) ToString() string {
	b, err := json.Marshal(n)
	if err != nil {
		panic(err)
	}
	return string(b)
}
