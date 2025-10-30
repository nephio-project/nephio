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
	"fmt"

	"github.com/kptdev/krm-functions-sdk/go/fn"
	kptfilelibv1 "github.com/nephio-project/nephio/krm-functions/lib/kptfile/v1"
	"github.com/nephio-project/nephio/krm-functions/lib/ref"
	kptv1 "github.com/nephio-project/porch/pkg/kpt/api/kptfile/v1"
	corev1 "k8s.io/api/core/v1"
)

// delete object is always a ownkind -> 2 refs
func (r *sdk) deleteChildObject(kind gvkKind, refs []corev1.ObjectReference, obj object, msg string) error {
	// get condition
	c, err := kptfilelibv1.GetConditionByRef(refs, msg, kptv1.ConditionFalse, nil)
	if err != nil {
		// this is an internal error -> return here since this also validate the refs
		return err
	}
	var e error
	// set the condition in the kptfile
	if err := r.kptfile.SetConditions(c); err != nil {
		// this is an internal error -> return
		e = errors.Join(e, err)
		fn.Logf("cannot set condition in kptfile objref: %s, err: %v\n", ref.GetRefsString(refs...), err.Error())
		r.rl.Results.ErrorE(err)
	}
	// set delete annotation on the obj before updating the resourcelist and inventory
	if err := obj.obj.SetAnnotation(SpecializerDelete, "true"); err != nil {
		e = errors.Join(e, err)
		fn.Logf("cannot set annotation on obj objref: %s, err: %v\n", ref.GetRefsString(refs...), err.Error())
		r.rl.Results.ErrorE(err)
	}
	// set the obj in the resourcelist and inventory
	if err := r.setObjectInResourceList(ownGVKKind, refs, obj); err != nil {
		e = errors.Join(e, err)
		fn.Logf("cannot set resource in resourceList objref: %s, err: %v\n", ref.GetRefsString(refs...), err.Error())
		r.rl.Results.ErrorE(err)
	}
	return e
}

// upsertChildObject is always a ownkind -> 2 refs
// ec: existing condition used to avoid overriding information that was already present
func (r *sdk) upsertChildObject(kind gvkKind, refs []corev1.ObjectReference, obj object, ec *kptv1.Condition, msg string, status kptv1.ConditionStatus, alwaysUpdate bool) error {
	// get condition
	c, err := kptfilelibv1.GetConditionByRef(refs, msg, status, ec)
	if err != nil {
		// this is an internal error -> return here since this also validate the refs
		return err
	}
	// for an existing condition use the present condition since it can contain data from a for resource we cannot replace and don't know as it was set
	// by a fn before this fn was called. e.g. reason in a forResource
	if ec != nil {
		c = *ec
		c.Message = msg
		c.Status = status
	}

	var e error
	// set the condition in the kptfile
	if err := r.kptfile.SetConditions(c); err != nil {
		// this is an internal error -> return
		e = errors.Join(e, err)
		fn.Logf("cannot set condition in kptfile objref: %s, err: %v\n", ref.GetRefsString(refs...), err.Error())
		r.rl.Results.ErrorE(err)
	}
	if alwaysUpdate {
		if err := r.setObjectInResourceList(kind, refs, obj); err != nil {
			e = errors.Join(e, err)
			fn.Logf("cannot set resource in resourceList objref: %s, err: %v\n", ref.GetRefsString(refs...), err.Error())
			r.rl.Results.ErrorE(err)
		}
		return e
	}
	// set the obj in the resourcelist and inventory
	if obj.ownKind == ChildRemote || obj.ownKind == ChildLocal {
		if err := r.setObjectInResourceList(ownGVKKind, refs, obj); err != nil {
			e = errors.Join(e, err)
			fn.Logf("cannot set resource in resourceList objref: %s, err: %v\n", ref.GetRefsString(refs...), err.Error())
			r.rl.Results.ErrorE(err)
		}
	}
	return e
}

// deleteCondition deletes the condition from the kptfile and inventory
// used only for forResources and ownResources
func (r *sdk) deleteCondition(kind gvkKind, refs []corev1.ObjectReference) error {
	c, err := kptfilelibv1.GetConditionByRef(refs, "", kptv1.ConditionFalse, nil)
	if err != nil {
		// this is an internal error -> return here since this also validate the refs
		return err
	}
	var e error
	// delete the condition from the kptfile
	if err := r.kptfile.DeleteCondition(c.Type); err != nil {
		e = errors.Join(e, err)
		fn.Logf("cannot delete condition from Kptfile objref: %s, err: %v\n", ref.GetRefsString(refs...), err.Error())
		r.rl.Results.ErrorE(err)
	}
	// delete the condition from the inventory
	if err := r.inv.delete(&gvkKindCtx{gvkKind: kind}, refs); err != nil {
		e = errors.Join(e, err)
		fn.Logf("cannot delete condition from inventory objref: %s, err: %v\n", ref.GetRefsString(refs...), err.Error())
		r.rl.Results.ErrorE(err)
	}
	return e
}

// setCondition sets the condition in the kptfile and inventory
// used for forResources, ??
func (r *sdk) setCondition(kind gvkKind, refs []corev1.ObjectReference, msg string, status kptv1.ConditionStatus, failed bool) error {
	c, err := kptfilelibv1.GetConditionByRef(refs, msg, status, nil)
	if err != nil {
		// this is an internal error -> return here since this also validate the refs
		return err
	}
	var e error
	// set the condition in the kptfile
	if err := r.kptfile.SetConditions(c); err != nil {
		e = errors.Join(e, err)
		fn.Logf("cannot set condition in Kptfile objref: %s, err: %v\n", ref.GetRefsString(refs...), err.Error())
		r.rl.Results.ErrorE(err)
	}
	// set the condition from the inventory
	if err := r.inv.set(&gvkKindCtx{gvkKind: kind}, refs, &c, false, failed); err != nil {
		e = errors.Join(e, err)
		fn.Logf("cannot set condition in inventory objref: %s, err: %v\n", ref.GetRefsString(refs...), err.Error())
		r.rl.Results.ErrorE(err)
	}
	return e
}

func (r *sdk) setObjectInResourceList(kind gvkKind, refs []corev1.ObjectReference, obj object) error {
	if r.debug {
		fn.Logf("setObjectInResourceList: kind: %s, refs: %v, obj: %v\n", kind, refs, obj.obj)
	}
	if !ref.IsRefsValid(refs) {
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
		if err := r.inv.set(&gvkKindCtx{gvkKind: kind}, []corev1.ObjectReference{forRef}, &obj.obj, false, false); err != nil {
			fn.Logf("error updating stage1 resource to the inventory: %v\n", err.Error())
			r.rl.Results = append(r.rl.Results, fn.ErrorResult(err))
			return err
		}
		return nil
	}
	objRef := refs[1]
	if err := r.rl.UpsertObjectToItems(&obj.obj, nil, true); err != nil {
		fn.Logf("error updating stage1 resource: %v\n", err.Error())
		r.rl.Results = append(r.rl.Results, fn.ErrorResult(err))
		return err
	}
	// update the resource status back in the inventory
	if err := r.inv.set(&gvkKindCtx{gvkKind: kind}, []corev1.ObjectReference{forRef, objRef}, &obj.obj, false, false); err != nil {
		fn.Logf("error updating stage1 resource to the inventory: %v\n", err.Error())
		r.rl.Results = append(r.rl.Results, fn.ErrorResult(err))
		return err
	}
	return nil
}

func (r *sdk) deleteObjFromResourceList(obj *fn.KubeObject) {
	for idx, o := range r.rl.Items {
		if ref.IsGVKNNEqual(o, obj) {
			r.rl.Items = append(r.rl.Items[:idx], r.rl.Items[idx+1:]...)
		}
	}
}
