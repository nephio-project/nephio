package v1

import (
	"errors"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	kptv1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type KptFile interface {
	UnMarshal() (*kptv1.KptFile, error)
	Marshal() ([]byte, error)
	ParseKubeObject() (*fn.KubeObject, error)
	GetKptFile() *kptv1.KptFile
	SetConditions(...kptv1.Condition)
	DeleteCondition(ct string)
	GetCondition(ct string) *kptv1.Condition
	GetConditions() []kptv1.Condition
}

// NewMutator creates a new mutator for the kptfile
// It expects a raw byte slice as input representing the serialized yaml file
func NewMutator(b string) KptFile {
	return &kptFile{
		raw: []byte(b),
	}
}

func NewEmpty() KptFile {
	return &kptFile{
		kptfile: &kptv1.KptFile{},
	}
}

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
	raw     []byte
	kptfile *kptv1.KptFile
}

// Unmarshal decodes the raw document within the in byte slice and assigns decoded values into the out value.
// it leverages the  "sigs.k8s.io/yaml" library
func (r *kptFile) UnMarshal() (*kptv1.KptFile, error) {
	k := &kptv1.KptFile{}
	if err := yaml.Unmarshal(r.raw, k); err != nil {
		return nil, err
	}
	r.kptfile = k
	return k, nil
}

// Marshal serializes the value provided into a YAML document based on "sigs.k8s.io/yaml".
// The structure of the generated document will reflect the structure of the value itself.
func (r *kptFile) Marshal() ([]byte, error) {
	if r.kptfile == nil {
		return nil, errors.New("cannot marshal unitialized kptfile")
	}
	b, err := yaml.Marshal(r.kptfile)
	if err != nil {
		return nil, err
	}
	r.raw = b
	return b, err
}

// ParseKubeObject returns a fn sdk KubeObject; if something failed an error
// is returned
func (r *kptFile) ParseKubeObject() (*fn.KubeObject, error) {
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
	// validate is the status is set, if not initialize the condition slice
	if r.GetKptFile().Status == nil {
		r.GetKptFile().Status = &kptv1.Status{
			Conditions: []kptv1.Condition{},
		}
	}

	// for each nex condition check if the kind
	for _, new := range c {
		exists := false
		for i, existing := range r.GetKptFile().Status.Conditions {
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
	if r.GetKptFile().Status == nil || len(r.GetKptFile().Status.Conditions) == 0 {
		return []kptv1.Condition{}
	}
	return r.GetKptFile().Status.Conditions
}
