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
	corev1 "k8s.io/api/core/v1"
)

// call the global watch callbacks to provide info to the fns in a generic way
// so they dont have to parse the complete resourcelist
// Also it provide readiness feedback when an error is returned
func (r *sdk) callGlobalWatches() {
	for _, resCtx := range r.inv.get(watchGVKKind, []corev1.ObjectReference{{}}) {
		fn.Logf("run watch: %v\n", resCtx.existingResource)
		if resCtx.gvkKindCtx.callbackFn != nil {
			if err := resCtx.gvkKindCtx.callbackFn(resCtx.existingResource); err != nil {
				fn.Logf("populatechildren not ready: watch callback failed: %v\n", err.Error())
				r.rl.Results = append(r.rl.Results, fn.ErrorConfigObjectResult(err, resCtx.existingResource))
				r.ready = false
			}
		}
	}
}
