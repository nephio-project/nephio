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
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
)

type ResourceList struct {
	fn.ResourceList
}

// AddResult adds a result with error and corresponding KubeObject by
// appending a result to the result slice in the resourceList
func (r *ResourceList) AddResult(err error, obj *fn.KubeObject) {
	r.Results = append(r.Results, fn.ErrorConfigObjectResult(err, obj))
}

// GetResults gets the results slice from the resourceList
func (r *ResourceList) GetResults() fn.Results {
	return r.Results
}

// GetObjects return an fn sdk KubeObject by comparing the APIVersion, Kind, Name and Namespace
// if the object is found the corresponding obj is returned, if not nil is returned
func (r *ResourceList) GetObjects(obj *fn.KubeObject) fn.KubeObjects {
	return r.Items.Where(func(ko *fn.KubeObject) bool { return isGVKNNEqual(ko, obj) })
}

// GetObjects returns all items from the resourceList
func (r *ResourceList) GetAllObjects() fn.KubeObjects {
	return r.Items
}

// SetObject sets the object in the resourceList items. It either updates/overrides
// the entry if it exists or appends the entry if it does not exist in the resourceList
// It uses APIVersion, Kind, Name and Namespace to check the object uniqueness
func (r *ResourceList) SetObject(obj *fn.KubeObject) error {
	if err := r.UpsertObjectToItems(obj, nil, true); err != nil {
		return err
	}
	return nil
}

// DeleteObject deletes the object from the resourceList if it exists.
func (r *ResourceList) DeleteObject(obj *fn.KubeObject) {
	for idx, o := range r.Items {
		if isGVKNNEqual(o, obj) {
			r.Items = append(r.Items[:idx], r.Items[idx+1:]...)
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
