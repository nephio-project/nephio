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
	"fmt"
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	nephiodeployv1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	nephioreqv1alpha1 "github.com/nephio-project/api/nf_requirements/v1alpha1"
	infrav1alpha1 "github.com/nephio-project/nephio-controller-poc/apis/infra/v1alpha1"
	kptcondsdk "github.com/nephio-project/nephio/krm-functions/lib/condkptsdk"
	"github.com/nephio-project/nephio/krm-functions/lib/kubeobject"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"reflect"
)

type NfType interface {
	*nephiodeployv1alpha1.UPFDeployment | *nephiodeployv1alpha1.AMFDeployment | *nephiodeployv1alpha1.SMFDeployment
}

func Run[T NfType](rl *fn.ResourceList, gvk schema.GroupVersionKind) (bool, error) {
	nfDeployFn := NewMutator[T](gvk)

	var err error

	kptfile := rl.Items.GetRootKptfile()
	nfDeployFn.pkgName = kptfile.GetName()

	nfDeployFn.sdk, err = kptcondsdk.New(
		rl,
		&kptcondsdk.Config{
			For: corev1.ObjectReference{
				APIVersion: nephiodeployv1alpha1.GroupVersion.Identifier(),
				Kind:       nfDeployFn.gvk.Kind,
			},
			Watch: map[corev1.ObjectReference]kptcondsdk.WatchCallbackFn{
				{
					APIVersion: infrav1alpha1.GroupVersion.Identifier(),
					Kind:       reflect.TypeOf(infrav1alpha1.ClusterContext{}).Name(),
				}: nfDeployFn.ClusterContextCallBackFn,
				{
					APIVersion: nephioreqv1alpha1.GroupVersion.Identifier(),
					Kind:       reflect.TypeOf(nephioreqv1alpha1.Capacity{}).Name(),
				}: nfDeployFn.CapacityContextCallBackFn,
				{
					APIVersion: nephioreqv1alpha1.GroupVersion.Identifier(),
					Kind:       nephioreqv1alpha1.InterfaceKind,
				}: nfDeployFn.InterfaceCallBackFn,
				{
					APIVersion: nephioreqv1alpha1.GroupVersion.Identifier(),
					Kind:       nephioreqv1alpha1.DataNetworkKind,
				}: nfDeployFn.DnnCallBackFn,
			},
			GenerateResourceFn: nfDeployFn.GenerateResourceFn,
		},
	)

	if err != nil {
		rl.Results.ErrorE(err)
		return false, err
	}

	return nfDeployFn.sdk.Run()
}

func (h *NfDeployFn[T1]) ClusterContextCallBackFn(o *fn.KubeObject) error {
	var cluster infrav1alpha1.ClusterContext
	err := o.As(&cluster)
	if err != nil {
		return err
	}
	h.site = *cluster.Spec.SiteCode
	return nil
}

func (h *NfDeployFn[T1]) CapacityContextCallBackFn(o *fn.KubeObject) error {
	var capacity nephioreqv1alpha1.Capacity
	err := o.As(&capacity)
	if err != nil {
		return err
	}

	h.capacity = capacity
	return nil
}

func (h *NfDeployFn[T1]) InterfaceCallBackFn(o *fn.KubeObject) error {
	var itfce *nephioreqv1alpha1.Interface
	err := o.As(&itfce)
	if err != nil {
		return err
	}

	if itfce.Status.IPAllocationStatus == nil || itfce.Status.VLANAllocationStatus == nil {
		return nil
	}

	itfcIPAllocStatus := itfce.Status.IPAllocationStatus
	itfcVlanAllocStatus := itfce.Status.VLANAllocationStatus

	itfcConfig := nephiodeployv1alpha1.InterfaceConfig{
		Name: itfce.Name,
		IPv4: &nephiodeployv1alpha1.IPv4{
			Address: itfcIPAllocStatus.AllocatedPrefix,
			Gateway: &itfcIPAllocStatus.Gateway,
		},
		VLANID: &itfcVlanAllocStatus.AllocatedVlanID,
	}

	h.SetInterfaceConfig(itfcConfig, itfce.Spec.NetworkInstance.Name)
	return nil
}

func (h *NfDeployFn[T1]) DnnCallBackFn(o *fn.KubeObject) error {
	var dnnReq nephioreqv1alpha1.DataNetwork
	err := o.As(&dnnReq)
	if err != nil {
		return err
	}

	if dnnReq.Status.Pools == nil {
		return nil
	}

	var pools []nephiodeployv1alpha1.Pool
	for _, pool := range dnnReq.Status.Pools {
		pools = append(pools, nephiodeployv1alpha1.Pool{Prefix: pool.IPAllocation.AllocatedPrefix})
	}

	dnn := nephiodeployv1alpha1.DataNetwork{
		Name: &dnnReq.Spec.NetworkInstance.Name,
		Pool: pools,
	}

	h.AddDNNToNetworkInstance(dnn, dnnReq.Spec.NetworkInstance.Name)

	return nil
}

func (h *NfDeployFn[T1]) GenerateResourceFn(nfDeploymentObj *fn.KubeObject, _ fn.KubeObjects) (*fn.KubeObject, error) {
	var err error

	if nfDeploymentObj == nil {
		nfDeploymentObj = fn.NewEmptyKubeObject()

		err = nfDeploymentObj.SetAPIVersion(nephiodeployv1alpha1.GroupVersion.String())
		if err != nil {
			return nil, err
		}

		err = nfDeploymentObj.SetKind(h.gvk.Kind)
		if err != nil {
			return nil, err
		}

		err = nfDeploymentObj.SetName(h.pkgName)
		if err != nil {
			return nil, err
		}
	}

	nfKoExt, err := kubeobject.NewFromKubeObject[T1](nfDeploymentObj)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	nfSpec := &nephiodeployv1alpha1.NFDeploymentSpec{}

	h.FillCapacityDetails(nfSpec)

	if err != nil {
		return nil, err
	}

	for networkInstanceName, itfceConfig := range h.interfaceConfigsMap {
		h.AddInterfaceToNetworkInstance(itfceConfig.Name, networkInstanceName)
	}

	nfSpec.Interfaces = h.GetAllInterfaceConfig()
	nfSpec.NetworkInstances = h.GetAllNetworkInstance()

	err = nfKoExt.SetSpec(nfSpec)

	nf, _ := nfKoExt.GetGoStruct()
	fmt.Println(nf)
	return &nfKoExt.KubeObject, err
}
