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
	kptv1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	kptfilelibv1 "github.com/nephio-project/nephio/krm-functions/lib/kptfile/v1"
	corev1 "k8s.io/api/core/v1"
)

func (r *sdk) populateChildren() error {
	// if no own resources exist there is not need to run this
	if len(r.cfg.Owns) == 0 {
		return nil
	}
	// validate if we are ready, if not we return
	// TBD if we need to cleanup own resources
	if !r.ready || !r.inv.isReady() {
		// TBD cleanup own resources
		return nil
	}

	fn.Log("populate children: ready:", r.ready)
	for forRef, resCtx := range r.inv.get(forGVKKind, []corev1.ObjectReference{{}}) {
		forObj := resCtx.existingResource
		fn.Log("PopulateOwnResourcesFn", forObj)
		if r.cfg.PopulateOwnResourcesFn != nil && forObj != nil {
			res, err := r.cfg.PopulateOwnResourcesFn(forObj)
			if err != nil {
				fn.Logf("error populating new resource: %v\n", err.Error())
				r.rl.Results = append(r.rl.Results, fn.ErrorConfigObjectResult(err, forObj))
				return err
			} else {
				for _, newObj := range res {
					objRef := corev1.ObjectReference{APIVersion: newObj.GetAPIVersion(), Kind: newObj.GetKind(), Name: newObj.GetName()}
					kc, ok := r.inv.isGVKMatch(&objRef)
					if !ok {
						fn.Logf("populate new resource: forRef %v objRef %v cannot find resource in gvkmap\n", forRef, objRef)
						return fmt.Errorf("populate new resource: forRef %v objRef %v cannot find resource in gvkmap", forRef, objRef)
					}
					fn.Logf("populate new resource: forRef %v objRef %v kc: %v\n", forRef, objRef, kc)
					// set owner reference on the new resource
					if err := newObj.SetAnnotation(FnRuntimeOwner, kptfilelibv1.GetConditionType(&forRef)); err != nil {
						fn.Logf("error setting new annotation: %v\n", err.Error())
						r.rl.Results = append(r.rl.Results, fn.ErrorResult(err))
						return err
					}
					// add the resource to the existing list as a new resource
					if err := r.inv.set(kc, []corev1.ObjectReference{forRef, objRef}, newObj, true); err != nil {
						fn.Logf("error setting new resource to the inventory: %v\n", err.Error())
						r.rl.Results = append(r.rl.Results, fn.ErrorResult(err))
						return err
					}
				}
			}
		}
	}
	return nil
}

// performs the update on the children after the diff in the stage1 of the pipeline
func (r *sdk) updateChildren() error {
	// perform a diff to validate the existing resource against the new resources
	diffMap, err := r.inv.diff()
	if err != nil {
		r.rl.Results = append(r.rl.Results, fn.ErrorConfigObjectResult(err, r.rl.Items.GetRootKptfile()))
		return err
	}
	fn.Logf("diff: %v\n", diffMap)

	// if the fn is not ready to act we stop immediately
	if !r.ready || !r.inv.isReady() {
		for forRef, diff := range diffMap {
			// delete the overall condition for the object
			if diff.deleteForCondition {
				fn.Logf("diff action -> delete for condition: %s\n", kptfilelibv1.GetConditionType(&forRef))
				if err := r.deleteConditionInKptFile(ownGVKKind, []corev1.ObjectReference{forRef}); err != nil {
					return err
				}
			}
			// delete all child resources by setting the annotation and set the condition to false
			for _, obj := range diff.deleteObjs {
				fn.Logf("diff action ->  delete set condition: %s\n", kptfilelibv1.GetConditionType(&obj.ref))
				if err := r.handleUpdate(actionDelete, ownGVKKind, []corev1.ObjectReference{forRef, obj.ref}, obj, kptv1.ConditionFalse, "not ready", true); err != nil {
					return err
				}
			}
		}
	} else {
		// act upon the diff
		for forRef, diff := range diffMap {
			// update conditions
			if diff.updateForCondition {
				fn.Logf("diff action ->  update for condition: %s\n", kptfilelibv1.GetConditionType(&forRef))
				if err := r.setConditionInKptFile(actionUpdate, ownGVKKind, []corev1.ObjectReference{forRef}, kptv1.ConditionFalse, "for condition"); err != nil {
					return err
				}
			}
			for _, obj := range diff.createConditions {
				fn.Logf("diff action ->  create condition: %s\n", kptfilelibv1.GetConditionType(&obj.ref))
				if err := r.setConditionInKptFile(actionUpdate, ownGVKKind, []corev1.ObjectReference{forRef, obj.ref}, kptv1.ConditionFalse, "condition again as it was deleted"); err != nil {
					return err
				}
			}
			for _, obj := range diff.deleteConditions {
				fn.Logf("diff action ->  delete condition: %s\n", kptfilelibv1.GetConditionType(&obj.ref))
				if err := r.deleteConditionInKptFile(ownGVKKind, []corev1.ObjectReference{forRef, obj.ref}); err != nil {
					return err
				}
			}
			// update resources
			for _, obj := range diff.createObjs {
				fn.Logf("diff action -> create obj: ref: %s, ownkind: %s\n", kptfilelibv1.GetConditionType(&obj.ref), obj.ownKind)
				if err := r.handleUpdate(actionCreate, ownGVKKind, []corev1.ObjectReference{forRef, obj.ref}, obj, kptv1.ConditionFalse, "resource", false); err != nil {
					return err
				}
			}
			for _, obj := range diff.updateObjs {
				fn.Logf("diff action -> update obj: %s\n", kptfilelibv1.GetConditionType(&obj.ref))
				if err := r.handleUpdate(actionUpdate, ownGVKKind, []corev1.ObjectReference{forRef, obj.ref}, obj, kptv1.ConditionFalse, "resource", false); err != nil {
					return err
				}
			}
			for _, obj := range diff.deleteObjs {
				fn.Logf("diff action -> delete obj: %s\n", kptfilelibv1.GetConditionType(&obj.ref))
				if err := r.handleUpdate(actionDelete, ownGVKKind, []corev1.ObjectReference{forRef, obj.ref}, obj, kptv1.ConditionFalse, "resource", true); err != nil {
					return err
				}
			}
			// this is a corner case, in case for object gets deleted and recreated
			// if the delete annotation is set, we need to cleanup the
			// delete annotation and set the condition to update
			for _, obj := range diff.updateDeleteAnnotations {
				fn.Log("diff action -> update delete annotation")
				if err := r.handleUpdate(actionCreate, ownGVKKind, []corev1.ObjectReference{forRef, obj.ref}, obj, kptv1.ConditionFalse, "resource", true); err != nil {
					return err
				}
			}
		}
	}
	// update the kptfile with the latest consitions
	return r.updateKptFile()
}
