package fnruntime

import (
	"fmt"
	"sync"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	kptv1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
)

type UpstreamInventory interface {
	AddExistingCondition(*corev1.ObjectReference, *kptv1.Condition)
	AddExistingResource(*corev1.ObjectReference, *fn.KubeObject)
	AddNewResource(*corev1.ObjectReference, *fn.KubeObject)
	Diff() (UpstreamInventoryDiff, error)
}

type UpstreamInventoryDiff struct {
	DeleteObjs       []*Object
	UpdateObjs       []*Object
	CreateObjs       []*Object
	DeleteConditions []*Object
	CreateConditions []*Object
	UpdateConditions []*Object
}

type Object struct {
	Ref corev1.ObjectReference
	Obj fn.KubeObject
}

func NewUpstreamInventory() UpstreamInventory {
	return &upstreamInventory{
		resources: map[corev1.ObjectReference]*upstreamInventoryCtx{},
	}
}

type upstreamInventory struct {
	m         sync.RWMutex
	resources map[corev1.ObjectReference]*upstreamInventoryCtx
}

type upstreamInventoryCtx struct {
	existingCondition *kptv1.Condition
	existingResource  *fn.KubeObject
	newResource       *fn.KubeObject
}

func (r *upstreamInventory) AddExistingCondition(ref *corev1.ObjectReference, c *kptv1.Condition) {
	r.m.Lock()
	defer r.m.Unlock()
	if _, ok := r.resources[*ref]; !ok {
		r.resources[*ref] = &upstreamInventoryCtx{}
	}
	r.resources[*ref].existingCondition = c

}

func (r *upstreamInventory) AddExistingResource(ref *corev1.ObjectReference, o *fn.KubeObject) {
	r.m.Lock()
	defer r.m.Unlock()
	if _, ok := r.resources[*ref]; !ok {
		r.resources[*ref] = &upstreamInventoryCtx{}
	}
	r.resources[*ref].existingResource = o
}

func (r *upstreamInventory) AddNewResource(ref *corev1.ObjectReference, o *fn.KubeObject) {
	r.m.Lock()
	defer r.m.Unlock()
	if _, ok := r.resources[*ref]; !ok {
		r.resources[*ref] = &upstreamInventoryCtx{}
	}
	r.resources[*ref].newResource = o
}

// Diff is based on the following principle: we have an inventory
// populated with the existing resource/condition info and we also
// have information on new resource/condition that would be created
// if nothinf existed.
// the diff compares these the eixisiting resource/condition inventory
// agsinst the new resource/condition inventory and provide CRUD operation
// based on the comparisons.
func (r *upstreamInventory) Diff() (UpstreamInventoryDiff, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	diff := UpstreamInventoryDiff{
		DeleteObjs:       []*Object{},
		UpdateObjs:       []*Object{},
		CreateObjs:       []*Object{},
		DeleteConditions: []*Object{},
		CreateConditions: []*Object{},
	}

	for ref, invCtx := range r.resources {
		// condition CRUD handling
		switch {
		// if there is no new resource, but we have a condition for that resource we should delete the condition
		case invCtx.newResource == nil && invCtx.existingCondition != nil:
			diff.DeleteConditions = append(diff.DeleteConditions, &Object{Ref: ref})
		// if there is a new resource, but we have no condition for that resource someone deleted it
		// and we have to recreate that condition
		case invCtx.newResource != nil && invCtx.existingCondition == nil:
			diff.CreateConditions = append(diff.CreateConditions, &Object{Ref: ref, Obj: *invCtx.newResource})
		}
		// resource handling
		switch {
		// if the existing resource does not exist but the new resource exist we have to create the new resource
		case invCtx.existingResource == nil && invCtx.newResource != nil:
			// create resource
			diff.CreateObjs = append(diff.CreateObjs, &Object{Ref: ref, Obj: *invCtx.newResource})
		// if the new resource does not exist and but the resource exist we have to delete the exisiting resource
		case invCtx.existingResource != nil && invCtx.newResource == nil:
			// delete resource
			diff.DeleteObjs = append(diff.DeleteObjs, &Object{Ref: ref, Obj: *invCtx.existingResource})
		// if both exisiting/new resource exists check the differences of the spec
		// dependening on the outcome update the resource with the new information
		case invCtx.existingResource != nil && invCtx.newResource != nil:
			// check diff
			existingSpec, ok, err := invCtx.existingResource.NestedStringMap("spec")
			if err != nil {
				return UpstreamInventoryDiff{}, err
			}
			if !ok {
				return UpstreamInventoryDiff{}, fmt.Errorf("cannot get spec of exisitng object: %v", ref)
			}
			newSpec, ok, err := invCtx.newResource.NestedStringMap("spec")
			if err != nil {
				return UpstreamInventoryDiff{}, err
			}
			if !ok {
				return UpstreamInventoryDiff{}, fmt.Errorf("cannot get spec of new object: %v", ref)
			}
			if d := cmp.Diff(existingSpec, newSpec); d != "" {
				diff.UpdateObjs = append(diff.UpdateObjs, &Object{Ref: ref, Obj: *invCtx.newResource})
			}
		}
	}
	return diff, nil
}
