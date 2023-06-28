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
	"strings"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	infrav1alpha1 "github.com/nephio-project/api/infra/v1alpha1"
	nephioreqv1alpha1 "github.com/nephio-project/api/nf_requirements/v1alpha1"
	"github.com/nephio-project/nephio/krm-functions/lib/condkptsdk"
	ko "github.com/nephio-project/nephio/krm-functions/lib/kubeobject"
	resourcev1alpha1 "github.com/nokia/k8s-ipam/apis/resource/common/v1alpha1"
	ipamv1alpha1 "github.com/nokia/k8s-ipam/apis/resource/ipam/v1alpha1"
	vlanv1alpha1 "github.com/nokia/k8s-ipam/apis/resource/vlan/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
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
					Kind:       ipamv1alpha1.IPClaimKind,
				}: condkptsdk.ChildRemote,
				{
					APIVersion: vlanv1alpha1.GroupVersion.Identifier(),
					Kind:       vlanv1alpha1.VLANClaimKind,
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

	afs := getAddressFamilies(itfce.Spec.IpFamilyPolicy)
	purpose := o.GetAnnotation(resourcev1alpha1.NephioNetworkNameKey)

	// When the CNIType is not set this is a loopback interface
	if itfce.Spec.CNIType != "" {
		if !f.IsCNITypePresent(itfce.Spec.CNIType) {
			return nil, fmt.Errorf("cniType not supported in workload cluster; workload cluster CNI(s): %v, interface cniType requested: %s", f.workloadCluster.Spec.CNIs, itfce.Spec.CNIType)
		}
		// add IPClaim of type network
		for _, af := range afs {
			meta := metav1.ObjectMeta{
				Name:        fmt.Sprintf("%s-%s-%s", getForName(o.GetAnnotations()), o.GetName(), string(af)),
				Annotations: getAnnotations(o.GetAnnotations()),
			}
			obj, err := f.getIPClaim(meta, *itfce.Spec.NetworkInstance, ipamv1alpha1.PrefixKindNetwork, af, purpose)
			if err != nil {
				return nil, err
			}
			resources = append(resources, obj)
		}

		if itfce.Spec.AttachmentType == nephioreqv1alpha1.AttachmentTypeVLAN {
			// add VLANClaim
			meta := metav1.ObjectMeta{
				Name:        fmt.Sprintf("%s-%s", getForName(o.GetAnnotations()), o.GetName()),
				Annotations: f.getAnnotationsWithvlanClaimName(itfce),
			}
			obj, err := f.getVLANClaim(meta)
			if err != nil {
				return nil, err
			}
			resources = append(resources, obj)
		}

		// claim nad
		meta := metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-%s", getForName(o.GetAnnotations()), o.GetName()),
			Annotations: getAnnotations(o.GetAnnotations()),
		}
		o, err = f.getNAD(meta)
		if err != nil {
			return nil, err
		}
		resources = append(resources, o)
	} else {
		// add IPClaim of type loopback
		for _, af := range afs {
			meta := metav1.ObjectMeta{
				Name:        fmt.Sprintf("%s-%s-%s", getForName(o.GetAnnotations()), o.GetName(), string(af)),
				Annotations: getAnnotations(o.GetAnnotations()),
			}
			o, err := f.getIPClaim(meta, *itfce.Spec.NetworkInstance, ipamv1alpha1.PrefixKindLoopback, af, purpose)
			if err != nil {
				return nil, err
			}
			resources = append(resources, o)
		}

	}
	return resources, nil
}

func (f *itfceFn) updateItfceResource(forObj *fn.KubeObject, objs fn.KubeObjects) (fn.KubeObjects, error) {
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

	ipclaims := objs.Where(fn.IsGroupVersionKind(ipamv1alpha1.IPClaimGroupVersionKind))
	sort.Slice(ipclaims, func(i, j int) bool {
		return ipclaims[i].GetName() < ipclaims[j].GetName()
	})
	for _, ipclaim := range ipclaims {
		// Dont care about the name since the condSDK sorts the data
		// based on owner reference
		claim, err := ko.NewFromKubeObject[ipamv1alpha1.IPClaim](ipclaim)
		if err != nil {
			return nil, err
		}
		claimGoStruct, err := claim.GetGoStruct()
		if err != nil {
			return nil, err
		}
		itfce.Status.UpsertIPClaim(claimGoStruct.Status)
	}
	vlanclaims := objs.Where(fn.IsGroupVersionKind(vlanv1alpha1.VLANClaimGroupVersionKind))
	for _, vlanclaim := range vlanclaims {
		claim, err := ko.NewFromKubeObject[vlanv1alpha1.VLANClaim](vlanclaim)
		if err != nil {
			return nil, err
		}
		claimGoStruct, err := claim.GetGoStruct()
		if err != nil {
			return nil, err
		}
		itfce.Status.VLANClaimStatus = &claimGoStruct.Status
	}
	// set the status
	err = itfceKOE.SetStatus(itfce)
	return fn.KubeObjects{&itfceKOE.KubeObject}, err
}

func (f *itfceFn) getVLANClaim(meta metav1.ObjectMeta) (*fn.KubeObject, error) {
	claim := vlanv1alpha1.BuildVLANClaim(
		meta,
		vlanv1alpha1.VLANClaimSpec{
			VLANIndex: corev1.ObjectReference{
				Name: f.workloadCluster.Spec.ClusterName,
			},
		},
		vlanv1alpha1.VLANClaimStatus{},
	)

	return fn.NewFromTypedObject(claim)
}

func (f *itfceFn) getIPClaim(meta metav1.ObjectMeta, ni corev1.ObjectReference, kind ipamv1alpha1.PrefixKind, af nephioreqv1alpha1.IPFamily, purpose string) (*fn.KubeObject, error) {
	matchLabels := map[string]string{
		resourcev1alpha1.NephioClusterNameKey:   f.workloadCluster.Spec.ClusterName,
		resourcev1alpha1.NephioAddressFamilyKey: string(af),
	}
	if purpose != "" {
		matchLabels[resourcev1alpha1.NephioNetworkNameKey] = purpose
	}

	claim := ipamv1alpha1.BuildIPClaim(
		meta,
		ipamv1alpha1.IPClaimSpec{
			Kind:            kind,
			NetworkInstance: ni,
			ClaimLabels: resourcev1alpha1.ClaimLabels{
				Selector: &metav1.LabelSelector{
					MatchLabels: matchLabels,
				},
			},
		},
		ipamv1alpha1.IPClaimStatus{},
	)
	return fn.NewFromTypedObject(claim)
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

func (f *itfceFn) getAnnotationsWithvlanClaimName(itfce *nephioreqv1alpha1.Interface) map[string]string {
	a := getAnnotations(itfce.GetAnnotations())
	a[condkptsdk.SpecializervlanClaimName] = fmt.Sprintf("%s-%s-bd", itfce.Spec.NetworkInstance.Name, f.workloadCluster.Spec.ClusterName)
	return a
}

func getAnnotations(annotations map[string]string) map[string]string {
	a := map[string]string{}
	for k, v := range annotations {
		if k == filters.LocalConfigAnnotation {
			a[k] = v
		}
	}
	if owner, ok := annotations[condkptsdk.SpecializerFor]; ok {
		a[condkptsdk.SpecializerFor] = owner
		return a
	}
	a[condkptsdk.SpecializerFor] = annotations[condkptsdk.SpecializerOwner]
	return a
}

func getForName(annotations map[string]string) string {
	// forName is the resource that is the root resource of the specialization
	// e.g. UPFDeployment, SMFDeployment, AMFDeployment
	forFullName := annotations[condkptsdk.SpecializerOwner]
	if owner, ok := annotations[condkptsdk.SpecializerFor]; ok {
		forFullName = owner
	}
	split := strings.Split(forFullName, ".")
	return split[len(split)-1]
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
