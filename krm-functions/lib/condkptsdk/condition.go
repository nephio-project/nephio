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
	kptv1 "github.com/nephio-project/porch/v4/pkg/kpt/api/kptfile/v1"
	corev1 "k8s.io/api/core/v1"
)

type ConditionReason string

// Reasons the specialization is at
const (
	ConditionReasonReady      ConditionReason = "Ready"
	ConditionReasonFailed     ConditionReason = "Failed"
	ConditionReasonSpecialize ConditionReason = "Specialize"
)

func getSpecializationConditionType() string {
	return kptfilelibv1.GetConditionType(&corev1.ObjectReference{
		APIVersion: "nephio.org",
		Kind:       "Specializer",
		Name:       "specialize",
	})
}

// Initialize returns a condition that initializes specialization
func initialize() kptv1.Condition {
	return kptv1.Condition{
		Type:    getSpecializationConditionType(),
		Status:  kptv1.ConditionFalse,
		Reason:  string(ConditionReasonSpecialize),
		Message: "initialized",
	}
}

// failed returns a condition that indicates the specialization has failed with a msg
func failed(msg string) kptv1.Condition {
	return kptv1.Condition{
		Type:    getSpecializationConditionType(),
		Status:  kptv1.ConditionFalse,
		Reason:  string(ConditionReasonFailed),
		Message: msg,
	}
}

// notReady returns a condition that indicates the specialization is notReady
func notReady() kptv1.Condition {
	return kptv1.Condition{
		Type:    getSpecializationConditionType(),
		Status:  kptv1.ConditionFalse,
		Reason:  string(ConditionReasonSpecialize),
		Message: "not ready",
	}
}

// ready returns a condition that indicates the specialization is ready
func ready() kptv1.Condition {
	return kptv1.Condition{
		Type:    getSpecializationConditionType(),
		Status:  kptv1.ConditionTrue,
		Reason:  string(ConditionReasonReady),
		Message: "",
	}
}

func (r *sdk) failForConditions(msg string) {
	forObjs := r.rl.Items.Where(fn.IsGroupVersionKind(r.cfg.For.GroupVersionKind()))
	for _, forObj := range forObjs {
		if err := r.kptfile.SetConditionRefFailed(corev1.ObjectReference{APIVersion: forObj.GetAPIVersion(), Kind: forObj.GetKind(), Name: forObj.GetName()}, msg); err != nil {
			fn.Logf("set fail for condition failed, err: %s\n", err.Error())
			r.rl.Results.ErrorE(err)
		}
	}
}
