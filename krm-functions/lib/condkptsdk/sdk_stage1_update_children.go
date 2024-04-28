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
	"errors"
	"sort"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	kptv1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	kptfilelibv1 "github.com/nephio-project/nephio/krm-functions/lib/kptfile/v1"
	"github.com/nephio-project/nephio/krm-functions/lib/ref"
	corev1 "k8s.io/api/core/v1"
)

// performs the update on the children after the diff in the stage1 of the pipeline
func (r *sdk) updateChildren() {
	// perform a diff to validate the existing resource against the new resources
	diffMap := r.inv.diff()
	// if the fn is not ready we delete the for condition and its children
	if !r.inv.isReady() {
		for forRef, diff := range diffMap {
			var e error
			// delete the overall condition for the object
			if diff.deleteForCondition {
				if r.debug {
					fn.Logf("stage1: diff action -> delete for condition objRef: %s\n", ref.GetRefsString(forRef))
				}
				// deletes the for condition from the kptfile and inventory
				if err := r.deleteCondition(forGVKKind, []corev1.ObjectReference{forRef}); err != nil {
					// the errors are already logged, we set the result in the for condition
					if err := errors.Join(e, err); err != nil {
						fn.Logf("join error, err: %s\n", err.Error())
						r.rl.Results.ErrorE(err)
					}
				}
			}
			// delete all child resources by setting the annotation and set the condition to false
			for _, obj := range diff.deleteObjs {
				if r.debug {
					fn.Logf("stage1: diff action -> delete child objRef: %s\n", ref.GetRefsString(forRef, obj.ref))
				}
				if err := r.deleteChildObject(ownGVKKind, []corev1.ObjectReference{forRef, obj.ref}, obj, "not ready"); err != nil {
					// the errors are already logged, we set the result in the for condition
					if err := errors.Join(e, err); err != nil {
						fn.Logf("join error, err: %s\n", err.Error())
						r.rl.Results.ErrorE(err)
					}
				}
			}
			// handle all errors and set them in the condition
			if e != nil {
				if err := r.kptfile.SetConditionRefFailed(forRef, e.Error()); err != nil {
					// we continue but put the result in the resourcelist as this is the only way to convey the message
					fn.Logf("stage1: cannot set the condition objRef: %s err: %v", ref.GetRefsString(forRef), err.Error())
					r.rl.Results.ErrorE(err)
				}
			}
		}
		return
	}
	// we are in ready status -> act upon the diff
	for _, forRef := range diffMapKeysInDeterministicOrder(diffMap) {
		forRef := forRef // to get rid of the gosec error: G601 (CWE-118): Implicit memory aliasing in for loop.
		diff := diffMap[forRef]

		var e error
		// update conditions
		if diff.updateForCondition {
			if r.debug {
				fn.Logf("stage1: diff action -> update for condition objRef: %s\n", ref.GetRefsString(forRef))
			}
			if err := r.setCondition(forGVKKind, []corev1.ObjectReference{forRef}, "update for condition", kptv1.ConditionFalse, false); err != nil {
				// the errors are already logged, we set the result in the for condition
				if err := errors.Join(e, err); err != nil {
					fn.Logf("join error, err: %s\n", err.Error())
					r.rl.Results.ErrorE(err)
				}
			}

		}
		sortObjects(diff.createConditions)
		for _, obj := range diff.createConditions {
			if r.debug {
				fn.Logf("stage1: diff action -> create condition objRef: %s\n", ref.GetRefsString(forRef, obj.ref))
			}
			status := kptv1.ConditionFalse
			msg := "create condition"
			if obj.ownKind == ChildLocal {
				// For local resources the condition can be set to true upon create
				status = kptv1.ConditionTrue
				msg = "child local resource -> done"
			}
			if obj.ownKind == ChildInitial {
				msg = "create initial resource condition"
			}
			if err := r.setCondition(ownGVKKind, []corev1.ObjectReference{forRef, obj.ref}, msg, status, false); err != nil {
				// the errors are already logged, we set the result in the for condition
				if err := errors.Join(e, err); err != nil {
					fn.Logf("join error, err: %s\n", err.Error())
					r.rl.Results.ErrorE(err)
				}
			}
		}
		for _, obj := range diff.deleteConditions {
			if r.debug {
				fn.Logf("stage1: diff action -> delete condition objRef: %s\n", ref.GetRefsString(forRef, obj.ref))
			}
			if err := r.deleteCondition(ownGVKKind, []corev1.ObjectReference{forRef, obj.ref}); err != nil {
				// the errors are already logged, we set the result in the for condition
				if err := errors.Join(e, err); err != nil {
					fn.Logf("join error, err: %s\n", err.Error())
					r.rl.Results.ErrorE(err)
				}
			}
		}
		// update resources
		sortObjects(diff.createObjs)
		for _, obj := range diff.createObjs {
			if r.debug {
				fn.Logf("stage1: diff action -> create obj: ref: %s, ownkind: %s\n", kptfilelibv1.GetConditionType(&obj.ref), obj.ownKind)
				// #nosec G601
			}
			status := kptv1.ConditionFalse
			if obj.ownKind == ChildLocal {
				// For local resources the condition can be set to true upon create
				status = kptv1.ConditionTrue
			}
			if err := r.upsertChildObject(ownGVKKind, []corev1.ObjectReference{forRef, obj.ref}, obj, nil, "create initial resource", status, false); err != nil {
				// the errors are already logged, we set the result in the for condition
				if err := errors.Join(e, err); err != nil {
					fn.Logf("join error, err: %s\n", err.Error())
					r.rl.Results.ErrorE(err)
				}
			}
		}
		sortObjects(diff.updateObjs)
		for _, obj := range diff.updateObjs {
			if r.debug {
				fn.Logf("stage1: diff action -> update obj: %s\n", kptfilelibv1.GetConditionType(&obj.ref))
				// #nosec G601
			}
			if err := r.upsertChildObject(ownGVKKind, []corev1.ObjectReference{forRef, obj.ref}, obj, nil, "update resource", kptv1.ConditionFalse, false); err != nil {
				// the errors are already logged, we set the result in the for condition
				if err := errors.Join(e, err); err != nil {
					fn.Logf("join error, err: %s\n", err.Error())
					r.rl.Results.ErrorE(err)
				}
			}
		}
		for _, obj := range diff.deleteObjs {
			if r.debug {
				fn.Logf("stage1: diff action -> delete obj: %s\n", kptfilelibv1.GetConditionType(&obj.ref))
				// #nosec G601
			}
			if err := r.deleteChildObject(ownGVKKind, []corev1.ObjectReference{forRef, obj.ref}, obj, "delete resource"); err != nil {
				// the errors are already logged, we set the result in the for condition
				if err := errors.Join(e, err); err != nil {
					fn.Logf("join error, err: %s\n", err.Error())
					r.rl.Results.ErrorE(err)
				}
			}
		}
		// this is a corner case, in case for object gets deleted and recreated
		// if the delete annotation is set, we need to cleanup the
		// delete annotation and set the condition to update
		for _, obj := range diff.updateDeleteAnnotations {
			if r.debug {
				fn.Log("stage1: diff action -> update delete annotation")
			}
			if err := r.upsertChildObject(ownGVKKind, []corev1.ObjectReference{forRef, obj.ref}, obj, nil, "update resource", kptv1.ConditionFalse, false); err != nil {
				// the errors are already logged, we set the result in the for condition
				if err := errors.Join(e, err); err != nil {
					fn.Logf("join error, err: %s\n", err.Error())
					r.rl.Results.ErrorE(err)
				}
			}
		}
		// handle all errors and set them in the condition
		if e != nil {
			if err := r.kptfile.SetConditionRefFailed(forRef, e.Error()); err != nil {
				// we continue but put the result in the resourcelist as this is the only way to convey the message
				fn.Logf("stage1: cannot set the condition objRef: %s err: %v", ref.GetRefsString(forRef), err.Error())
				r.rl.Results.ErrorE(err)
			}
		}
	}
}

func diffMapKeysInDeterministicOrder(diffMap map[corev1.ObjectReference]*inventoryDiff) []corev1.ObjectReference {
	keys := make([]corev1.ObjectReference, 0, len(diffMap))
	for k := range diffMap {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].String() < keys[j].String()
	})
	return keys
}

func sortObjects(objs []object) {
	sort.Slice(objs, func(i, j int) bool {
		return objs[i].ref.String() < objs[j].ref.String()
	})
}
