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
	"fmt"
	"reflect"
	"sort"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	infrav1alpha1 "github.com/nephio-project/api/infra/v1alpha1"
	nephioreqv1alpha1 "github.com/nephio-project/api/nf_requirements/v1alpha1"
	workloadv1alpha1 "github.com/nephio-project/api/workload/v1alpha1"
	"github.com/nephio-project/nephio/krm-functions/lib/condkptsdk"
	ko "github.com/nephio-project/nephio/krm-functions/lib/kubeobject"
	"github.com/nokia/k8s-ipam/pkg/iputil"
	corev1 "k8s.io/api/core/v1"
)

func Run(rl *fn.ResourceList) (bool, error) {
	nfDeployFn := NewFunction()

	var err error
	/*
		kptfile := rl.Items.GetRootKptfile()
		if kptfile == nil {
			fn.Log("mandatory Kptfile is missing from the package")
			rl.Results.Errorf("mandatory Kptfile is missing from the package")
			return false, fmt.Errorf("mandatory Kptfile is missing from the package")
		}

		nfDeployFn.pkgName = kptfile.GetName()
	*/

	nfDeployFn.sdk, err = condkptsdk.New(
		rl,
		&condkptsdk.Config{
			For: corev1.ObjectReference{
				APIVersion: workloadv1alpha1.GroupVersion.Identifier(),
				Kind:       workloadv1alpha1.NFDeploymentKind,
			},
			Owns: map[corev1.ObjectReference]condkptsdk.ResourceKind{
				{
					APIVersion: nephioreqv1alpha1.GroupVersion.Identifier(),
					Kind:       nephioreqv1alpha1.CapacityKind,
				}: condkptsdk.ChildLocal,
				{
					APIVersion: nephioreqv1alpha1.GroupVersion.Identifier(),
					Kind:       nephioreqv1alpha1.InterfaceKind,
				}: condkptsdk.ChildInitial,
				{
					APIVersion: nephioreqv1alpha1.GroupVersion.Identifier(),
					Kind:       nephioreqv1alpha1.DataNetworkKind,
				}: condkptsdk.ChildInitial,
				{
					APIVersion: nephioreqv1alpha1.GroupVersion.Identifier(),
					Kind:       nephioreqv1alpha1.DependencyKind,
				}: condkptsdk.ChildInitial,
			},
			Watch: map[corev1.ObjectReference]condkptsdk.WatchCallbackFn{
				{
					APIVersion: infrav1alpha1.GroupVersion.Identifier(),
					Kind:       reflect.TypeOf(infrav1alpha1.WorkloadCluster{}).Name(),
				}: nfDeployFn.WorkloadClusterCallbackFn,
			},
			PopulateOwnResourcesFn: nfDeployFn.desiredOwnedResourceList,
			UpdateResourceFn:       nfDeployFn.UpdateResourceFn,
			Root:                   true,
		},
	)

	if err != nil {
		rl.Results.ErrorE(err)
		return false, err
	}

	return nfDeployFn.sdk.Run()
}

func (f *NfDeployFn) WorkloadClusterCallbackFn(o *fn.KubeObject) error {
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

// desiredOwnedResourceList returns with the list of all child KubeObjects
// belonging to the parent Interface "for object"
func (f *NfDeployFn) desiredOwnedResourceList(o *fn.KubeObject) (fn.KubeObjects, error) {
	if f.workloadCluster == nil {
		// no WorkloadCluster resource in the package
		return nil, fmt.Errorf("workload cluster is missing from the kpt package")
	}
	return fn.KubeObjects{}, nil
}

func (f *NfDeployFn) CapacityUpdate(o *fn.KubeObject) error {
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

func (f *NfDeployFn) InterfaceUpdate(o *fn.KubeObject) error {
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

	itfcIPAllocStatus := itfce.Status.IPClaimStatus
	itfcVlanAllocStatus := itfce.Status.VLANClaimStatus

	// validate if status is not nil
	if itfcIPAllocStatus == nil || itfcVlanAllocStatus == nil {
		return nil
	}

	var ipv4 *workloadv1alpha1.IPv4
	var ipv6 *workloadv1alpha1.IPv6
	for _, ifStatus := range itfcIPAllocStatus {
		//fn.Logf("prefix prefix:%v\n", ifStatus)
		if ifStatus.Prefix != nil {
			pi, err := iputil.New(*ifStatus.Prefix)
			if err != nil {
				fn.Logf("prefix parsing:%v\n", err)
				return err
			}
			if pi.IsIpv6() {
				ipv6 = &workloadv1alpha1.IPv6{
					Address: *ifStatus.Prefix,
					Gateway: ifStatus.Gateway,
				}
			} else {
				ipv4 = &workloadv1alpha1.IPv4{
					Address: *ifStatus.Prefix,
					Gateway: ifStatus.Gateway,
				}
			}
		}
	}

	itfcConfig := workloadv1alpha1.InterfaceConfig{
		Name:   itfce.Name,
		IPv4:   ipv4,
		IPv6:   ipv6,
		VLANID: itfcVlanAllocStatus.VLANID,
	}

	f.SetInterfaceConfig(itfcConfig, itfce.Spec.NetworkInstance.Name)
	return nil
}

func (f *NfDeployFn) DnnUpdate(o *fn.KubeObject) error {
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

	pools := []workloadv1alpha1.Pool{}
	if len(dnnReq.Status.Pools) != 0 {
		for _, pool := range dnnReq.Status.Pools {
			if pool.IPClaim.Prefix != nil {
				pools = append(pools, workloadv1alpha1.Pool{Prefix: *pool.IPClaim.Prefix})
			}
		}
	}

	dnn := workloadv1alpha1.DataNetwork{
		Name: &dnnReq.Name,
		Pool: pools,
	}

	f.AddDNNToNetworkInstance(dnn, dnnReq.Spec.NetworkInstance.Name)

	return nil
}

func (f *NfDeployFn) DependencyUpdate(o *fn.KubeObject) error {
	depKOE, err := ko.NewFromKubeObject[nephioreqv1alpha1.Dependency](o)
	if err != nil {
		return err
	}

	dep, err := depKOE.GetGoStruct()
	if err != nil {
		return err
	}
	for _, ref := range dep.Status.Injected {
		f.AddDependencyRef(ref)
	}
	return nil
}

func (f *NfDeployFn) UpdateResourceFn(nfDeploymentObj *fn.KubeObject, objs fn.KubeObjects) (fn.KubeObjects, error) {
	if nfDeploymentObj == nil {
		return nil, fmt.Errorf("expected a for object but got nil")
	}

	var err error

	nfKoExt, err := ko.NewFromKubeObject[workloadv1alpha1.NFDeployment](nfDeploymentObj)
	if err != nil {
		return nil, err
	}

	var nf *workloadv1alpha1.NFDeployment
	nf, err = nfKoExt.GetGoStruct()
	if err != nil {
		return nil, err
	}

	capObjs := objs.Where(fn.IsGroupVersionKind(nephioreqv1alpha1.CapacityGroupVersionKind))
	for _, o := range capObjs {
		if err := f.CapacityUpdate(o); err != nil {
			return nil, err
		}
	}
	dnnObjs := objs.Where(fn.IsGroupVersionKind(nephioreqv1alpha1.DataNetworkGroupVersionKind))
	for _, o := range dnnObjs {
		if err := f.DnnUpdate(o); err != nil {
			return nil, err
		}
	}
	itfceObjs := objs.Where(fn.IsGroupVersionKind(nephioreqv1alpha1.InterfaceGroupVersionKind))
	for _, o := range itfceObjs {
		if err := f.InterfaceUpdate(o); err != nil {
			return nil, err
		}
	}

	depObjs := objs.Where(fn.IsGroupVersionKind(nephioreqv1alpha1.DependencyGroupVersionKind))
	for _, o := range depObjs {
		if err := f.DependencyUpdate(o); err != nil {
			return nil, err
		}
	}

	f.FillCapacityDetails(nf)

	if len(nf.Spec.ParametersRefs) == 0 {
		nf.Spec.ParametersRefs = f.paramRef
	} else {
		nf.Spec.ParametersRefs = append(nf.Spec.ParametersRefs, f.paramRef...)
	}

	//sort the paramRefs
	sort.Slice(nf.Spec.ParametersRefs, func(i, j int) bool {
		if nf.Spec.ParametersRefs[i].Name != nil && nf.Spec.ParametersRefs[j].Name != nil {
			return *nf.Spec.ParametersRefs[i].Name <= *nf.Spec.ParametersRefs[j].Name
		}
		return true
	})

	for networkInstanceName, itfceConfigs := range f.interfaceConfigsMap {
		for _, itfceConfig := range itfceConfigs {
			f.AddInterfaceToNetworkInstance(itfceConfig.Name, networkInstanceName)
		}

		//sort the added interfaces
		sort.Slice(f.networkInstance[networkInstanceName].Interfaces, func(i, j int) bool {
			return f.networkInstance[networkInstanceName].Interfaces[i] <= f.networkInstance[networkInstanceName].Interfaces[j]
		})
	}

	nf.Spec.Interfaces = f.GetAllInterfaceConfig()
	nf.Spec.NetworkInstances = f.GetAllNetworkInstance()

	err = nfKoExt.SetSpec(nf)

	return fn.KubeObjects{&nfKoExt.KubeObject}, err
}
