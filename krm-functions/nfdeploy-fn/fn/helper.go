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

package fn

import (
	"sort"

	infrav1alpha1 "github.com/nephio-project/api/infra/v1alpha1"
	nephioreqv1alpha1 "github.com/nephio-project/api/nf_requirements/v1alpha1"
	workloadv1alpha1 "github.com/nephio-project/api/workload/v1alpha1"
	kptcondsdk "github.com/nephio-project/nephio/krm-functions/lib/condkptsdk"
	corev1 "k8s.io/api/core/v1"
)

type NfDeployFn struct {
	sdk             kptcondsdk.KptCondSDK
	workloadCluster *infrav1alpha1.WorkloadCluster
	capacity        *nephioreqv1alpha1.Capacity
	//pkgName             string
	networkInstance     map[string]workloadv1alpha1.NetworkInstance
	interfaceConfigsMap map[string][]workloadv1alpha1.InterfaceConfig
	paramRef            []workloadv1alpha1.ObjectReference
}

func NewFunction() NfDeployFn {
	return NfDeployFn{
		interfaceConfigsMap: make(map[string][]workloadv1alpha1.InterfaceConfig),
		networkInstance:     make(map[string]workloadv1alpha1.NetworkInstance),
		paramRef:            []workloadv1alpha1.ObjectReference{},
	}
}

func (h *NfDeployFn) SetInterfaceConfig(interfaceConfig workloadv1alpha1.InterfaceConfig, networkInstanceName string) {
	// don't add to empty networkInstanceName, ideally should not happen
	if len(networkInstanceName) == 0 {
		return
	}

	if len(h.interfaceConfigsMap[networkInstanceName]) != 0 {
		h.interfaceConfigsMap[networkInstanceName] = append(h.interfaceConfigsMap[networkInstanceName], interfaceConfig)
	} else {
		h.interfaceConfigsMap[networkInstanceName] = []workloadv1alpha1.InterfaceConfig{interfaceConfig}
	}

}

func (h *NfDeployFn) AddDNNToNetworkInstance(dnn workloadv1alpha1.DataNetwork, networkInstanceName string) {
	if nI, ok := h.networkInstance[networkInstanceName]; ok {
		nI.DataNetworks = append(nI.DataNetworks, dnn)
		h.networkInstance[networkInstanceName] = nI
	} else {
		h.networkInstance[networkInstanceName] = workloadv1alpha1.NetworkInstance{
			Name:         networkInstanceName,
			DataNetworks: []workloadv1alpha1.DataNetwork{dnn},
		}
	}
}

func (h *NfDeployFn) AddInterfaceToNetworkInstance(interfaceName, networkInstanceName string) {
	if nI, ok := h.networkInstance[networkInstanceName]; ok {
		nI.Interfaces = append(nI.Interfaces, interfaceName)
		h.networkInstance[networkInstanceName] = nI
	} else {
		h.networkInstance[networkInstanceName] = workloadv1alpha1.NetworkInstance{
			Name:       networkInstanceName,
			Interfaces: []string{interfaceName},
		}
	}
}

func (h *NfDeployFn) GetAllNetworkInstance() []workloadv1alpha1.NetworkInstance {
	networkInstances := make([]workloadv1alpha1.NetworkInstance, 0)

	for _, nI := range h.networkInstance {
		networkInstances = append(networkInstances, nI)
	}

	// sort networkInstance based on name
	sort.Slice(networkInstances, func(i, j int) bool {
		return networkInstances[i].Name < networkInstances[j].Name
	})

	return networkInstances
}

func (h *NfDeployFn) GetAllInterfaceConfig() []workloadv1alpha1.InterfaceConfig {
	interfaceConfigs := make([]workloadv1alpha1.InterfaceConfig, 0)

	for _, ic := range h.interfaceConfigsMap {
		interfaceConfigs = append(interfaceConfigs, ic...)
	}

	//sort it based on resource names
	sort.Slice(interfaceConfigs, func(i, j int) bool {
		return interfaceConfigs[i].Name < interfaceConfigs[j].Name
	})

	return interfaceConfigs
}

func (h *NfDeployFn) FillCapacityDetails(nf *workloadv1alpha1.NFDeployment) {
	if h.capacity == nil {
		return
	}

	nf.Spec.Capacity = &h.capacity.Spec
}

func (h *NfDeployFn) AddDependencyRef(ref corev1.ObjectReference) {
	if !h.checkDependencyExist(ref) {
		h.paramRef = append(h.paramRef, workloadv1alpha1.ObjectReference{
			Name:       &ref.Name,
			Kind:       ref.Kind,
			APIVersion: ref.APIVersion,
		})
	}
}

func (h *NfDeployFn) checkDependencyExist(ref corev1.ObjectReference) bool {
	for _, p := range h.paramRef {
		if *p.Name == ref.Name && p.Kind == ref.Kind && p.APIVersion == ref.APIVersion {
			return true
		}
	}
	return false
}
