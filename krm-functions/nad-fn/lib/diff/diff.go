/*
Copyright 2023 Samsung.

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

package diff

import (
	"fmt"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	kptv1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
)

// Inventory interface will refer to all go-functions this library supports
type Inventory interface {
	AddExistingCondition(*corev1.ObjectReference, *kptv1.Condition)
	AddExistingResource(*corev1.ObjectReference, *fn.KubeObject)
	AddNewResource(*corev1.ObjectReference, *fn.KubeObject)
	Diff() (InventoryDiff, error)
}

// InventoryDiff struct is the output of Diff go-function implemented in this library
type InventoryDiff struct {
	DeleteObjs       []*Object
	UpdateObjs       []*Object
	CreateObjs       []*Object
	DeleteConditions []*Object
	CreateConditions []*Object
	UpdateConditions []*Object
}

// Object is used to put the GVKN in context to its respective KubeObject
type Object struct {
	Ref corev1.ObjectReference
	Obj fn.KubeObject
}

// New creates an Inventory interface to be used by kpt-functions
func New() Inventory {
	return &inventory{
		resources: map[corev1.ObjectReference]*inventoryCtx{},
	}
}

type inventory struct {
	resources map[corev1.ObjectReference]*inventoryCtx
}

type inventoryCtx struct {
	existingCondition *kptv1.Condition
	existingResource  *fn.KubeObject
	newResource       *fn.KubeObject
}

// AddExistingCondition function is to be called for storing all existing conditions as part of KPTFile
// During the DIFF these conditions are referred to make appropriate decisions
func (r *inventory) AddExistingCondition(ref *corev1.ObjectReference, c *kptv1.Condition) {
	if _, ok := r.resources[*ref]; !ok {
		r.resources[*ref] = &inventoryCtx{}
	}
	r.resources[*ref].existingCondition = c

}

// AddExistingResource function is to be called for storing all existing KubeObject resources which are inputed into this function
// The Key used here would need to have the GVKN
// During the DIFF these resources are referred to make appropriate decisions
func (r *inventory) AddExistingResource(ref *corev1.ObjectReference, o *fn.KubeObject) {
	if _, ok := r.resources[*ref]; !ok {
		r.resources[*ref] = &inventoryCtx{}
	}
	r.resources[*ref].existingResource = o
}

// AddNewResource function is to be called for storing all new KubeObject resources which are created via your function logic
// The Key used here would need to have the GVKN
// During the DIFF these resources are referred to make appropriate decisions
func (r *inventory) AddNewResource(ref *corev1.ObjectReference, o *fn.KubeObject) {
	if _, ok := r.resources[*ref]; !ok {
		r.resources[*ref] = &inventoryCtx{}
	}
	r.resources[*ref].newResource = o
}

// Diff function is to be called Only after calling-out appropriate AddExistingCondition, AddExistingResource, AddNewResource fn
// Diff is done on all existing condition, resources and new-resources
// Diff will result in creating InventoryDiff struct that will tell the kpt-fn various:
// resources to Delete, Update and Create
// conditions to Delete, Update and Create
func (r *inventory) Diff() (InventoryDiff, error) {
	diff := InventoryDiff{
		DeleteObjs:       []*Object{},
		UpdateObjs:       []*Object{},
		CreateObjs:       []*Object{},
		DeleteConditions: []*Object{},
		CreateConditions: []*Object{},
		UpdateConditions: []*Object{},
	}

	for ref, invCtx := range r.resources {
		if invCtx.newResource == nil && invCtx.existingCondition != nil {
			diff.DeleteConditions = append(diff.DeleteConditions, &Object{Ref: ref})
		}
		if invCtx.newResource != nil && invCtx.existingCondition == nil {
			diff.CreateConditions = append(diff.CreateConditions, &Object{Ref: ref})
		}
		if invCtx.existingResource == nil && invCtx.newResource != nil {
			// create resource
			diff.CreateObjs = append(diff.CreateObjs, &Object{Ref: ref, Obj: *invCtx.newResource})
		}
		if invCtx.existingResource != nil && invCtx.newResource == nil {
			// delete resource
			diff.DeleteObjs = append(diff.DeleteObjs, &Object{Ref: ref, Obj: *invCtx.existingResource})
		}
		if invCtx.existingResource != nil && invCtx.newResource != nil {
			// check diff
			existingSpec, ok, err := invCtx.existingResource.NestedStringMap("spec")
			if err != nil {
				return InventoryDiff{}, err
			}
			if !ok {
				return InventoryDiff{}, fmt.Errorf("cannot get spec of exisitng object: %v", ref)
			}
			newSpec, ok, err := invCtx.newResource.NestedStringMap("spec")
			if err != nil {
				return InventoryDiff{}, err
			}
			if !ok {
				return InventoryDiff{}, fmt.Errorf("cannot get spec of new object: %v", ref)
			}
			if d := cmp.Diff(existingSpec, newSpec); d != "" {
				diff.UpdateObjs = append(diff.UpdateObjs, &Object{Ref: ref, Obj: *invCtx.newResource})
				diff.UpdateConditions = append(diff.UpdateConditions, &Object{Ref: ref})
			}
		}
	}
	return diff, nil
}
