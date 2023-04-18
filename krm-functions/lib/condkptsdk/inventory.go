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
	"sync"

	corev1 "k8s.io/api/core/v1"
)

type Inventory interface {
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
	get(k gvkKind, ref *corev1.ObjectReference) map[corev1.ObjectReference]*resourceCtx
	list() [][]sdkObjectReference
	// readiness
	isReady() bool
	getReadyMap() map[corev1.ObjectReference]*readyCtx
	// diff
	diff() (map[corev1.ObjectReference]*inventoryDiff, error)
}

func newInventory(cfg *Config) (Inventory, error) {
	r := &inventory{
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

type inventory struct {
	m      sync.RWMutex
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
)
