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
    nephioreqv1alpha1 "github.com/nephio-project/api/nf_requirements/v1alpha1"
    "sigs.k8s.io/yaml"
)

type Interface interface {
    // ParseKubeObject returns a fn sdk KubeObject; if something failed an error
    // is returned
    ParseKubeObject() (*fn.KubeObject, error)
    // GetInterface returns the interface as a go struct
    GetInterface() *nephioreqv1alpha1.Interface
}

// New creates a new parser interface
// It expects a raw byte slice as input representing the serialized yaml file
func New(b string) (Interface, error) {
    i := &nephioreqv1alpha1.Interface{}
    if err := yaml.Unmarshal([]byte(b), i); err != nil {
        return nil, err
    }
    return &itfce{
        itfce: i,
    }, nil
}

type itfce struct {
    itfce *nephioreqv1alpha1.Interface
}

// Marshal serializes the value provided into a YAML document based on "sigs.k8s.io/yaml".
// The structure of the generated document will reflect the structure of the value itself.
func (r *itfce) marshal() ([]byte, error) {
    if r.itfce == nil {
        return nil, errors.New("cannot marshal unitialized interface")
    }
    return yaml.Marshal(r.itfce)
}

// ParseKubeObject returns a fn sdk KubeObject; if something failed an error
// is returned
func (r *itfce) ParseKubeObject() (*fn.KubeObject, error) {
    b, err := r.marshal()
    if err != nil {
        return nil, err
    }
    return fn.ParseKubeObject(b)
}

// GetInterface returns the interface as a go struct
func (r *itfce) GetInterface() *nephioreqv1alpha1.Interface {
    return r.itfce
}