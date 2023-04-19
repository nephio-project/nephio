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

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	kptv1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	corev1 "k8s.io/api/core/v1"
)

// defines the kind of gvks supported by the inventory
type gvkKind string

const (
	forGVKKind   gvkKind = "for"
	ownGVKKind   gvkKind = "own"
	watchGVKKind gvkKind = "watch"
)

// to make the resource list of the inventory geenric we add the gvkKind on top of the objectReference
type sdkObjectReference struct {
	gvkKind gvkKind
	ref     corev1.ObjectReference
}

type gvkKindCtx struct {
	gvkKind    gvkKind
	ownKind    ResourceKind    // only used for kind == own
	callbackFn WatchCallbackFn // only used for global watches
}

type resourceCtx struct {
	gvkKindCtx
	existingCondition *kptv1.Condition // contains owner in the condition reason
	existingResource  *fn.KubeObject   // contains the owner in the owner annotation
	newResource       *fn.KubeObject
}

type newResource bool

type resources struct {
	resourceCtx
	resources map[sdkObjectReference]*resources
}

func (r *inventory) set(kc *gvkKindCtx, refs []corev1.ObjectReference, x any, new newResource) error {
	r.m.Lock()
	defer r.m.Unlock()

	fn.Logf("set: kc: %v, refs: %v, resource: %v, new: %t\n", kc, refs, x, new)
	sdkRefs, err := getSdkRefs(kc.gvkKind, refs)
	if err != nil {
		return err
	}
	return r.resources.set2(sdkRefs, kc, x, new)
	//return r.resources.set(kc, refs, x, new)
}

func (r *inventory) delete(kc *gvkKindCtx, refs []corev1.ObjectReference) error {
	r.m.Lock()
	defer r.m.Unlock()

	fn.Logf("delete: kc: %v, refs: %v\n", kc, refs)

	sdkRefs, err := getSdkRefs(kc.gvkKind, refs)
	if err != nil {
		return err
	}

	return r.resources.delete2(sdkRefs)
	//return r.resources.delete(kc, refs)
}

func (r *inventory) get(k gvkKind, refs []corev1.ObjectReference) map[corev1.ObjectReference]*resourceCtx {
	r.m.RLock()
	defer r.m.RUnlock()

	fn.Logf("get: kind: %v, refs: %v\n", k, refs)

	sdkRefs, err := getSdkRefs(k, refs)
	if err != nil {
		fn.Logf("cannot get sdkrefs :%v\n", err)
		return map[corev1.ObjectReference]*resourceCtx{}
	}

	return r.resources.get2(sdkRefs)

	//return r.resources.get(k, refs, map[corev1.ObjectReference]*resourceCtx{})
}

func (r *inventory) list() [][]sdkObjectReference {
	r.m.RLock()
	defer r.m.RUnlock()

	return r.resources.list()
}

func (r *resources) list() [][]sdkObjectReference {
	entries := [][]sdkObjectReference{}
	for parentSdkRef, res := range r.resources {
		entries = append(entries, []sdkObjectReference{parentSdkRef})
		for sdkRef := range res.resources {
			entries = append(entries, []sdkObjectReference{parentSdkRef, sdkRef})
		}
	}
	return entries
}

/*
func (r *resources) get(k gvkKind, ref *corev1.ObjectReference, resCtxs map[corev1.ObjectReference]*resourceCtx) map[corev1.ObjectReference]*resourceCtx {
	if ref != nil {
		// when ref is not nil we need to do another lookup in the forResourceMap
		// since for has the children
		sdkRef := sdkObjectReference{gvkKind: forGVKKind, ref: *ref}
		res, ok := r.resources[sdkRef]
		fn.Logf("get resource with ref: %v, kind: %s, resources: %v\n", sdkRef, k, res.resources)
		if !ok {
			return resCtxs
		}
		return res.get(k, nil, resCtxs)
	}
	fn.Log("get resources", r.resources)
	for sdkref, res := range r.resources {
		fn.Log("get sdkref", sdkref)
		if sdkref.gvkKind == k {
			var ec *kptv1.Condition
			var eo, no *fn.KubeObject
			if res.existingCondition != nil {
				x := *res.existingCondition
				ec = &x
			}
			if res.existingResource != nil {
				x := *res.existingResource
				eo = &x
			}
			if res.newResource != nil {
				x := *res.newResource
				no = &x
			}

			resCtxs[sdkref.ref] = &resourceCtx{
				gvkKindCtx:        res.gvkKindCtx,
				existingCondition: ec,
				existingResource:  eo,
				newResource:       no,
			}
		}
	}
	return resCtxs
}

func (r *resources) set(kc *gvkKindCtx, refs []corev1.ObjectReference, x any, new newResource) error {
	if err := validateWalk(kc.gvkKind, refs); err != nil {
		fn.Logf("cannot set -> walk validation failed :%v\n", err)
		return err
	}
	return r.walk(actionCreate, kc, refs, x, new)
}

func (r *resources) delete(kc *gvkKindCtx, refs []corev1.ObjectReference) error {
	if err := validateWalk(kc.gvkKind, refs); err != nil {
		fn.Logf("cannot get -> walk validation failed :%v\n", err)
		return err
	}
	return r.walk(actionDelete, kc, refs, nil, false)
}

// walk implements a generic walk over the resources with action create or delete that represent set/delete
func (r *resources) walk(a action, kc *gvkKindCtx, refs []corev1.ObjectReference, x any, new newResource) error {
	//fn.Logf("entry tree action: %s, kind: kc: %v refs: %v\n", a, kc, refs)
	if len(refs) > 1 {
		// continue walk
		sdkRef := sdkObjectReference{gvkKind: forGVKKind, ref: refs[0]}
		// continue with the walk
		// check if the reference is initialized
		if !r.isInitialized(sdkRef) {
			// if the walkaction is set we need to initialize the resource tree
			if a == actionCreate {
				r.init(sdkRef)
			} else {
				// when the tree is not initialized we dont have to proceed as the
				// object does not exists
				return nil
			}
		}
		return r.resources[sdkRef].walk(a, kc, refs[1:], x, new)
	}
	// act on the reference
	var sdkRef *sdkObjectReference
	if len(refs) == 0 {
		sdkRef = &sdkObjectReference{gvkKind: kc.gvkKind, ref: refs[0]}
	}
	// perform action
	fn.Logf("walk action: %s, sdkref: %v\n", a, sdkRef)

	switch a {
	case actionCreate, actionUpdate:
		if sdkRef != nil {
			if !r.isInitialized(*sdkRef) {
				r.init(*sdkRef)
			}
			switch d := x.(type) {
			case *kptv1.Condition:
				fn.Logf("add existing condition: %v\n", sdkRef)
				x := *d
				r.resources[*sdkRef].resourceCtx.existingCondition = &x
			case *fn.KubeObject:
				r.resources[*sdkRef].gvkKindCtx = *kc
				if new {
					x := *d
					r.resources[*sdkRef].resourceCtx.newResource = &x
				} else {
					x := *d
					r.resources[*sdkRef].resourceCtx.existingResource = &x
				}
			default:
				return fmt.Errorf("cannot insert unsupported object: %v", x)
			}
		}
	case actionDelete:
		if sdkRef != nil {
			if r.isInitialized(*sdkRef) {
				// right now we only have action delete for the exisitng Condition
				r.resources[*sdkRef].resourceCtx.existingCondition = nil
			}
		}
	case actionGet:

	}
	return nil
}
*/

// isInitialized checks if the resources are initialized
func (r *resources) isInitialized(sdkRef sdkObjectReference) bool {
	if _, ok := r.resources[sdkRef]; !ok {
		return false
	}
	return true
}

// init initialize the resources
func (r *resources) init(sdkRef sdkObjectReference) {
	r.resources[sdkRef] = &resources{
		resources: map[sdkObjectReference]*resources{},
	}
}

/*
// validateWalk checks if the attributes of the walk are valid. if not an error is returned
func validateWalk(k gvkKind, refs []corev1.ObjectReference) error {
	switch len(refs) {
	case 0:
		return fmt.Errorf("cannot walk resource tree with empty ref")
	case 1:
		if k == ownGVKKind {
			return fmt.Errorf("cannot walk resource tree with depth %d other than using for or watch, got: %s", len(refs), k)
		}
		if err := validateGVKNRef(refs[0]); err != nil {
			return fmt.Errorf("cannot walk resource tree with depth %d, nil reference, got: %v", len(refs), refs)
		}
		return nil
	case 2:
		if k == forGVKKind {
			return fmt.Errorf("cannot walk resource tree with depth %d other than own or watch, got: %s", len(refs), k)
		}
		if validateGVKNRef(refs[0]) != nil && validateGVKNRef(refs[1]) != nil {
			return fmt.Errorf("cannot walk resource tree with depth %d, nil reference, got: %v", len(refs), refs)
		}
		return nil
	default:
		return fmt.Errorf("cannot walk resource tree with depth > 2, got %d", len(refs))
	}
}
*/

func getSdkRefs(k gvkKind, refs []corev1.ObjectReference) ([]sdkObjectReference, error) {
	switch len(refs) {
	case 0:
		return nil, fmt.Errorf("cannot walk resource tree with empty ref")
	case 1:
		if k != forGVKKind && k != watchGVKKind {
			return nil, fmt.Errorf("refs with len 1 only allowed for for/watch")
		}
		return []sdkObjectReference{{gvkKind: k, ref: refs[0]}}, nil
	case 2:
		if k == forGVKKind {
			return nil, fmt.Errorf("refs with len 2 only allowed for own/watch")
		}
		return []sdkObjectReference{{gvkKind: forGVKKind, ref: refs[0]}, {gvkKind: k, ref: refs[1]}}, nil
	default:
		return nil, fmt.Errorf("refs with len > 2, got %d", len(refs))
	}

}

func (r *resources) set2(refs []sdkObjectReference, kc *gvkKindCtx, x any, new newResource) error {
	if len(refs) == 0 {
		switch d := x.(type) {
		case *kptv1.Condition:
			fn.Logf("add existing condition: %v\n", x)
			x := *d
			r.resourceCtx.existingCondition = &x
			return nil
		case *fn.KubeObject:
			fn.Logf("add existing resource: %v\n", x)
			r.gvkKindCtx = *kc
			if new {
				x := *d
				r.resourceCtx.newResource = &x
			} else {
				x := *d
				r.resourceCtx.existingResource = &x
			}
			return nil
		default:
			return fmt.Errorf("unsupported object: %v", x)
		}

	}
	// check if resource exists
	if !r.isInitialized(refs[0]) {
		r.init(refs[0])
	}
	return r.resources[refs[0]].set2(refs[1:], kc, x, new)
}

func (r *resources) delete2(refs []sdkObjectReference) error {
	if len(refs) == 0 {
		r.resourceCtx.existingCondition = nil
		return nil
	}
	// check if resource exists
	if !r.isInitialized(refs[0]) {
		return fmt.Errorf("not found")
	}
	return r.resources[refs[0]].delete2(refs[1:])

}

func (r *resources) get2(refs []sdkObjectReference) map[corev1.ObjectReference]*resourceCtx {
	if len(refs) == 0 {
		resCtxs := map[corev1.ObjectReference]*resourceCtx{}
		if len(r.resources) == 0 {
			// specific get
			resCtxs[corev1.ObjectReference{}] = r.resourceCtx.Deepcopy()
		} else {
			// wildcard get
			for sdkRef, resCtx := range r.resources {
				resCtxs[sdkRef.ref] = resCtx.Deepcopy()
			}
		}
		return resCtxs
	}
	// check if resource exists
	if _, ok := r.resources[refs[0]]; !ok {
		fn.Logf("ref not found")
		return map[corev1.ObjectReference]*resourceCtx{}
	}
	return r.resources[refs[0]].get2(refs[1:])
}

func (in *resourceCtx) Deepcopy() *resourceCtx {
	if in == nil {
		return nil
	}
	out := new(resourceCtx)
	in.DeepCopyInto(out)
	return out
}

func (in *resourceCtx) DeepCopyInto(out *resourceCtx) {
	*out = *in
	if in.existingCondition != nil {
		in, out := &in.existingCondition, &out.existingCondition
		*out = new(kptv1.Condition)
		**out = **in
	}
	if in.existingResource != nil {
		in, out := &in.existingResource, &out.existingResource
		*out = new(fn.KubeObject)
		**out = **in
	}
	if in.newResource != nil {
		in, out := &in.newResource, &out.newResource
		*out = new(fn.KubeObject)
		**out = **in
	}
}
