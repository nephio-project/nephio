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

package mutator

import (
	"fmt"
	"reflect"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	infrav1alpha1 "github.com/nephio-project/api/infra/v1alpha1"
	nephioreqv1alpha1 "github.com/nephio-project/api/nf_requirements/v1alpha1"
	"github.com/nephio-project/nephio/krm-functions/lib/condkptsdk"
	interfacelibv1alpha1 "github.com/nephio-project/nephio/krm-functions/lib/interface/v1alpha1"
	ipalloclibv1alpha1 "github.com/nephio-project/nephio/krm-functions/lib/ipalloc/v1alpha1"
	nadlibv1 "github.com/nephio-project/nephio/krm-functions/lib/nad/v1"
	ipamv1alpha1 "github.com/nokia/k8s-ipam/apis/alloc/ipam/v1alpha1"
	vlanv1alpha1 "github.com/nokia/k8s-ipam/apis/alloc/vlan/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mutatorCtx struct {
	fnCondSdk      condkptsdk.KptCondSDK
	clusterContext infrav1alpha1.ClusterContext
}

func Run(rl *fn.ResourceList) (bool, error) {
	m := mutatorCtx{}
	var err error
	m.fnCondSdk, err = condkptsdk.New(
		rl,
		&condkptsdk.Config{
			For: corev1.ObjectReference{
				APIVersion: nadv1.SchemeGroupVersion.Identifier(),
				Kind:       reflect.TypeOf(nadv1.NetworkAttachmentDefinition{}).Name(),
			},
			Watch: map[corev1.ObjectReference]condkptsdk.WatchCallbackFn{
				{
					APIVersion: infrav1alpha1.GroupVersion.Identifier(),
					Kind:       reflect.TypeOf(infrav1alpha1.ClusterContext{}).Name(),
				}: m.ClusterContextCallbackFn,
				{
					APIVersion: ipamv1alpha1.GroupVersion.Identifier(),
					Kind:       ipamv1alpha1.IPAllocationKind,
				}: nil,
				{
					APIVersion: vlanv1alpha1.GroupVersion.Identifier(),
					Kind:       vlanv1alpha1.VLANAllocationKind,
				}: nil,
				{
					APIVersion: nephioreqv1alpha1.GroupVersion.Identifier(),
					Kind:       nephioreqv1alpha1.InterfaceKind,
				}: nil,
			},
			PopulateOwnResourcesFn: nil,
			GenerateResourceFn:     m.generateResourceFn,
		},
	)
	if err != nil {
		rl.Results = append(rl.Results, fn.ErrorResult(err))
	}
	return m.fnCondSdk.Run()
}

func (r *mutatorCtx) ClusterContextCallbackFn(o *fn.KubeObject) error {
	var cluster infrav1alpha1.ClusterContext
	err := o.As(&cluster)
	if err != nil {
		return err
	}
	err = cluster.Spec.Validate()
	if err != nil {
		return err
	}
	r.clusterContext = cluster
	return nil
}

func (r *mutatorCtx) generateResourceFn(forObj *fn.KubeObject, objs fn.KubeObjects) (*fn.KubeObject, error) {
	ipAllocationObjs := objs.Where(fn.IsGroupVersionKind(ipamv1alpha1.IPAllocationGroupVersionKind))
	vlanAllocationObjs := objs.Where(fn.IsGroupVersionKind(vlanv1alpha1.VLANAllocationGroupVersionKind))
	interfaceObjs := objs.Where(fn.IsGroupVersionKind(nephioreqv1alpha1.InterfaceGroupVersionKind))

	// verify all needed objects exist
	if interfaceObjs.Len() == 0 {
		return nil, fmt.Errorf("expected %s object to generate the nad", nephioreqv1alpha1.InterfaceKind)
	}
	if ipAllocationObjs.Len() == 0 && vlanAllocationObjs.Len() == 0 {
		return nil, fmt.Errorf("expected one of %s or %s objects to generate the nad", ipamv1alpha1.IPAllocationKind, vlanv1alpha1.VLANAllocationKind)
	}
	if r.clusterContext.Spec.CNIConfig == nil {
		return nil, fmt.Errorf("expected ClusterContext object to generate the nad")
	}

	// generate an empty nad struct
	nad, err := nadlibv1.NewFromGoStruct(&nadv1.NetworkAttachmentDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: nadv1.SchemeGroupVersion.Identifier(),
			Kind:       reflect.TypeOf(nadv1.NetworkAttachmentDefinition{}).Name(),
		},
		ObjectMeta: metav1.ObjectMeta{Name: objs[0].GetName()},
	})
	if err != nil {
		return nil, err
	}

	if ipAllocationObjs.Len() == 0 && vlanAllocationObjs.Len() != 0 {
		nad.SetCniSpecType(nadlibv1.VlanAllocOnly)
	}
	if nad.GetCniSpecType() != nadlibv1.VlanAllocOnly {
		for _, itfce := range interfaceObjs {
			i, err := interfacelibv1alpha1.NewFromKubeObject(itfce)
			if err != nil {
				return nil, err
			}
			interfaceGoStruct, err := i.GetGoStruct()
			if err != nil {
				return nil, err
			}
			if r.clusterContext.Spec.CNIConfig.CNIType == "" {
				err = nad.SetCNIType(string(interfaceGoStruct.Spec.CNIType))
			} else if r.clusterContext.Spec.CNIConfig.CNIType == string(interfaceGoStruct.Spec.CNIType) {
				err = nad.SetCNIType(string(interfaceGoStruct.Spec.CNIType))
			} else {
				return nil, fmt.Errorf("CNIType mismatch between interface and clustercontext")
			}
			if err != nil {
				return nil, err
			}
			err = nad.SetNadMaster(r.clusterContext.Spec.CNIConfig.MasterInterface)
			if err != nil {
				return nil, err
			}
		}

		for _, ipAllocation := range ipAllocationObjs {
			alloc, err := ipalloclibv1alpha1.NewFromKubeObject(ipAllocation)
			if err != nil {
				return nil, err
			}
			allocGoStruct, err := alloc.GetGoStruct()
			if err != nil {
				return nil, err
			}
			err = nad.SetIpamAddress([]nadlibv1.Addresses{{
				Address: allocGoStruct.Status.AllocatedPrefix,
				Gateway: allocGoStruct.Status.Gateway,
			}})
			if err != nil {
				return nil, err
			}
		}
	}

	for _, vlanAllocation := range vlanAllocationObjs {
		vlanID, _, _ := vlanAllocation.NestedInt([]string{"status", "vlanID"}...)
		err = nad.SetVlan(vlanID)
		if err != nil {
			return nil, err
		}
	}

	return &nad.K.KubeObject, nil
}
