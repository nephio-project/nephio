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
	"strconv"

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
	Type            string       `json:"type,omitempty"`
	Capabilities    Capabilities `json:"capabilities,omitempty"`
	Master          string       `json:"master,omitempty"`
	Mode            string       `json:"mode,omitempty"`
	Ipam            Ipam         `json:"ipam,omitempty"`
	VlanId          int          `json:"vlanId,omitempty"`          // cniType vlan
	LinkInContainer bool         `json:"linkInContainer,omitempty"` // cniType vlan
	Bridge          string       `json:"bridge,omitempty"`          // cniType bridge
	Vlan            int          `json:"vlan,omitempty"`            // cniType bridge
	VlanTrunk       []VlanTrunk  `json:"vlanTrunk,omitempty"`       // cniType bridge
	Name            string       `json:"name,omitempty"`            // cniType bridge
}

type Capabilities struct {
	Ips bool `json:"ips,omitempty"`
	Mac bool `json:"mac,omitempty"`
}

type Ipam struct {
	Type      string    `json:"type,omitempty"`
	Addresses []Address `json:"addresses,omitempty"`
	Routes    []Route   `json:"routes,omitempty"`
}

type Address struct {
	Address string `json:"address,omitempty"`
	Gateway string `json:"gateway,omitempty"`
}

type Route struct {
	Destination string `json:"dst,omitempty"`
	Gateway     string `json:"gw,omitempty"`
}

type VlanTrunk struct {
	MinID int `json:"minID,omitempty"`
	MaxID int `json:"maxID,omitempty"`
	ID    int `json:"id,omitempty"`
}

type CniSpecType int64

const (
	OtherType CniSpecType = iota // 0
	VlanClaimOnly
	IpVlanType
	SriovType
	MacVlanType
	VlanType
	BridgeType
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
		if len(nadConfigStruct.Plugins) == 0 {
			nadConfigStruct.Plugins = []PluginCniType{
				{
					Capabilities: Capabilities{Ips: true},
					Mode:         ModeL2,
					Ipam: Ipam{
						Type: StaticNadType,
					},
				},
			}
		}
		return nadConfigStruct, nil
	case MacVlanType:
		if len(nadConfigStruct.Plugins) == 0 {
			nadConfigStruct.Plugins = []PluginCniType{
				{
					Capabilities: Capabilities{Ips: true},
					Mode:         ModeBridge,
					Ipam: Ipam{
						Type: StaticNadType,
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
		if len(nadConfigStruct.Plugins) == 0 {
			nadConfigStruct.Plugins = []PluginCniType{
				{
					Capabilities: Capabilities{
						Ips: true,
					},
					Mode: ModeBridge,
					Ipam: Ipam{
						Type: StaticNadType,
					},
				},
			}
		}
		return nadConfigStruct, nil
	case VlanType:
		if len(nadConfigStruct.Plugins) == 0 {
			nadConfigStruct.Plugins = []PluginCniType{
				{
					Ipam: Ipam{
						Type: StaticNadType,
					},
				},
			}
		}
		return nadConfigStruct, nil
	case BridgeType:
		if len(nadConfigStruct.Plugins) == 0 {
			nadConfigStruct.Plugins = []PluginCniType{
				{
					Ipam: Ipam{
						Type: StaticNadType,
					},
				},
			}
		}
		return nadConfigStruct, nil
	case OtherType:
		if len(nadConfigStruct.Plugins) == 0 {
			nadConfigStruct.Plugins = []PluginCniType{
				{
					Capabilities: Capabilities{
						Ips: true,
					},
					Mode: ModeBridge,
					Ipam: Ipam{
						Type: StaticNadType,
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

func (r *NadStruct) GetIpamAddress() ([]Address, error) {
	existingNadConfig, err := r.getNadConfig()
	if err != nil {
		return []Address{}, err
	}
	for _, plugin := range existingNadConfig.Plugins {
		if plugin.Type == TuningType {
			continue
		} else {
			return plugin.Ipam.Addresses, nil
		}
	}
	return []Address{}, nil
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
	case "vlan":
		r.CniSpecType = VlanType
	case "bridge":
		r.CniSpecType = BridgeType
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

func (r *NadStruct) SetVlanID(vlanID int) error {
	if vlanID == 0 {
		return fmt.Errorf("unknown vlanID")
	} else {
		nadConfigStruct, err := r.getNadConfig()
		if err != nil {
			return err
		}
		for i, plugin := range nadConfigStruct.Plugins {
			if plugin.Type == TuningType {
				continue
			} else {
				nadConfigStruct.Plugins[i].VlanId = vlanID
			}
		}
		return r.setNadConfig(nadConfigStruct)
	}
}

func (r *NadStruct) SetBridgeVlan(vlanID int) error {
	if vlanID == 0 {
		return fmt.Errorf("unknown vlanID")
	} else {
		nadConfigStruct, err := r.getNadConfig()
		if err != nil {
			return err
		}
		for i, plugin := range nadConfigStruct.Plugins {
			if plugin.Type == TuningType {
				continue
			} else {
				nadConfigStruct.Plugins[i].Vlan = vlanID
			}
		}
		return r.setNadConfig(nadConfigStruct)
	}
}

func (r *NadStruct) SetBridgeTrunk(vlanID int) error {
	if vlanID == 0 {
		return fmt.Errorf("unknown vlanID")
	} else {
		nadConfigStruct, err := r.getNadConfig()
		if err != nil {
			return err
		}
		for i, plugin := range nadConfigStruct.Plugins {
			if plugin.Type == TuningType {
				continue
			} else {
				nadConfigStruct.Plugins[i].VlanTrunk = []VlanTrunk{
					{
						ID: vlanID,
					},
				}
			}
		}
		return r.setNadConfig(nadConfigStruct)
	}
}

func (r *NadStruct) SetBridgeName(vlanID int) error {
	if vlanID == 0 {
		return fmt.Errorf("unknown vlanID")
	} else {
		nadConfigStruct, err := r.getNadConfig()
		if err != nil {
			return err
		}
		for i, plugin := range nadConfigStruct.Plugins {
			if plugin.Type == TuningType {
				continue
			} else {
				nadConfigStruct.Plugins[i].Bridge = fmt.Sprintf("cni%s", strconv.Itoa(vlanID))
				nadConfigStruct.Plugins[i].Name = fmt.Sprintf("cni%s", strconv.Itoa(vlanID))
			}
		}
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

func (r *NadStruct) SetIpamAddress(addresses []Address) error {
	if addresses == nil {
		return fmt.Errorf("unknown IPAM addresses")
	} else {
		nadConfigStruct, err := r.getNadConfig()
		if err != nil {
			return err
		}
		for i, plugin := range nadConfigStruct.Plugins {
			if plugin.Type == TuningType {
				continue
			} else {
				nadConfigStruct.Plugins[i].Ipam.Addresses = addresses
			}
		}
		return r.setNadConfig(nadConfigStruct)
	}
}

func (r *NadStruct) SetIpamRoutes(routes []Route) error {
	if routes == nil {
		return nil
	} else {
		nadConfigStruct, err := r.getNadConfig()
		if err != nil {
			return err
		}
		for i, plugin := range nadConfigStruct.Plugins {
			if plugin.Type == TuningType {
				continue
			} else {
				nadConfigStruct.Plugins[i].Ipam.Routes = routes
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
