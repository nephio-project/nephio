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
	"fmt"
	"testing"

	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestParseKubeObjectByNADInterface(t *testing.T) {
	type object struct {
		namespace string
		config    string
		name      string
	}
	testItems := []struct {
		input object
	}{
		{
			input: object{
				name:      "a",
				namespace: "b",
				config:    "c",
			},
		},
		{
			input: object{
				name:      "d",
				namespace: "e",
				config:    "",
			},
		},
	}
	for _, tt := range testItems {
		nadReceived := NewGenerator(metav1.ObjectMeta{
			Name:      tt.input.name,
			Namespace: tt.input.namespace,
		}, nadv1.NetworkAttachmentDefinitionSpec{
			Config: tt.input.config,
		})

		kubeObject, _ := nadReceived.ParseKubeObject()
		fmt.Println(kubeObject)
		if kubeObject.GetName() != tt.input.name {
			t.Errorf("TestParseKubeObjectByNADInterface: -want%s, +got:\n%s", tt.input.name, kubeObject.GetName())
		} else if kubeObject.GetNamespace() != tt.input.namespace {
			t.Errorf("TestParseKubeObjectByNADInterface: -want%s, +got:\n%s", tt.input.namespace, kubeObject.GetNamespace())
		}
		out, bool, _ := kubeObject.NestedStringMap("spec")
		if out["config"] != tt.input.config && bool {
			t.Errorf("TestParseKubeObjectByNADInterface: -want%s, +got:\n%s", tt.input.config, out["config"])
		}
	}
}

func TestParseKubeObjectBynadStruct(t *testing.T) {

	type object struct {
		namespace string
		config    string
		name      string
	}
	testItems := []struct {
		input object
	}{
		{
			input: object{
				name:      "a",
				namespace: "b",
				config:    "c",
			},
		},
		{
			input: object{
				name:      "d",
				namespace: "e",
				config:    "",
			},
		},
	}
	for _, tt := range testItems {
		test1 := nad{
			meta: metav1.ObjectMeta{
				Name:      tt.input.name,
				Namespace: tt.input.namespace,
			},
			spec: nadv1.NetworkAttachmentDefinitionSpec{
				Config: tt.input.config,
			},
		}
		kubeObject, _ := test1.ParseKubeObject()
		if kubeObject.GetName() != tt.input.name {
			t.Errorf("TestParseKubeObjectBynadStruct: -want%s, +got:\n%s", tt.input.name, kubeObject.GetName())
		} else if kubeObject.GetNamespace() != tt.input.namespace {
			t.Errorf("TestParseKubeObjectBynadStruct: -want%s, +got:\n%s", tt.input.namespace, kubeObject.GetNamespace())
		}
		out, bool, _ := kubeObject.NestedStringMap("spec")
		if out["config"] != tt.input.config && bool {
			t.Errorf("TestParseKubeObjectBynadStruct: -want%s, +got:\n%s", tt.input.config, out["config"])
		}
	}

}
