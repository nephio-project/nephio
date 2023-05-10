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
	"sync"

	corev1 "k8s.io/api/core/v1"
)

type inventory interface {
	// initializeGVKInventory initializes the GVK with the generic GVK
	// resources as specified in the SDKConfig
	// used to provide faster loopup if the GVK is relevant for the fn/controller
	// and to provide context if there is a match
	initializeGVKInventory(cfg *Config) error
	addGVKObjectReference(kc *gvkKindCtx, ref corev1.ObjectReference) error
	isGVKMatch(ref *corev1.ObjectReference) (*gvkKindCtx, bool)
	// runtime crud operations on the inventory
	set(kc *gvkKindCtx, refs []corev1.ObjectReference, x any, new newResource) error
	delete(kc *gvkKindCtx, refs []corev1.ObjectReference) error
	get(k gvkKind, refs []corev1.ObjectReference) map[corev1.ObjectReference]*resourceCtx
	list() [][]sdkObjectReference
	// readiness
	isReady() bool
	getReadyMap() map[corev1.ObjectReference]*readyCtx
	// diff
	diff() (map[corev1.ObjectReference]*inventoryDiff, error)
}

func newInventory(cfg *Config) (inventory, error) {
	r := &inv{
		gvkResources: map[corev1.ObjectReference]*gvkKindCtx{},
		resources: &resources{
			resources: map[sdkObjectReference]*resources{},
		},
	}
	if err := r.initializeGVKInventory(cfg); err != nil {
		return nil, err
	}
	return r, nil
}

type inv struct {
	m sync.RWMutex
	//hasOwn bool
	// gvkResource contain the gvk based resource from config
	// they dont contain the names but allow for faster lookups
	// when walking the resource list or condition list
	gvkResources map[corev1.ObjectReference]*gvkKindCtx
	// resources contain the runtime resources collected and updated
	// during the execution
	resources *resources
}

type action string

const (
	actionCreate action = "create"
	actionDelete action = "delete"
	actionUpdate action = "update"
	actionGet    action = "get"
)

// initializeGVKInventory initializes the GVK with the generic GVK
// resources as specified in the SDKConfig
// used to provide faster lookup if the GVK is relevant for the fn/controller
// and to provide context if there is a match
func (r *inv) initializeGVKInventory(cfg *Config) error {
	if err := validateGVKRef(cfg.For); err != nil {
		return err
	}
	if err := r.addGVKObjectReference(&gvkKindCtx{gvkKind: forGVKKind}, cfg.For); err != nil {
		return err
	}
	for ref, ok := range cfg.Owns {
		if err := validateGVKRef(ref); err != nil {
			return err
		}
		if err := r.addGVKObjectReference(&gvkKindCtx{gvkKind: ownGVKKind, ownKind: ok}, ref); err != nil {
			return err
		}
	}
	for ref, cb := range cfg.Watch {
		if err := validateGVKRef(ref); err != nil {
			return err
		}
		if err := r.addGVKObjectReference(&gvkKindCtx{gvkKind: watchGVKKind, callbackFn: cb}, ref); err != nil {
			return err
		}
	}
	if cfg.GenerateResourceFn == nil {
		return fmt.Errorf("a function always needs a GenerateResource function")
	}
	return nil
}

func (r *inv) addGVKObjectReference(kc *gvkKindCtx, ref corev1.ObjectReference) error {
	r.m.Lock()
	defer r.m.Unlock()

	// validates if we GVK(s) were added to the same context
	if resCtx, ok := r.gvkResources[corev1.ObjectReference{APIVersion: ref.APIVersion, Kind: ref.Kind}]; ok {
		return fmt.Errorf("another resource with a different kind %s already exists", resCtx.gvkKind)
	}
	r.gvkResources[corev1.ObjectReference{APIVersion: ref.APIVersion, Kind: ref.Kind}] = kc
	return nil
}

func (r *inv) isGVKMatch(ref *corev1.ObjectReference) (*gvkKindCtx, bool) {
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
