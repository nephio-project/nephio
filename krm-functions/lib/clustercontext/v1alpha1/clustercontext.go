package v1alpha1

import (
	"errors"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	infrav1alpha1 "github.com/nephio-project/nephio-controller-poc/apis/infra/v1alpha1"
	"sigs.k8s.io/yaml"
)

type ClusterContext interface {
	// Unmarshal decodes the raw document within the in byte slice and assigns decoded values into the out value.
	// it leverages the  "sigs.k8s.io/yaml" library
	UnMarshal() (*infrav1alpha1.ClusterContext, error)
	// Marshal serializes the value provided into a YAML document based on "sigs.k8s.io/yaml".
	// The structure of the generated document will reflect the structure of the value itself.
	Marshal() ([]byte, error)
	// ParseKubeObject returns a fn sdk KubeObject; if something failed an error
	// is returned
	ParseKubeObject() (*fn.KubeObject, error)
}

// NewMutator creates a new mutator for the ClusterContext
// It expects a raw byte slice as input representing the serialized yaml file
func NewMutator(b string) ClusterContext {
	return &clusterContext{
		raw: []byte(b),
	}
}

type clusterContext struct {
	raw     []byte
	cluster *infrav1alpha1.ClusterContext
}

func (r *clusterContext) UnMarshal() (*infrav1alpha1.ClusterContext, error) {
	c := &infrav1alpha1.ClusterContext{}
	if err := yaml.Unmarshal(r.raw, c); err != nil {
		return nil, err
	}
	r.cluster = c
	return c, nil
}

// Marshal serializes the value provided into a YAML document based on "sigs.k8s.io/yaml".
// The structure of the generated document will reflect the structure of the value itself.
func (r *clusterContext) Marshal() ([]byte, error) {
	if r.cluster == nil {
		return nil, errors.New("cannot marshal unitialized cluster")
	}
	b, err := yaml.Marshal(r.cluster)
	if err != nil {
		return nil, err
	}
	r.raw = b
	return b, err
}

// ParseKubeObject returns a fn sdk KubeObject; if something failed an error
// is returned
func (r *clusterContext) ParseKubeObject() (*fn.KubeObject, error) {
	b, err := r.Marshal()
	if err != nil {
		return nil, err
	}
	return fn.ParseKubeObject(b)
}
