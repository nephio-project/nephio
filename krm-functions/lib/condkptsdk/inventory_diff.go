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

package condkptsdk

import (
	"fmt"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	v1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/google/go-cmp/cmp"
	"github.com/nephio-project/nephio/krm-functions/lib/ref"
	corev1 "k8s.io/api/core/v1"
)

type inventoryDiff struct {
	deleteForCondition bool
	updateForCondition bool
	deleteObjs         []object
	updateObjs         []object
	createObjs         []object
	deleteConditions   []object
	createConditions   []object // different types exist here based on childResource Type (childInitial, childLocal)
	//updateConditions []*object
	updateDeleteAnnotations []object
}

type object struct {
	ref     corev1.ObjectReference
	obj     fn.KubeObject
	ownKind ResourceKind
}

// Diff is based on the following principle: we have an inventory
// populated with the existing resource/condition info and we also
// have information on new resource/condition that would be created
// if nothing existed.
// the diff compares the existing resource/condition inventory
// against the new resource/condition inventory and provide CRUD operation
// based on that comparisons.
func (r *inv) diff() map[corev1.ObjectReference]*inventoryDiff {
	r.m.RLock()
	defer r.m.RUnlock()
	diffMap := map[corev1.ObjectReference]*inventoryDiff{}

	for forRef, forResCtx := range r.get(forGVKKind, []corev1.ObjectReference{{}}) {
		diffMap[forRef] = &inventoryDiff{
			deleteObjs:       []object{},
			updateObjs:       []object{},
			createObjs:       []object{},
			deleteConditions: []object{},
			createConditions: []object{},
			//createInitialConditions: []object{},
			//createTrueConditions:    []object{},
			updateDeleteAnnotations: []object{},
		}
		// if the existing for resource is not present we need to cleanup
		// all child resources and conditions
		//fn.Logf("diff: forRef: %v, existingResource: %v\n", forRef, resCtx.existingResource)
		if forResCtx.existingResource == nil {
			for ownRef, resCtx := range r.get(ownGVKKind, []corev1.ObjectReference{forRef, {}}) {
				if r.debug {
					fn.Logf("delete resource and conditions: objRef: %s\n", ref.GetRefsString(forRef, ownRef))
				}
				diffMap[forRef].deleteForCondition = true
				if resCtx.existingCondition != nil {
					diffMap[forRef].deleteConditions = append(diffMap[forRef].deleteConditions, object{ref: ownRef, ownKind: resCtx.ownKind})
				}
				if resCtx.existingResource != nil {
					if resCtx.ownKind != ChildInitial {
						diffMap[forRef].deleteObjs = append(diffMap[forRef].deleteObjs, object{ref: ownRef, obj: *resCtx.existingResource, ownKind: resCtx.ownKind})
					}
				}
			}
		} else {
			for ownRef, resCtx := range r.get(ownGVKKind, []corev1.ObjectReference{forRef, {}}) {
				if r.debug {
					fn.Logf("diff: objRef: %s, existingResource: %v, newResource: %v, existing condition: %v\n", ref.GetRefsString(forRef, ownRef), resCtx.existingResource != nil, resCtx.newResource != nil, resCtx.existingCondition != nil)
				}
				// condition diff handling
				switch {
				case resCtx.newResource == nil && resCtx.existingCondition == nil:
					if forResCtx.existingCondition == nil || (forResCtx.existingCondition != nil && forResCtx.existingCondition.Status != v1.ConditionFalse) {
						diffMap[forRef].updateForCondition = true
					}
					if r.ready {
						diffMap[forRef].createConditions = append(diffMap[forRef].createConditions, object{ref: ownRef, ownKind: resCtx.ownKind})
					}
				// if there is no new resource, but we have a condition for that resource we should delete the condition
				// if the ownKind is not ChildInitial || ChildLocal -> this would happen in stage 2 of upf deployment
				case resCtx.newResource == nil && resCtx.existingCondition != nil:
					if forResCtx.existingCondition == nil || (forResCtx.existingCondition != nil && forResCtx.existingCondition.Status != v1.ConditionFalse) {
						diffMap[forRef].updateForCondition = true
					}
					if resCtx.ownKind != ChildInitial && resCtx.ownKind != ChildLocal {
						diffMap[forRef].deleteConditions = append(diffMap[forRef].deleteConditions, object{ref: ownRef, ownKind: resCtx.ownKind})
					}
				// if there is a new resource, but we have no condition for that resource someone deleted it
				// and we have to recreate that condition
				case resCtx.newResource != nil && resCtx.existingCondition == nil:
					if forResCtx.existingCondition == nil || (forResCtx.existingCondition != nil && forResCtx.existingCondition.Status != v1.ConditionFalse) {
						diffMap[forRef].updateForCondition = true
					}
					diffMap[forRef].createConditions = append(diffMap[forRef].createConditions, object{ref: ownRef, obj: *resCtx.newResource, ownKind: resCtx.ownKind})
				}

				// resource diff handling
				switch {
				// if the existing resource does not exist but the new resource exist we have to create the new resource
				case resCtx.existingResource == nil && resCtx.newResource != nil:
					// create resource
					if forResCtx.existingCondition == nil || (forResCtx.existingCondition != nil && forResCtx.existingCondition.Status != v1.ConditionFalse) {
						diffMap[forRef].updateForCondition = true
					}
					diffMap[forRef].createObjs = append(diffMap[forRef].createObjs, object{ref: ownRef, obj: *resCtx.newResource, ownKind: resCtx.ownKind})
				// if the new resource does not exist and but the resource exist we have to delete the existing resource
				case resCtx.existingResource != nil && resCtx.newResource == nil:
					// delete resource
					if forResCtx.existingCondition == nil || (forResCtx.existingCondition != nil && forResCtx.existingCondition.Status != v1.ConditionFalse) {
						diffMap[forRef].updateForCondition = true
					}
					if resCtx.ownKind != ChildInitial && resCtx.ownKind != ChildLocal {
						diffMap[forRef].deleteObjs = append(diffMap[forRef].deleteObjs, object{ref: ownRef, obj: *resCtx.existingResource, ownKind: resCtx.ownKind})
					}
				// if both existing/new resource exists check the differences of the spec
				// depending on the outcome update the resource with the new information
				case resCtx.existingResource != nil && resCtx.newResource != nil:
					// for child remote condition a diff is not needed since the object
					// is created remotely
					if resCtx.ownKind != ChildRemoteCondition {
						// check diff
						existingSpec, err := getSpec(resCtx.existingResource)
						if err != nil {
							fn.Logf("cannot get spec from existing obj, err: %v\n", err)
							continue
						}
						newSpec, err := getSpec(resCtx.newResource)
						if err != nil {
							fn.Logf("cannot get spec from existing obj, err: %v\n", err)
							continue
						}

						if d := cmp.Diff(existingSpec, newSpec); d != "" {
							if forResCtx.existingCondition == nil || (forResCtx.existingCondition != nil && forResCtx.existingCondition.Status != v1.ConditionFalse) {
								diffMap[forRef].updateForCondition = true
							}
							diffMap[forRef].updateObjs = append(diffMap[forRef].updateObjs, object{ref: ownRef, obj: *resCtx.newResource, ownKind: resCtx.ownKind})
						}
						// this is a corner case, in case for object gets deleted and recreated
						// if the delete annotation is set, we need to cleanup the
						// delete annotation and set the condition to update
						a := resCtx.existingResource.GetAnnotations()
						if _, ok := a[SpecializerDelete]; ok {
							//fn.Logf("delete annotation: %v\n", a)
							if _, ok := a[SpecializerDelete]; ok {
								if forResCtx.existingCondition.Status != v1.ConditionFalse {
									diffMap[forRef].updateForCondition = true
								}
								diffMap[forRef].updateDeleteAnnotations = append(diffMap[forRef].updateDeleteAnnotations, object{ref: ownRef, obj: *resCtx.newResource, ownKind: resCtx.ownKind})
							}
						}
					}
				}
			}
		}
	}
	return diffMap
}

func getSpec(o *fn.KubeObject) (map[string]any, error) {
	spec := &map[string]any{}
	ok, err := o.NestedResource(spec, "spec")
	if err != nil {
		return nil, fmt.Errorf("cannot get spec from obj, err %v", err)
	}
	if !ok {
		return nil, fmt.Errorf("cannot get spec from obj, not found")
	}
	return *spec, nil
}
