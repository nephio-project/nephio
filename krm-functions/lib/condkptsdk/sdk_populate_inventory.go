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
	"github.com/nephio-project/nephio/krm-functions/lib/ref"
	corev1 "k8s.io/api/core/v1"
)

// populateInventory populates the inventory with the conditions and resources
// related to the config
func (r *sdk) populateInventory() error {
	// To make filtering easier the inventory distinguishes global resources
	// versus specific resources associated to a forInstance (specified through the SDK Config).
	// To perform this filtering we use the concept of the forOwnerRef, which is
	// an ownerReference associated to the forGVK
	// A watchedResource matching the forOwnerRef is associated to the specific
	// forInventory context. If no match was found to the forOwnerRef the watchedResource is associated
	// to the global context
	var forOwnerRef *corev1.ObjectReference
	// keeps track a map to link the forOnwer name to the specific for resourceName
	// used by NAD since we dont do the intelligent diff, we need to handle the mapping
	// used only to populate the inventory for specific watches
	forOwnerRefNameMap := map[string]string{}

	// We first run through the conditions to check if an ownRef is associated
	// to the for resource objects. We call this the forOwnerRef
	// When a forOwnerRef exists it is used to associate a watch resource to the
	// inventory specific to the for resource or globally.
	for _, c := range r.kptfile.GetConditions() {
		// get the specific inventory context from the conditionType
		objRef := kptfilelibv1.GetGVKNFromConditionType(c.Type)
		// check if the conditionType is coming from a for KRM resource
		kindCtx, ok := r.inv.isGVKMatch(objRef)
		if ok && kindCtx.gvkKind == forGVKKind {
			// get the ownerRef from the conditionReason
			// to see if the forOwnerref is present and if so initialize the forOwnerRef using the GVK
			// information
			ownerRef := kptfilelibv1.GetGVKNFromConditionType(c.Reason)
			if err := ref.ValidateGVKRef(*ownerRef); err == nil {
				forOwnerRef = &corev1.ObjectReference{APIVersion: ownerRef.APIVersion, Kind: ownerRef.Kind}
				forOwnerRefNameMap[ownerRef.Name] = objRef.Name
				if r.debug {
					fn.Logf("forOwnerRefNameMap: refKind: %s, refName: %s, forOwnRefName: %s\n", objRef.Kind, objRef.Name, ownerRef.Name)
				}
			}
		}
	}
	// Now we have the forOwnerRef we run through the condition again to populate the remaining
	// resources in the inventory
	for _, c := range r.kptfile.GetConditions() {
		ref := kptfilelibv1.GetGVKNFromConditionType(c.Type)
		ownerRef := kptfilelibv1.GetGVKNFromConditionType(c.Reason)
		x := c
		if err := r.populate(forOwnerRefNameMap, forOwnerRef, ref, ownerRef, &x, r.kptfile.Kptfile); err != nil {
			return err
		}
	}
	for _, o := range r.rl.Items {
		ref := &corev1.ObjectReference{APIVersion: o.GetAPIVersion(), Kind: o.GetKind(), Name: o.GetName()}
		ownerRef := kptfilelibv1.GetGVKNFromConditionType(o.GetAnnotation(SpecializerOwner))
		if err := r.populate(forOwnerRefNameMap, forOwnerRef, ref, ownerRef, o, o); err != nil {
			return err
		}
	}
	return nil
}

func (r *sdk) populate(forOwnerRefNameMap map[string]string, forOwnerRef, objRef, ownerRef *corev1.ObjectReference, x any, relatedObject *fn.KubeObject) error {
	// we lookup in the GVK context we initialized in the beginning to validate
	// if the gvk is relevant for this fn/controller
	// what the gvk Kind is about through the kindContext
	gvkKindCtx, ok := r.inv.isGVKMatch(ref.GetGVKRefFromGVKNref(objRef))
	if !ok {
		// it can be that a resource in the kpt package is not relevant for this fn/controller
		// As such we return
		if r.debug {
			fn.Logf("stag1: populate no match, ref: %v \n", objRef)
		}
		return nil
	}

	switch gvkKindCtx.gvkKind {
	case forGVKKind:
		if r.debug {
			fn.Logf("stag1: set existing object in inventory, kind %s, ref: %v ownerRef: %v\n", gvkKindCtx.gvkKind, objRef, nil)
		}
		if err := r.inv.set(gvkKindCtx, []corev1.ObjectReference{*objRef}, x, false, false); err != nil {
			fn.Logf("stag1: cannot set existing object in the inventory: %v\n", err.Error())
			//r.rl.Results = append(r.rl.Results, fn.ErrorConfigObjectResult(err, relatedObject))
			return err
		}
	case ownGVKKind:
		ownerKindCtx, ok := r.inv.isGVKMatch(ownerRef)
		if !ok || ownerKindCtx.gvkKind != forGVKKind {
			// this means the resource was added from a different kind
			// we don't need to add this to the inventory
			// with wildcards this is extremely important, otherwise we end up adding everything to the inventory
			// context
			if r.debug {
				fn.Logf("stag1: populate ownkind different owner, ownerRef %v, ownKind: %v ref: %v \n", ownerRef, ownerKindCtx, objRef)
			}
			return nil
		}
		if r.debug {
			fn.Logf("stage1: set existing object in inventory, kind %s, ref: %v ownerRef: %v\n", gvkKindCtx.gvkKind, objRef, ownerRef)
		}
		if err := r.inv.set(gvkKindCtx, []corev1.ObjectReference{*ownerRef, *objRef}, x, false, false); err != nil {
			fn.Logf("stage1: cannot set existing resource to the inventory: %v\n", err.Error())
			//r.rl.Results = append(r.rl.Results, fn.ErrorConfigObjectResult(err, relatedObject))
			return err
		}
	case watchGVKKind:
		// check if the watch is specific or global
		// if no forOwnerRef is set the watch is global
		// if a forOwnerref is set we check if either the ownerRef or ref is match the GVK
		// the specifics of the name is sorted out later
		if forOwnerRef != nil && (forOwnerRef.APIVersion == ownerRef.APIVersion && forOwnerRef.Kind == ownerRef.Kind ||
			forOwnerRef.APIVersion == objRef.APIVersion && forOwnerRef.Kind == objRef.Kind) {
			// this is a specific watch

			// The name is a bit complicated ->
			// in general we take the ownerref
			// when the forOwnerRef matches we take the name of the ref since the ownerref here is owned by another resource
			// e.g. interface in NAD context is owned by nfdeploy, so we take the name of the ref iso ownerref
			name := forOwnerRefNameMap[ownerRef.Name]
			if forOwnerRef.APIVersion == objRef.APIVersion && forOwnerRef.Kind == objRef.Kind {
				name = forOwnerRefNameMap[objRef.Name]
			}
			forRef := &corev1.ObjectReference{APIVersion: r.cfg.For.APIVersion, Kind: r.cfg.For.Kind, Name: name}

			if r.debug {
				fn.Logf("stage1: set existing object in inventory, kind %s, forRef: %v, ref: %v ownerRef: %v\n", gvkKindCtx.gvkKind, forRef, objRef, ownerRef)
			}
			if err := r.inv.set(gvkKindCtx, []corev1.ObjectReference{*forRef, *objRef}, x, false, false); err != nil {
				fn.Logf("stage1: cannot set existing resource to the inventory: %v\n", err.Error())
				//r.rl.Results = append(r.rl.Results, fn.ErrorConfigObjectResult(err, relatedObject))
				return err
			}
		} else {
			// don't add a resource to the global watch if the ownerref was set, since this would be an intermediate
			// resource owned by another for. Since we aggregate the status back we don't care about this and hence
			// we don't add the object to the inventory
			if ref.ValidateGVKNRef(*ownerRef) != nil { // this mean ownerref is empty
				// this is a global watch
				if r.debug {
					fn.Logf("stage1: set existing object in inventory, kind %s, ref: %v ownerRef: %v\n", gvkKindCtx.gvkKind, objRef, nil)
				}
				if err := r.inv.set(gvkKindCtx, []corev1.ObjectReference{*objRef}, x, false, false); err != nil {
					fn.Logf("stage1: cannot set existing resource to the inventory: %v\n", err.Error())
					//r.rl.Results = append(r.rl.Results, fn.ErrorConfigObjectResult(err, relatedObject))
					return err
				}
			}
		}
	}
	return nil
}
