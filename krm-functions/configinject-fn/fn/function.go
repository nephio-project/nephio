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
	porchconfigv1alpha1 "github.com/GoogleContainerTools/kpt/porch/api/porchconfig/v1alpha1"
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
	sdkConfig       *condkptsdk.Config
}

func New(c client.Client) *FnR {
	f := &FnR{
		Client: c,
	}
	f.sdkConfig = &condkptsdk.Config{
		For: corev1.ObjectReference{
			APIVersion: nephioreqv1alpha1.GroupVersion.Identifier(),
			Kind:       nephioreqv1alpha1.DependencyKind,
		},
		Owns: map[corev1.ObjectReference]condkptsdk.ResourceKind{
			{
				APIVersion: "ref.nephio.org",
				Kind:       "Config",
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
	}
	return f
}

func (f *FnR) GetConfig() condkptsdk.Config {
	return *f.sdkConfig
}

func (f *FnR) Run(rl *fn.ResourceList) (bool, error) {
	sdk, err := condkptsdk.New(
		rl,
		f.sdkConfig,
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

	//get "parent"| Dependency struct
	dep, err := ko.KubeObjectToStruct[nephioreqv1alpha1.Dependency](forObj) // TO BE CHANGED
	if err != nil {
		return nil, err
	}
	depPackageName := dep.Spec.PackageName

	ctx := context.Background()
	// list the package revisions
	prl := &porchv1alpha1.PackageRevisionList{}
	if err := f.List(ctx, prl); err != nil {
		return nil, err
	}
	// list the repo(s)
	repos := &porchconfigv1alpha1.RepositoryList{}
	if err := f.List(ctx, repos); err != nil {
		return nil, err
	}
	// build a repo map for faster lookup
	repomap := map[string]porchconfigv1alpha1.Repository{}
	for _, repo := range repos.Items {
		repomap[repo.Name] = repo
	}

	resources := fn.KubeObjects{}
	// walk through all the package revisions and check if the dependent resources are ready
	// we assume there needs to be 1 dependency that resolves
	found := false
	for _, pr := range prl.Items {
		repo, ok := repomap[pr.Spec.RepositoryName]
		if !ok {
			return nil, fmt.Errorf("configinject repo name not found: %s", pr.Spec.RepositoryName)
		}
		// only analyse the packages with the packageName contained in the dependency requirement resource
		// and only look at repo(s) where the deployment is true
		// TBD: do we need to check the latest revision ?
		if pr.Spec.PackageName == depPackageName && repo.Spec.Deployment {
			fn.Logf("configinject repo %s\n", pr.Spec.RepositoryName)
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
				return nil, fmt.Errorf("mandatory Kptfile is missing from the package %s, repo %s", pr.Spec.PackageName, pr.Spec.RepositoryName)
			}
			kf := kptfilelibv1.KptFile{Kptfile: kfko}

			// get the dependency objects in the package and check its status
			gvk := schema.GroupVersionKind{
				Group:   nephiodeployv1alpha1.GroupVersion.Group,
				Version: nephiodeployv1alpha1.GroupVersion.Version,
				Kind:    nephiodeployv1alpha1.UPFDeploymentKind,
			}
			for _, ref := range dep.Spec.Injectors {
				/*
					for _, o := range rl.Items {
						fn.Logf("configinject resource apiVersion: %s kind: %s\n", o.GetAPIVersion(), o.GetKind())
					}
				*/
				fn.Logf("configinject dependency gvk: %s\n", gvk.String())

				depObjs := rl.Items.Where(fn.IsGroupVersionKind(ref.GroupVersionKind()))
				if len(depObjs) == 0 {
					fn.Logf("configinject dependency not ready: the package %s in repo %s does not contain a resource with %s\n", pr.Spec.PackageName, pr.Spec.RepositoryName, gvk.String())
					return nil, fmt.Errorf("dependency not ready: the package %s in repo %s does not contain a resource with %s", pr.Spec.PackageName, pr.Spec.RepositoryName, gvk.String())
				}
				for _, o := range depObjs {
					ct := kptfilelibv1.GetConditionType(&corev1.ObjectReference{
						APIVersion: o.GetAPIVersion(),
						Kind:       o.GetKind(),
						Name:       o.GetName(),
					})
					c := kf.GetCondition(ct)
					if c == nil {
						fn.Logf("configinject dependency not ready: the package %s in repo %s does not contain a condition for %s\n", pr.Spec.PackageName, pr.Spec.RepositoryName, ct)
						return nil, fmt.Errorf("dependency not ready: the package %s in repo %s does not contain a condition for %s", pr.Spec.PackageName, pr.Spec.RepositoryName, ct)
					}
					if c.Status != kptv1.ConditionTrue {
						// we fail fast if the condition is not true
						fn.Logf("configinject dependency not ready: the package %s in repo %s has a condition which is False for: %s\n", pr.Spec.PackageName, pr.Spec.RepositoryName, c.Type)
						return nil, fmt.Errorf("dependency not ready: the package %s in repo %s has a condition which is False for: %s", pr.Spec.PackageName, pr.Spec.RepositoryName, c.Type)
					}
					// encapsulates the resource in another CR
					newObj, err := GetConfigKubeObject(forObj, o)
					if err != nil {
						return nil, err
					}
					fn.Logf("configinject newObj : %v\n", newObj.String())
					found = true
					resources = append(resources, newObj)
				}
			}
		}
	}
	if !found {
		fn.Logf("configinject dependency not ready: expecting at least 1 package %s with the corresponding reference\n", depPackageName)
		return nil, fmt.Errorf("dependency not ready: expecting at least 1 package %s with the corresponding reference", depPackageName)
	}
	return resources, nil
}

// updateDependencyResource adds the resources to the status
func (f *FnR) updateDependencyResource(forObj *fn.KubeObject, objs fn.KubeObjects) (*fn.KubeObject, error) {
	if forObj == nil {
		return nil, fmt.Errorf("expected a for object but got nil")
	}

	// get "parent"| Dependency struct
	depObj, err := ko.NewFromKubeObject[nephioreqv1alpha1.Dependency](forObj)
	if err != nil {
		return nil, err
	}
	dep, err := depObj.GetGoStruct()
	if err != nil {
		return nil, err
	}

	dep.Status.Injected = []corev1.ObjectReference{}
	for _, o := range objs {
		dep.Status.Injected = append(dep.Status.Injected, corev1.ObjectReference{APIVersion: o.GetAPIVersion(), Kind: o.GetKind()})
	}

	if err := depObj.SetStatus(dep); err != nil {
		return nil, err
	}

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

	newCfgObj := BuildConfig(metav1.ObjectMeta{
		Name:      o.GetName(),
		Namespace: forObj.GetAnnotation(condkptsdk.SpecializerNamespace),
	},
		configv1alpha1.NetworkSpec{
			Config: runtime.RawExtension{Object: u},
		},
		configv1alpha1.NetworkStatus{},
	)
	return fn.NewFromTypedObject(newCfgObj)
}

// BuildNetworkConfig returns a Network from a client Object a crName and
// an Network Spec/Status
func BuildConfig(meta metav1.ObjectMeta, spec configv1alpha1.NetworkSpec, status configv1alpha1.NetworkStatus) *configv1alpha1.Network {
	return &configv1alpha1.Network{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "ref.nephio.org",
			Kind:       "Config",
		},
		ObjectMeta: meta,
		Spec:       spec,
		Status:     status,
	}
}
