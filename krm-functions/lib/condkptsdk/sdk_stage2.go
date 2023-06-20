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
	"github.com/nephio-project/nephio/krm-functions/lib/ref"
	corev1 "k8s.io/api/core/v1"
)

// updateResource updates or generates the resource when the status is declared ready
// First readiness is validated in 2 steps:
// - global readiness: when key resources are missing
// - per instance readiness: when certain parts of an instance readiness is missing
func (r *sdk) updateResource() {
	if r.debug {
		fn.Logf("updateResource isReady: %t\n", r.inv.isReady())
	}
	if !r.inv.isReady() {
		// when the overall status is not ready delete all resources
		// TODO if we need to check the delete annotation
		// TODO check if the owned resources were dynamic or static
		readyMap := r.inv.getReadyMap()
		for _, readyCtx := range readyMap {
			if readyCtx.forObj != nil {
				if len(r.cfg.Owns) == 0 {
					r.deleteObjFromResourceList(readyCtx.forObj)
				}
			}
		}
		return
	}
	// the overall status is ready, so lets check the readiness map
	readyMap := r.inv.getReadyMap()
	for forRef, readyCtx := range readyMap {
		if r.debug {
			fn.Logf("updateResource readyMap: objRef %s, readyCtx: %v\n", ref.GetRefsString(forRef), readyCtx)
		}
		// if the for is not ready delete the object
		if !readyCtx.ready || readyCtx.failed {
			/*
				TODO defines what to do here
			*/
			continue
		}
		if r.cfg.UpdateResourceFn != nil {
			objs := fn.KubeObjects{}
			for _, o := range readyCtx.owns {
				x := o
				objs = append(objs, &x)
			}
			for _, o := range readyCtx.watches {
				x := o
				objs = append(objs, &x)
			}
			forObj, err := r.handleUpdateResource(forRef, readyCtx.forObj, readyCtx.forCondition, objs)
			if err != nil {
				fn.Logf("cannot handleUpdateResource objRef %s, err: %v\n", ref.GetRefsString(forRef), err.Error())
				if err := r.kptfile.SetConditionRefFailed(forRef, err.Error()); err != nil {
					fn.Logf("set condition failed error, err: %s\n", err.Error())
					r.rl.Results.ErrorE(err)
				}
				continue
			}
			if forObj != nil {
				if err := r.upsertChildObject(forGVKKind, []corev1.ObjectReference{forRef}, object{obj: *forObj}, readyCtx.forCondition, "update done", kptv1.ConditionTrue, true); err != nil {
					fn.Logf("cannot update resourcelist and inventory after handleUpdateResource: objRef %s, err: %v\n", ref.GetRefsString(forRef), err.Error())
				}
			}
		}
	}
}

// updateResource performs the fn/controller callback and handles the response
// by updating the condition and resource in kptfile/resourcelist
func (r *sdk) handleUpdateResource(forRef corev1.ObjectReference, forObj *fn.KubeObject, forCondition *kptv1.Condition, objs fn.KubeObjects) (*fn.KubeObject, error) {
	newObj, err := r.cfg.UpdateResourceFn(forObj, objs)
	if err != nil {
		return newObj, err
	}
	if newObj == nil {
		// this happens right now because the NAD gets an interface and the interface can have a default pod network
		// for which no nad is to be created. hence the nil object
		// once we do the intelligent diff this can be changed back to an error, since NAD does not have to watch the interface
		if r.debug {
			fn.Logf("cannot generate resource GenerateResourceFn returned nil, objRef: %s\n", ref.GetRefsString(forRef))
		}
		return nil, nil
		/*
			fn.Logf("cannot generate resource GenerateResourceFn returned nil, for: %v\n", forRef)
			r.rl.Results = append(r.rl.Results, fn.ErrorResult(fmt.Errorf("cannot generate resource GenerateResourceFn returned nil, for: %v", forRef)))
			return fmt.Errorf("cannot generate resource GenerateResourceFn returned nil, for: %v", forRef)
		*/
	}
	// if forCondition reason was set, set the annotation back with the owner
	if forCondition != nil && forCondition.Reason != "" {
		if err := newObj.SetAnnotation(SpecializerOwner, forCondition.Reason); err != nil {
			fn.Logf("error setting new annotation: %v\n", err.Error())
			r.rl.Results = append(r.rl.Results, fn.ErrorConfigObjectResult(err, newObj))
			return newObj, err
		}

	}
	return newObj, nil
	//return r.handleUpdate(actionUpdate, forGVKKind, []corev1.ObjectReference{forRef}, object{obj: *newObj}, forCondition, kptv1.ConditionTrue, "done", true)
}
