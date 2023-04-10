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
	// AddResult add a result with error and corresponding kubeObject by
	// appending a result to the result slice in the resourcelist
	AddResult(err error, obj *fn.KubeObject)
	// GetResults gets the results slice from the resourcelist
	GetResults() fn.Results
	// GetObject return an fn sdk KubeObject by comparing the APIVersion, Kind and Name
	// if the object is found the corresponding obj is returned, if not nil is returned
	GetObject(obj *fn.KubeObject) *fn.KubeObject
	// GetObjects returns all items from the resourcelist
	GetObjects() fn.KubeObjects
	// SetObject sets the object in the resourcelist items. It either updates/overrides
	// the entry if it exists or appends the entry if it does not exist in the resourcelist
	// It uses APIVersion, Kind and Name to check the object uniqueness
	SetObject(obj *fn.KubeObject)
	// DeleteObject deletes the object from the resourcelist if it exists.
	DeleteObject(obj *fn.KubeObject)
}

// New creates a new ResourceList interface
// concurrency is handles in the methods
func New(rl *fn.ResourceList) ResourceList {
	return &resourceList{
		rl: rl,
	}
}

type resourceList struct {
	m  sync.RWMutex
	rl *fn.ResourceList
}

// AddResult add a result with error and corresponding kubeObject by
// appending a result to the result slice in the resourcelist
func (r *resourceList) AddResult(err error, obj *fn.KubeObject) {
	r.m.Lock()
	defer r.m.Unlock()
	r.rl.Results = append(r.rl.Results, fn.ErrorConfigObjectResult(err, obj))
}

// GetResults gets the results slice from the resourcelist
func (r *resourceList) GetResults() fn.Results {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.rl.Results
}

// GetObject return an fn sdk KubeObject by comparing the APIVersion, Kind and Name
// if the object is found the corresponding obj is returned, if not nil is returned
func (r *resourceList) GetObject(obj *fn.KubeObject) *fn.KubeObject {
	r.m.RLock()
	defer r.m.RUnlock()
	for _, o := range r.rl.Items {
		if isGVKNEqual(o, obj) {
			return o
		}
	}
	return nil
}

// GetObjects returns all items from the resourcelist
func (r *resourceList) GetObjects() fn.KubeObjects {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.rl.Items
}

// SetObject sets the object in the resourcelist items. It either updates/overrides
// the entry if it exists or appends the entry if it does not exist in the resourcelist
// It uses APIVersion, Kind and Name to check the object uniqueness
func (r *resourceList) SetObject(obj *fn.KubeObject) {
	r.m.Lock()
	defer r.m.Unlock()
	exists := false
	for idx, o := range r.rl.Items {
		if isGVKNEqual(o, obj) {
			r.rl.Items[idx] = obj
			exists = true
			break
		}
	}
	if !exists {
		r.addObject(obj)
	}
}

// addObject is a helper function to append an object to the resourcelist
func (r *resourceList) addObject(obj *fn.KubeObject) {
	r.rl.Items = append(r.rl.Items, obj)
}

// DeleteObject deletes the object from the resourcelist if it exists.
func (r *resourceList) DeleteObject(obj *fn.KubeObject) {
	r.m.Lock()
	defer r.m.Unlock()
	for idx, o := range r.rl.Items {
		if isGVKNEqual(o, obj) {
			r.rl.Items = append(r.rl.Items[:idx], r.rl.Items[idx+1:]...)
		}
	}
}

// isGVKNEqual validates if the APIVersion, Kind and Name of both fn.KubeObject are equal
func isGVKNEqual(curobj, newobj *fn.KubeObject) bool {
	if curobj.GetAPIVersion() == newobj.GetAPIVersion() && curobj.GetKind() == newobj.GetKind() && curobj.GetName() == newobj.GetName() {
		return true
	}
	return false
}
