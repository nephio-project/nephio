package mutator

import (
	nephiodeployv1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	kptcondsdk "github.com/nephio-project/nephio/krm-functions/lib/condkptsdk"
	"k8s.io/apimachinery/pkg/api/resource"
)

type NfDeployFn struct {
	sdk                           kptcondsdk.KptCondSDK
	site                          string
	capacityMaxUpLinkThroughPut   resource.Quantity
	capacityMaxDownLinkThroughPut resource.Quantity
	networkInstance               map[string]nephiodeployv1alpha1.NetworkInstance
	interfaceConfigsMap           map[string]nephiodeployv1alpha1.InterfaceConfig
}

func NewMutatorContext() NfDeployFn {
	return NfDeployFn{
		interfaceConfigsMap: make(map[string]nephiodeployv1alpha1.InterfaceConfig),
		networkInstance:     make(map[string]nephiodeployv1alpha1.NetworkInstance),
	}
}

func (h *NfDeployFn) SetInterfaceConfig(interfaceConfig nephiodeployv1alpha1.InterfaceConfig, networkInstanceName string) {
	h.interfaceConfigsMap[networkInstanceName] = interfaceConfig
}

func (h *NfDeployFn) SetCapacity(maxDownLinkThroughPut, maxUpLinkThrouput resource.Quantity) {
	h.capacityMaxDownLinkThroughPut = maxDownLinkThroughPut
	h.capacityMaxUpLinkThroughPut = maxUpLinkThrouput
}

func (h *NfDeployFn) AddDNNToNetworkInstance(dnn nephiodeployv1alpha1.DataNetwork, networkInstanceName string) {
	dnns := h.GetNetworkInstance(networkInstanceName).DataNetworks
	dnns = append(dnns, dnn)

	netWorkInstance := h.GetNetworkInstance(networkInstanceName)
	netWorkInstance.DataNetworks = dnns
	h.networkInstance[networkInstanceName] = netWorkInstance
}

func (h *NfDeployFn) AddInterfaceToNetworkInstance(interfaceName, networkInstanceName string) {
	networkInstance := h.GetNetworkInstance(networkInstanceName)

	networkInstance.Interfaces = append(networkInstance.Interfaces, interfaceName)
}

func (h *NfDeployFn) GetNetworkInstance(networkInstanceName string) nephiodeployv1alpha1.NetworkInstance {
	if v, ok := h.networkInstance[networkInstanceName]; ok {
		return v
	}
	h.networkInstance[networkInstanceName] = nephiodeployv1alpha1.NetworkInstance{
		Name: networkInstanceName,
	}
	return h.networkInstance[networkInstanceName]
}

func (h *NfDeployFn) GetAllNetworkInstance() []nephiodeployv1alpha1.NetworkInstance {
	networkInstances := make([]nephiodeployv1alpha1.NetworkInstance, 0)

	for _, nI := range h.networkInstance {
		networkInstances = append(networkInstances, nI)
	}

	return networkInstances
}

func (h *NfDeployFn) GetAllInterfaceConfig() []nephiodeployv1alpha1.InterfaceConfig {
	interfaceConfigs := make([]nephiodeployv1alpha1.InterfaceConfig, 0)

	for _, ic := range h.GetInterfaceConfigMap() {
		interfaceConfigs = append(interfaceConfigs, ic)
	}

	return interfaceConfigs
}

func (h *NfDeployFn) GetInterfaceConfigMap() map[string]nephiodeployv1alpha1.InterfaceConfig {
	return h.interfaceConfigsMap
}

func (h *NfDeployFn) GetDNNs(networkInstanceName string) []nephiodeployv1alpha1.DataNetwork {
	return h.networkInstance[networkInstanceName].DataNetworks
}
