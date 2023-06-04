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
	"reflect"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	infrav1alpha1 "github.com/nephio-project/api/infra/v1alpha1"
	nephiodeployv1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	nephioreqv1alpha1 "github.com/nephio-project/api/nf_requirements/v1alpha1"
	kptcondsdk "github.com/nephio-project/nephio/krm-functions/lib/condkptsdk"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ko "github.com/nephio-project/nephio/krm-functions/lib/kubeobject"
)

func Run[T any, PT PtrIsNFDeployemnt[T]](rl *fn.ResourceList, gvk schema.GroupVersionKind) (bool, error) {
	nfDeployFn := NewFunction[T, PT](gvk)

	var err error

	kptfile := rl.Items.GetRootKptfile()
	if kptfile == nil {
		fn.Log("mandatory Kptfile is missing from the package")
		rl.Results.Errorf("mandatory Kptfile is missing from the package")
		return false, fmt.Errorf("mandatory Kptfile is missing from the package")
	}

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
					Kind:       reflect.TypeOf(infrav1alpha1.WorkloadCluster{}).Name(),
				}: nfDeployFn.WorkloadClusterCallbackFn,
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
			UpdateResourceFn: nfDeployFn.UpdateResourceFn,
		},
	)

	if err != nil {
		rl.Results.ErrorE(err)
		return false, err
	}

	return nfDeployFn.sdk.Run()
}

func (f *NfDeployFn[T, PT]) WorkloadClusterCallbackFn(o *fn.KubeObject) error {
	var err error

	if f.workloadCluster != nil {
		return fmt.Errorf("multiple WorkloadCluster objects found in the kpt package")
	}
	f.workloadCluster, err = ko.KubeObjectToStruct[infrav1alpha1.WorkloadCluster](o)
	if err != nil {
		return err
	}

	// validate check the specifics of the spec, like mandatory fields
	return f.workloadCluster.Spec.Validate()
}

func (f *NfDeployFn[T, PT]) CapacityContextCallBackFn(o *fn.KubeObject) error {
	capacityKOE, err := ko.NewFromKubeObject[nephioreqv1alpha1.Capacity](o)
	if err != nil {
		return err
	}

	capacity, err := capacityKOE.GetGoStruct()
	if err != nil {
		return err
	}

	f.capacity = capacity
	return nil
}

func (f *NfDeployFn[T, PT]) InterfaceCallBackFn(o *fn.KubeObject) error {
	itfcKOE, err := ko.NewFromKubeObject[nephioreqv1alpha1.Interface](o)
	if err != nil {
		return err
	}

	itfce, err := itfcKOE.GetGoStruct()
	if err != nil {
		return err
	}

	fn.Logf("Interface Spec validate:%v\n", itfce.Spec.Validate())
	// validate check the specifics of the spec, like mandatory fields
	if err := itfce.Spec.Validate(); err != nil {
		return err
	}

	itfcIPAllocStatus := itfce.Status.IPAllocationStatus
	itfcVlanAllocStatus := itfce.Status.VLANAllocationStatus

	// validate if status is not nil
	if itfcIPAllocStatus == nil || itfcVlanAllocStatus == nil {
		return nil
	}

	itfcConfig := nephiodeployv1alpha1.InterfaceConfig{
		Name: itfce.Name,
		IPv4: &nephiodeployv1alpha1.IPv4{
			Address: *itfcIPAllocStatus.Prefix,
			Gateway: itfcIPAllocStatus.Gateway,
		},
		VLANID: itfcVlanAllocStatus.VLANID,
	}

	f.SetInterfaceConfig(itfcConfig, itfce.Spec.NetworkInstance.Name)
	return nil
}

func (f *NfDeployFn[T, PT]) DnnCallBackFn(o *fn.KubeObject) error {
	dnnReqKOE, err := ko.NewFromKubeObject[nephioreqv1alpha1.DataNetwork](o)
	if err != nil {
		return err
	}

	dnnReq, err := dnnReqKOE.GetGoStruct()
	if err != nil {
		return err
	}

	if dnnReq.Status.Pools == nil {
		return nil
	}

	var pools []nephiodeployv1alpha1.Pool
	for _, pool := range dnnReq.Status.Pools {
		pools = append(pools, nephiodeployv1alpha1.Pool{Prefix: *pool.IPAllocation.Prefix})
	}

	dnn := nephiodeployv1alpha1.DataNetwork{
		Name: &dnnReq.Spec.NetworkInstance.Name,
		Pool: pools,
	}

	f.AddDNNToNetworkInstance(dnn, dnnReq.Spec.NetworkInstance.Name)

	return nil
}

func (f *NfDeployFn[T, PT]) UpdateResourceFn(nfDeploymentObj *fn.KubeObject, _ fn.KubeObjects) (*fn.KubeObject, error) {
	var err error

	if f.workloadCluster == nil {
		// no WorkloadCluster resource in the package
		return nil, fmt.Errorf("workload cluster is missing from the kpt package")
	}

	if nfDeploymentObj == nil {
		nfDeploymentObj = fn.NewEmptyKubeObject()

		err = nfDeploymentObj.SetAPIVersion(nephiodeployv1alpha1.GroupVersion.String())
		if err != nil {
			return nil, err
		}

		err = nfDeploymentObj.SetKind(f.gvk.Kind)
		if err != nil {
			return nil, err
		}

		err = nfDeploymentObj.SetName(f.pkgName)
		if err != nil {
			return nil, err
		}
	}

	nfKoExt, err := ko.NewFromKubeObject[T](nfDeploymentObj)
	if err != nil {
		return nil, err
	}

	var nf PT
	nf, err = nfKoExt.GetGoStruct()
	if err != nil {
		return nil, err
	}

	nfSpec := nf.GetNFDeploymentSpec()

	f.FillCapacityDetails(nfSpec)

	for networkInstanceName, itfceConfig := range f.interfaceConfigsMap {
		f.AddInterfaceToNetworkInstance(itfceConfig.Name, networkInstanceName)
	}

	nfSpec.Interfaces = f.GetAllInterfaceConfig()
	nfSpec.NetworkInstances = f.GetAllNetworkInstance()
	nf.SetNFDeploymentSpec(nfSpec)

	err = nfKoExt.SetSpec((*T)(nf))

	return &nfKoExt.KubeObject, err
}
