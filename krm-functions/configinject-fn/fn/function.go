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
	"context"
	"fmt"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	kptv1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	porchv1alpha1 "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configv1alpha1 "github.com/henderiw-nephio/network/apis/config/v1alpha1"
	infrav1alpha1 "github.com/nephio-project/api/infra/v1alpha1"
	nephiodeployv1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	nephioreqv1alpha1 "github.com/nephio-project/api/nf_requirements/v1alpha1"
	"github.com/nephio-project/nephio/krm-functions/lib/condkptsdk"
	kptfilelibv1 "github.com/nephio-project/nephio/krm-functions/lib/kptfile/v1"
	"github.com/nephio-project/nephio/krm-functions/lib/kptrl"
	ko "github.com/nephio-project/nephio/krm-functions/lib/kubeobject"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FnR struct {
	client.Client
	workloadCluster *infrav1alpha1.WorkloadCluster
}

func (f *FnR) Run(rl *fn.ResourceList) (bool, error) {
	sdk, err := condkptsdk.New(
		rl,
		&condkptsdk.Config{
			For: corev1.ObjectReference{
				APIVersion: nephioreqv1alpha1.GroupVersion.Identifier(),
				Kind:       nephioreqv1alpha1.DependencyKind, // TO BE CHANGED TO DEPENDENCY
			},
			Owns: map[corev1.ObjectReference]condkptsdk.ResourceKind{
				{
					APIVersion: nephiodeployv1alpha1.GroupVersion.Identifier(),
					Kind:       nephiodeployv1alpha1.ConfigKind, // TO BE CHANGED TO CONFIG
				}: condkptsdk.ChildLocal,
			},
			Watch: map[corev1.ObjectReference]condkptsdk.WatchCallbackFn{
				{
					APIVersion: infrav1alpha1.GroupVersion.Identifier(),
					Kind:       infrav1alpha1.WorkloadClusterKind,
				}: f.WorkloadClusterCallbackFn,
			},
			PopulateOwnResourcesFn: f.desiredOwnedResourceList,
			UpdateResourceFn:       f.updateDependencyResource,
		},
	)
	if err != nil {
		rl.Results.ErrorE(err)
		return false, err
	}
	return sdk.Run()
}

// WorkloadClusterCallbackFn provides a callback for the workload cluster
// resources in the resourceList
func (f *FnR) WorkloadClusterCallbackFn(o *fn.KubeObject) error {
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

func (f *FnR) desiredOwnedResourceList(forObj *fn.KubeObject) (fn.KubeObjects, error) {
	if f.workloadCluster == nil {
		// no WorkloadCluster resource in the package
		return nil, fmt.Errorf("workload cluster is missing from the kpt package")
	}

	// get "parent"| Dependency struct
	dep, err := ko.KubeObjectToStruct[nephioreqv1alpha1.Dependency](forObj) // TO BE CHANGED
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	// list the package revisions
	prl := &porchv1alpha1.PackageRevisionList{}
	if err := f.List(ctx, prl); err != nil {
		return nil, err
	}
	resources := fn.KubeObjects{}
	// walk through all the package revisions and check if the dependent resources are ready
	for _, pr := range prl.Items {
		// only analyse the packages with the packageName contained in the dependency requirement resource
		// TBD: do we need to check the latest revision ?
		if pr.Spec.PackageName == dep.Name {
			// get the package resources of the revision
			prr := &porchv1alpha1.PackageRevisionResources{}
			if err := f.Get(ctx, types.NamespacedName{Namespace: pr.Namespace, Name: pr.Name}, prr); err != nil {
				return nil, err
			}
			// get the resource list from the package
			rl, err := kptrl.GetResourceList(prr.Spec.Resources)
			if err != nil {
				return nil, err
			}
			// get the kptfile from the resourcelist to check condition status
			kfko := rl.Items.GetRootKptfile()
			if kfko == nil {
				return nil, fmt.Errorf("mandatory Kptfile is missing from the package")
			}
			kf := kptfilelibv1.KptFile{Kptfile: kfko}

			// get the dependency objects in the package and check its status
			depObjs := rl.Items.Where(fn.IsGroupVersionKind(schema.GroupVersionKind{}))
			if len(depObjs) == 0 {
				return nil, fmt.Errorf("dependency not ready: the package does not contain a resource with %s", schema.GroupVersionKind{}.String())
			}
			for _, o := range depObjs {
				ct := kptfilelibv1.GetConditionType(&corev1.ObjectReference{
					APIVersion: o.GetAPIVersion(),
					Kind:       o.GetKind(),
					Name:       o.GetName(),
				})
				c := kf.GetCondition(ct)
				if c == nil {
					return nil, fmt.Errorf("dependency not ready: no condition for %s", ct)
				}
				if c.Status != kptv1.ConditionTrue {
					// we fail fast if the condition is not true
					return nil, fmt.Errorf("dependency not ready: the condition is not true for: %s", c.Type)
				}
				// append the resource to the resources
				newObj, err := GetConfigKubeObject(forObj, o)
				if err != nil {
					return nil, err
				}
				resources = append(resources, newObj)
			}
		}
	}
	return resources, nil
}

// updateDependencyResource adds the resources to the status
func (f *FnR) updateDependencyResource(forObj *fn.KubeObject, objs fn.KubeObjects) (*fn.KubeObject, error) {
	if forObj == nil {
		return nil, fmt.Errorf("expected a for object but got nil")
	}

	configRefs := []corev1.ObjectReference{}
	for _, o := range objs {
		configRefs = append(configRefs, corev1.ObjectReference{APIVersion: o.GetAPIVersion(), Kind})
	}

	// get "parent"| Dependency struct
	depObj, err := ko.KubeObjectToStruct[nephioreqv1alpha1.Dependency](forObj) // TO BE CHANGED
	if err != nil {
		return nil, err
	}
	dep, err := depObj.GetGoStruct()
	if err != nil {
		return nil, err
	}
	dep.Status.injected = configRefs
	depObj.SetStatus(dep)

	return &depObj.KubeObject, err
}

func GetConfigKubeObject(forObj, o *fn.KubeObject) (*fn.KubeObject, error) {
	x, err := ko.NewFromKubeObject[unstructured.Unstructured](o)
	if err != nil {
		return nil, err
	}

	u, err := x.GetGoStruct()
	if err != nil {
		return nil, err
	}

	newCfgObj := configv1alpha1.BuildNetworkConfig(metav1.ObjectMeta{
		Name:      o.GetName(),
		Namespace: forObj.GetNamespace(),
	},
		configv1alpha1.NetworkSpec{
			Config: runtime.RawExtension{Object: u},
		},
		configv1alpha1.NetworkStatus{},
	)
	return fn.NewFromTypedObject(newCfgObj)
}
