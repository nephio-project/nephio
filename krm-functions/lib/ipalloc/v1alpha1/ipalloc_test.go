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
	ipamv1alpha1 "github.com/nokia/k8s-ipam/apis/alloc/ipam/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var normal = `apiVersion: ipam.alloc.nephio.org/v1alpha1
kind: IPAllocation
# test comment a
metadata:
  name: n3
  annotations:
    config.kubernetes.io/local-config: "true"
spec:
  kind: network
  networkInstance:
    name: vpc-ran # test comment c
  # test comment d
  selector:
    matchLabels:
      a: b
`

var empty = `apiVersion: ipam.alloc.nephio.org/v1alpha1
kind: IPAllocation
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
		input       *ipamv1alpha1.IPAllocation
		errExpected bool
	}{
		"Normal": {
			input: &ipamv1alpha1.IPAllocation{
				TypeMeta: metav1.TypeMeta{
					APIVersion: ipamv1alpha1.SchemeBuilder.GroupVersion.Identifier(),
					Kind:       ipamv1alpha1.IPAllocationKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "a",
				},
				Spec: ipamv1alpha1.IPAllocationSpec{
					PrefixKind: ipamv1alpha1.PrefixKindNetwork,
					NetworkInstance: &corev1.ObjectReference{
						Name: "x",
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
	i, err := NewFromYAML([]byte(normal))
	if err != nil {
		t.Errorf("cannot unmarshal file: %s", err.Error())
	}

	cases := map[string]struct {
		wantKind string
		wantName string
	}{
		"ParseObject": {
			wantKind: "IPAllocation",
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
		wantPK ipamv1alpha1.PrefixKind
		wantML map[string]string
		wantNI *corev1.ObjectReference
	}{
		"Normal": {
			file:   normal,
			wantPK: ipamv1alpha1.PrefixKindNetwork,
			wantML: map[string]string{
				"a": "b",
			},
			wantNI: &corev1.ObjectReference{
				Name: "vpc-ran",
			},
		},
		"Empty": {
			file:   empty,
			wantPK: "",
			wantML: nil,
			wantNI: nil,
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

			gotPK := g.Spec.PrefixKind
			if diff := cmp.Diff(tc.wantPK, gotPK); diff != "" {
				t.Errorf("PrefixKind: -want, +got:\n%s", diff)
			}

			var gotML map[string]string
			if g.Spec.Selector != nil {
				gotML = g.Spec.Selector.MatchLabels
			}
			if diff := cmp.Diff(tc.wantML, gotML); diff != "" {
				t.Errorf("MatchLabels: -want, +got:\n%s", diff)
			}
			gotNI := g.Spec.NetworkInstance
			if diff := cmp.Diff(tc.wantNI, gotNI); diff != "" {
				t.Errorf("NetworkInstance: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestSetSpec(t *testing.T) {
	cases := map[string]struct {
		file string
		t    ipamv1alpha1.IPAllocationSpec
	}{
		"Override": {
			file: normal,
			t: ipamv1alpha1.IPAllocationSpec{
				NetworkInstance: &corev1.ObjectReference{
					Name:      "x",
					Namespace: "y",
				},
				PrefixKind: ipamv1alpha1.PrefixKindLoopback,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"a": "b",
						"c": "d",
						"e": "f",
					},
				},
				CreatePrefix: true,
				Prefix:       "10.0.0.0/24",
			},
		},
		"Change": {
			file: normal,
			t: ipamv1alpha1.IPAllocationSpec{
				PrefixKind: ipamv1alpha1.PrefixKindLoopback,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"a": "b",
						"c": "d",
						"e": "f",
					},
				},
				CreatePrefix: true,
				Prefix:       "10.0.0.0/24",
			},
		},
		"Empty": {
			file: empty,
			t: ipamv1alpha1.IPAllocationSpec{
				NetworkInstance: &corev1.ObjectReference{
					Name:      "x",
					Namespace: "y",
				},
				PrefixKind: ipamv1alpha1.PrefixKindLoopback,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"a": "b",
						"c": "d",
						"e": "f",
					},
				},
				CreatePrefix: true,
				Prefix:       "10.0.0.0/24",
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
		t    ipamv1alpha1.IPAllocationStatus
	}{
		"Override": {
			file: normal,
			t: ipamv1alpha1.IPAllocationStatus{
				AllocatedPrefix: "10.0.0.1/24",
				Gateway:         "10.0.0.254",
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
