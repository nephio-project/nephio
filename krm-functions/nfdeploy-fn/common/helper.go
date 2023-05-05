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

package common

import (
	"sort"

	infrav1alpha1 "github.com/nephio-project/api/infra/v1alpha1"
	nephiodeployv1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	nephioreqv1alpha1 "github.com/nephio-project/api/nf_requirements/v1alpha1"
	kptcondsdk "github.com/nephio-project/nephio/krm-functions/lib/condkptsdk"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type PtrIsNFDeployemnt[T any] interface {
	*T
	nephiodeployv1alpha1.NFDeployment
}

type NfDeployFn[T any, PT PtrIsNFDeployemnt[T]] struct {
	sdk                 kptcondsdk.KptCondSDK
	clusterContext      *infrav1alpha1.ClusterContext
	gvk                 schema.GroupVersionKind
	capacity            *nephioreqv1alpha1.Capacity
	pkgName             string
	networkInstance     map[string]nephiodeployv1alpha1.NetworkInstance
	interfaceConfigsMap map[string]nephiodeployv1alpha1.InterfaceConfig
}

func NewMutator[T any, PT PtrIsNFDeployemnt[T]](gvk schema.GroupVersionKind) NfDeployFn[T, PT] {
	return NfDeployFn[T, PT]{
		interfaceConfigsMap: make(map[string]nephiodeployv1alpha1.InterfaceConfig),
		networkInstance:     make(map[string]nephiodeployv1alpha1.NetworkInstance),
		gvk:                 gvk,
	}
}

func (h *NfDeployFn[T, PT]) SetInterfaceConfig(interfaceConfig nephiodeployv1alpha1.InterfaceConfig, networkInstanceName string) {
	// dont add to empty networkInstanceName, ideally should not happen
	if len(networkInstanceName) == 0 {
		return
	}

	h.interfaceConfigsMap[networkInstanceName] = interfaceConfig
}

func (h *NfDeployFn[T, PT]) AddDNNToNetworkInstance(dnn nephiodeployv1alpha1.DataNetwork, networkInstanceName string) {
	if nI, ok := h.networkInstance[networkInstanceName]; ok {
		nI.DataNetworks = append(nI.DataNetworks, dnn)
		h.networkInstance[networkInstanceName] = nI
	} else {
		h.networkInstance[networkInstanceName] = nephiodeployv1alpha1.NetworkInstance{
			Name:         networkInstanceName,
			DataNetworks: []nephiodeployv1alpha1.DataNetwork{dnn},
		}
	}
}

func (h *NfDeployFn[T, PT]) AddInterfaceToNetworkInstance(interfaceName, networkInstanceName string) {
	if nI, ok := h.networkInstance[networkInstanceName]; ok {
		nI.Interfaces = append(nI.Interfaces, interfaceName)
		h.networkInstance[networkInstanceName] = nI
	} else {
		h.networkInstance[networkInstanceName] = nephiodeployv1alpha1.NetworkInstance{
			Name:       networkInstanceName,
			Interfaces: []string{interfaceName},
		}
	}

	return
}

func (h *NfDeployFn[T, PT]) GetAllNetworkInstance() []nephiodeployv1alpha1.NetworkInstance {
	networkInstances := make([]nephiodeployv1alpha1.NetworkInstance, 0)

	for _, nI := range h.networkInstance {
		networkInstances = append(networkInstances, nI)
	}

	// sort networkInstance based on name
	sort.Slice(networkInstances, func(i, j int) bool {
		return networkInstances[i].Name < networkInstances[j].Name
	})

	return networkInstances
}

func (h *NfDeployFn[T, PT]) GetAllInterfaceConfig() []nephiodeployv1alpha1.InterfaceConfig {
	interfaceConfigs := make([]nephiodeployv1alpha1.InterfaceConfig, 0)

	for _, ic := range h.interfaceConfigsMap {
		interfaceConfigs = append(interfaceConfigs, ic)
	}

	//sort it based on resource names
	sort.Slice(interfaceConfigs, func(i, j int) bool {
		return interfaceConfigs[i].Name < interfaceConfigs[j].Name
	})

	return interfaceConfigs
}

func (h *NfDeployFn[T, PT]) FillCapacityDetails(spec *nephiodeployv1alpha1.NFDeploymentSpec) {
	if spec.Capacity == nil {
		spec.Capacity = &nephioreqv1alpha1.CapacitySpec{}
	}

	if !h.capacity.Spec.MaxUplinkThroughput.IsZero() {
		spec.Capacity.MaxUplinkThroughput = h.capacity.Spec.MaxUplinkThroughput
	}

	if !h.capacity.Spec.MaxDownlinkThroughput.IsZero() {
		spec.Capacity.MaxDownlinkThroughput = h.capacity.Spec.MaxDownlinkThroughput
	}

	if h.capacity.Spec.MaxSessions != 0 {
		spec.Capacity.MaxSessions = h.capacity.Spec.MaxSessions
	}

	if h.capacity.Spec.MaxSubscribers != 0 {
		spec.Capacity.MaxSubscribers = h.capacity.Spec.MaxSubscribers
	}

	if h.capacity.Spec.MaxNFConnections != 0 {
		spec.Capacity.MaxNFConnections = h.capacity.Spec.MaxNFConnections
	}
}
