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

package specializerreconciler

import (
	"strings"

	kptv1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	porchv1alpha1 "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
)

// getPorchConditions converts kpt conditions to porch conditions
func getPorchConditions(cs []kptv1.Condition) []porchv1alpha1.Condition {
	var prConditions []porchv1alpha1.Condition
	for _, c := range cs {
		prConditions = append(prConditions, porchv1alpha1.Condition{
			Type:    c.Type,
			Reason:  c.Reason,
			Status:  porchv1alpha1.ConditionStatus(c.Status),
			Message: c.Message,
		})
	}
	return prConditions
}

// hasSpecificTypeConditions checks if the package revision has forResource Conditions
// we don't care if the conditions are true or false because we can refresh the allocations
// with this approach
func hasSpecificTypeConditions(conditions []porchv1alpha1.Condition, conditionType string) bool {
	for _, c := range conditions {
		if strings.HasPrefix(c.Type, conditionType+".") {
			return true
		}
	}
	return false
}
