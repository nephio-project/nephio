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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type PtrIsNFDeployment[T any] interface {
	*T
	nephiodeployv1alpha1.NFDeployment
}

type NfDeployFn[T any, PT PtrIsNFDeployment[T]] struct {
	sdk             kptcondsdk.KptCondSDK
	workloadCluster *infrav1alpha1.WorkloadCluster
	gvk             schema.GroupVersionKind
	capacity        *nephioreqv1alpha1.Capacity
	//pkgName             string
	networkInstance     map[string]nephiodeployv1alpha1.NetworkInstance
	interfaceConfigsMap map[string]nephiodeployv1alpha1.InterfaceConfig
	configRefs          []corev1.ObjectReference
}

func NewFunction[T any, PT PtrIsNFDeployment[T]](gvk schema.GroupVersionKind) NfDeployFn[T, PT] {
	return NfDeployFn[T, PT]{
		interfaceConfigsMap: make(map[string]nephiodeployv1alpha1.InterfaceConfig),
		networkInstance:     make(map[string]nephiodeployv1alpha1.NetworkInstance),
		configRefs:          []corev1.ObjectReference{},
		gvk:                 gvk,
	}
}

func (h *NfDeployFn[T, PT]) SetInterfaceConfig(interfaceConfig nephiodeployv1alpha1.InterfaceConfig, networkInstanceName string) {
	// don't add to empty networkInstanceName, ideally should not happen
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
	if h.capacity == nil {
		return
	}

	spec.Capacity = &h.capacity.Spec
}

func (h *NfDeployFn[T, PT]) AddDependencyRef(ref corev1.ObjectReference) {
	h.configRefs = append(h.configRefs, ref)
}
