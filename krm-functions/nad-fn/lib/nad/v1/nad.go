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
	"reflect"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

type NAD interface {
	ParseKubeObject() (*fn.KubeObject, error)
}

// NewGenerator creates a new generator for the nad
// It expects a raw byte slice as input representing the serialized yaml file
func NewGenerator(meta metav1.ObjectMeta, spec nadv1.NetworkAttachmentDefinitionSpec) NAD {
	return &nad{
		meta: meta,
		spec: spec,
	}
}

type nad struct {
	meta metav1.ObjectMeta
	spec nadv1.NetworkAttachmentDefinitionSpec
}

func (r *nad) ParseKubeObject() (*fn.KubeObject, error) {
	ipa := &nadv1.NetworkAttachmentDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: nadv1.SchemeGroupVersion.Identifier(),
			Kind:       reflect.TypeOf(nadv1.NetworkAttachmentDefinition{}).Name(),
		},
		ObjectMeta: r.meta,
		Spec:       r.spec,
	}
	b, err := yaml.Marshal(ipa)
	if err != nil {
		return nil, err
	}
	return fn.ParseKubeObject(b)
}
