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

const FnRuntimeOwner = "fnruntime.nephio.org/owner"
const FnRuntimeDelete = "fnruntime.nephio.org/delete"

type KptCondSDK interface {
	Run() (bool, error)
}
type ResourceKind string

const (
	// ChildRemoteCondition defines a GVK resource for which only conditions need to be created
	ChildRemoteCondition ResourceKind = "remoteCondition"
	// ChildRemote defines a GVK resource for which conditions and resources need to be created
	ChildRemote ResourceKind = "remote"
	// ChildLocal defines a GVK resource for which no conditions need to be created
	ChildLocal ResourceKind = "local"
)

type Config struct {
	For                    corev1.ObjectReference
	Owns                   map[corev1.ObjectReference]ResourceKind    // ResourceKind distinguishes ResourceKindNone and ResourceKindFull
	Watch                  map[corev1.ObjectReference]WatchCallbackFn // Used for watches to non specific resources
	PopulateOwnResourcesFn PopulateOwnResourcesFn
	GenerateResourceFn     GenerateResourceFn
}

type PopulateOwnResourcesFn func(*fn.KubeObject) (fn.KubeObjects, error)

// the list of objects contains the owns and the specific watches
type GenerateResourceFn func(*fn.KubeObject, fn.KubeObjects) (*fn.KubeObject, error)

func GenerateResourceFnNop(*fn.KubeObject, fn.KubeObjects) (*fn.KubeObject, error) { return nil, nil }

type WatchCallbackFn func(*fn.KubeObject) error

func New(rl *fn.ResourceList, cfg *Config) (KptCondSDK, error) {
	inv, err := newInventory(cfg)
	if err != nil {
		return nil, err
	}
	r := &sdk{
		cfg:   cfg,
		inv:   inv,
		rl:    rl,
		ready: true,
	}
	return r, nil
}

type sdk struct {
	cfg   *Config
	inv   inventory
	rl    *fn.ResourceList
	kptf  kptfilelibv1.KptFile
	ready bool
}

func (r *sdk) Run() (bool, error) {
	if r.rl.Items.Len() == 0 {
		r.rl.Results = append(r.rl.Results, fn.ErrorResult(fmt.Errorf("no resources present in the resourcelist")))
		return false, fmt.Errorf("no resources present in the resourcelist")
	}
	// get the kptfile first as we need it in various places
	kptfile := r.rl.Items.GetRootKptfile()
	if kptfile == nil {
		fn.Log("mandatory Kptfile is missing from the package")
		r.rl.Results.Errorf("mandatory Kptfile is missing from the package")
		return false, fmt.Errorf("mandatory Kptfile is missing from the package")
	}

	var err error
	r.kptf, err = kptfilelibv1.New(kptfile.String())
	if err != nil {
		fn.Logf("cannot unmarshal kptfile, err: %v\n", err)
		r.rl.Results = append(r.rl.Results, fn.ErrorResult(err))
		return false, err
	}

	// initialize inventory
	if err := r.populateInventory(); err != nil {
		return false, err
	}
	// list the result of inventory -> used for debug only
	r.listInventory()
	// call the global watches is used to inform the fn/controller
	// of global watch data. The fn/controller can use it to parse the data
	// and/or return an error is certain info is missing
	r.callGlobalWatches()
	// stage 1 of the sdk pipeline
	// populate the child resources as if nothing existed
	if err := r.populateChildren(); err != nil {
		return false, err
	}
	// list the result of inventory -> used for debug only
	r.listInventory()
	// update the children based on the diff between existing and new resources/conditions
	if err := r.updateChildren(); err != nil {
		return false, err
	}
	// stage 2 of the sdk pipeline
	if err := r.generateResource(); err != nil {
		return false, err
	}

	return true, nil
}
