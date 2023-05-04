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
	vlan1alpha1 "github.com/nokia/k8s-ipam/apis/alloc/vlan/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var normal = `apiVersion: vlan.alloc.nephio.org/v1alpha1
kind: VLANAllocation
# test comment a
metadata:
  name: n3
  annotations:
    config.kubernetes.io/local-config: "true"
spec:
  vlanDatabase:
  - name: vpc-ran # test comment c
  # test comment d
  selector:
    matchLabels:
      a: b
`

var empty = `apiVersion: ipam.alloc.nephio.org/v1alpha1
kind: VLANAllocation
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
		"Normal": {
			input:       []byte(normal),
			errExpected: false,
		},
		"Nil": {
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
		input       *vlan1alpha1.VLANAllocation
		errExpected bool
	}{
		"Normal": {
			input: &vlan1alpha1.VLANAllocation{
				TypeMeta: metav1.TypeMeta{
					APIVersion: vlan1alpha1.SchemeBuilder.GroupVersion.Identifier(),
					Kind:       vlan1alpha1.VLANAllocationKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "a",
				},
				Spec: vlan1alpha1.VLANAllocationSpec{
					VLANDatabases: []*corev1.ObjectReference{
						{Name: "x"},
					},
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"a": "b",
						},
					},
				},
			},
			errExpected: false,
		},
		"Nil": {
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

func TestGetKubeObject(t *testing.T) {
	i, err := NewFromYAML([]byte(normal))
	if err != nil {
		t.Errorf("cannot unmarshal file: %s", err.Error())
	}

	cases := map[string]struct {
		wantKind string
		wantName string
	}{
		"ParseObject": {
			wantKind: "VLANAllocation",
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
		file   string
		wantML map[string]string
		wantDB []*corev1.ObjectReference
	}{
		"Normal": {
			file: normal,
			wantML: map[string]string{
				"a": "b",
			},
			wantDB: []*corev1.ObjectReference{
				{Name: "vpc-ran"},
			},
		},
		"Empty": {
			file:   empty,
			wantML: nil,
			wantDB: nil,
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

			var gotML map[string]string
			if g.Spec.Selector != nil {
				gotML = g.Spec.Selector.MatchLabels
			}
			if diff := cmp.Diff(tc.wantML, gotML); diff != "" {
				t.Errorf("MatchLabels: -want, +got:\n%s", diff)
			}
			gotDB := g.Spec.VLANDatabases
			if diff := cmp.Diff(tc.wantDB, gotDB); diff != "" {
				t.Errorf("NetworkInstance: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestSetSpec(t *testing.T) {
	cases := map[string]struct {
		file string
		t    vlan1alpha1.VLANAllocationSpec
	}{
		"Override": {
			file: normal,
			t: vlan1alpha1.VLANAllocationSpec{
				VLANDatabases: []*corev1.ObjectReference{
					{Name: "x", Namespace: "y"},
				},
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"a": "b",
						"c": "d",
						"e": "f",
					},
				},
			},
		},
		"Change": {
			file: normal,
			t: vlan1alpha1.VLANAllocationSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"a": "b",
						"c": "d",
						"e": "f",
					},
				},
			},
		},
		"Empty": {
			file: empty,
			t: vlan1alpha1.VLANAllocationSpec{
				VLANDatabases: []*corev1.ObjectReference{
					{Name: "x", Namespace: "y"},
				},
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"a": "b",
						"c": "d",
						"e": "f",
					},
				},
			},
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			err = i.SetSpec(tc.t)
			assert.NoError(t, err)

			got, err := i.GetGoStruct()
			assert.NoError(t, err)

			if diff := cmp.Diff(tc.t, got.Spec); diff != "" {
				t.Errorf("-want, +got:\n%s", diff)
			}
		})
	}
}

func TestSetStatus(t *testing.T) {
	cases := map[string]struct {
		file string
		t    vlan1alpha1.VLANAllocationStatus
	}{
		"Override": {
			file: normal,
			t: vlan1alpha1.VLANAllocationStatus{
				AllocatedVlanID: 100,
			},
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			err = i.SetStatus(tc.t)
			assert.NoError(t, err)

			got, err := i.GetGoStruct()
			assert.NoError(t, err)

			if diff := cmp.Diff(tc.t, got.Status); diff != "" {
				t.Errorf("-want, +got:\n%s", diff)
			}
		})
	}
}
