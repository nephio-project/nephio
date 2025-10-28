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
	"testing"

	"github.com/google/go-cmp/cmp"
	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	"github.com/kptdev/krm-functions-sdk/go/fn"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

var nadTestSriov = `apiVersion: "k8s.cni.cncf.io/v1"
kind: NetworkAttachmentDefinition
metadata:
  creationTimestamp: null
  name: upf-us-central1-n3
spec:
  config: '{"cniVersion":"0.3.1","vlan": 1001, "plugins":[{"type":"sriov","capabilities":{"ips":true,"mac":false},"master":"bond0","mode":"bridge","ipam":{"type":"static","addresses":[{"address":"10.0.0.3/24","gateway":"10.0.0.1"}]}}]}'
`

var nadTestIpVlan = `apiVersion: "k8s.cni.cncf.io/v1"
kind: NetworkAttachmentDefinition
metadata:
  creationTimestamp: null
  name: upf-us-central1-n3
spec:
  config: '{"cniVersion":"0.3.1","plugins":[{"type":"ipvlan","capabilities":{"ips":true},"master":"eth1","mode":"l2","ipam":{"type":"static","addresses":[{"address":"16.0.0.2/24","gateway":"16.0.0.1"}]}}]}'
`

var nadTestMacVlan = `apiVersion: "k8s.cni.cncf.io/v1"
kind: NetworkAttachmentDefinition
metadata:
  creationTimestamp: null
  name: upf-us-central1-n3
spec:
  config: '{"cniVersion":"0.3.1","plugins":[{"type":"macvlan","capabilities":{"ips":true},"master":"eth1","mode":"bridge","ipam":{"type":"static","addresses":[{"address":"14.0.0.2/24","gateway":"14.0.0.1"}]}},{"type":"tuning","capabilities":{"mac":true},"ipam":{}}]}'
`

var nadTestEmpty = `apiVersion: "k8s.cni.cncf.io/v1"
kind: NetworkAttachmentDefinition
metadata:
  creationTimestamp: null
  name: upf-us-central1-n3
`

func TestNewFromYAML(t *testing.T) {
	cases := map[string]struct {
		input       []byte
		errExpected bool
	}{
		"TestNewFromYAMLNormal": {
			input:       []byte(nadTestSriov),
			errExpected: false,
		},
		"TestNewFromYAMLNil": {
			input:       nil,
			errExpected: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := NewFromYAML(tc.input)

			if tc.errExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewFromGoStruct(t *testing.T) {
	configSpec := &NadConfig{
		Vlan: 007,
	}
	config, err := configSpec.String()
	if err != nil {
		t.Errorf(err.Error())
	}
	cases := map[string]struct {
		input       *nadv1.NetworkAttachmentDefinition
		errExpected bool
	}{
		"TestNewFromGoStructNormal": {
			input: &nadv1.NetworkAttachmentDefinition{
				TypeMeta: metav1.TypeMeta{
					APIVersion: nadv1.SchemeGroupVersion.Identifier(),
					Kind:       reflect.TypeFor[nadv1.NetworkAttachmentDefinition]().Name(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "a",
				},
				Spec: nadv1.NetworkAttachmentDefinitionSpec{
					Config: config,
				},
			},
			errExpected: false,
		},
		"TestNewFromGoStructNil": {
			input:       nil,
			errExpected: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := NewFromGoStruct(tc.input)

			if tc.errExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewFromKubeObject(t *testing.T) {
	cases := map[string]struct {
		input       fn.KubeObject
		errExpected bool
	}{
		"TestEmptyKubeObject": {
			input:       fn.KubeObject{},
			errExpected: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := NewFromKubeObject(&tc.input)
			if tc.errExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetKubeObject(t *testing.T) {
	i, err := NewFromYAML([]byte(nadTestSriov))
	if err != nil {
		t.Errorf("cannot unmarshal file: %s", err.Error())
	}

	cases := map[string]struct {
		wantKind string
		wantName string
	}{
		"ParseObject": {
			wantKind: "NetworkAttachmentDefinition",
			wantName: "upf-us-central1-n3",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			if diff := cmp.Diff(tc.wantKind, i.K.KubeObject.GetKind()); diff != "" {
				t.Errorf("TestGetKubeObject: -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantName, i.K.KubeObject.GetName()); diff != "" {
				t.Errorf("TestGetKubeObject: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestGetGoStruct(t *testing.T) {
	cases := map[string]struct {
		file string
		want string
	}{
		"TestGetGoStructNormal": {
			file: nadTestSriov,
			want: "0.3.1",
		},
		"TestGetGoStructEmpty": {
			file: nadTestEmpty,
			want: "",
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			g, err := i.K.GetGoStruct()
			assert.NoError(t, err)
			got := g.Spec.Config
			configSpec := &NadConfig{}
			if err := yaml.Unmarshal([]byte(got), configSpec); err != nil {
				t.Errorf("YAML Unmarshal error: %s", err)
			}
			if diff := cmp.Diff(tc.want, configSpec.CniVersion); diff != "" {
				t.Errorf("TestGetAttachmentType: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestGetSpec(t *testing.T) {
	cases := map[string]struct {
		file string
		want int
	}{
		"GetAttachmentTypeNormal": {
			file: nadTestSriov,
			want: 224,
		},
		"GetAttachmentTypeEmpty": {
			file: nadTestEmpty,
			want: 0,
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			got := i.GetConfigSpec()
			if diff := cmp.Diff(tc.want, len(got)); diff != "" {
				t.Errorf("TestGetAttachmentType: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestGetCNIType(t *testing.T) {

	cases := map[string]struct {
		file string
		want string
	}{
		"GetAttachmentTypeNormal": {
			file: nadTestSriov,
			want: "sriov",
		},
		"GetAttachmentTypeEmpty": {
			file: nadTestEmpty,
			want: "",
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			got, err := i.GetCNIType()
			if err != nil {
				t.Errorf("cannot get CNIType: %s", err.Error())
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("TestGetAttachmentType: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestGetVlan(t *testing.T) {
	cases := map[string]struct {
		file string
		want int
	}{
		"GetAttachmentTypeNormal": {
			file: nadTestSriov,
			want: 1001,
		},
		"GetAttachmentTypeEmpty": {
			file: nadTestEmpty,
			want: 0,
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			got, err := i.GetVlan()
			if err != nil {
				t.Errorf("cannot get vlan: %s", err.Error())
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("TestGetAttachmentType: -want, +got:\n%s", diff)
			}
		})
	}

}

func TestGetNadMaster(t *testing.T) {

	cases := map[string]struct {
		file string
		want string
	}{
		"GetAttachmentTypeNormal": {
			file: nadTestSriov,
			want: "bond0",
		},
		"GetAttachmentTypeEmpty": {
			file: nadTestEmpty,
			want: "",
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			got, err := i.GetNadMaster()
			if err != nil {
				t.Errorf("cannot get nad master: %s", err.Error())
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("TestGetAttachmentType: -want, +got:\n%s", diff)
			}
		})
	}
}
func TestGetIpamAddress(t *testing.T) {

	cases := map[string]struct {
		file string
		want []Address
	}{
		"GetAttachmentTypeNormal": {
			file: nadTestSriov,
			want: []Address{
				{Address: "10.0.0.3/24", Gateway: "10.0.0.1"},
			},
		},
		"GetAttachmentTypeEmpty": {
			file: nadTestEmpty,
			want: nil,
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			got, err := i.GetIpamAddress()
			if err != nil {
				t.Errorf("cannot get ipam addresses: %s", err.Error())
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("TestGetAttachmentType: -want, +got:\n%s", diff)
			}
		})
	}

}

func TestSetConfigSpec(t *testing.T) {
	cases := map[string]struct {
		file        string
		value       *nadv1.NetworkAttachmentDefinitionSpec
		errExpected bool
		length      int
	}{
		"SetAttachmentTypeNormal": {
			file: nadTestSriov,
			value: &nadv1.NetworkAttachmentDefinitionSpec{
				Config: "{\"cniVersion\": \"0.3.1\"}",
			},
			errExpected: false,
			length:      23,
		},
		"SetAttachmentTypeEmpty": {
			file:        nadTestEmpty,
			value:       &nadv1.NetworkAttachmentDefinitionSpec{},
			errExpected: false,
			length:      0,
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			err := i.SetConfigSpec(tc.value)
			if tc.errExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				got := i.GetConfigSpec()
				if diff := cmp.Diff(tc.length, len(got)); diff != "" {
					t.Errorf("TestSetAttachmentType: -want, +got:\n%s", diff)
				}
			}

		})
	}
}

func TestSetCNIType(t *testing.T) {
	cases := map[string]struct {
		file        string
		value       string
		errExpected bool
	}{
		"SetAttachmentTypeOther": {
			file:        nadTestSriov,
			value:       "calico",
			errExpected: false,
		},
		"SetAttachmentTypeSriov": {
			file:        nadTestSriov,
			value:       "sriov",
			errExpected: false,
		},
		"SetAttachmentTypeIpVlan": {
			file:        nadTestIpVlan,
			value:       "ipvlan",
			errExpected: false,
		},
		"SetAttachmentTypeMacVlan": {
			file:        nadTestMacVlan,
			value:       "macvlan",
			errExpected: false,
		},
		"SetAttachmentTypeEmpty": {
			file:        nadTestEmpty,
			value:       "",
			errExpected: true,
		},
		"SetNewAttachmentTypeOther": {
			file:        nadTestEmpty,
			value:       "calico",
			errExpected: false,
		},
		"SetNewAttachmentTypeSriov": {
			file:        nadTestEmpty,
			value:       "sriov",
			errExpected: false,
		},
		"SetNewAttachmentTypeIpVlan": {
			file:        nadTestEmpty,
			value:       "ipvlan",
			errExpected: false,
		},
		"SetNewAttachmentTypeMacVlan": {
			file:        nadTestEmpty,
			value:       "macvlan",
			errExpected: false,
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			err := i.SetCNIType(tc.value)
			if tc.errExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				got, err := i.GetCNIType()
				if err != nil {
					t.Errorf("cannot get CNIType: %s", err.Error())
				}
				if diff := cmp.Diff(tc.value, got); diff != "" {
					t.Errorf("TestSetAttachmentType: -want, +got:\n%s", diff)
				}
			}

		})
	}

}

func TestSetVlan(t *testing.T) {
	cases := map[string]struct {
		file        string
		value       int
		errExpected bool
	}{
		"SetAttachmentTypeNormal": {
			file:        nadTestSriov,
			value:       2002,
			errExpected: false,
		},
		"SetAttachmentTypeEmpty": {
			file:        nadTestEmpty,
			value:       0,
			errExpected: true,
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			err := i.SetVlan(tc.value)
			if tc.errExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				got, err := i.GetVlan()
				if err != nil {
					t.Errorf("cannot get vlan: %s", err.Error())
				}
				if diff := cmp.Diff(tc.value, got); diff != "" {
					t.Errorf("TestSetAttachmentType: -want, +got:\n%s", diff)
				}
			}

		})
	}

}

func TestSetNadMaster(t *testing.T) {
	cases := map[string]struct {
		file        string
		value       string
		errExpected bool
	}{
		"SetAttachmentTypeNormal": {
			file:        nadTestSriov,
			value:       "eno1",
			errExpected: false,
		},
		"SetAttachmentTypeEmpty": {
			file:        nadTestEmpty,
			value:       "",
			errExpected: true,
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			err := i.SetNadMaster(tc.value)
			if tc.errExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				got, err := i.GetNadMaster()
				if err != nil {
					t.Errorf("cannot get nad master: %s", err.Error())
				}
				if diff := cmp.Diff(tc.value, got); diff != "" {
					t.Errorf("TestSetAttachmentType: -want, +got:\n%s", diff)
				}
			}

		})
	}

}

func TestSetNadAddress(t *testing.T) {
	cases := map[string]struct {
		file        string
		value       []Address
		errExpected bool
	}{
		"SetAttachmentTypeNormal": {
			file: nadTestSriov,
			value: []Address{
				{Address: "10.0.0.3/24", Gateway: "10.0.0.1"},
			},
			errExpected: false,
		},
		"SetAttachmentTypeEmpty": {
			file:        nadTestEmpty,
			value:       nil,
			errExpected: true,
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			err := i.SetIpamAddress(tc.value)
			if tc.errExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				got, err := i.GetIpamAddress()
				if err != nil {
					t.Errorf("cannot get ipam addresses: %s", err.Error())
				}
				if diff := cmp.Diff(tc.value, got); diff != "" {
					t.Errorf("TestSetAttachmentType: -want, +got:\n%s", diff)
				}
			}

		})
	}

}
