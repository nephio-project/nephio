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

package v1alpha1

import (
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	nephiodeployv1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	"github.com/nephio-project/nephio/krm-functions/lib/kubeobject"
)

type NfDeploymentType interface {
	*nephiodeployv1alpha1.UPFDeployment | *nephiodeployv1alpha1.SMFDeployment | *nephiodeployv1alpha1.AMFDeployment
}

type NfDeployment[T1 NfDeploymentType] struct {
	kubeobject.KubeObjectExt[T1]
}

func NewFromKubeObject[T NfDeploymentType](o *fn.KubeObject) (*NfDeployment[T], error) {
	r, err := kubeobject.NewFromKubeObject[T](o)
	if err != nil {
		return nil, err
	}
	return &NfDeployment[T]{*r}, nil
}

// NewFromYAML creates a new parser interface
// It expects a raw byte slice as input representing the serialized yaml file
func NewFromYAML[T NfDeploymentType](b []byte) (*NfDeployment[T], error) {
	r, err := kubeobject.NewFromYaml[T](b)
	if err != nil {
		return nil, err
	}
	return &NfDeployment[T]{*r}, nil
}

func (r *NfDeployment[T1]) SetSpec(spec nephiodeployv1alpha1.NFDeploymentSpec) error {
	return r.KubeObjectExt.SetSpec(spec)
}

func (r *NfDeployment[T1]) SetStatus(spec nephiodeployv1alpha1.NFDeploymentStatus) error {
	return r.KubeObjectExt.SetStatus(spec)
}

// NewFromGoStruct creates a new parser interface
// It expects a go struct representing the interface krm resource
func NewFromGoStruct[T NfDeploymentType](x T) (*NfDeployment[T], error) {
	r, err := kubeobject.NewFromGoStruct[T](x)
	if err != nil {
		return nil, err
	}
	return &NfDeployment[T]{*r}, nil
}
