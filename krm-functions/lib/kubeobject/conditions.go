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

package kubeobject

import (
	"fmt"
	"sort"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	kptv1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	kptfilelibv1 "github.com/nephio-project/nephio/krm-functions/lib/kptfile/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	statusFieldName     = "status"
	conditionsFieldName = "conditions"
)

var (
	BoolToConditionStatus = map[bool]kptv1.ConditionStatus{
		true:  kptv1.ConditionTrue,
		false: kptv1.ConditionFalse,
	}
)

// KptPackageConditions provides an API to manipulate the conditions of a kpt package,
// more precisely the list of conditions in the status of the Kptfile resource
type KptPackageConditions struct {
	kptfile *fn.KubeObject
}

// NewKptPackageConditions creates a new `KptPackageConditions` instance from the list of resources in the kpt package
func NewKptPackageConditions(items fn.KubeObjects) (*KptPackageConditions, error) {
	var ret KptPackageConditions
	ret.kptfile = items.GetRootKptfile()
	if ret.kptfile == nil {
		return nil, fmt.Errorf(" Kptfile is missing from the package")

	}
	return &ret, nil
}

// status returns with the status field of the Kptfile as a SubObject
func (kpc *KptPackageConditions) status() *fn.SubObject {
	return kpc.kptfile.UpsertMap(statusFieldName)
}

func (kpc *KptPackageConditions) conditions() fn.SliceSubObjects {
	return kpc.status().GetSlice(conditionsFieldName)
}

func (kpc *KptPackageConditions) setConditions(conditions fn.SliceSubObjects) error {
	sort.SliceStable(conditions, func(i, j int) bool {
		return conditions[i].GetString("type") < conditions[j].GetString("type")
	})
	return kpc.status().SetSlice(conditions, conditionsFieldName)
}

// AsStructs returns with (a copy of) the list of current conditions of the kpt package
func (kpc *KptPackageConditions) AsStructs() []kptv1.Condition {
	var status kptv1.Status
	err := kpc.status().As(&status)
	if err != nil {
		return nil
	}
	return status.Conditions
}

// Get returns with the condition whose type is `conditionType` as its first return value, and
// whether the component exists or not as its second return value
func (kpc *KptPackageConditions) Get(conditionType string) (kptv1.Condition, bool) {
	for _, cond := range kpc.AsStructs() {
		if cond.Type == conditionType {
			return cond, true
		}
	}
	return kptv1.Condition{}, false
}

// Set creates or updates the given condition using the Type field as the primary key
func (kpc *KptPackageConditions) Set(condition kptv1.Condition) error {
	conds := kpc.conditions()
	for _, conditionSubObj := range conds {
		if conditionSubObj.GetString("type") == condition.Type {
			if err := UpdateStringField(conditionSubObj, string(condition.Status), "status"); err != nil {
				return err
			}
			if err := UpdateStringField(conditionSubObj, condition.Reason, "reason"); err != nil {
				return err
			}
			if err := UpdateStringField(conditionSubObj, condition.Message, "message"); err != nil {
				return err
			}
			return nil
		}
	}
	ko, err := fn.NewFromTypedObject(condition)
	if err != nil {
		return err
	}
	conds = append(conds, &ko.SubObject)
	return kpc.setConditions(conds)
}

// DeleteByTpe deletes all conditions with the given type
func (kpc *KptPackageConditions) DeleteByType(conditionType string) error {
	oldConditions := kpc.conditions()
	newConditions := make([]*fn.SubObject, 0, len(oldConditions))
	for _, c := range oldConditions {
		if c.GetString("type") != conditionType {
			newConditions = append(newConditions, c)
		}
	}
	return kpc.setConditions(newConditions)
}

// DeleteByObjectReference deletes the condition belonging to the referenced object
func (kpc *KptPackageConditions) DeleteByObjectReference(ref corev1.ObjectReference) error {
	return kpc.DeleteByType(kptfilelibv1.GetConditionType(&ref))
}
