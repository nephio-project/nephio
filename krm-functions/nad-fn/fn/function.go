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

package fn

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	infrav1alpha1 "github.com/nephio-project/api/infra/v1alpha1"
	nephioreqv1alpha1 "github.com/nephio-project/api/nf_requirements/v1alpha1"
	"github.com/nephio-project/nephio/krm-functions/lib/condkptsdk"
	ko "github.com/nephio-project/nephio/krm-functions/lib/kubeobject"
	nadlibv1 "github.com/nephio-project/nephio/krm-functions/lib/nad/v1"
	ipamv1alpha1 "github.com/nokia/k8s-ipam/apis/resource/ipam/v1alpha1"
	vlanv1alpha1 "github.com/nokia/k8s-ipam/apis/resource/vlan/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const defaultPODNetwork = "default"

type nadFn struct {
	sdk             condkptsdk.KptCondSDK
	workloadCluster *infrav1alpha1.WorkloadCluster
}

func Run(rl *fn.ResourceList) (bool, error) {
	myFn := nadFn{}
	var err error
	myFn.sdk, err = condkptsdk.New(
		rl,
		&condkptsdk.Config{
			For: corev1.ObjectReference{
				APIVersion: nadv1.SchemeGroupVersion.Identifier(),
				Kind:       reflect.TypeOf(nadv1.NetworkAttachmentDefinition{}).Name(),
			},
			Watch: map[corev1.ObjectReference]condkptsdk.WatchCallbackFn{
				{
					APIVersion: infrav1alpha1.GroupVersion.Identifier(),
					Kind:       reflect.TypeOf(infrav1alpha1.WorkloadCluster{}).Name(),
				}: myFn.WorkloadClusterCallbackFn,
				{
					APIVersion: ipamv1alpha1.GroupVersion.Identifier(),
					Kind:       ipamv1alpha1.IPClaimKind,
				}: nil,
				{
					APIVersion: vlanv1alpha1.GroupVersion.Identifier(),
					Kind:       vlanv1alpha1.VLANClaimKind,
				}: nil,
				{
					APIVersion: nephioreqv1alpha1.GroupVersion.Identifier(),
					Kind:       nephioreqv1alpha1.InterfaceKind,
				}: nil,
			},
			PopulateOwnResourcesFn: nil,
			UpdateResourceFn:       myFn.updateResourceFn,
		},
	)
	if err != nil {
		rl.Results.ErrorE(err)
		return false, err
	}
	return myFn.sdk.Run()
}

// WorkloadClusterCallbackFn provides a callback for the workload cluster
// resources in the resourceList
func (f *nadFn) WorkloadClusterCallbackFn(o *fn.KubeObject) error {
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

func (f *nadFn) updateResourceFn(_ *fn.KubeObject, objs fn.KubeObjects) (*fn.KubeObject, error) {
	if f.workloadCluster == nil {
		// no WorkloadCluster resource in the package
		return nil, fmt.Errorf("workload cluster is missing from the kpt package")
	}
	ipClaimObjs := objs.Where(fn.IsGroupVersionKind(ipamv1alpha1.IPClaimGroupVersionKind))
	vlanClaimObjs := objs.Where(fn.IsGroupVersionKind(vlanv1alpha1.VLANClaimGroupVersionKind))
	interfaceObjs := objs.Where(fn.IsGroupVersionKind(nephioreqv1alpha1.InterfaceGroupVersionKind))

	fn.Logf("nad updateResourceFn: ifObj: %d, ipClaimObj: %d, vlanClaimObj: %d\n", len(interfaceObjs), len(ipClaimObjs), len(vlanClaimObjs))
	// verify all needed objects exist
	if interfaceObjs.Len() == 0 {
		return nil, fmt.Errorf("expected %s object to generate the nad", nephioreqv1alpha1.InterfaceKind)
	}

	itfceKOE, err := ko.NewFromKubeObject[nephioreqv1alpha1.Interface](interfaceObjs[0])
	if err != nil {
		return nil, err
	}

	itfce, err := itfceKOE.GetGoStruct()
	if err != nil {
		return nil, err
	}

	// nothing to be done
	if itfce.Spec.NetworkInstance.Name == defaultPODNetwork {
		return nil, nil
	}

	if ipClaimObjs.Len() == 0 && vlanClaimObjs.Len() == 0 {
		return nil, fmt.Errorf("expected one of %s or %s objects to generate the nad", ipamv1alpha1.IPClaimKind, vlanv1alpha1.VLANClaimKind)
	}

	// generate an empty nad struct
	nad, err := nadlibv1.NewFromGoStruct(&nadv1.NetworkAttachmentDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: nadv1.SchemeGroupVersion.Identifier(),
			Kind:       reflect.TypeOf(nadv1.NetworkAttachmentDefinition{}).Name(),
		},
		ObjectMeta: metav1.ObjectMeta{Name: interfaceObjs[0].GetName()},
	})
	if err != nil {
		return nil, err
	}

	if ipClaimObjs.Len() == 0 && vlanClaimObjs.Len() != 0 {
		nad.CniSpecType = nadlibv1.VlanClaimOnly
	}
	if nad.CniSpecType != nadlibv1.VlanClaimOnly {
		for _, itfce := range interfaceObjs {
			i, err := ko.NewFromKubeObject[nephioreqv1alpha1.Interface](itfce)
			if err != nil {
				return nil, err
			}

			itfceGoStruct, err := i.GetGoStruct()
			if err != nil {
				return nil, err
			}

			if !f.IsCNITypePresent(itfceGoStruct.Spec.CNIType) {
				return nil, fmt.Errorf("cniType not supported in workload cluster; workload cluster CNI(s): %v, interface cniType requested: %s", f.workloadCluster.Spec.CNIs, itfceGoStruct.Spec.CNIType)
			}

			if err := nad.SetCNIType(string(itfceGoStruct.Spec.CNIType)); err != nil {
				return nil, err
			}
			err = nad.SetNadMaster(*f.workloadCluster.Spec.MasterInterface) // since we validated the workload cluster before it is safe to do this
			if err != nil {
				return nil, err
			}
		}

		var nadAddresses []nadlibv1.Addresses
		for _, ipClaim := range ipClaimObjs {
			claim, err := ko.NewFromKubeObject[ipamv1alpha1.IPClaim](ipClaim)
			if err != nil {
				return nil, err
			}

			ipclaimGoStruct, err := claim.GetGoStruct()
			if err != nil {
				return nil, err
			}
			address := ""
			gateway := ""
			if ipclaimGoStruct.Status.Prefix != nil {
				address = *ipclaimGoStruct.Status.Prefix
			}
			if ipclaimGoStruct.Status.Gateway != nil {
				gateway = *ipclaimGoStruct.Status.Gateway
			}
			if !contains(nadAddresses, address) {
				nadAddresses = append(nadAddresses, nadlibv1.Addresses{
					Address: address,
					Gateway: gateway,
				})
			}
		}
		sort.Slice(nadAddresses, func(i, j int) bool {
			return nadAddresses[i].Address < nadAddresses[j].Address
		})
		err = nad.SetIpamAddress(nadAddresses)
		if err != nil {
			return nil, err
		}
	}

	for _, vlanClaim := range vlanClaimObjs {
		vlanID, _, _ := vlanClaim.NestedInt([]string{"status", "vlanID"}...)
		err = nad.SetVlan(vlanID)
		if err != nil {
			return nil, err
		}
	}

	return &nad.K.KubeObject, nil
}

func (f *nadFn) IsCNITypePresent(itfceCNIType nephioreqv1alpha1.CNIType) bool {
	for _, cni := range f.workloadCluster.Spec.CNIs {
		if nephioreqv1alpha1.CNIType(cni) == itfceCNIType {
			return true
		}
	}
	return false
}

func contains(s []nadlibv1.Addresses, e string) bool {
	for _, a := range s {
		if a.Address == e {
			return true
		}
	}
	return false
}
