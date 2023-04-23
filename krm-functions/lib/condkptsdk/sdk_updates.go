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

// handleUpdate sets the condition and resource based on the action
// action: create/update/delete
// kind: own/for/watch
func (r *sdk) handleUpdate(a action, kind gvkKind, refs []corev1.ObjectReference, obj *object, c *kptv1.Condition, status kptv1.ConditionStatus, msg string, ignoreOwnKind bool) error {
	// set the condition
	if err := r.setConditionInKptFile(a, kind, refs, c, status, msg); err != nil {
		fn.Logf("error setting condition in kptfile: %v\n", err.Error())
		r.rl.Results = append(r.rl.Results, fn.ErrorResult(err))
		return err
	}
	// update resource
	if a == actionDelete {
		if err := obj.obj.SetAnnotation(FnRuntimeDelete, "true"); err != nil {
			fn.Logf("error setting annotation: %v\n", err.Error())
			r.rl.Results = append(r.rl.Results, fn.ErrorResult(err))
			return err
		}
	}
	// set resource
	if ignoreOwnKind {
		if err := r.setObjectInResourceList(kind, refs, obj); err != nil {
			fn.Logf("error setting resource in resourceList: %v\n", err.Error())
			r.rl.Results = append(r.rl.Results, fn.ErrorResult(err))
			return err
		}
	} else {
		if obj.ownKind == ChildRemote {
			if err := r.setObjectInResourceList(kind, refs, obj); err != nil {
				fn.Logf("error setting resource in resourceList: %v\n", err.Error())
				r.rl.Results = append(r.rl.Results, fn.ErrorResult(err))
				return err
			}
		}
	}
	return nil
}

func (r *sdk) deleteConditionInKptFile(kind gvkKind, refs []corev1.ObjectReference) error {
	if !isRefsValid(refs) {
		return fmt.Errorf("cannot set resource in resourcelist as the object has no valid refs: %v", refs)
	}
	forRef := refs[0]
	if len(refs) == 1 {
		// delete condition
		r.kptf.DeleteCondition(kptfilelibv1.GetConditionType(&forRef))
		// update the status back in the inventory
		if err := r.inv.delete(&gvkKindCtx{gvkKind: kind}, []corev1.ObjectReference{forRef}); err != nil {
			fn.Logf("error deleting stage1 resource to the inventory: %v\n", err.Error())
			r.rl.Results = append(r.rl.Results, fn.ErrorResult(err))
			return err
		}
	} else {
		objRef := refs[1]
		// delete condition
		r.kptf.DeleteCondition(kptfilelibv1.GetConditionType(&objRef))
		// update the status back in the inventory
		if err := r.inv.delete(&gvkKindCtx{gvkKind: kind}, []corev1.ObjectReference{forRef, objRef}); err != nil {
			fn.Logf("error deleting stage1 resource to the inventory: %v\n", err.Error())
			r.rl.Results = append(r.rl.Results, fn.ErrorResult(err))
			return err
		}
	}
	return nil
}

func (r *sdk) setConditionInKptFile(a action, kind gvkKind, refs []corev1.ObjectReference, c *kptv1.Condition, status kptv1.ConditionStatus, msg string) error {
	if !isRefsValid(refs) {
		return fmt.Errorf("cannot set resource in resourcelist as the object has no valid refs: %v", refs)
	}
	if c != nil {
		c.Message = fmt.Sprintf("%s %s", a, msg)
		c.Status = status
		r.kptf.SetConditions(*c)
		return nil
	}
	forRef := refs[0]
	if len(refs) == 1 {
		c := kptv1.Condition{
			Type:    kptfilelibv1.GetConditionType(&forRef),
			Status:  status,
			Message: fmt.Sprintf("%s %s", a, msg),
		}
		r.kptf.SetConditions(c)
	} else {
		objRef := refs[1]
		c := kptv1.Condition{
			Type:    kptfilelibv1.GetConditionType(&objRef),
			Status:  status,
			Reason:  fmt.Sprintf("%s.%s", kptfilelibv1.GetConditionType(&r.cfg.For), forRef.Name),
			Message: fmt.Sprintf("%s %s", a, msg),
		}
		r.kptf.SetConditions(c)
		// update the condition status back in the inventory
		if err := r.inv.set(&gvkKindCtx{gvkKind: kind}, []corev1.ObjectReference{forRef, objRef}, &c, false); err != nil {
			fn.Logf("error updating stage1 resource to the inventory: %v\n", err.Error())
			r.rl.Results = append(r.rl.Results, fn.ErrorResult(err))
			return err
		}
	}
	return nil
}

func (r *sdk) setObjectInResourceList(kind gvkKind, refs []corev1.ObjectReference, obj *object) error {
	if !isRefsValid(refs) {
		return fmt.Errorf("cannot set resource in resourcelist as the object has no valid refs: %v", refs)
	}
	forRef := refs[0]
	if len(refs) == 1 {
		if err := r.rl.UpsertObjectToItems(&obj.obj, nil, true); err != nil {
			fn.Logf("error updating stage1 resource to the inventory: %v\n", err.Error())
			r.rl.Results = append(r.rl.Results, fn.ErrorResult(err))
			return err
		}
		// update the resource status back in the inventory
		if err := r.inv.set(&gvkKindCtx{gvkKind: kind}, []corev1.ObjectReference{forRef}, &obj.obj, false); err != nil {
			fn.Logf("error updating stage1 resource to the inventory: %v\n", err.Error())
			r.rl.Results = append(r.rl.Results, fn.ErrorResult(err))
			return err
		}
	} else {
		objRef := refs[1]
		if err := r.rl.UpsertObjectToItems(&obj.obj, nil, true); err != nil {
			fn.Logf("error updating stage1 resource: %v\n", err.Error())
			r.rl.Results = append(r.rl.Results, fn.ErrorResult(err))
			return err
		}
		// update the resource status back in the inventory
		if err := r.inv.set(&gvkKindCtx{gvkKind: kind}, []corev1.ObjectReference{forRef, objRef}, &obj.obj, false); err != nil {
			fn.Logf("error updating stage1 resource to the inventory: %v\n", err.Error())
			r.rl.Results = append(r.rl.Results, fn.ErrorResult(err))
			return err
		}
	}
	return nil
}

func (r *sdk) updateKptFile() error {
	kptfile, err := r.kptf.ParseKubeObject()
	if err != nil {
		fn.Log(err)
		r.rl.Results = append(r.rl.Results, fn.ErrorConfigObjectResult(err, r.rl.Items.GetRootKptfile()))
		return err
	}
	if err := r.rl.UpsertObjectToItems(kptfile, nil, true); err != nil {
		fn.Log(err)
		r.rl.Results = append(r.rl.Results, fn.ErrorConfigObjectResult(err, r.rl.Items.GetRootKptfile()))
		return err
	}
	return nil
}

func (r *sdk) deleteObjFromResourceList(obj *fn.KubeObject) {
	for idx, o := range r.rl.Items {
		if isGVKNNEqual(o, obj) {
			r.rl.Items = append(r.rl.Items[:idx], r.rl.Items[idx+1:]...)
		}
	}
}
