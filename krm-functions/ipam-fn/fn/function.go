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

package fn

import (
	"context"
	"fmt"
	"strings"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/nephio-project/nephio/krm-functions/lib/condkptsdk"
	"github.com/nephio-project/nephio/krm-functions/lib/kubeobject"
	ipamv1alpha1 "github.com/nokia/k8s-ipam/apis/alloc/ipam/v1alpha1"
	"github.com/nokia/k8s-ipam/pkg/proxy/clientproxy"
	corev1 "k8s.io/api/core/v1"
)

type FnR struct {
	ClientProxy clientproxy.Proxy[*ipamv1alpha1.NetworkInstance, *ipamv1alpha1.IPAllocation]
}

func (f *FnR) Run(rl *fn.ResourceList) (bool, error) {
	sdk, err := condkptsdk.New(
		rl,
		&condkptsdk.Config{
			For: corev1.ObjectReference{
				APIVersion: ipamv1alpha1.GroupVersion.Identifier(),
				Kind:       ipamv1alpha1.IPAllocationKind,
			},
			PopulateOwnResourcesFn: nil,
			UpdateResourceFn:       f.updateIPAllocationResource,
		},
	)
	if err != nil {
		rl.Results.ErrorE(err)
		return false, err
	}
	return sdk.Run()
}

// updateIPAllocationResource provides an ip allocation for a given IPAllocation KRM resource
// in the package by calling the ipam backend
func (f *FnR) updateIPAllocationResource(forObj *fn.KubeObject, objs fn.KubeObjects) (*fn.KubeObject, error) {
	if forObj == nil {
		return nil, fmt.Errorf("expected a for object but got nil")
	}
	fn.Logf("ipalloc: %v\n", forObj)
	allocKOE, err := kubeobject.NewFromKubeObject[ipamv1alpha1.IPAllocation](forObj)
	if err != nil {
		return nil, err
	}
	alloc, err := allocKOE.GetGoStruct()
	if err != nil {
		return nil, err
	}
	newalloc := alloc.DeepCopy()
	newalloc.Name = getNewName(alloc.GetAnnotations(), alloc.GetName())
	fn.Logf("ipalloc newName: %s\n", newalloc.Name)
	resp, err := f.ClientProxy.Allocate(context.Background(), newalloc, nil)
	if err != nil {
		return nil, err
	}
	alloc.Status = resp.Status

	if alloc.Status.Prefix != nil {
		fn.Logf("ipalloc resp prefix: %v\n", *resp.Status.Prefix)
	}
	if alloc.Status.Gateway != nil {
		fn.Logf("ipalloc resp gateway: %v\n", *resp.Status.Gateway)
	}
	// set the status
	err = allocKOE.SetStatus(resp)
	return &allocKOE.KubeObject, err
}

func getNewName(annotations map[string]string, origName string) string {
	split := strings.Split(annotations[condkptsdk.SpecializerPurpose], ".")
	return fmt.Sprintf("%s-%s", split[len(split)-1], origName)
}
