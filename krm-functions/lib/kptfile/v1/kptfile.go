/*
Copyright 2023 Nephio.

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

package v1

import (
	"strings"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	ko "github.com/nephio-project/nephio/krm-functions/lib/kubeobject"
	kptv1 "github.com/nephio-project/porch/pkg/kpt/api/kptfile/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	infoFieldName           = "info"
	readinessGatesFieldName = "readinessGates"
	statusFieldName         = "status"
	conditionsFieldName     = "conditions"
)

type KptFile struct {
	Kptfile *fn.KubeObject
}

// status returns with the status field of the Kptfile as a SubObject
func (r *KptFile) info() *fn.SubObject {
	return r.Kptfile.UpsertMap(infoFieldName)
}

func (r *KptFile) GetReadinessGates() []kptv1.ReadinessGate {
	var info kptv1.PackageInfo
	err := r.info().As(&info)
	if err != nil {
		return nil
	}
	return info.ReadinessGates
}

func (r *KptFile) HasReadinessGate(ct string) bool {
	for _, rg := range r.GetReadinessGates() {
		if rg.ConditionType == ct {
			return true
		}
	}
	return false
}

func (r *KptFile) SetReadinessGates(cs ...string) error {
	ergs := r.GetReadinessGates()
	for _, nct := range cs {
		found := false
		for _, erg := range ergs {
			if erg.ConditionType == nct {
				found = true
				break
			}
		}
		if !found {
			rg := kptv1.ReadinessGate{ConditionType: nct}
			ergs = append(ergs, rg)
		}
	}

	return ko.SetNestedFieldKeepFormatting(r.Kptfile, ergs, infoFieldName, readinessGatesFieldName)
}

// status returns with the status field of the Kptfile as a SubObject
func (r *KptFile) status() *fn.SubObject {
	return r.Kptfile.UpsertMap(statusFieldName)
}

// GetConditions returns with (a copy of) the list of current conditions of the kpt package
func (r *KptFile) GetConditions() []kptv1.Condition {
	var status kptv1.Status
	err := r.status().As(&status)
	if err != nil {
		return nil
	}
	return status.Conditions
}

// Get returns with the condition whose type is `conditionType` as its first return value, and
// whether the component exists or not as its second return value
func (r *KptFile) GetCondition(conditionType string) *kptv1.Condition {
	for _, cond := range r.GetConditions() {
		if cond.Type == conditionType {
			return &cond
		}
	}
	return nil
}

// SetConditions overwrites the existing condition or append the condition to the list if it does not exist
func (r *KptFile) SetConditions(ncs ...kptv1.Condition) error {
	ecs := r.GetConditions()
	for _, nc := range ncs {
		found := false
		for i, ec := range ecs {
			if ec.Type == nc.Type {
				// overwrite existing condition
				ecs[i] = nc
				found = true
				break
			}
		}
		if !found {
			ecs = append(ecs, nc)
		}
	}
	return ko.SetNestedFieldKeepFormatting(r.Kptfile, ecs, statusFieldName, conditionsFieldName)
}

// DeleteCondition deletes the conditions from the list with a given type
func (r *KptFile) DeleteCondition(ct string) error {
	ecs := r.GetConditions()
	for idx, c := range ecs {
		if c.Type == ct {
			ecs = append(ecs[:idx], ecs[idx+1:]...)
		}
	}
	return ko.SetNestedFieldKeepFormatting(r.Kptfile, ecs, statusFieldName, conditionsFieldName)
}

func (r *KptFile) DeleteConditionRef(ref corev1.ObjectReference) error {
	return r.DeleteCondition(GetConditionType(&ref))
}

func (r *KptFile) SetConditionRefFailed(ref corev1.ObjectReference, msg string) error {
	// item 0 is the forRef
	condType := GetConditionType(&ref)

	// if the condition exists update the msg and status
	c := r.GetCondition(condType)
	if c != nil {
		c.Message = msg
		c.Status = kptv1.ConditionFalse
		return r.SetConditions(*c)
	}
	// if the condition does not exist create a new condition for the for object
	return r.SetConditions(kptv1.Condition{
		Type:    condType,
		Status:  kptv1.ConditionFalse,
		Message: msg,
	})
}

func (r *KptFile) IsReady(ctPrefix string) bool {
	found := false
	for _, c := range r.GetConditions() {
		if strings.HasPrefix(c.Type, ctPrefix) {
			found = true
			if c.Status == kptv1.ConditionFalse {
				return false
			}
		}
	}
	return found
}
