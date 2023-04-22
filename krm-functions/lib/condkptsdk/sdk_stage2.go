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

// generateResource updates or generates the resource when the status is declared ready
// First readiness is validated in 2 steps:
// - global readiness: when key resources are missing
// - per instance readiness: when certain parts of an instance readiness is missing
func (r *sdk) generateResource() error {
	fn.Logf("generateResource isReady: %t\n", r.inv.isReady())
	if !r.ready || !r.inv.isReady() {
		// when the overal status is not ready delete all resources
		// TBD if we need to check the delete annotation
		readyMap := r.inv.getReadyMap()
		for _, readyCtx := range readyMap {
			if readyCtx.forObj != nil {
				if len(r.cfg.Owns) == 0 {
					r.deleteObjFromResourceList(readyCtx.forObj)
				}
			}
		}
		return nil
	}
	// the overall status is ready, so lets check the readiness map
	readyMap := r.inv.getReadyMap()
	if len(readyMap) == 0 {
		// this is when the global resource is not found
		if err := r.handleGenerateUpdate(corev1.ObjectReference{APIVersion: r.cfg.For.APIVersion, Kind: r.cfg.For.Kind, Name: r.kptf.GetKptFile().Name}, nil, fn.KubeObjects{}); err != nil {
			return err
		}
	}
	for forRef, readyCtx := range readyMap {
		fn.Logf("generateResource readyMap: forRef %v, readyCtx: %v\n", forRef, readyCtx)
		// if the for is not ready delete the object
		if !readyCtx.ready {
			if readyCtx.forObj != nil {
				// TBD if this is the right approach -> avoids deleting interface
				if len(r.cfg.Owns) == 0 {
					r.deleteObjFromResourceList(readyCtx.forObj)

				}
			}
			continue
		}
		if r.cfg.GenerateResourceFn != nil {
			objs := fn.KubeObjects{}
			for _, o := range readyCtx.owns {
				x := o
				objs = append(objs, &x)
			}
			for _, o := range readyCtx.watches {
				x := o
				objs = append(objs, &x)
			}
			if err := r.handleGenerateUpdate(forRef, readyCtx.forObj, objs); err != nil {
				return err
			}
		}
	}
	// update the kptfile with the latest conditions
	return r.updateKptFile()
}

// handleGenerateUpdate performs the fn/controller callback and handles the response
// by updating the condition and resource in kptfile/resourcelist
func (r *sdk) handleGenerateUpdate(forRef corev1.ObjectReference, forObj *fn.KubeObject, objs fn.KubeObjects) error {
	newObj, err := r.cfg.GenerateResourceFn(forObj, objs)
	if err != nil {
		fn.Logf("error generating new resource: %v\n", err.Error())
		r.rl.Results = append(r.rl.Results, fn.ErrorResult(fmt.Errorf("cannot generate resource GenerateResourceFn returned nil, for: %v", forRef)))
		return err
	}
	if newObj == nil {
		fn.Logf("cannot generate resource GenerateResourceFn returned nil, for: %v\n", forRef)
		r.rl.Results = append(r.rl.Results, fn.ErrorResult(fmt.Errorf("cannot generate resource GenerateResourceFn returned nil, for: %v", forRef)))
		return fmt.Errorf("cannot generate resource GenerateResourceFn returned nil, for: %v", forRef)
	}
	// set owner reference on the new resource if not having owns
	// as you ste it to yourself
	if len(r.cfg.Owns) == 0 {
		if err := newObj.SetAnnotation(FnRuntimeOwner, kptfilelibv1.GetConditionType(&forRef)); err != nil {
			fn.Logf("error setting new annotation: %v\n", err.Error())
			r.rl.Results = append(r.rl.Results, fn.ErrorConfigObjectResult(err, newObj))
			return err
		}
	}
	// add the resource to the kptfile and updates the resource in the resourcelist
	return r.handleUpdate(actionUpdate, forGVKKind, []corev1.ObjectReference{forRef}, &object{obj: *newObj}, kptv1.ConditionTrue, "done", true)
}
