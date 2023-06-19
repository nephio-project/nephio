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

type readyCtx struct {
	ready        bool
	failed       bool
	forObj       *fn.KubeObject
	forCondition *kptv1.Condition
	owns         map[corev1.ObjectReference]fn.KubeObject
	watches      map[corev1.ObjectReference]fn.KubeObject
}

func (r *inv) setReady(b bool) {
	r.ready = b
}

func (r *inv) isReady() bool {
	return r.ready
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
	for forRef, forResCtx := range r.get(forGVKKind, []corev1.ObjectReference{{}}) {
		readyMap[forRef] = &readyCtx{
			ready:        true,
			failed:       forResCtx.failed,
			owns:         map[corev1.ObjectReference]fn.KubeObject{},
			watches:      map[corev1.ObjectReference]fn.KubeObject{},
			forObj:       forResCtx.existingResource,
			forCondition: forResCtx.existingCondition,
		}
		for ref, resCtx := range r.get(ownGVKKind, []corev1.ObjectReference{forRef, {}}) {
			if r.debug {
				fn.Logf("getReadyMap: own ref: %v, resCtx condition %v\n", ref, resCtx.existingCondition)
			}
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
			if r.debug {
				fn.Logf("getReadyMap: watch ref: %v, resCtx condition %v\n", ref, resCtx.existingCondition)
			}
			if resCtx.existingCondition == nil || resCtx.existingCondition.Status == kptv1.ConditionFalse {
				// ignore validating condition if the the owner reference is equal to the watch resource
				// e.g. interface watch in case of nad forFilter
				if forResCtx != nil && forResCtx.existingCondition != nil && kptfilelibv1.GetConditionType(&ref) != forResCtx.existingCondition.Reason {
					if r.debug {
						fn.Logf("getReadyMap: watch ref: %v not ready, forOwnreason: %s, resType %s", ref, forResCtx.existingCondition.Reason, kptfilelibv1.GetConditionType(&ref))
					}
					readyMap[forRef].ready = false
				}
			}
			if resCtx.existingResource != nil {
				readyMap[forRef].watches[ref] = *resCtx.existingResource
			}
		}
	}
	return readyMap
}
