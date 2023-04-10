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

package kptrl

import (
	"sync"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
)

type ResourceList interface {
	// AddResult adds a result with error and corresponding kubeObject by
	// appending a result to the result slice in the resourceList
	AddResult(err error, obj *fn.KubeObject)
	// GetResults gets the results slice from the resourceList
	GetResults() fn.Results
	// GetObject return an fn sdk KubeObject by comparing the APIVersion, Kind, Name and Namespace
	// if the object is found the corresponding obj is returned, if not nil is returned
	GetObject(obj *fn.KubeObject) *fn.KubeObject
	// GetObjects returns all items from the resourceList
	GetObjects() fn.KubeObjects
	// SetObject sets the object in the resourceList items. It either updates/overrides
	// the entry if it exists or appends the entry if it does not exist in the resourceList
	// It uses APIVersion, Kind, Name and Namespace to check the object uniqueness
	SetObject(obj *fn.KubeObject) error
	// DeleteObject deletes the object from the resourceList if it exists.
	DeleteObject(obj *fn.KubeObject)
}

// New creates a new ResourceList interface
// concurrency is handles in the methods
func New(rl *fn.ResourceList) ResourceList {
	return &resourceList{
		rl: rl,
	}
}

// Typically in functions there is no concurrency, but this library may be used in specializers
// running concurrently in the same binary, and may share the same resourceList".
type resourceList struct {
	m  sync.RWMutex
	rl *fn.ResourceList
}

// AddResult adds a result with error and corresponding KubeObject by
// appending a result to the result slice in the resourceList
func (r *resourceList) AddResult(err error, obj *fn.KubeObject) {
	r.m.Lock()
	defer r.m.Unlock()
	r.rl.Results = append(r.rl.Results, fn.ErrorConfigObjectResult(err, obj))
}

// GetResults gets the results slice from the resourceList
func (r *resourceList) GetResults() fn.Results {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.rl.Results
}

// GetObject return an fn sdk KubeObject by comparing the APIVersion, Kind, Name and Namespace
// if the object is found the corresponding obj is returned, if not nil is returned
func (r *resourceList) GetObject(obj *fn.KubeObject) *fn.KubeObject {
	r.m.RLock()
	defer r.m.RUnlock()
	for _, o := range r.rl.Items {
		if isGVKNNEqual(o, obj) {
			return o
		}
	}
	return nil
}

// GetObjects returns all items from the resourceList
func (r *resourceList) GetObjects() fn.KubeObjects {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.rl.Items
}

// SetObject sets the object in the resourceList items. It either updates/overrides
// the entry if it exists or appends the entry if it does not exist in the resourceList
// It uses APIVersion, Kind, Name and Namespace to check the object uniqueness
func (r *resourceList) SetObject(obj *fn.KubeObject) error {
	r.m.Lock()
	defer r.m.Unlock()

	if err := r.rl.UpsertObjectToItems(obj, nil, true); err != nil {
		return err
	}
	return nil
}

// DeleteObject deletes the object from the resourceList if it exists.
func (r *resourceList) DeleteObject(obj *fn.KubeObject) {
	r.m.Lock()
	defer r.m.Unlock()
	for idx, o := range r.rl.Items {
		if isGVKNNEqual(o, obj) {
			r.rl.Items = append(r.rl.Items[:idx], r.rl.Items[idx+1:]...)
		}
	}
}

// isGVKNEqual validates if the APIVersion, Kind, Name and Namespace of both fn.KubeObject are equal
func isGVKNNEqual(curobj, newobj *fn.KubeObject) bool {
	if curobj.GetAPIVersion() == newobj.GetAPIVersion() &&
		curobj.GetKind() == newobj.GetKind() &&
		curobj.GetName() == newobj.GetName() &&
		curobj.GetNamespace() == newobj.GetNamespace() {
		return true
	}
	return false
}
