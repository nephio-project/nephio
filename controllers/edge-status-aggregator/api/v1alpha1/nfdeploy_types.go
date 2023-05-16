/*
Copyright 2022-2023 The Nephio Authors.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Plmn struct {
	MCC int `json:"mcc,omitempty" yaml:"mcc,omitempty"`
	MNC int `json:"mnc,omitempty" yaml:"mnc,omitempty"`
}

type Connectivity struct {
	NeighborName string `json:"neighborName,omitempty" yaml:"neighborName,omitempty"`
}

type Site struct {
	Id             string         `json:"id,omitempty" yaml:"id,omitempty"`
	ClusterName    string         `json:"clusterName,omitempty" yaml:"clusterName,omitempty"`
	NFType         string         `json:"nfType,omitempty" yaml:"nfType,omitempty"`
	NFTypeName     string         `json:"nfTypeName,omitempty" yaml:"nfTypeName,omitempty"`
	NFVendor       string         `json:"nfVendor,omitempty" yaml:"nfVendor,omitempty"`
	NFVersion      string         `json:"nfVersion,omitempty" yaml:"nfVersion,omitempty"`
	IPAddrBlock    []string       `json:"ipAddrBlock,omitempty" yaml:"ipAddrBlock,omitempty"`
	Connectivities []Connectivity `json:"connectivities,omitempty" yaml:"connectivities,omitempty"`
}

// NfDeploySpec defines the desired state of NfDeploy
type NfDeploySpec struct {
	Plmn     Plmn   `json:"plmn,omitempty" yaml:"plmn,omitempty"`
	Capacity string `json:"capacity,omitempty" yaml:"capacity,omitempty"`
	Sites    []Site `json:"sites,omitempty" yaml:"sites,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// NfDeploy is the Schema for the nfdeploys API
type NfDeploy struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata"`

	Spec   NfDeploySpec   `json:"spec,omitempty"`
	Status NfDeployStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NfDeployList contains a list of NfDeploy
type NfDeployList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NfDeploy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NfDeploy{}, &NfDeployList{})
}
