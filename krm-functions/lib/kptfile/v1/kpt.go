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
	"errors"
	"sync"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	kptv1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"sigs.k8s.io/yaml"
)

type KptFile interface {
	// Marshal serializes the value provided into a YAML document based on "sigs.k8s.io/yaml".
	// The structure of the generated document will reflect the structure of the value itself.
	Marshal() ([]byte, error)
	// ParseKubeObject returns a fn sdk KubeObject; if something failed an error
	// is returned
	ParseKubeObject() (*fn.KubeObject, error)
	// GetKptFile returns the Kptfile as a go struct
	GetKptFile() *kptv1.KptFile
	// SetConditions sets the conditions in the kptfile. It either updates the entry if it exists
	// or appends the entry if it does not exist.
	SetConditions(...kptv1.Condition)
	// DeleteCondition deletes the condition equal to the conditionType if it exists
	DeleteCondition(ct string)
	// GetCondition returns the condition for the given ConditionType if it exists,
	// otherwise returns nil
	GetCondition(ct string) *kptv1.Condition
	// GetConditions returns all the conditions in the kptfile. if not initialized it
	// returns an emoty slice
	GetConditions() []kptv1.Condition
}

// New creates a new parser interface
// It expects a raw byte slice as input representing the serialized yaml file
func New(b string) (KptFile, error) {
	k := &kptv1.KptFile{}
	if err := yaml.Unmarshal([]byte(b), k); err != nil {
		return nil, err
	}
	return &kptFile{
		kptfile: k,
	}, nil
}

type kptFile struct {
	m       sync.RWMutex
	kptfile *kptv1.KptFile
}

// Marshal serializes the value provided into a YAML document based on "sigs.k8s.io/yaml".
// The structure of the generated document will reflect the structure of the value itself.
func (r *kptFile) Marshal() ([]byte, error) {
	if r.kptfile == nil {
		return nil, errors.New("cannot marshal unitialized kptfile")
	}
	return yaml.Marshal(r.kptfile)
}

// ParseKubeObject returns a fn sdk KubeObject; if something failed an error
// is returned
func (r *kptFile) ParseKubeObject() (*fn.KubeObject, error) {
	r.m.RLock()
	defer r.m.RUnlock()
	b, err := r.Marshal()
	if err != nil {
		return nil, err
	}
	return fn.ParseKubeObject(b)
}

// GetKptFile returns the Kptfile as a go struct
func (r *kptFile) GetKptFile() *kptv1.KptFile {
	return r.kptfile
}

// SetConditions sets the conditions in the kptfile. It either updates the entry if it exists
// or appends the entry if it does not exist.
func (r *kptFile) SetConditions(c ...kptv1.Condition) {
	r.m.Lock()
	defer r.m.Unlock()
	// validate is the status is set, if not initialize the condition slice
	if r.GetKptFile().Status == nil {
		r.GetKptFile().Status = &kptv1.Status{
			Conditions: []kptv1.Condition{},
		}
	} else {
		// initialize conditions if not initialized
		if r.GetKptFile().Status.Conditions == nil {
			r.GetKptFile().Status = &kptv1.Status{
				Conditions: []kptv1.Condition{},
			}
		}
	}

	// for each new condition check if the type is already in the slice
	// if not add it, if not override it.
	for _, new := range c {
		exists := false
		for i, existing := range r.GetKptFile().Status.Conditions {
			// if the condition exists we update the conditions in the kpt file
			// to the new condition
			if existing.Type != new.Type {
				continue
			}
			r.GetKptFile().Status.Conditions[i] = new
			exists = true
		}
		if !exists {
			r.GetKptFile().Status.Conditions = append(r.GetKptFile().Status.Conditions, new)
		}
	}
}

// DeleteCondition deletes the condition equal to the conditionType if it exists
func (r *kptFile) DeleteCondition(ct string) {
	r.m.Lock()
	defer r.m.Unlock()
	if r.GetKptFile().Status == nil || len(r.GetKptFile().Status.Conditions) == 0 {
		return
	}

	for idx, c := range r.GetKptFile().Status.Conditions {
		if c.Type == ct {
			r.GetKptFile().Status.Conditions = append(r.GetKptFile().Status.Conditions[:idx], r.GetKptFile().Status.Conditions[idx+1:]...)
		}
	}
}

// GetCondition returns the condition for the given ConditionType if it exists,
// otherwise returns nil
func (r *kptFile) GetCondition(ct string) *kptv1.Condition {
	r.m.RLock()
	defer r.m.RUnlock()
	if r.GetKptFile().Status == nil || len(r.GetKptFile().Status.Conditions) == 0 {
		return nil
	}

	for _, c := range r.GetKptFile().Status.Conditions {
		if c.Type == ct {
			return &c
		}
	}
	return nil
}

// GetConditions returns all the conditions in the kptfile. if not initialized it
// returns an emoty slice
func (r *kptFile) GetConditions() []kptv1.Condition {
	r.m.RLock()
	defer r.m.RUnlock()
	if r.GetKptFile().Status == nil || len(r.GetKptFile().Status.Conditions) == 0 {
		return []kptv1.Condition{}
	}
	return r.GetKptFile().Status.Conditions
}
