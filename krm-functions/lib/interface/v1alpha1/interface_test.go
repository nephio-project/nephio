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
	ipamv1alpha1 "github.com/nokia/k8s-ipam/apis/alloc/ipam/v1alpha1"
	vlanv1alpha1 "github.com/nokia/k8s-ipam/apis/alloc/vlan/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
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
		file   string
		wantAT nephioreqv1alpha1.AttachmentType
		wantCT nephioreqv1alpha1.CNIType
		wantNI *corev1.ObjectReference
	}{
		"Normal": {
			file:   itface,
			wantAT: nephioreqv1alpha1.AttachmentTypeVLAN,
			wantCT: nephioreqv1alpha1.CNITypeSRIOV,
			wantNI: &corev1.ObjectReference{
				Name: "vpc-ran",
			},
		},
		"Empty": {
			file:   itfaceEmpty,
			wantAT: "",
			wantCT: "",
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

			gotAT := g.Spec.AttachmentType
			if diff := cmp.Diff(tc.wantAT, gotAT); diff != "" {
				t.Errorf("AttachmentType: -want, +got:\n%s", diff)
			}
			gotCT := g.Spec.CNIType
			if diff := cmp.Diff(tc.wantCT, gotCT); diff != "" {
				t.Errorf("CNIType: -want, +got:\n%s", diff)
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
		t    nephioreqv1alpha1.InterfaceSpec
	}{
		"Override": {
			file: itface,
			t: nephioreqv1alpha1.InterfaceSpec{
				NetworkInstance: &corev1.ObjectReference{
					Name:      "a",
					Namespace: "b",
				},
				AttachmentType: nephioreqv1alpha1.AttachmentTypeNone,
				CNIType:        nephioreqv1alpha1.CNITypeIPVLAN,
			},
		},
		"Change": {
			file: itface,
			t: nephioreqv1alpha1.InterfaceSpec{
				CNIType: nephioreqv1alpha1.CNITypeIPVLAN,
			},
		},
		"Empty": {
			file: itfaceEmpty,
			t: nephioreqv1alpha1.InterfaceSpec{
				NetworkInstance: &corev1.ObjectReference{
					Name:      "a",
					Namespace: "b",
				},
				AttachmentType: nephioreqv1alpha1.AttachmentTypeNone,
				CNIType:        nephioreqv1alpha1.CNITypeIPVLAN,
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
	x := uint16(10)
	cases := map[string]struct {
		file string
		t    nephioreqv1alpha1.InterfaceStatus
	}{
		"Override": {
			file: itface,
			t: nephioreqv1alpha1.InterfaceStatus{
				IPAllocationStatus: []ipamv1alpha1.IPAllocationStatus{
					{
						Prefix:  pointer.String("10.0.0.2/24"),
						Gateway: pointer.String("10.0.0.1"),
					},
				},
				VLANAllocationStatus: &vlanv1alpha1.VLANAllocationStatus{
					VLANID: &x,
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
