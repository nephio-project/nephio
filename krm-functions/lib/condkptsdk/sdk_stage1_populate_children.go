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
	kptfilelibv1 "github.com/nephio-project/nephio/krm-functions/lib/kptfile/v1"
	"github.com/nephio-project/nephio/krm-functions/lib/ref"
	kptv1 "github.com/nephio-project/porch/v4/pkg/kpt/api/kptfile/v1"
	corev1 "k8s.io/api/core/v1"
)

func (r *sdk) populateChildren() {
	if r.debug {
		fn.Log("stage1: populate children")
	}
	for forRef, resCtx := range r.inv.get(forGVKKind, []corev1.ObjectReference{{}}) {
		forObj := resCtx.existingResource
		if r.debug {
			fn.Logf("stage1: populateOwnResourcesFn objRef: %s\n", ref.GetRefsString(forRef))
		}
		if r.cfg.PopulateOwnResourcesFn != nil && forObj != nil {
			res, err := r.cfg.PopulateOwnResourcesFn(forObj)
			if err != nil {
				msg := fmt.Sprintf("stage1: cannot populate new resource err: %v", err.Error())
				// set the condition in the inventory and update the condition
				if err := r.setCondition(forGVKKind, []corev1.ObjectReference{forRef}, msg, kptv1.ConditionFalse, true); err != nil {
					// we continue but put the result in the resourcelist as this is the only way to convey the message
					fn.Logf("stage1: cannot set the condition objRef: %s err: %v", ref.GetRefsString(forRef), err.Error())
					r.rl.Results.ErrorE(err)
				}
				// we continue to the next forReference
				continue
			}
			for _, newObj := range res {
				objRef := corev1.ObjectReference{APIVersion: newObj.GetAPIVersion(), Kind: newObj.GetKind(), Name: newObj.GetName()}
				kc, ok := r.inv.isGVKMatch(&objRef)
				if !ok {
					msg := fmt.Sprintf("stage1: cannot find new resource in gvkmap: objRef: %s", ref.GetRefsString(forRef, objRef))
					if r.debug {
						fn.Log(msg)
					}
					if err := r.kptfile.SetConditionRefFailed(forRef, msg); err != nil {
						// we continue but put the result in the resourcelist as this is the only way to convey the message
						fn.Logf("stage1: cannot set the condition objRef: %s err: %v", ref.GetRefsString(forRef), err.Error())
						r.rl.Results.ErrorE(err)
					}
					continue
				}
				if r.debug {
					fn.Logf("stage1: populate new resource: objRef: %s kc: %v\n", ref.GetRefsString(forRef, objRef), kc)
				}
				// set owner reference on the new resource
				if err := newObj.SetAnnotation(SpecializerOwner, kptfilelibv1.GetConditionType(&forRef)); err != nil {
					msg := fmt.Sprintf("stage1: cannot set new annotation objRef: %s, err: %v", ref.GetRefsString(forRef), err.Error())
					if err := r.kptfile.SetConditionRefFailed(forRef, msg); err != nil {
						// we continue but put the result in the resourcelist as this is the only way to convey the message
						fn.Logf("stage1: cannot set the condition objRef: %s, err: %v", ref.GetRefsString(forRef), err.Error())
						r.rl.Results.ErrorE(err)
					}
					continue
				}
				// add the resource to the existing list as a new resource
				if err := r.inv.set(kc, []corev1.ObjectReference{forRef, objRef}, newObj, true, false); err != nil {
					msg := fmt.Sprintf("stage1: cannot set new resource to the inventory objRef: %s, err: %v\n", ref.GetRefsString(forRef), err.Error())
					if err := r.kptfile.SetConditionRefFailed(forRef, msg); err != nil {
						// we continue but put the result in the resourcelist as this is the only way to convey the message
						fn.Logf("stage1: cannot set the condition objRef: %s, err: %v", ref.GetRefsString(forRef), err.Error())
						r.rl.Results.ErrorE(err)
					}
					continue
				}
			}
		}
	}
}
