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

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	infrav1alpha1 "github.com/nephio-project/api/infra/v1alpha1"
	nephioreqv1alpha1 "github.com/nephio-project/api/nf_requirements/v1alpha1"
	"github.com/nephio-project/nephio/krm-functions/lib/condkptsdk"
	"github.com/nephio-project/nephio/krm-functions/lib/kubeobject"
	allocv1alpha1 "github.com/nokia/k8s-ipam/apis/alloc/common/v1alpha1"
	ipamv1alpha1 "github.com/nokia/k8s-ipam/apis/alloc/ipam/v1alpha1"
	vlanv1alpha1 "github.com/nokia/k8s-ipam/apis/alloc/vlan/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	vlanlibv1alpha1 "github.com/nephio-project/nephio/krm-functions/lib/vlanalloc/v1alpha1"
)

const defaultPODNetwork = "default"

type itfceFn struct {
	sdk condkptsdk.KptCondSDK
	//clusterContext *infrav1alpha1.ClusterContext
	siteCode string
	cniType  string
}

func Run(rl *fn.ResourceList) (bool, error) {
	myFn := itfceFn{}
	var err error
	myFn.sdk, err = condkptsdk.New(
		rl,
		&condkptsdk.Config{
			For: corev1.ObjectReference{
				APIVersion: nephioreqv1alpha1.GroupVersion.Identifier(),
				Kind:       nephioreqv1alpha1.InterfaceKind,
			},
			Owns: map[corev1.ObjectReference]condkptsdk.ResourceKind{
				{
					APIVersion: nadv1.SchemeGroupVersion.Identifier(),
					Kind:       reflect.TypeOf(nadv1.NetworkAttachmentDefinition{}).Name(),
				}: condkptsdk.ChildRemoteCondition,
				{
					APIVersion: ipamv1alpha1.GroupVersion.Identifier(),
					Kind:       ipamv1alpha1.IPAllocationKind,
				}: condkptsdk.ChildRemote,
				{
					APIVersion: vlanv1alpha1.GroupVersion.Identifier(),
					Kind:       vlanv1alpha1.VLANAllocationKind,
				}: condkptsdk.ChildRemote,
			},
			Watch: map[corev1.ObjectReference]condkptsdk.WatchCallbackFn{
				{
					APIVersion: infrav1alpha1.GroupVersion.Identifier(),
					Kind:       reflect.TypeOf(infrav1alpha1.ClusterContext{}).Name(),
				}: myFn.ClusterContextCallbackFn,
			},
			PopulateOwnResourcesFn: myFn.desiredOwnedResourceList,
			GenerateResourceFn:     myFn.updateItfceResource,
		},
	)
	if err != nil {
		rl.Results.ErrorE(err)
		return false, nil
	}
	return myFn.sdk.Run()
}

// ClusterContextCallbackFn provides a callback for the cluster context
// resources in the resourceList
func (r *itfceFn) ClusterContextCallbackFn(o *fn.KubeObject) error {
	clusterKOE, err := kubeobject.NewFromKubeObject[infrav1alpha1.ClusterContext](o)
	if err != nil {
		return err
	}
	clusterContext, err := clusterKOE.GetGoStruct()
	if err != nil {
		return err
	}

	fn.Logf("clusterctxt validate:%v\n", clusterContext.Spec.Validate())
	// validate check the specifics of the spec, like mandatory fields
	if err := clusterContext.Spec.Validate(); err != nil {
		return err
	}
	r.siteCode = *clusterContext.Spec.SiteCode
	r.cniType = clusterContext.Spec.CNIConfig.CNIType
	return nil
}

// desiredOwnedResourceList returns with the list of all child KubeObjects
// belonging to the parent Interface "for object"
func (r *itfceFn) desiredOwnedResourceList(o *fn.KubeObject) (fn.KubeObjects, error) {
	// resources contain the list of child resources
	// belonging to the parent object
	resources := fn.KubeObjects{}

	itfceKOE, err := kubeobject.NewFromKubeObject[nephioreqv1alpha1.Interface](o)
	if err != nil {
		return nil, err
	}

	itfce, err := itfceKOE.GetGoStruct()
	if err != nil {
		return nil, err
	}

	// Nothing to be done in case the interface is attached to
	// the default pod network since this is all handled in the
	// k8s cluster via the CNI.
	if itfce.Spec.NetworkInstance.Name == defaultPODNetwork {
		return fn.KubeObjects{}, nil
	}

	// meta is the generic object meta attached to all derived child objects
	meta := metav1.ObjectMeta{
		Name: o.GetName(),
	}
	// When the CNIType is not set this is a loopback interface
	if itfce.Spec.CNIType != "" {
		if itfce.Spec.CNIType != nephioreqv1alpha1.CNIType(r.cniType) {
			return nil, fmt.Errorf("cluster cniType not supported: cluster cniType: %s, interface cniType: %s", r.cniType, itfce.Spec.CNIType)
		}
		// add IP allocation of type network
		o, err := r.getIPAllocation(meta, *itfce.Spec.NetworkInstance, ipamv1alpha1.PrefixKindNetwork)
		if err != nil {
			return nil, err
		}
		resources = append(resources, o)

		if itfce.Spec.AttachmentType == nephioreqv1alpha1.AttachmentTypeVLAN {
			// add VLAN allocation
			o, err := r.getVLANAllocation(meta)
			if err != nil {
				return nil, err
			}
			resources = append(resources, o)
		}

		// allocate nad
		o, err = r.getNAD(meta)
		if err != nil {
			return nil, err
		}
		resources = append(resources, o)
	} else {
		// add IP allocation of type loopback
		o, err := r.getIPAllocation(meta, *itfce.Spec.NetworkInstance, ipamv1alpha1.PrefixKindLoopback)
		if err != nil {
			return nil, err
		}
		resources = append(resources, o)
	}
	return resources, nil
}

func (r *itfceFn) updateItfceResource(forObj *fn.KubeObject, objs fn.KubeObjects) (*fn.KubeObject, error) {
	if forObj == nil {
		return nil, fmt.Errorf("expected a for object but got nil")
	}
	itfceKOE, err := kubeobject.NewFromKubeObject[nephioreqv1alpha1.Interface](forObj)
	if err != nil {
		return nil, err
	}
	itfce, err := itfceKOE.GetGoStruct()
	if err != nil {
		return nil, err
	}

	ipallocs := objs.Where(fn.IsGroupVersionKind(ipamv1alpha1.IPAllocationGroupVersionKind))
	for _, ipalloc := range ipallocs {
		if ipalloc.GetName() == forObj.GetName() {
			alloc, err := kubeobject.NewFromKubeObject[ipamv1alpha1.IPAllocation](ipalloc)
			if err != nil {
				return nil, err
			}
			allocGoStruct, err := alloc.GetGoStruct()
			if err != nil {
				return nil, err
			}
			itfce.Status.IPAllocationStatus = &allocGoStruct.Status
		}
	}
	vlanallocs := objs.Where(fn.IsGroupVersionKind(vlanv1alpha1.VLANAllocationGroupVersionKind))
	for _, vlanalloc := range vlanallocs {
		if vlanalloc.GetName() == forObj.GetName() {
			alloc, err := vlanlibv1alpha1.NewFromKubeObject(vlanalloc)
			if err != nil {
				return nil, err
			}
			//alloc, err := ko.NewFromKubeObject[*vlanv1alpha1.VLANAllocation](vlanalloc)
			//if err != nil {
			//	return nil, err
			//}
			allocGoStruct, err := alloc.GetGoStruct()
			if err != nil {
				return nil, err
			}
			itfce.Status.VLANAllocationStatus = &allocGoStruct.Status
		}
	}
	// set the status
	err = itfceKOE.SetStatus(itfce)
	return &itfceKOE.KubeObject, err
}

func (r *itfceFn) getVLANAllocation(meta metav1.ObjectMeta) (*fn.KubeObject, error) {
	alloc := vlanv1alpha1.BuildVLANAllocation(
		meta,
		vlanv1alpha1.VLANAllocationSpec{
			VLANDatabase: corev1.ObjectReference{
				Name: r.siteCode,
			},
		},
		vlanv1alpha1.VLANAllocationStatus{},
	)

	return fn.NewFromTypedObject(alloc)
}

func (r *itfceFn) getIPAllocation(meta metav1.ObjectMeta, ni corev1.ObjectReference, kind ipamv1alpha1.PrefixKind) (*fn.KubeObject, error) {
	alloc := ipamv1alpha1.BuildIPAllocation(
		meta,
		ipamv1alpha1.IPAllocationSpec{
			Kind:            kind,
			NetworkInstance: ni,
			AllocationLabels: allocv1alpha1.AllocationLabels{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						allocv1alpha1.NephioSiteKey: r.siteCode,
					},
				},
			},
		},
		ipamv1alpha1.IPAllocationStatus{},
	)
	return fn.NewFromTypedObject(alloc)
}

func (r *itfceFn) getNAD(meta metav1.ObjectMeta) (*fn.KubeObject, error) {
	nad := BuildNetworkAttachmentDefinition(
		meta,
		nadv1.NetworkAttachmentDefinitionSpec{},
	)
	return fn.NewFromTypedObject(nad)
}

func BuildNetworkAttachmentDefinition(meta metav1.ObjectMeta, spec nadv1.NetworkAttachmentDefinitionSpec) *nadv1.NetworkAttachmentDefinition {
	return &nadv1.NetworkAttachmentDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: nadv1.SchemeGroupVersion.Identifier(),
			Kind:       reflect.TypeOf(nadv1.NetworkAttachmentDefinition{}).Name(),
		},
		ObjectMeta: meta,
		Spec:       spec,
	}
}
