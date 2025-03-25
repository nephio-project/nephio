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
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	infrav1alpha1 "github.com/nephio-project/api/infra/v1alpha1"
	nephioreqv1alpha1 "github.com/nephio-project/api/nf_requirements/v1alpha1"
	nephiorefv1alpha1 "github.com/nephio-project/api/references/v1alpha1"
	"github.com/nephio-project/nephio/krm-functions/lib/condkptsdk"
	kptfilelibv1 "github.com/nephio-project/nephio/krm-functions/lib/kptfile/v1"
	"github.com/nephio-project/nephio/krm-functions/lib/kptrl"
	ko "github.com/nephio-project/nephio/krm-functions/lib/kubeobject"
	porchv1alpha1 "github.com/nephio-project/porch/api/porch/v1alpha1"
	porchconfigv1alpha1 "github.com/nephio-project/porch/api/porchconfig/v1alpha1"
	kptv1 "github.com/nephio-project/porch/pkg/kpt/api/kptfile/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const revisionPrefix = "v"

type FnR struct {
	client.Reader
	workloadCluster *infrav1alpha1.WorkloadCluster
	sdkConfig       *condkptsdk.Config
}

func New(c client.Reader) *FnR {
	f := &FnR{
		Reader: c,
	}
	f.sdkConfig = &condkptsdk.Config{
		For: corev1.ObjectReference{
			APIVersion: nephioreqv1alpha1.GroupVersion.Identifier(),
			Kind:       nephioreqv1alpha1.DependencyKind,
		},
		Owns: map[corev1.ObjectReference]condkptsdk.ResourceKind{
			{
				APIVersion: nephiorefv1alpha1.GroupVersion.Identifier(),
				Kind:       reflect.TypeFor[nephiorefv1alpha1.Config]().Name(),
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
	// walk through all the package revisions and build a map of the pr(s) that
	// have the package name and are in a repo that has deployment true
	// The map will contain the latest published revision of the pr, if no pr
	// is published it will have a reference to this pr
	// we assume there needs to be 1 dependency that resolves
	prmap := map[string]*porchv1alpha1.PackageRevision{}
	for _, pr := range prl.Items {
		repo, ok := repomap[pr.Spec.RepositoryName]
		if !ok {
			return nil, fmt.Errorf("configinject repo name not found: %s", pr.Spec.RepositoryName)
		}
		// only analyze the packages with
		// - the packageName contained in the dependency requirement resource
		// - repo has deployment true
		// - package is published
		if pr.Spec.PackageName == depPackageName && repo.Spec.Deployment {
			fn.Logf("configinject repo %s\n", pr.Spec.RepositoryName)

			prName := fmt.Sprintf("%s-%s", pr.Spec.RepositoryName, pr.Spec.PackageName)
			fn.Logf("configinject pr name %s\n", prName)

			if porchv1alpha1.LifecycleIsPublished(pr.Spec.Lifecycle) {
				if pr.Spec.Revision > 0 {
					fn.Logf("configinject revision %d\n", pr.Spec.Revision)

					newRev := pr.Spec.Revision
					if latestPR, ok := prmap[prName]; ok {
						// both the latest pr and the new pr are published
						// update the map with the latest pr
						// if the revision of the new pr is better than the one of the latest pr in the map
						latestRev := latestPR.Spec.Revision
						if newRev > latestRev {
							prmap[prName] = pr.DeepCopy()
						}
					} else {
						prmap[prName] = pr.DeepCopy()
					}
				}
			}
		}
	}

	for prName, pr := range prmap {
		if pr == nil {
			msg := fmt.Sprintf("configinject dependency not ready: no published package %s\n", prName)
			fn.Logf("%s\n", msg)
			// if 1 package is not ready we fail fast
			return nil, fmt.Errorf("%s", msg)
		}
	}

	// at this stage all packages are published
	for _, pr := range prmap {
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
		for _, ref := range dep.Spec.Injectors {
			fn.Logf("configinject dependency gvk: %s\n", ref.GroupVersionKind().String())

			depObjs := rl.Items.Where(fn.IsGroupVersionKind(ref.GroupVersionKind()))
			if len(depObjs) == 0 {
				fn.Logf("configinject dependency not ready: the package %s in repo %s does not contain a resource with %s\n", pr.Spec.PackageName, pr.Spec.RepositoryName, ref.GroupVersionKind().String())
				return nil, fmt.Errorf("dependency not ready: the package %s in repo %s does not contain a resource with %s", pr.Spec.PackageName, pr.Spec.RepositoryName, ref.GroupVersionKind().String())
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
				resources = append(resources, newObj)
			}
		}
	}
	if len(prmap) == 0 {
		fn.Logf("configinject dependency not ready: expecting at least 1 package %s with the corresponding reference\n", depPackageName)
		return nil, fmt.Errorf("dependency not ready: expecting at least 1 package %s with the corresponding reference", depPackageName)
	}
	return resources, nil
}

// updateDependencyResource adds the resources to the status
func (f *FnR) updateDependencyResource(forObj *fn.KubeObject, objs fn.KubeObjects) (fn.KubeObjects, error) {
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
		dep.Status.Injected = append(dep.Status.Injected, corev1.ObjectReference{
			APIVersion: o.GetAPIVersion(),
			Kind:       o.GetKind(),
			Name:       o.GetName(),
			Namespace:  o.GetNamespace()})
	}

	sort.Slice(dep.Status.Injected, func(i, j int) bool {
		return dep.Status.Injected[i].Name < dep.Status.Injected[j].Name
	})

	if err := depObj.SetStatus(dep); err != nil {
		return nil, err
	}

	return fn.KubeObjects{&depObj.KubeObject}, nil
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
		Name:      fmt.Sprintf("%s-%s", getForName(forObj.GetAnnotations()), o.GetName()),
		Namespace: forObj.GetAnnotation(condkptsdk.SpecializerNamespace),
	},
		nephiorefv1alpha1.ConfigSpec{
			Config: runtime.RawExtension{Object: u},
		},
	)
	return fn.NewFromTypedObject(newCfgObj)
}

// BuildConfig returns a Network from a client Object a crName and
// an Network Spec/Status
func BuildConfig(meta metav1.ObjectMeta, spec nephiorefv1alpha1.ConfigSpec) *nephiorefv1alpha1.Config {
	return &nephiorefv1alpha1.Config{
		TypeMeta: metav1.TypeMeta{
			APIVersion: nephiorefv1alpha1.GroupVersion.Identifier(),
			Kind:       "Config",
		},
		ObjectMeta: meta,
		Spec:       spec,
	}
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

func getRevisionNbr(rev string) (int, error) {
	rev = strings.TrimPrefix(rev, revisionPrefix)
	return strconv.Atoi(rev)
}
