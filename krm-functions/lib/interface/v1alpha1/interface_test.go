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
	"testing"

	"github.com/google/go-cmp/cmp"
	nephioreqv1alpha1 "github.com/nephio-project/api/nf_requirements/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var itface = `apiVersion: req.nephio.org/v1alpha1
kind: Interface
# test comment a
metadata:
  name: n3
  annotations:
    config.kubernetes.io/local-config: "true"
spec:
  # test comment b
  networkInstance:
    name: vpc-ran # test comment c
  # test comment d
  cniType: sriov # test comment e
  attachmentType: vlan # test comment f
`

var itfaceEmpty = `apiVersion: req.nephio.org/v1alpha1
kind: Interface
metadata:
  name: n3
  annotations:
    config.kubernetes.io/local-config: "true"
`

func TestNewFromYAML(t *testing.T) {
	cases := map[string]struct {
		input       []byte
		errExpected bool
	}{
		"TestNewFromYAMLNormal": {
			input:       []byte(itface),
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
	cases := map[string]struct {
		input       *nephioreqv1alpha1.Interface
		errExpected bool
	}{
		"TestNewFromGoStructNormal": {
			input: &nephioreqv1alpha1.Interface{
				TypeMeta: metav1.TypeMeta{
					APIVersion: nephioreqv1alpha1.SchemeBuilder.GroupVersion.Identifier(),
					Kind:       nephioreqv1alpha1.InterfaceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "a",
				},
				Spec: nephioreqv1alpha1.InterfaceSpec{
					AttachmentType: nephioreqv1alpha1.AttachmentTypeVLAN,
					CNIType:        nephioreqv1alpha1.CNITypeSRIOV,
					NetworkInstance: &corev1.ObjectReference{
						Name: "x",
					},
				},
			},
			errExpected: false,
		},
		"TestNewFromGoStructNil": {
			input:       nil,
			errExpected: false, // new approach does not return an error
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

func TestGetKubeObject(t *testing.T) {
	i, err := NewFromYAML([]byte(itface))
	if err != nil {
		t.Errorf("cannot unmarshal file: %s", err.Error())
	}

	cases := map[string]struct {
		wantKind string
		wantName string
	}{
		"ParseObject": {
			wantKind: "Interface",
			wantName: "n3",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			if diff := cmp.Diff(tc.wantKind, i.GetKind()); diff != "" {
				t.Errorf("TestGetKubeObject: -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantName, i.GetName()); diff != "" {
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
			file: itface,
			want: "vlan",
		},
		"TestGetGoStructEmpty": {
			file: itfaceEmpty,
			want: "",
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			g, err := i.GetGoStruct()
			assert.NoError(t, err)
			got := g.Spec.AttachmentType
			if diff := cmp.Diff(tc.want, string(got)); diff != "" {
				t.Errorf("TestGetAttachmentType: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestGetAttachmentType(t *testing.T) {
	cases := map[string]struct {
		file string
		want string
	}{
		"GetAttachmentTypeNormal": {
			file: itface,
			want: "vlan",
		},
		"GetAttachmentTypeEmpty": {
			file: itfaceEmpty,
			want: "",
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			got := i.GetAttachmentType()
			if diff := cmp.Diff(tc.want, got); diff != "" {
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
		"GetCNITypeNormal": {
			file: itface,
			want: "sriov",
		},
		"GetCNITypeEmpty": {
			file: itfaceEmpty,
			want: "",
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			got := i.GetCNIType()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("TestGetCNIType: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestGetNetworkInstanceName(t *testing.T) {
	cases := map[string]struct {
		file string
		want string
	}{
		"GetNetworkInstanceNameNormal": {
			file: itface,
			want: "vpc-ran",
		},
		"GetNetworkInstanceNameEmpty": {
			file: itfaceEmpty,
			want: "",
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			got := i.GetNetworkInstanceName()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("TestGetNetworkInstanceName: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestSetAttachmentType(t *testing.T) {
	cases := map[string]struct {
		file        string
		value       nephioreqv1alpha1.AttachmentType
		errExpected bool
	}{
		"SetAttachmentTypeNormal": {
			file:        itface,
			value:       "none",
			errExpected: false,
		},
		"SetAttachmentTypeEmpty": {
			file:        itfaceEmpty,
			value:       "vlan",
			errExpected: false,
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			err := i.SetAttachmentType(tc.value)
			if tc.errExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				got := i.GetAttachmentType()
				if diff := cmp.Diff(tc.value, nephioreqv1alpha1.AttachmentType(got)); diff != "" {
					t.Errorf("TestSetAttachmentType: -want, +got:\n%s", diff)
				}
			}

		})
	}
}

func TestSetCNIType(t *testing.T) {
	cases := map[string]struct {
		file        string
		value       nephioreqv1alpha1.CNIType
		errExpected bool
	}{
		"SetCNITypeNormal": {
			file:        itface,
			value:       "ipvlan",
			errExpected: false,
		},
		"SetCNITypeEmpty": {
			file:        itfaceEmpty,
			value:       "sriov",
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
				got := i.GetCNIType()
				if diff := cmp.Diff(tc.value, nephioreqv1alpha1.CNIType(got)); diff != "" {
					t.Errorf("TestSetCNIType: -want, +got:\n%s", diff)
				}
			}
		})
	}
}

func TestSetNetworkInstanceName(t *testing.T) {
	cases := map[string]struct {
		file        string
		value       string
		errExpected bool
	}{
		"SetNetworkInstanceNameNormal": {
			file:        itface,
			value:       "a",
			errExpected: false,
		},
		"SetNetworkInstanceNameEmpty": {
			file:        itfaceEmpty,
			value:       "b",
			errExpected: false,
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			err := i.SetNetworkInstanceName(tc.value)
			if tc.errExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				got := i.GetNetworkInstanceName()
				if diff := cmp.Diff(tc.value, got); diff != "" {
					t.Errorf("TestSetNetworkInstanceName: -want, +got:\n%s", diff)
				}
			}
		})
	}
}

func TestSetSpec(t *testing.T) {
	cases := map[string]struct {
		file                  string
		spec                  *nephioreqv1alpha1.InterfaceSpec
		errExpected           bool
		defaultCNIType        string
		defaultAttachmentType string
	}{
		"SetInterfaceSpecNormal": {
			file:                  itface,
			defaultCNIType:        "sriov",
			defaultAttachmentType: "vlan",
			spec: &nephioreqv1alpha1.InterfaceSpec{
				NetworkInstance: &corev1.ObjectReference{
					Name: "test",
				},
				AttachmentType: nephioreqv1alpha1.AttachmentTypeNone,
				CNIType:        nephioreqv1alpha1.CNITypeIPVLAN,
			},
			errExpected: false,
		},
		"SetInterfaceSpecDefault": {
			file:                  itface,
			defaultCNIType:        "",
			defaultAttachmentType: "",
			spec: &nephioreqv1alpha1.InterfaceSpec{
				NetworkInstance: &corev1.ObjectReference{
					Name: "test",
				},
			},
			errExpected: false,
		},
		"SetInterfaceSpecEmpty": {
			file:                  itfaceEmpty,
			defaultCNIType:        "",
			defaultAttachmentType: "",
			spec: &nephioreqv1alpha1.InterfaceSpec{
				NetworkInstance: &corev1.ObjectReference{
					Name: "test",
				},
			},
			errExpected: false,
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			err := i.SetSpec(tc.spec)
			if tc.errExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tc.spec.NetworkInstance != nil {
					got := i.GetNetworkInstanceName()
					if diff := cmp.Diff(tc.spec.NetworkInstance.Name, got); diff != "" {
						t.Errorf("TestSetInterfaceSpec: -want, +got:\n%s", diff)
					}
				}
				if tc.spec.AttachmentType != "" {
					got := i.GetAttachmentType()
					if diff := cmp.Diff(string(tc.spec.AttachmentType), got); diff != "" {
						t.Errorf("TestSetInterfaceSpec: -want, +got:\n%s", diff)
					}
				} else {
					got := i.GetAttachmentType()
					if diff := cmp.Diff(tc.defaultAttachmentType, got); diff != "" {
						t.Errorf("TestSetInterfaceSpec: -want, +got:\n%s", diff)
					}
				}
				if tc.spec.CNIType != "" {
					got := i.GetCNIType()
					if diff := cmp.Diff(string(tc.spec.CNIType), got); diff != "" {
						t.Errorf("TestSetInterfaceSpec: -want, +got:\n%s", diff)
					}
				} else {
					got := i.GetCNIType()
					if diff := cmp.Diff(tc.defaultCNIType, got); diff != "" {
						t.Errorf("TestSetInterfaceSpec: -want, +got:\n%s", diff)
					}
				}
			}
		})
	}
}

func TestDeleteCNIType(t *testing.T) {
	cases := map[string]struct {
		file string
	}{
		"DeleteCNIType": {
			file: itface,
		},
		"DeleteCNITypeEmpty": {
			file: itfaceEmpty,
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			_, err := i.DeleteCNIType()
			assert.NoError(t, err)

		})
	}
}

func TestDeleteAttachmentType(t *testing.T) {
	cases := map[string]struct {
		file string
	}{
		"TestDeleteAttachmentType": {
			file: itface,
		},
		"TestDeleteAttachmentTypeEmpty": {
			file: itfaceEmpty,
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			_, err := i.DeleteAttachmentType()
			assert.NoError(t, err)

		})
	}
}

func TestYamlComments(t *testing.T) {
	cases := map[string]struct {
		input       []byte
		errExpected bool
	}{
		"TestNewFromYAMLNormal": {
			input:       []byte(itface),
			errExpected: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			i, err := NewFromYAML(tc.input)

			if tc.errExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if err := i.SetCNIType("ipvlan"); err != nil {
				assert.NoError(t, err)
			}
			if err := i.SetAttachmentType("none"); err != nil {
				assert.NoError(t, err)
			}
			if err := i.SetNetworkInstanceName("new"); err != nil {
				assert.NoError(t, err)
			}

			if err := i.SetCNIType("sriov"); err != nil {
				assert.NoError(t, err)
			}
			if err := i.SetAttachmentType("vlan"); err != nil {
				assert.NoError(t, err)
			}
			if err := i.SetNetworkInstanceName("vpc-ran"); err != nil {
				assert.NoError(t, err)
			}

			if i.String() != string(tc.input) {
				t.Errorf("expected output to be %q, but got %q", string(tc.input), i.String())
			}
		})
	}
}
