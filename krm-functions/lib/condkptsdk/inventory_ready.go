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
	corev1 "k8s.io/api/core/v1"
)

type readyCtx struct {
	ready        bool
	forObj       *fn.KubeObject
	forCondition *kptv1.Condition
	owns         map[corev1.ObjectReference]fn.KubeObject
	watches      map[corev1.ObjectReference]fn.KubeObject
}

// isReady provide the overall ready status by validating the global
// watch resource. Used in stage1 and stage2
// if the global watched resource(s) dont exist we are not ready
// if the global watched resource(s) have a False condition status we are not ready
func (r *inv) isReady() bool {
	r.m.RLock()
	defer r.m.RUnlock()
	// check readiness, we start positive
	ready := true
	// the readiness is determined by the global watch resources
	for watchRef, resCtx := range r.get(watchGVKKind, []corev1.ObjectReference{{}}) {
		fn.Logf("isReady: watchRef: %v, resCtx: %#v\n", watchRef, *resCtx)
		if resCtx.existingCondition != nil {
			fn.Logf("isReady: watchRef: %v, condition: %#v\n", watchRef, resCtx.existingCondition)
		}
		// if global watched resource does not exist we fail readiness
		// if the condition is present and the status is False something is pending, so we
		// fail readiness
		if resCtx.existingResource == nil {
			return false
		} else {
			if resCtx.existingCondition != nil &&
				resCtx.existingCondition.Status == kptv1.ConditionFalse {
				return false
			}
		}
	}
	return ready
}

// getReadyMap provides a readyMap based on the information of the children
// of the forResource
// Both own and watches that are dependent on the forResource are validated for
// readiness
// The readyMap is used only in stage 2 of the sdk
func (r *inv) getReadyMap() map[corev1.ObjectReference]*readyCtx {
	r.m.RLock()
	defer r.m.RUnlock()

	readyMap := map[corev1.ObjectReference]*readyCtx{}
	for forRef, resCtx := range r.get(forGVKKind, []corev1.ObjectReference{{}}) {
		readyMap[forRef] = &readyCtx{
			ready:        true,
			owns:         map[corev1.ObjectReference]fn.KubeObject{},
			watches:      map[corev1.ObjectReference]fn.KubeObject{},
			forObj:       resCtx.existingResource,
			forCondition: resCtx.existingCondition,
		}
		for ref, resCtx := range r.get(ownGVKKind, []corev1.ObjectReference{forRef, {}}) {
			fn.Logf("getReadyMap: own ref: %v, resCtx condition %v\n", ref, resCtx.existingCondition)
			if resCtx.existingCondition == nil ||
				resCtx.existingCondition.Status == kptv1.ConditionFalse {
				readyMap[forRef].ready = false
			}
			if resCtx.existingResource != nil {
				readyMap[forRef].owns[ref] = *resCtx.existingResource
			}
		}
		for ref, resCtx := range r.get(watchGVKKind, []corev1.ObjectReference{forRef, {}}) {
			// TBD we need to look at some watches that we want to check the condition for and others not
			fn.Logf("getReadyMap: watch ref: %v, resCtx condition %v\n", ref, resCtx.existingCondition)
			if resCtx.existingCondition == nil || resCtx.existingCondition.Status == kptv1.ConditionFalse {
				readyMap[forRef].ready = false
			}
			if _, ok := readyMap[forRef].watches[ref]; !ok {
				readyMap[forRef].watches[ref] = *resCtx.existingResource
			}
			if resCtx.existingResource != nil {
				readyMap[forRef].watches[ref] = *resCtx.existingResource
			}
		}
	}
	return readyMap
}
