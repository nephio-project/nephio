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
	"testing"

	"github.com/google/go-cmp/cmp"
	ipamv1alpha1 "github.com/nokia/k8s-ipam/apis/alloc/ipam/v1alpha1"
	"github.com/nokia/k8s-ipam/pkg/iputil"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var yamlObject = `apiVersion: ipam.nephio.org/v1alpha1
kind: IPAllocation
# test comment a
metadata:
  name: n3
  annotations:
    config.kubernetes.io/local-config: "true"
spec:
  # test comment b
  kind: network
  networkInstance:
    name: vpc-ran # test comment c
  # test comment d
  selector: 
    matchLabels: # test comment e
      nephio.org/site: edge1 # test comment f
  index: 10
  prefixLength: 24
  addressFamily: ipv4
  prefix: 10.0.0.3/24
  createPrefix: true
  labels:
    a: b
    c: d
status:
  prefix: 10.0.0.3/24
  gateway: 10.0.0.1
`

var yamlObjectEmpty = `apiVersion: ipam.nephio.org/v1alpha1
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
			input:       []byte(yamlObject),
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
					PrefixKind: ipamv1alpha1.PrefixKindLoopback,
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
	i, err := NewFromYAML([]byte(yamlObject))
	if err != nil {
		t.Errorf("cannot unmarshal file: %s", err.Error())
	}

	cases := map[string]struct {
		wantKind string
		wantName string
	}{
		"Normal": {
			wantKind: "IPAllocation",
			wantName: "n3",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			if diff := cmp.Diff(tc.wantKind, i.GetKubeObject().GetKind()); diff != "" {
				t.Errorf("TestGetKubeObject: -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantName, i.GetKubeObject().GetName()); diff != "" {
				t.Errorf("TestGetKubeObject: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestGetGoStruct(t *testing.T) {
	cases := map[string]struct {
		file string
		want ipamv1alpha1.PrefixKind
	}{
		"Normal": {
			file: yamlObject,
			want: ipamv1alpha1.PrefixKindNetwork,
		},
		"Empty": {
			file: yamlObjectEmpty,
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
			got := g.Spec.PrefixKind
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("TestGetGoStruct: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestGetPrefixKind(t *testing.T) {
	cases := map[string]struct {
		file string
		want ipamv1alpha1.PrefixKind
	}{
		"Normal": {
			file: yamlObject,
			want: ipamv1alpha1.PrefixKindNetwork,
		},
		"Empty": {
			file: yamlObjectEmpty,
			want: "",
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			got := i.GetPrefixKind()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("TestGetPrefixKind: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestGetNetworkInstanceName(t *testing.T) {
	cases := map[string]struct {
		file string
		want string
	}{
		"Noraml": {
			file: yamlObject,
			want: "vpc-ran",
		},
		"Empty": {
			file: yamlObjectEmpty,
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

func TestGetAddressFamily(t *testing.T) {
	cases := map[string]struct {
		file string
		want iputil.AddressFamily
	}{
		"Normal": {
			file: yamlObject,
			want: iputil.AddressFamilyIpv4,
		},
		"Empty": {
			file: yamlObjectEmpty,
			want: "",
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			got := i.GetAddressFamily()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("TestGetAddressFamily: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestGetPrefix(t *testing.T) {
	cases := map[string]struct {
		file string
		want string
	}{
		"Normal": {
			file: yamlObject,
			want: "10.0.0.3/24",
		},
		"Empty": {
			file: yamlObjectEmpty,
			want: "",
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			got := i.GetPrefix()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("TestGetPrefix: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestGetPrefixLength(t *testing.T) {
	cases := map[string]struct {
		file string
		want uint8
	}{
		"Normal": {
			file: yamlObject,
			want: 24,
		},
		"Empty": {
			file: yamlObjectEmpty,
			want: 0,
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			got := i.GetPrefixLength()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("TestGetPrefixLength: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestGetIndex(t *testing.T) {
	cases := map[string]struct {
		file string
		want uint32
	}{
		"Normal": {
			file: yamlObject,
			want: 10,
		},
		"Empty": {
			file: yamlObjectEmpty,
			want: 0,
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			got := i.GetIndex()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("TestGetIndex: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestGetSelectorLabels(t *testing.T) {
	cases := map[string]struct {
		file string
		want map[string]string
	}{
		"Normal": {
			file: yamlObject,
			want: map[string]string{
				"nephio.org/site": "edge1",
			},
		},
		"Empty": {
			file: yamlObjectEmpty,
			want: map[string]string{},
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			got := i.GetSelectorLabels()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("TestGetSelectorLabels: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestGetSpecLabels(t *testing.T) {
	cases := map[string]struct {
		file string
		want map[string]string
	}{
		"Normal": {
			file: yamlObject,
			want: map[string]string{
				"a": "b",
				"c": "d",
			},
		},
		"Empty": {
			file: yamlObjectEmpty,
			want: map[string]string{},
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			got := i.GetSpecLabels()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("TestGetSpecLabels: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestGetCreatePrefix(t *testing.T) {
	cases := map[string]struct {
		file string
		want bool
	}{
		"Normal": {
			file: yamlObject,
			want: true,
		},
		"Empty": {
			file: yamlObjectEmpty,
			want: false,
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			got := i.GetCreatePrefix()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("TestGetCreatePrefix: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestGetAllocatedPrefix(t *testing.T) {
	cases := map[string]struct {
		file string
		want string
	}{
		"Normal": {
			file: yamlObject,
			want: "10.0.0.3/24",
		},
		"Empty": {
			file: yamlObjectEmpty,
			want: "",
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			got := i.GetAllocatedPrefix()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("TestGetAllocatedPrefix: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestGetAllocatedGateway(t *testing.T) {
	cases := map[string]struct {
		file string
		want string
	}{
		"Normal": {
			file: yamlObject,
			want: "10.0.0.1",
		},
		"Empty": {
			file: yamlObjectEmpty,
			want: "",
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			got := i.GetAllocatedGateway()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("TestGetAllocatedGateway: -want, +got:\n%s", diff)
			}
		})
	}
}
