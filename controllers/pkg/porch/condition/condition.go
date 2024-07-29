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

package porchcondition

import (
	"strings"

	porchv1alpha1 "github.com/nephio-project/porch/api/porch/v1alpha1"
	kptv1 "github.com/nephio-project/porch/pkg/kpt/api/kptfile/v1"
)

// GetPorchConditions converts kpt conditions to porch conditions
func GetPorchConditions(cs []kptv1.Condition) []porchv1alpha1.Condition {
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

// HasSpecificTypeConditions checks if the package revision has forResource Conditions
// we don't care if the conditions are true or false because we can refresh the allocations
// with this approach
func HasSpecificTypeConditions(conditions []porchv1alpha1.Condition, conditionType string) bool {
	for _, c := range conditions {
		if strings.HasPrefix(c.Type, conditionType+".") {
			return true
		}
	}
	return false
}

// Check ReadinessGates checks if the package has met all readiness gates
func PackageRevisionIsReady(readinessGates []porchv1alpha1.ReadinessGate, conditions []porchv1alpha1.Condition) bool {
	// Index our conditions
	conds := make(map[string]porchv1alpha1.Condition)
	for _, c := range conditions {
		conds[c.Type] = c
	}

	// Check if the readiness gates are met
	for _, g := range readinessGates {
		if _, ok := conds[g.ConditionType]; !ok {
			return false
		}
		if conds[g.ConditionType].Status != "True" {
			return false
		}
	}

	return true
}
