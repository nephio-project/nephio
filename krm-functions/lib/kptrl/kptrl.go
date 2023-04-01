/*
 Copyright 2023 Nephio.

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
	AddResult(err error, obj *fn.KubeObject)
	GetResults() fn.Results
	GetObject(obj *fn.KubeObject) *fn.KubeObject
	GetObjects() fn.KubeObjects
	SetObject(obj *fn.KubeObject)
	DeleteObject(obj *fn.KubeObject)
}

func New(rl *fn.ResourceList) ResourceList {
	return &resourceList{
		rl: rl,
	}
}

type resourceList struct {
	m  sync.RWMutex
	rl *fn.ResourceList
}

func (r *resourceList) AddResult(err error, obj *fn.KubeObject) {
	r.m.Lock()
	defer r.m.Unlock()
	r.rl.Results = append(r.rl.Results, fn.ErrorConfigObjectResult(err, obj))
}

func (r *resourceList) GetResults() fn.Results {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.rl.Results
}

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

func (r *resourceList) GetObjects() fn.KubeObjects {
	r.m.RLock()
	defer r.m.RUnlock()
	return r.rl.Items
}

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

func (r *resourceList) addObject(obj *fn.KubeObject) {
	r.rl.Items = append(r.rl.Items, obj)
}

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
