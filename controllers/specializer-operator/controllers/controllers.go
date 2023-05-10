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

package controllers

import (
	"context"

	"github.com/nephio-project/nephio/controllers/specializer-operator/controllers/config"
	"github.com/nephio-project/nephio/controllers/specializer-operator/controllers/ipam"
	"github.com/nephio-project/nephio/controllers/specializer-operator/controllers/vlan"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Setup specializer controllers.
func Setup(ctx context.Context, mgr ctrl.Manager, cfg config.SpecializerControllerConfig) error {
	for _, setup := range []func(context.Context, ctrl.Manager, config.SpecializerControllerConfig) error{
		vlan.Setup,
		ipam.Setup,
	} {
		if err := setup(ctx, mgr, cfg); err != nil {
			return err
		}
	}
	return nil
}
