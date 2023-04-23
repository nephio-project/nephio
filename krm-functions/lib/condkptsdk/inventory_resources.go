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
	"github.com/google/go-cmp/cmp"
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

func (r *inv) set(kc *gvkKindCtx, refs []corev1.ObjectReference, x any, new newResource) error {
	r.m.Lock()
	defer r.m.Unlock()

	fn.Logf("set: kc: %v, refs: %v, resource: %v, new: %t\n", kc, refs, x, new)
	sdkRefs, err := getSdkRefs(kc.gvkKind, refs)
	if err != nil {
		return err
	}
	return r.resources.set(sdkRefs, kc, x, new)
	//return r.resources.set(kc, refs, x, new)
}

func (r *inv) delete(kc *gvkKindCtx, refs []corev1.ObjectReference) error {
	r.m.Lock()
	defer r.m.Unlock()

	fn.Logf("delete: kc: %v, refs: %v\n", kc, refs)

	sdkRefs, err := getSdkRefs(kc.gvkKind, refs)
	if err != nil {
		return err
	}

	return r.resources.delete(sdkRefs)
	//return r.resources.delete(kc, refs)
}

func (r *inv) get(k gvkKind, refs []corev1.ObjectReference) map[corev1.ObjectReference]*resourceCtx {
	r.m.RLock()
	defer r.m.RUnlock()

	fn.Logf("get: kind: %v, refs: %v\n", k, refs)

	sdkRefs, err := getSdkRefs(k, refs)
	if err != nil {
		fn.Logf("cannot get sdkrefs :%v\n", err)
		return map[corev1.ObjectReference]*resourceCtx{}
	}

	return r.resources.get(sdkRefs)
}

func (r *inv) list() [][]sdkObjectReference {
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

func (r *resources) set(refs []sdkObjectReference, kc *gvkKindCtx, x any, new newResource) error {
	if len(refs) == 0 {
		switch d := x.(type) {
		case *kptv1.Condition:
			fn.Logf("add existing condition: %v\n", x)
			x := *d
			r.resourceCtx.existingCondition = &x
			return nil
		case *fn.KubeObject:

			r.gvkKindCtx = *kc
			x := *d
			if new {
				fn.Logf("add new resource: %v\n", x)
				r.resourceCtx.newResource = &x
			} else {
				fn.Logf("add existing resource: %v\n", x)
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
	return r.resources[refs[0]].set(refs[1:], kc, x, new)
}

func (r *resources) delete(refs []sdkObjectReference) error {
	if len(refs) == 0 {
		r.resourceCtx.existingCondition = nil
		return nil
	}
	// check if resource exists
	if !r.isInitialized(refs[0]) {
		return fmt.Errorf("not found")
	}
	return r.resources[refs[0]].delete(refs[1:])

}

func (r *resources) get(refs []sdkObjectReference) map[corev1.ObjectReference]*resourceCtx {
	if len(refs) == 0 {
		resCtxs := map[corev1.ObjectReference]*resourceCtx{}
		if len(r.resources) == 0 {
			// specific get
			resCtxs[corev1.ObjectReference{}] = r.resourceCtx.Deepcopy()
		} else {
			// wildcard get
			for sdkRef, res := range r.resources {
				resCtxs[sdkRef.ref] = res.resourceCtx.Deepcopy()
			}
		}
		return resCtxs
	}
	// check if resource exists
	fn.Logf("get2 objectRef refs: %v, empty: %v\n", refs[0].ref, corev1.ObjectReference{})
	if cmp.Equal(refs[0].ref, corev1.ObjectReference{}) {
		fn.Log("empty objectReference")
		resCtxs := map[corev1.ObjectReference]*resourceCtx{}
		for sdkRef, res := range r.resources {
			if sdkRef.gvkKind == refs[0].gvkKind {
				resCtxs[sdkRef.ref] = res.resourceCtx.Deepcopy()
			}
		}
		return resCtxs
	}
	if _, ok := r.resources[refs[0]]; !ok {
		fn.Log("ref not found")
		return map[corev1.ObjectReference]*resourceCtx{}
	}
	return r.resources[refs[0]].get(refs[1:])
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
