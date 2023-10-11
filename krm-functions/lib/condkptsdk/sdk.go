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
	kptfilelibv1 "github.com/nephio-project/nephio/krm-functions/lib/kptfile/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	SpecializerOwner         = "specializer.nephio.org/owner"
	SpecializerDelete        = "specializer.nephio.org/delete"
	SpecializerDebug         = "specializer.nephio.org/debug"
	SpecializerFor           = "specializer.nephio.org/for"
	SpecializervlanClaimName = "specializer.nephio.org/vlanClaimName"
	SpecializerNamespace     = "specializer.nephio.org/namespace"
)

type KptCondSDK interface {
	Run() (bool, error)
}
type ResourceKind string

const (
	// ChildRemoteCondition defines a GVK resource for which only conditions need to be created
	ChildRemoteCondition ResourceKind = "remoteCondition"
	// ChildRemote defines a GVK resource for which conditions and resources need to be created
	ChildRemote ResourceKind = "remote"
	// ChildLocal defines a GVK resource for which conditions will be created as true
	ChildLocal ResourceKind = "local"
	// ChildInitial defines a GVK resource which is an initial resource part fo the package and should not be deleted
	ChildInitial ResourceKind = "initial"
)

type Config struct {
	Root                   bool
	For                    corev1.ObjectReference
	Owns                   map[corev1.ObjectReference]ResourceKind    // ResourceKind distinguishes different types of child resources.
	Watch                  map[corev1.ObjectReference]WatchCallbackFn // Used for watches to non specific resources
	PopulateOwnResourcesFn PopulateOwnResourcesFn
	UpdateResourceFn       UpdateResourceFn
}

type PopulateOwnResourcesFn func(*fn.KubeObject) (fn.KubeObjects, error)

// the list of objects contains the owns and the specific watches
type UpdateResourceFn func(*fn.KubeObject, fn.KubeObjects) (fn.KubeObjects, error)

func UpdateResourceFnNop(*fn.KubeObject, fn.KubeObjects) (fn.KubeObjects, error) { return nil, nil }

type WatchCallbackFn func(*fn.KubeObject) error

func New(rl *fn.ResourceList, cfg *Config) (KptCondSDK, error) {
	inv, err := newInventory(cfg)
	if err != nil {
		return nil, err
	}
	r := &sdk{
		cfg: cfg,
		inv: inv,
		rl:  rl,
		//ready: true,
	}
	return r, nil
}

type sdk struct {
	cfg     *Config
	inv     inventory
	rl      *fn.ResourceList
	kptfile kptfilelibv1.KptFile
	debug   bool // set based on for annotation
}

func (r *sdk) Run() (bool, error) {
	if r.rl.Items.Len() == 0 {
		r.rl.Results.Infof("no resources present in the resourcelist")
		return true, nil
	}
	// get the kptfile
	// used to add/delete/update conditions
	// used to add readiness gate
	kfko := r.rl.Items.GetRootKptfile()
	if kfko == nil {
		msg := "mandatory Kptfile is missing from the package"
		fn.Log(msg)
		r.rl.Results.Errorf(msg)
		return false, fmt.Errorf(msg)
	}
	r.kptfile = kptfilelibv1.KptFile{Kptfile: kfko}

	if r.cfg.Root {
		if err := r.ensureConditionsAndGates(); err != nil {
			msg := "cannot ensure specialize conditions and readiness gates"
			fn.Logf("%s, error: %s\n", msg, err.Error())
			r.rl.Results.Errorf("%s, error: %s\n", msg, err.Error())
			return false, fmt.Errorf(err.Error(), msg)
		}
	}

	// check if debug needs to be enabled.
	// Debugging can be enabled by setting the SpecializerDebug annotation on the for resource
	r.setDebug()
	// initialize inventory
	if err := r.populateInventory(); err != nil {
		r.failForConditions(fmt.Sprintf("stage1: cannot populate inventory, err: %s", err.Error()))
		return true, nil
	}
	// list the result of inventory -> used for debug only
	if r.debug {
		r.listInventory()
	}
	// call the global watches is used to inform the fn/controller
	// of global watch data. The fn/controller can use it to parse the data
	// and/or return an error is certain info is missing
	if err := r.callGlobalWatches(); err != nil {
		// the for condition status is updated but we don't return since
		// we might act upon the readiness status, set by the global watch return status
		if r.cfg.Root {
			if err := r.kptfile.SetConditions(failed(err.Error())); err != nil {
				fn.Logf("set conditions, err: %s\n", err.Error())
				r.rl.Results.ErrorE(err)
			}
		} else {
			// only add a fail condition for a for that exists for the particular function
			// the challenge here is if the name get changed e.g. AMF example to AMF amf-cluster01
			// the condition does not clear -> this could be solved with a generic specialize condition
			// where the name is turned into a generic specializer name
			r.failForConditions(err.Error())
		}
	}
	// stage 1 of the sdk pipeline
	// populate the child resources as if nothing existed; errors are put in the conditions of the for resources
	// we only call the populate children if we are in ready status and if there are own resources. As such
	// we don't populate the children and the next part in stage 1 will act upon the result
	if r.inv.isReady() && len(r.cfg.Owns) > 0 {
		r.populateChildren()
	}

	// list the result of inventory -> used for debug only
	if r.debug {
		r.listInventory()
	}
	// update the children based on the diff between existing and new resources/conditions
	// updates resourceList, conditions and inventory
	// the error and condition update is handled in the fn as we can have multiple for resource
	r.updateChildren()

	// stage 2 of the sdk pipeline -> update resources (forObj and adjacent resources)
	// the error and condition update is handled in the fn as we can have multiple for resource
	r.updateResources()

	// handle readiness condition -> if all conditions of the for resource are true we can declare readiness
	if r.cfg.Root {
		// when not ready leave the condition as is
		if r.inv.isReady() {
			ctPrefix := kptfilelibv1.GetConditionType(&corev1.ObjectReference{APIVersion: r.cfg.For.APIVersion, Kind: r.cfg.For.Kind})
			if r.kptfile.IsReady(ctPrefix) {
				if err := r.kptfile.SetConditions(ready()); err != nil {
					fn.Logf("set conditions, err: %s\n", err.Error())
					r.rl.Results.ErrorE(err)
				}
			} else {
				if err := r.kptfile.SetConditions(notReady()); err != nil {
					fn.Logf("set conditions, err: %s\n", err.Error())
					r.rl.Results.ErrorE(err)
				}
			}
		}
	}
	return true, nil
}

func (r *sdk) setDebug() {
	// check if debug needs to be enabled.
	// Debugging can be enabled by setting the SpecializerDebug annotation on the for resource
	forObjs := r.rl.Items.Where(fn.IsGroupVersionKind(r.cfg.For.GroupVersionKind()))
	for _, forObj := range forObjs {
		if forObj.GetAnnotation(SpecializerDebug) != "" {
			r.debug = true
			r.inv.setdebug()
		}
	}
}

func (r *sdk) ensureConditionsAndGates() error {
	specializeCTType := getSpecializationConditionType()
	if err := r.kptfile.SetReadinessGates(specializeCTType); err != nil {
		return err
	}
	// if the specialization condition type is not set set it
	// if set don't touch it
	if r.kptfile.GetCondition(specializeCTType) == nil {
		if err := r.kptfile.SetConditions(initialize()); err != nil {
			fn.Logf("set conditions, err: %s\n", err.Error())
			r.rl.Results.ErrorE(err)
		}
	}
	return nil
}
