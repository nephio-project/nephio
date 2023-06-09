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
	CniVersion    = "0.3.1"
	ModeBridge    = "bridge"
	ModeL2        = "l2"
	StaticNadType = "static"
	TuningType    = "tuning"
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

type CniSpecType int64

const (
	OtherType CniSpecType = iota // 0
	VlanClaimOnly
	IpVlanType
	SriovType
	MacVlanType
)

type NadStruct struct {
	K           kubeobject.KubeObjectExt[nadv1.NetworkAttachmentDefinition]
	CniSpecType CniSpecType
}

// NewFromKubeObject creates a new parser interface
// It expects a *fn.KubeObject as input representing the serialized yaml file
func NewFromKubeObject(b *fn.KubeObject) (*NadStruct, error) {
	p, err := kubeobject.NewFromKubeObject[nadv1.NetworkAttachmentDefinition](b)
	if err != nil {
		return nil, err
	}
	return &NadStruct{K: *p}, nil
}

// NewFromYAML creates a new parser interface
// It expects a raw byte slice as input representing the serialized yaml file
func NewFromYAML(b []byte) (*NadStruct, error) {
	p, err := kubeobject.NewFromYaml[nadv1.NetworkAttachmentDefinition](b)
	if err != nil {
		return nil, err
	}
	return &NadStruct{K: *p}, nil
}

// NewFromGoStruct creates a new parser interface
// It expects a go struct representing the interface krm resource
func NewFromGoStruct(b *nadv1.NetworkAttachmentDefinition) (*NadStruct, error) {
	p, err := kubeobject.NewFromGoStruct(b)
	if err != nil {
		return nil, err
	}
	return &NadStruct{K: *p}, nil
}

func (r *NadStruct) getNadConfig() (NadConfig, error) {
	nadConfigStruct := NadConfig{}
	configSpec := r.GetConfigSpec()
	if configSpec == "" {
		configSpec = "{}"
	}
	if err := json.Unmarshal([]byte(configSpec), &nadConfigStruct); err != nil {
		return nadConfigStruct, fmt.Errorf("invalid NAD Config, %s", err)
	}
	nadConfigStruct.CniVersion = CniVersion
	switch cniSpecType := r.CniSpecType; cniSpecType {
	case VlanClaimOnly:
		return nadConfigStruct, nil
	case IpVlanType:
		if nadConfigStruct.Plugins == nil || len(nadConfigStruct.Plugins) == 0 {
			nadConfigStruct.Plugins = []PluginCniType{
				{
					Capabilities: Capabilities{
						Ips: true,
					},
					Mode: ModeL2,
					Ipam: Ipam{
						Type: StaticNadType,
						Addresses: []Addresses{
							{},
						},
					},
				},
			}
		}
		return nadConfigStruct, nil
	case MacVlanType:
		if nadConfigStruct.Plugins == nil || len(nadConfigStruct.Plugins) == 0 {
			nadConfigStruct.Plugins = []PluginCniType{
				{
					Capabilities: Capabilities{Ips: true},
					Mode:         ModeBridge,
					Ipam: Ipam{
						Type: StaticNadType,
						Addresses: []Addresses{
							{},
						},
					},
				},
				{
					Capabilities: Capabilities{
						Mac: true,
					},
					Type: TuningType,
				},
			}
		}
		return nadConfigStruct, nil
	case SriovType:
		if nadConfigStruct.Plugins == nil || len(nadConfigStruct.Plugins) == 0 {
			nadConfigStruct.Plugins = []PluginCniType{
				{
					Capabilities: Capabilities{
						Ips: true,
					},
					Mode: ModeBridge,
					Ipam: Ipam{
						Type: StaticNadType,
						Addresses: []Addresses{
							{},
						},
					},
				},
			}
		}
		return nadConfigStruct, nil
	case OtherType:
		if nadConfigStruct.Plugins == nil || len(nadConfigStruct.Plugins) == 0 {
			nadConfigStruct.Plugins = []PluginCniType{
				{
					Capabilities: Capabilities{
						Ips: true,
					},
					Mode: ModeBridge,
					Ipam: Ipam{
						Type: StaticNadType,
						Addresses: []Addresses{
							{},
						},
					},
				},
			}
		}
		return nadConfigStruct, nil
	}
	return nadConfigStruct, nil
}

func (r *NadStruct) setNadConfig(config NadConfig) error {
	b, err := json.Marshal(config)
	if err != nil {
		return err
	}
	return r.K.SetNestedString(string(b), ConfigType...)
}

func (r *NadStruct) getStringValue(fields ...string) string {
	s, ok, err := r.K.NestedString(fields...)
	if err != nil {
		return ""
	}
	if !ok {
		return ""
	}
	return s
}

func (r *NadStruct) GetConfigSpec() string {
	return r.getStringValue(ConfigType...)
}

func (r *NadStruct) GetCNIType() (string, error) {
	existingNadConfig, err := r.getNadConfig()
	if err != nil {
		return "", err
	}
	for _, plugin := range existingNadConfig.Plugins {
		if plugin.Type == TuningType {
			continue
		} else {
			return plugin.Type, nil
		}
	}
	return "", nil
}

func (r *NadStruct) GetVlan() (int, error) {
	existingNadConfig, err := r.getNadConfig()
	if err != nil {
		return 0, err
	}
	return existingNadConfig.Vlan, nil
}

func (r *NadStruct) GetNadMaster() (string, error) {
	existingNadConfig, err := r.getNadConfig()
	if err != nil {
		return "", err
	}
	for _, plugin := range existingNadConfig.Plugins {
		if plugin.Type == TuningType {
			continue
		} else {
			return plugin.Master, nil
		}
	}
	return "", nil
}

func (r *NadStruct) GetIpamAddress() ([]Addresses, error) {
	existingNadConfig, err := r.getNadConfig()
	if err != nil {
		return []Addresses{}, err
	}
	for _, plugin := range existingNadConfig.Plugins {
		if plugin.Type == TuningType {
			continue
		} else {
			return plugin.Ipam.Addresses, nil
		}
	}
	return []Addresses{}, nil
}

// SetConfigSpec sets the spec attributes in the kubeobject according the go struct
func (r *NadStruct) SetConfigSpec(spec *nadv1.NetworkAttachmentDefinitionSpec) error {
	return r.K.SetNestedString(spec.Config, ConfigType...)
}

func (r *NadStruct) SetCNIType(cniType string) error {
	switch cniSpecType := cniType; cniSpecType {
	case "":
		return fmt.Errorf("unknown cniType")
	case "ipvlan":
		r.CniSpecType = IpVlanType
	case "macvlan":
		r.CniSpecType = MacVlanType
	case "sriov":
		r.CniSpecType = SriovType
	}
	nadConfigStruct, err := r.getNadConfig()
	if err != nil {
		return err
	}
	for i, plugin := range nadConfigStruct.Plugins {
		if plugin.Type == TuningType {
			continue
		} else {
			nadConfigStruct.Plugins[i].Type = cniType
		}
	}
	return r.setNadConfig(nadConfigStruct)
}

func (r *NadStruct) SetVlan(vlanType int) error {
	if vlanType == 0 {
		return fmt.Errorf("unknown vlanType")
	} else {
		nadConfigStruct, err := r.getNadConfig()
		if err != nil {
			return err
		}
		nadConfigStruct.Vlan = vlanType
		return r.setNadConfig(nadConfigStruct)
	}
}

func (r *NadStruct) SetNadMaster(nadMaster string) error {
	if nadMaster == "" {
		return fmt.Errorf("unknown nad master interface")
	} else {
		nadConfigStruct, err := r.getNadConfig()
		if err != nil {
			return err
		}
		for i, plugin := range nadConfigStruct.Plugins {
			if plugin.Type == TuningType {
				continue
			} else {
				nadConfigStruct.Plugins[i].Master = nadMaster
			}
		}
		return r.setNadConfig(nadConfigStruct)
	}
}

func (r *NadStruct) SetIpamAddress(ipam []Addresses) error {
	if ipam == nil {
		return fmt.Errorf("unknown IPAM address")
	} else {
		nadConfigStruct, err := r.getNadConfig()
		if err != nil {
			return err
		}
		for i, plugin := range nadConfigStruct.Plugins {
			if plugin.Type == TuningType {
				continue
			} else {
				nadConfigStruct.Plugins[i].Ipam.Addresses = ipam
			}
		}
		return r.setNadConfig(nadConfigStruct)
	}
}

func (n *NadConfig) String() (string, error) {
	b, err := json.Marshal(n)
	if err != nil {
		return "", fmt.Errorf("to string conversion error: %s", err)
	}
	return string(b), nil
}
