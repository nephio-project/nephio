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
func (r *sdk) handleUpdate(a action, kind gvkKind, refs []corev1.ObjectReference, obj object, c *kptv1.Condition, status kptv1.ConditionStatus, msg string, ignoreOwnKind bool) error {
	// set the condition
	var err error
	if c == nil {
		err = r.setConditionByRef(a, kind, refs, status, msg)
	} else {
		err = r.setConditionInKptFile(a, *c, status, msg)
	}
	if err != nil {
		fn.Logf("error setting condition in kptfile: %v\n", err.Error())
		r.rl.Results.ErrorE(err)
		return err
	}

	// update resource
	if a == actionDelete {
		if err := obj.obj.SetAnnotation(SpecializerDelete, "true"); err != nil {
			fn.Logf("error setting annotation: %v\n", err.Error())
			r.rl.Results.ErrorE(err)
			return err
		}
	}
	// set resource
	if ignoreOwnKind {
		if err := r.setObjectInResourceList(kind, refs, obj); err != nil {
			fn.Logf("error setting resource in resourceList: %v\n", err.Error())
			r.rl.Results.ErrorE(err)
			return err
		}
	} else {
		if obj.ownKind == ChildRemote {
			if err := r.setObjectInResourceList(kind, refs, obj); err != nil {
				fn.Logf("error setting resource in resourceList: %v\n", err.Error())
				r.rl.Results.ErrorE(err)
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
		err := r.conditions.DeleteByObjectReference(forRef)
		if err != nil {
			fn.Logf("error deleting condition from Kptfile: %v\n", err.Error())
			r.rl.Results.ErrorE(err)
			return err
		}

		// update the status back in the inventory
		if err := r.inv.delete(&gvkKindCtx{gvkKind: kind}, []corev1.ObjectReference{forRef}); err != nil {
			fn.Logf("error deleting condition from inventory: %v\n", err.Error())
			r.rl.Results.ErrorE(err)
			return err
		}
	} else {
		objRef := refs[1]
		// delete condition
		err := r.conditions.DeleteByObjectReference(objRef)
		if err != nil {
			fn.Logf("error deleting condition from Kptfile: %v\n", err.Error())
			r.rl.Results.ErrorE(err)
			return err
		}
		// update the status back in the inventory
		if err := r.inv.delete(&gvkKindCtx{gvkKind: kind}, []corev1.ObjectReference{forRef, objRef}); err != nil {
			fn.Logf("error deleting condition from inventory: %v\n", err.Error())
			r.rl.Results.ErrorE(err)
			return err
		}
	}
	return nil
}

func (r *sdk) setConditionInKptFile(a action, c kptv1.Condition, status kptv1.ConditionStatus, msg string) error {
	c.Message = fmt.Sprintf("%s %s", a, msg)
	c.Status = status
	return r.conditions.Set(c)
}

func (r *sdk) setConditionByRef(a action, kind gvkKind, refs []corev1.ObjectReference, status kptv1.ConditionStatus, msg string) error {
	if !isRefsValid(refs) {
		return fmt.Errorf("cannot set resource in resource list as the object has no valid refs: %v", refs)
	}
	forRef := refs[0]
	if len(refs) == 1 {
		c := kptv1.Condition{
			Type:    kptfilelibv1.GetConditionType(&forRef),
			Status:  status,
			Message: fmt.Sprintf("%s %s", a, msg),
		}
		return r.conditions.Set(c)
	}
	objRef := refs[1]
	c := kptv1.Condition{
		Type:    kptfilelibv1.GetConditionType(&objRef),
		Status:  status,
		Reason:  fmt.Sprintf("%s.%s", kptfilelibv1.GetConditionType(&r.cfg.For), forRef.Name),
		Message: fmt.Sprintf("%s %s", a, msg),
	}
	err := r.conditions.Set(c)
	if err != nil {
		r.rl.Results.ErrorE(err)
		return err
	}
	// update the condition status back in the inventory
	if err := r.inv.set(&gvkKindCtx{gvkKind: kind}, []corev1.ObjectReference{forRef, objRef}, &c, false); err != nil {
		fn.Logf("error updating stage1 resource to the inventory: %v\n", err.Error())
		r.rl.Results.ErrorE(err)
		return err
	}
	return nil
}

func (r *sdk) setObjectInResourceList(kind gvkKind, refs []corev1.ObjectReference, obj object) error {
	if r.debug {
		fn.Logf("setObjectInResourceList: kind: %s, refs: %v, obj: %v\n", kind, refs, obj.obj)
	}
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

func (r *sdk) deleteObjFromResourceList(obj *fn.KubeObject) {
	for idx, o := range r.rl.Items {
		if isGVKNNEqual(o, obj) {
			r.rl.Items = append(r.rl.Items[:idx], r.rl.Items[idx+1:]...)
		}
	}
}
