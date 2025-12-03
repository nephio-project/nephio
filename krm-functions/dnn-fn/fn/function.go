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
	"reflect"
	"strings"

	"fmt"

	"github.com/kptdev/krm-functions-sdk/go/fn"
	infrav1alpha1 "github.com/nephio-project/api/infra/v1alpha1"
	nephioreqv1alpha1 "github.com/nephio-project/api/nf_requirements/v1alpha1"
	"github.com/nephio-project/nephio/krm-functions/lib/condkptsdk"
	ko "github.com/nephio-project/nephio/krm-functions/lib/kubeobject"
	resourcev1alpha1 "github.com/nokia/k8s-ipam/apis/resource/common/v1alpha1"
	ipamv1alpha1 "github.com/nokia/k8s-ipam/apis/resource/ipam/v1alpha1"
	"github.com/nokia/k8s-ipam/pkg/iputil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
)

func init() {
	_ = nephioreqv1alpha1.AddToScheme(ko.TheScheme)
	_ = infrav1alpha1.AddToScheme(ko.TheScheme)
	_ = ipamv1alpha1.AddToScheme(ko.TheScheme)
}

type dnnFn struct {
	sdk             condkptsdk.KptCondSDK
	workloadCluster *infrav1alpha1.WorkloadCluster
	rl              *fn.ResourceList
}

// Run is the entry point of the KRM function (called by the upstream fn SDK)
func Run(rl *fn.ResourceList) (bool, error) {
	var err error
	myFn := dnnFn{rl: rl}

	myFn.sdk, err = condkptsdk.New(
		rl,
		&condkptsdk.Config{
			For: corev1.ObjectReference{
				APIVersion: nephioreqv1alpha1.GroupVersion.Identifier(),
				Kind:       nephioreqv1alpha1.DataNetworkKind,
			},
			Owns: map[corev1.ObjectReference]condkptsdk.ResourceKind{
				{
					APIVersion: ipamv1alpha1.GroupVersion.Identifier(),
					Kind:       ipamv1alpha1.IPClaimKind,
				}: condkptsdk.ChildRemote,
			},
			Watch: map[corev1.ObjectReference]condkptsdk.WatchCallbackFn{
				{
					APIVersion: infrav1alpha1.GroupVersion.Identifier(),
					Kind:       reflect.TypeFor[infrav1alpha1.WorkloadCluster]().Name(),
				}: myFn.WorkloadClusterCallbackFn,
			},
			PopulateOwnResourcesFn: myFn.desiredOwnedResourceList,
			UpdateResourceFn:       myFn.updateDnnResource,
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
func (f *dnnFn) WorkloadClusterCallbackFn(o *fn.KubeObject) error {
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

// desiredOwnedResourceList returns with the list of all KubeObjects that the DNN "for object" should own in the package
func (f *dnnFn) desiredOwnedResourceList(o *fn.KubeObject) (fn.KubeObjects, error) {
	if f.workloadCluster == nil {
		// no WorkloadCluster resource in the package
		return nil, fmt.Errorf("workload cluster is missing from the kpt package")
	}

	// get "parent"| DNN struct
	dnn, err := ko.KubeObjectToStruct[nephioreqv1alpha1.DataNetwork](o)
	if err != nil {
		return nil, err
	}

	// add IpClaim for each pool
	resources := fn.KubeObjects{}
	for _, pool := range dnn.Spec.Pools {

		af := iputil.AddressFamilyIpv4
		if pool.IPFamily == nephioreqv1alpha1.IPFamilyIPv6 {
			af = iputil.AddressFamilyIpv6
		}

		ipClaim := ipamv1alpha1.BuildIPClaim(
			metav1.ObjectMeta{
				Name:        fmt.Sprintf("%s-%s-%s", getForName(o.GetAnnotations()), dnn.Name, pool.Name),
				Annotations: getAnnotations(dnn.GetAnnotations()),
			},
			ipamv1alpha1.IPClaimSpec{
				Kind:            ipamv1alpha1.PrefixKindPool,
				NetworkInstance: dnn.Spec.NetworkInstance,
				PrefixLength:    &pool.PrefixLength,
				CreatePrefix:    ptr.To(true),
				ClaimLabels: resourcev1alpha1.ClaimLabels{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							resourcev1alpha1.NephioClusterNameKey:   f.workloadCluster.Spec.ClusterName, // NOTE: at this point WorkloadCluster is validated, so this is safe
							resourcev1alpha1.NephioAddressFamilyKey: af.String(),
						},
					},
				},
			},
			ipamv1alpha1.IPClaimStatus{},
		)

		ipClaimObj, err := fn.NewFromTypedObject(ipClaim)
		if err != nil {
			return nil, err
		}
		resources = append(resources, ipClaimObj)
	}
	return resources, nil
}

// updateDnnResource assembles the Status of the DNN "for object" from the status of the owned IPClaims
func (f *dnnFn) updateDnnResource(dnnObj_ *fn.KubeObject, owned fn.KubeObjects) (fn.KubeObjects, error) {
	dnnObj, err := ko.NewFromKubeObject[nephioreqv1alpha1.DataNetwork](dnnObj_)
	if err != nil {
		return nil, err
	}
	dnn, err := dnnObj.GetGoStruct()
	if err != nil {
		return nil, err
	}

	// get IPClaim status of all pools
	dnn.Status.Pools = nil
	ipclaims, _, err := ko.FilterByType[ipamv1alpha1.IPClaim](owned)
	if err != nil {
		return nil, err
	}
	for _, ipclaim := range ipclaims {
		if ipclaim.Spec.Kind == ipamv1alpha1.PrefixKindPool {
			poolName, found := strings.CutPrefix(ipclaim.Name, fmt.Sprintf("%s-%s-", getForName(dnn.Annotations), dnn.Name))
			if found {
				status := nephioreqv1alpha1.PoolStatus{
					Name:    poolName,
					IPClaim: ipclaim.Status,
				}
				dnn.Status.Pools = append(dnn.Status.Pools, status)
			} else {
				f.rl.Results.Warningf("found an IPClaim owned by DNN %q with a suspicious name: %v", dnn.Name, ipclaim.Name)
			}
		}
	}

	err = dnnObj.SetStatus(dnn)
	return fn.KubeObjects{&dnnObj.KubeObject}, err
}

func getAnnotations(annotations map[string]string) map[string]string {
	a := map[string]string{}
	for k, v := range annotations {
		if k == filters.LocalConfigAnnotation {
			a[k] = v
		}
	}
	if rootOwner, ok := annotations[condkptsdk.SpecializerFor]; ok {
		a[condkptsdk.SpecializerFor] = rootOwner
		return a
	}
	a[condkptsdk.SpecializerFor] = annotations[condkptsdk.SpecializerOwner]
	return a
}

func getForName(annotations map[string]string) string {
	// forName is the resource that is the root resource of the specialization
	// e.g. NFDeployment
	forFullName := annotations[condkptsdk.SpecializerOwner]
	if owner, ok := annotations[condkptsdk.SpecializerFor]; ok {
		forFullName = owner
	}
	split := strings.Split(forFullName, ".")

	return split[len(split)-1]
}
