package inventory

import (
	"fmt"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	kptv1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
)

type Inventory interface {
	AddExistingCondition(*corev1.ObjectReference, *kptv1.Condition)
	AddExistingResource(*corev1.ObjectReference, *fn.KubeObject)
	AddNewResource(*corev1.ObjectReference, *fn.KubeObject)
	Diff() (InventoryDiff, error)
}

type InventoryDiff struct {
	DeleteObjs       []*Object
	UpdateObjs       []*Object
	CreateObjs       []*Object
	DeleteConditions []*Object
	CreateConditions []*Object
}

type Object struct {
	Ref corev1.ObjectReference
	Obj fn.KubeObject
}

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

func (r *inventory) AddExistingCondition(ref *corev1.ObjectReference, c *kptv1.Condition) {
	if _, ok := r.resources[*ref]; !ok {
		r.resources[*ref] = &inventoryCtx{}
	}
	r.resources[*ref].existingCondition = c

}

func (r *inventory) AddExistingResource(ref *corev1.ObjectReference, o *fn.KubeObject) {
	if _, ok := r.resources[*ref]; !ok {
		r.resources[*ref] = &inventoryCtx{}
	}
	r.resources[*ref].existingResource = o
}

func (r *inventory) AddNewResource(ref *corev1.ObjectReference, o *fn.KubeObject) {
	if _, ok := r.resources[*ref]; !ok {
		r.resources[*ref] = &inventoryCtx{}
	}
	r.resources[*ref].newResource = o
}

func (r *inventory) Diff() (InventoryDiff, error) {
	diff := InventoryDiff{
		DeleteObjs:       []*Object{},
		UpdateObjs:       []*Object{},
		CreateObjs:       []*Object{},
		DeleteConditions: []*Object{},
		CreateConditions: []*Object{},
	}

	for ref, invCtx := range r.resources {
		switch {
		case invCtx.newResource == nil && invCtx.existingCondition != nil:
			diff.DeleteConditions = append(diff.DeleteConditions, &Object{Ref: ref})
		case invCtx.newResource != nil && invCtx.existingCondition == nil:
			diff.CreateConditions = append(diff.CreateConditions, &Object{Ref: ref, Obj: *invCtx.newResource})
		}
		switch {
		case invCtx.existingResource == nil && invCtx.newResource != nil:
			// create resource
			diff.CreateObjs = append(diff.CreateObjs, &Object{Ref: ref, Obj: *invCtx.newResource})
		case invCtx.existingResource != nil && invCtx.newResource == nil:
			// delete resource
			diff.DeleteObjs = append(diff.DeleteObjs, &Object{Ref: ref, Obj: *invCtx.existingResource})
		case invCtx.existingResource != nil && invCtx.newResource != nil:
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
			}
		}
	}
	return diff, nil
}
