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
	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	infrav1alpha1 "github.com/nephio-project/api/infra/v1alpha1"
	nephioreqv1alpha1 "github.com/nephio-project/api/nf_requirements/v1alpha1"
	"github.com/nephio-project/nephio/krm-functions/lib/condkptsdk"
	ko "github.com/nephio-project/nephio/krm-functions/lib/kubeobject"
	allocv1alpha1 "github.com/nokia/k8s-ipam/apis/alloc/common/v1alpha1"
	ipamv1alpha1 "github.com/nokia/k8s-ipam/apis/alloc/ipam/v1alpha1"
	vlanv1alpha1 "github.com/nokia/k8s-ipam/apis/alloc/vlan/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const defaultPODNetwork = "default"

type itfceFn struct {
	sdk             condkptsdk.KptCondSDK
	workloadCluster *infrav1alpha1.WorkloadCluster
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
					Kind:       reflect.TypeOf(infrav1alpha1.WorkloadCluster{}).Name(),
				}: myFn.WorkloadClusterCallbackFn,
			},
			PopulateOwnResourcesFn: myFn.desiredOwnedResourceList,
			UpdateResourceFn:       myFn.updateItfceResource,
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
func (f *itfceFn) WorkloadClusterCallbackFn(o *fn.KubeObject) error {
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
func (f *itfceFn) desiredOwnedResourceList(o *fn.KubeObject) (fn.KubeObjects, error) {
	if f.workloadCluster == nil {
		// no WorkloadCluster resource in the package
		return nil, fmt.Errorf("workload cluster is missing from the kpt package")
	}
	// resources contain the list of child resources
	// belonging to the parent object
	resources := fn.KubeObjects{}

	itfceKOE, err := ko.NewFromKubeObject[nephioreqv1alpha1.Interface](o)
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
		Name:        o.GetName(),
		Annotations: getAnnotations(o.GetAnnotations()),
	}

	afs := getAddressFamilies(itfce.Spec.IpFamilyPolicy)

	// When the CNIType is not set this is a loopback interface
	if itfce.Spec.CNIType != "" {
		if !f.IsCNITypePresent(itfce.Spec.CNIType) {
			return nil, fmt.Errorf("cniType not supported in workload cluster; workload cluster CNI(s): %v, interface cniType requested: %s", f.workloadCluster.Spec.CNIs, itfce.Spec.CNIType)
		}
		// add IP allocation of type network
		for _, af := range afs {
			meta := metav1.ObjectMeta{
				Name:        fmt.Sprintf("%s-%s", o.GetName(), string(af)),
				Annotations: getAnnotations(o.GetAnnotations()),
			}
			o, err := f.getIPAllocation(meta, *itfce.Spec.NetworkInstance, ipamv1alpha1.PrefixKindNetwork, af)
			if err != nil {
				return nil, err
			}
			resources = append(resources, o)
		}

		if itfce.Spec.AttachmentType == nephioreqv1alpha1.AttachmentTypeVLAN {
			// add VLAN allocation
			meta := metav1.ObjectMeta{
				Name:        o.GetName(),
				Annotations: f.getAnnotationsWithvlanAllocName(itfce),
			}
			o, err := f.getVLANAllocation(meta)
			if err != nil {
				return nil, err
			}
			resources = append(resources, o)
		}

		// allocate nad
		o, err = f.getNAD(meta)
		if err != nil {
			return nil, err
		}
		resources = append(resources, o)
	} else {
		// add IP allocation of type loopback
		for _, af := range afs {
			meta := metav1.ObjectMeta{
				Name:        fmt.Sprintf("%s-%s", o.GetName(), string(af)),
				Annotations: getAnnotations(o.GetAnnotations()),
			}
			o, err := f.getIPAllocation(meta, *itfce.Spec.NetworkInstance, ipamv1alpha1.PrefixKindLoopback, af)
			if err != nil {
				return nil, err
			}
			resources = append(resources, o)
		}

	}
	return resources, nil
}

func (f *itfceFn) updateItfceResource(forObj *fn.KubeObject, objs fn.KubeObjects) (*fn.KubeObject, error) {
	if forObj == nil {
		return nil, fmt.Errorf("expected a for object but got nil")
	}
	itfceKOE, err := ko.NewFromKubeObject[nephioreqv1alpha1.Interface](forObj)
	if err != nil {
		return nil, err
	}
	itfce, err := itfceKOE.GetGoStruct()
	if err != nil {
		return nil, err
	}

	ipallocs := objs.Where(fn.IsGroupVersionKind(ipamv1alpha1.IPAllocationGroupVersionKind))
	sort.Slice(ipallocs, func(i, j int) bool {
		return ipallocs[i].GetName() < ipallocs[j].GetName()
	})
	for _, ipalloc := range ipallocs {
		//if ipalloc.GetName() == forObj.GetName() {
		alloc, err := ko.NewFromKubeObject[ipamv1alpha1.IPAllocation](ipalloc)
		if err != nil {
			return nil, err
		}
		allocGoStruct, err := alloc.GetGoStruct()
		if err != nil {
			return nil, err
		}
		itfce.Status.UpsertIPAllocation(allocGoStruct.Status)
		//}
	}
	vlanallocs := objs.Where(fn.IsGroupVersionKind(vlanv1alpha1.VLANAllocationGroupVersionKind))
	for _, vlanalloc := range vlanallocs {
		//if vlanalloc.GetName() == forObj.GetName() {
		alloc, err := ko.NewFromKubeObject[vlanv1alpha1.VLANAllocation](vlanalloc)
		if err != nil {
			return nil, err
		}
		allocGoStruct, err := alloc.GetGoStruct()
		if err != nil {
			return nil, err
		}
		itfce.Status.VLANAllocationStatus = &allocGoStruct.Status
		//}
	}
	// set the status
	err = itfceKOE.SetStatus(itfce)
	return &itfceKOE.KubeObject, err
}

func (f *itfceFn) getVLANAllocation(meta metav1.ObjectMeta) (*fn.KubeObject, error) {
	alloc := vlanv1alpha1.BuildVLANAllocation(
		meta,
		vlanv1alpha1.VLANAllocationSpec{
			VLANDatabase: corev1.ObjectReference{
				Name: f.workloadCluster.Spec.ClusterName,
			},
		},
		vlanv1alpha1.VLANAllocationStatus{},
	)

	return fn.NewFromTypedObject(alloc)
}

func (f *itfceFn) getIPAllocation(meta metav1.ObjectMeta, ni corev1.ObjectReference, kind ipamv1alpha1.PrefixKind, af nephioreqv1alpha1.IPFamily) (*fn.KubeObject, error) {
	alloc := ipamv1alpha1.BuildIPAllocation(
		meta,
		ipamv1alpha1.IPAllocationSpec{
			Kind:            kind,
			NetworkInstance: ni,
			AllocationLabels: allocv1alpha1.AllocationLabels{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						allocv1alpha1.NephioClusterNameKey:   f.workloadCluster.Spec.ClusterName,
						allocv1alpha1.NephioAddressFamilyKey: string(af),
					},
				},
			},
		},
		ipamv1alpha1.IPAllocationStatus{},
	)
	return fn.NewFromTypedObject(alloc)
}

func (f *itfceFn) getNAD(meta metav1.ObjectMeta) (*fn.KubeObject, error) {
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

func (f *itfceFn) IsCNITypePresent(itfceCNIType nephioreqv1alpha1.CNIType) bool {
	for _, cni := range f.workloadCluster.Spec.CNIs {
		if nephioreqv1alpha1.CNIType(cni) == itfceCNIType {
			return true
		}
	}
	return false
}

func (f *itfceFn) getAnnotationsWithvlanAllocName(itfce *nephioreqv1alpha1.Interface) map[string]string {
	a := getAnnotations(itfce.GetAnnotations())
	a[condkptsdk.SpecializervlanAllocName] = fmt.Sprintf("%s-%s-bd", itfce.Spec.NetworkInstance.Name, f.workloadCluster.Spec.ClusterName)
	return a
}

func getAnnotations(annotations map[string]string) map[string]string {
	a := map[string]string{}
	if owner, ok := annotations[condkptsdk.SpecializerPurpose]; ok {
		a[condkptsdk.SpecializerPurpose] = owner
		return a
	}
	a[condkptsdk.SpecializerPurpose] = annotations[condkptsdk.SpecializerOwner]
	return a
}

func getAddressFamilies(pol nephioreqv1alpha1.IpFamilyPolicy) []nephioreqv1alpha1.IPFamily {
	afs := []nephioreqv1alpha1.IPFamily{}
	switch pol {
	case nephioreqv1alpha1.IpFamilyPolicyDualStack:
		afs = append(afs, nephioreqv1alpha1.IPFamilyIPv4)
		afs = append(afs, nephioreqv1alpha1.IPFamilyIPv6)
	case nephioreqv1alpha1.IpFamilyPolicyIPv6Only:
		afs = append(afs, nephioreqv1alpha1.IPFamilyIPv6)
	case nephioreqv1alpha1.IpFamilyPolicyIPv4Only:
		afs = append(afs, nephioreqv1alpha1.IPFamilyIPv4)
	default:
		afs = append(afs, nephioreqv1alpha1.IPFamilyIPv4)
	}
	return afs
}
