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
	// ParseKubeObject returns a fn sdk KubeObject; if something failed an error
	// is returned
	ParseKubeObject() (*fn.KubeObject, error)
	// GetClusterContext returns the ClusterContext as a go struct
	GetClusterContext() *infrav1alpha1.ClusterContext
}

// NewMutator creates a new mutator for the ClusterContext
// It expects a raw byte slice as input representing the serialized yaml file
func New(b string) (ClusterContext, error) {
	c := &infrav1alpha1.ClusterContext{}
	if err := yaml.Unmarshal([]byte(b), c); err != nil {
		return nil, err
	}
	return &clusterContext{
		cluster: c,
	}, nil
}

type clusterContext struct {
	cluster *infrav1alpha1.ClusterContext
}

// Marshal serializes the value provided into a YAML document based on "sigs.k8s.io/yaml".
// The structure of the generated document will reflect the structure of the value itself.
func (r *clusterContext) marshal() ([]byte, error) {
	if r.cluster == nil {
		return nil, errors.New("cannot marshal unitialized cluster")
	}
	b, err := yaml.Marshal(r.cluster)
	if err != nil {
		return nil, err
	}
	return b, err
}

// ParseKubeObject returns a fn sdk KubeObject; if something failed an error
// is returned
func (r *clusterContext) ParseKubeObject() (*fn.KubeObject, error) {
	b, err := r.marshal()
	if err != nil {
		return nil, err
	}
	return fn.ParseKubeObject(b)
}

// GetClusterContext returns the ClusterContext as a go struct
func (r *clusterContext) GetClusterContext() *infrav1alpha1.ClusterContext {
	return r.cluster
}
