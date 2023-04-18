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

	corev1 "k8s.io/api/core/v1"
)

// initializeGVKInventory initializes the GVK with the generic GVK
// resources as specified in the SDKConfig
// used to provide faster loopup if the GVK is relevant for the fn/controller
// and to provide context if there is a match
func (r *inventory) initializeGVKInventory(cfg *Config) error {
	if err := r.addGVKObjectReference(&gvkKindCtx{gvkKind: forGVKKind}, cfg.For); err != nil {
		return err
	}
	for ref, ok := range cfg.Owns {
		if err := r.addGVKObjectReference(&gvkKindCtx{gvkKind: ownGVKKind, ownKind: ok}, ref); err != nil {
			return err
		}
	}
	for ref, cb := range cfg.Watch {
		if err := r.addGVKObjectReference(&gvkKindCtx{gvkKind: watchGVKKind, callbackFn: cb}, ref); err != nil {
			return err
		}
	}
	return nil
}

func (r *inventory) addGVKObjectReference(kc *gvkKindCtx, ref corev1.ObjectReference) error {
	r.m.Lock()
	defer r.m.Unlock()

	// validates if we GVK(s) were added to the same context
	if resCtx, ok := r.gvkResources[corev1.ObjectReference{APIVersion: ref.APIVersion, Kind: ref.Kind}]; ok {
		return fmt.Errorf("another resource with a different kind %s already exists", resCtx.gvkKind)
	}
	r.gvkResources[corev1.ObjectReference{APIVersion: ref.APIVersion, Kind: ref.Kind}] = kc
	return nil
}

func (r *inventory) isGVKMatch(ref *corev1.ObjectReference) (*gvkKindCtx, bool) {
	r.m.RLock()
	defer r.m.RUnlock()
	if ref == nil {
		return nil, false
	}
	kindCtx, ok := r.gvkResources[corev1.ObjectReference{APIVersion: ref.APIVersion, Kind: ref.Kind}]
	if !ok {
		return nil, false
	}
	return kindCtx, true
}
