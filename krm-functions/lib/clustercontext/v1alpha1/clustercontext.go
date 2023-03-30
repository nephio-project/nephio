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
