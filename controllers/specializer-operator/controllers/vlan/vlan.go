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

package vlan

import (
	"context"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/nephio-project/nephio/controllers/pkg/specializerreconciler"
	"github.com/nephio-project/nephio/controllers/specializer-operator/controllers/config"
	function "github.com/nephio-project/nephio/krm-functions/vlan-fn/fn"
	vlanv1alpha1 "github.com/nokia/k8s-ipam/apis/alloc/vlan/v1alpha1"
	"github.com/nokia/k8s-ipam/pkg/proxy/clientproxy"
	"github.com/nokia/k8s-ipam/pkg/proxy/clientproxy/vlan"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func Setup(ctx context.Context, mgr ctrl.Manager, cfg config.SpecializerControllerConfig) error {
	r := &function.FnR{ClientProxy: vlan.New(
		ctx, clientproxy.Config{Address: cfg.Address},
	)}

	return specializerreconciler.Setup(mgr, specializerreconciler.Config{
		For: corev1.ObjectReference{
			APIVersion: vlanv1alpha1.SchemeBuilder.GroupVersion.Identifier(),
			Kind:       vlanv1alpha1.VLANAllocationKind,
		},
		PorchClient: cfg.PorchClient,
		KRMfunction: fn.ResourceListProcessorFunc(
			r.Run),
	})
}
