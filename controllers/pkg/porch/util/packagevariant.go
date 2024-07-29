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

package util

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"

	porchv1alpha1 "github.com/nephio-project/porch/api/porch/v1alpha1"
	porchconfig "github.com/nephio-project/porch/api/porchconfig/v1alpha1"
	pvapi "github.com/nephio-project/porch/controllers/packagevariants/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func PackageVariantReady(ctx context.Context, pr *porchv1alpha1.PackageRevision, c client.Client) (bool, error) {
	// If the package revision is owned by a PackageVariant, check the Ready condition
	// of the package variant.
	owned := false
	for _, ownerRef := range pr.GetOwnerReferences() {
		if ownerRef.Controller == nil || !*ownerRef.Controller {
			continue
		}
		if porchconfig.GroupVersion.String() != ownerRef.APIVersion {
			continue
		}
		if ownerRef.Kind != "PackageVariant" {
			continue
		}

		owned = true
		var pv pvapi.PackageVariant
		if err := c.Get(ctx, types.NamespacedName{Namespace: pr.Namespace, Name: ownerRef.Name}, &pv); err != nil {
			return false, err
		}

		for _, cond := range pv.Status.Conditions {
			if cond.Type != "Ready" {
				continue
			}

			if cond.Status == metav1.ConditionTrue {
				return true, nil
			}

			return false, nil
		}
	}

	// if the package revision is not owned by a packagevariant, consider it Ready
	// otherwise, falling through to here should be considered not Ready, since
	// the readiness condition was not found at all.

	return !owned, nil
}
