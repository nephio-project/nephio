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
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	kptfilelibv1 "github.com/nephio-project/nephio/krm-functions/lib/kptfile/v1"
	corev1 "k8s.io/api/core/v1"
)

// populateInventory populates the inventory with the conditions and resources
// related to the config
func (r *sdk) populateInventory() error {
	// To make filtering easier the inventory distinguishes global resources
	// versus specific resources associated to a forInstance (specified through the SDK Config).
	// To perform this filtering we use the concept of the forOwnerRef, which is
	// an ownerReference associated to the forGVK
	// A watchedResource matching the forOwnerRef is assocatiated to the specific
	// forInventory context. If no match was found to the forOwnerRef the watchedResource is associated
	// to the global context
	var forOwnerRef *corev1.ObjectReference
	// we assume the kpt file is always resource idx 0 in the resourcelist, the object is used
	// as a reference to errors when we encounter issues with the condition processing
	// since conditions are stored in the kptFile
	//o := r.rl.Items.GetRootKptfile()

	// We first run through the conditions to check if an ownRef is associated
	// to the for resource objects. We call this the forOwnerRef
	// When a forOwnerRef exists it is used to associate a watch resource to the
	// inventory specific to the for resource or globally.
	for _, c := range r.kptf.GetConditions() {
		// get the specific inventory context from the conditionType
		ref := kptfilelibv1.GetGVKNFromConditionType(c.Type)
		// check if the conditionType is coming from a for KRM resource
		kindCtx, ok := r.inv.isGVKMatch(ref)
		if ok && kindCtx.gvkKind == forGVKKind {
			// get the ownerRef from the conditionReason
			// to see if the forOwnerref is present and if so initialize the forOwnerRef using the GVK
			// information
			ownerRef := kptfilelibv1.GetGVKNFromConditionType(c.Reason)
			if err := validateGVKRef(*ownerRef); err == nil {
				forOwnerRef = &corev1.ObjectReference{APIVersion: ownerRef.APIVersion, Kind: ownerRef.Kind}
			}
		}
	}
	// Now we have the forOwnerRef we run through the condition again to populate the remaining
	// resources in the inventory
	for _, c := range r.kptf.GetConditions() {
		ref := kptfilelibv1.GetGVKNFromConditionType(c.Type)
		ownerRef := kptfilelibv1.GetGVKNFromConditionType(c.Reason)
		x := c
		if err := r.populate(forOwnerRef, ref, ownerRef, &x, r.rl.Items.GetRootKptfile()); err != nil {
			return err
		}
	}
	for _, o := range r.rl.Items {
		ref := &corev1.ObjectReference{APIVersion: o.GetAPIVersion(), Kind: o.GetKind(), Name: o.GetName()}
		ownerRef := kptfilelibv1.GetGVKNFromConditionType(o.GetAnnotation(FnRuntimeOwner))
		if err := r.populate(forOwnerRef, ref, ownerRef, o, o); err != nil {
			return err
		}
	}
	return nil
}

func (r *sdk) populate(forOwnerRef, ref, ownerRef *corev1.ObjectReference, x any, o *fn.KubeObject) error {
	// we lookup in the GVK context we initialized in the beginning to validate
	// if the gvk is relevant for this fn/controller
	// what the gvk Kind is about through the kindContext
	gvkKindCtx, ok := r.inv.isGVKMatch(getGVKRefFromGVKNref(ref))
	if !ok {
		// it can be that a resource in the kpt package is not relevant for this fn/controller
		// As such we return
		fn.Logf("populate no match, ref: %v \n", ref)
		return nil
	}

	switch gvkKindCtx.gvkKind {
	case forGVKKind:
		fn.Logf("set existing object in inventory, kind %s, ref: %v ownerRef: %v\n", gvkKindCtx.gvkKind, ref, nil)
		if err := r.inv.set(gvkKindCtx, []corev1.ObjectReference{*ref}, x, false); err != nil {
			fn.Logf("error setting exisiting object in the inventory: %v\n", err.Error())
			r.rl.Results = append(r.rl.Results, fn.ErrorConfigObjectResult(err, o))
			return err
		}
	case ownGVKKind:
		ownerKindCtx, ok := r.inv.isGVKMatch(ownerRef)
		if !ok || ownerKindCtx.gvkKind != forGVKKind {
			// this means the resource was added from a different kind
			// we dont need to add this to the inventory
			fn.Logf("populate ownkind different owner, ownerRef %v, ownKind: %s ref: %v \n", ownerRef, ownerKindCtx, ref)
			return nil
		}
		fn.Logf("set existing object in inventory, kind %s, ref: %v ownerRef: %v\n", gvkKindCtx.gvkKind, ref, ownerRef)
		if err := r.inv.set(gvkKindCtx, []corev1.ObjectReference{*ownerRef, *ref}, x, false); err != nil {
			fn.Logf("error setting exisiting resource to the inventory: %v\n", err.Error())
			r.rl.Results = append(r.rl.Results, fn.ErrorConfigObjectResult(err, o))
			return err
		}
	case watchGVKKind:
		// check if the watch is specific or global
		// if no forOwnerRef is set the watch is global
		// if a forOwnerref is set we check if either the ownerRef or ref is match the GVK
		// the specifics of the name is sorted out later
		if forOwnerRef != nil && (ownerRef.APIVersion == forOwnerRef.APIVersion && ownerRef.Kind == forOwnerRef.Kind ||
			ref.APIVersion == forOwnerRef.APIVersion && ref.Kind == forOwnerRef.Kind) {
			// this is a specific watch
			forRef := &corev1.ObjectReference{APIVersion: r.cfg.For.APIVersion, Kind: r.cfg.For.Kind, Name: ref.Name}

			fn.Logf("set existing object in inventory, kind %s, ref: %v ownerRef: %v\n", gvkKindCtx.gvkKind, ref, ownerRef)
			if err := r.inv.set(gvkKindCtx, []corev1.ObjectReference{*forRef, *ref}, x, false); err != nil {
				fn.Logf("error setting exisiting resource to the inventory: %v\n", err.Error())
				r.rl.Results = append(r.rl.Results, fn.ErrorConfigObjectResult(err, o))
				return err
			}
		} else {
			// dont add a resource to the global watch if the ownref was set, sicne this would be an intermediate
			// resource owned by another for. Since we aggergate the status back we dont care about this and hence
			// we dont add the object to the inventory
			if validateGVKNRef(*ownerRef) != nil { // this mean onwerref is empty
				// this is a global watch
				fn.Logf("set existing object in inventory, kind %s, ref: %v ownerRef: %v\n", gvkKindCtx.gvkKind, ref, nil)
				if err := r.inv.set(gvkKindCtx, []corev1.ObjectReference{*ref}, x, false); err != nil {
					fn.Logf("error setting exisiting resource to the inventory: %v\n", err.Error())
					r.rl.Results = append(r.rl.Results, fn.ErrorConfigObjectResult(err, o))
					return err
				}
			}
		}
	}
	return nil
}
