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

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/nephio-project/nephio/krm-functions/lib/condkptsdk"
	"github.com/nephio-project/nephio/krm-functions/lib/kubeobject"
	vlanv1alpha1 "github.com/nokia/k8s-ipam/apis/resource/vlan/v1alpha1"
	"github.com/nokia/k8s-ipam/pkg/proxy/clientproxy"
	corev1 "k8s.io/api/core/v1"
)

type FnR struct {
	ClientProxy clientproxy.Proxy[*vlanv1alpha1.VLANIndex, *vlanv1alpha1.VLANClaim]
}

func (f *FnR) Run(rl *fn.ResourceList) (bool, error) {
	sdk, err := condkptsdk.New(
		rl,
		&condkptsdk.Config{
			For: corev1.ObjectReference{
				APIVersion: vlanv1alpha1.GroupVersion.Identifier(),
				Kind:       vlanv1alpha1.VLANClaimKind,
			},
			PopulateOwnResourcesFn: nil,
			UpdateResourceFn:       f.updateVLANClaimResource,
		},
	)
	if err != nil {
		rl.Results.ErrorE(err)
		return false, err
	}
	return sdk.Run()
}

// updateVLANClaimResource claims a VLAN for a given VLANClaim KRM resource
// in the package by calling the vlan backend
func (f *FnR) updateVLANClaimResource(forObj *fn.KubeObject, objs fn.KubeObjects) (*fn.KubeObject, error) {
	if forObj == nil {
		return nil, fmt.Errorf("expected a for object but got nil")
	}
	fn.Logf("vlanclaim: %v\n", forObj)
	claimKOE, err := kubeobject.NewFromKubeObject[vlanv1alpha1.VLANClaim](forObj)
	if err != nil {
		return nil, err
	}
	claim, err := claimKOE.GetGoStruct()
	if err != nil {
		return nil, err
	}
	newclaim := claim.DeepCopy()
	newclaim.Name = claim.GetAnnotations()[condkptsdk.SpecializervlanClaimName]
	fn.Logf("vlanclaim newName: %s\n", newclaim.Name)
	resp, err := f.ClientProxy.Claim(context.Background(), newclaim, nil)
	if err != nil {
		return nil, err
	}
	claim.Status = resp.Status

	if claim.Status.VLANID != nil {
		fn.Logf("claim resp vlan: %v\n", *resp.Status.VLANID)
	}
	// set the status
	err = claimKOE.SetStatus(resp)
	return &claimKOE.KubeObject, err
}
