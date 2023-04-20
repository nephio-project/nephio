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
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	kptv1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	kptfilelibv1 "github.com/nephio-project/nephio/krm-functions/lib/kptfile/v1"
	corev1 "k8s.io/api/core/v1"
)

func (r *sdk) populateChildren() {
	// validate if the general watches are available to populate the ownResources
	// if no own resources exist there is not need to run this
	if len(r.cfg.Owns) > 0 && !r.inv.isReady() {
		return
	}

	fn.Log("populate children: ready:", r.ready)
	for forRef, resCtx := range r.inv.get(forGVKKind, nil) {
		forObj := resCtx.existingResource
		fn.Log("PopulateOwnResourcesFn", forObj)
		if r.cfg.PopulateOwnResourcesFn != nil && forObj != nil {
			res, err := r.cfg.PopulateOwnResourcesFn(forObj)
			if err != nil {
				fn.Logf("error populating new resource: %v\n", err.Error())
				r.rl.Results = append(r.rl.Results, fn.ErrorConfigObjectResult(err, forObj))
			} else {
				for _, newObj := range res {
					objRef := corev1.ObjectReference{APIVersion: newObj.GetAPIVersion(), Kind: newObj.GetKind(), Name: newObj.GetName()}
					kc, ok := r.inv.isGVKMatch(&objRef)
					if !ok {
						fn.Logf("populate new resource: forRef %v objRef %v cannot find resource in gvkmap\n", forRef, objRef)
					} else {
						fn.Logf("populate new resource: forRef %v objRef %v kc: %v\n", forRef, objRef, kc)
						// set owner reference on the new resource
						if err := newObj.SetAnnotation(FnRuntimeOwner, kptfilelibv1.GetConditionType(&forRef)); err != nil {
							fn.Logf("error setting new annotation: %v\n", err.Error())
							r.rl.Results = append(r.rl.Results, fn.ErrorConfigObjectResult(err, newObj))
						}
						// add the resource to the existing list as a new resource
						if err := r.inv.set(kc, []corev1.ObjectReference{forRef, objRef}, newObj, true); err != nil {
							fn.Logf("error setting new resource to the inventory: %v\n", err.Error())
							r.rl.Results = append(r.rl.Results, fn.ErrorConfigObjectResult(err, newObj))
						}
					}
				}
			}
		}
	}
}

// performs the update on the children after the diff in the stage1 of the pipeline
func (r *sdk) updateChildren() {
	// perform a diff to validate the existing resource against the new resources
	diffMap, err := r.inv.diff()
	if err != nil {
		r.rl.Results = append(r.rl.Results, fn.ErrorConfigObjectResult(err, r.rl.Items.GetRootKptfile()))
	}
	fn.Logf("diff: %v\n", diffMap)

	// if the fn is not ready to act we stop immediately
	if !r.inv.isReady() {
		for forRef, diff := range diffMap {
			// delete the overall condition for the object
			if diff.deleteForCondition {
				fn.Logf("diff action -> delete for condition: %s\n", kptfilelibv1.GetConditionType(&forRef))
				r.deleteConditionInKptFile(ownGVKKind, []*corev1.ObjectReference{&forRef})
			}
			// delete all child resources by setting the annotation and set the condition to false
			for _, obj := range diff.deleteObjs {
				fn.Logf("diff action ->  delete set condition: %s\n", kptfilelibv1.GetConditionType(&obj.ref))
				r.handleUpdate(actionDelete, ownGVKKind, []*corev1.ObjectReference{&forRef, &obj.ref}, obj, kptv1.ConditionFalse, "not ready", true)
			}
		}
	} else {
		// act upon the diff
		for forRef, diff := range diffMap {
			// update conditions
			if diff.updateForCondition {
				fn.Logf("diff action ->  update for condition: %s\n", kptfilelibv1.GetConditionType(&forRef))
				r.setConditionInKptFile(actionUpdate, ownGVKKind, []*corev1.ObjectReference{&forRef}, kptv1.ConditionFalse, "for condition")
			}
			for _, obj := range diff.createConditions {
				fn.Logf("diff action ->  create condition: %s\n", kptfilelibv1.GetConditionType(&obj.ref))
				r.setConditionInKptFile(actionUpdate, ownGVKKind, []*corev1.ObjectReference{&forRef, &obj.ref}, kptv1.ConditionFalse, "condition again as it was deleted")
			}
			for _, obj := range diff.deleteConditions {
				fn.Logf("diff action ->  delete condition: %s\n", kptfilelibv1.GetConditionType(&obj.ref))
				r.deleteConditionInKptFile(ownGVKKind, []*corev1.ObjectReference{&forRef, &obj.ref})
			}
			// update resources
			for _, obj := range diff.createObjs {
				fn.Logf("diff action -> create obj: ref: %s, ownkind: %s\n", kptfilelibv1.GetConditionType(&obj.ref), obj.ownKind)
				r.handleUpdate(actionCreate, ownGVKKind, []*corev1.ObjectReference{&forRef, &obj.ref}, obj, kptv1.ConditionFalse, "resource", false)
			}
			for _, obj := range diff.updateObjs {
				fn.Logf("diff action -> update obj: %s\n", kptfilelibv1.GetConditionType(&obj.ref))
				r.handleUpdate(actionUpdate, ownGVKKind, []*corev1.ObjectReference{&forRef, &obj.ref}, obj, kptv1.ConditionFalse, "resource", false)
			}
			for _, obj := range diff.deleteObjs {
				fn.Logf("diff action -> delete obj: %s\n", kptfilelibv1.GetConditionType(&obj.ref))
				r.handleUpdate(actionDelete, ownGVKKind, []*corev1.ObjectReference{&forRef, &obj.ref}, obj, kptv1.ConditionFalse, "resource", true)
			}
			// this is a corner case, in case for object gets deleted and recreated
			// if the delete annotation is set, we need to cleanup the
			// delete annotation and set the condition to update
			for _, obj := range diff.updateDeleteAnnotations {
				fn.Log("diff action -> update delete annotation")
				r.handleUpdate(actionCreate, ownGVKKind, []*corev1.ObjectReference{&forRef, &obj.ref}, obj, kptv1.ConditionFalse, "resource", true)
			}
		}
	}
	// update the kptfile with the latest consitions
	r.updateKptFile()
}
