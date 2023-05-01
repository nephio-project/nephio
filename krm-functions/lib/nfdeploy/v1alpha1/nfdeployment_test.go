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
	"github.com/google/go-cmp/cmp"
	nephiodeployv1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
	nephioreqv1alpha1 "github.com/nephio-project/api/nf_requirements/v1alpha1"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

var upfdeployment = `apiVersion: workload.nephio.org/v1alpha1
kind: UPFDeployment
 # test comment a
spec:
  # test comment b
  capacity:
    maxDownlinkThroughput: 10G
    maxUplinkThroughput: 10G
  interfaces:
  - name: n3
    ipv4:
      address: 10.0.0.3/24
      gateway: 10.0.0.1
    vlanID: 100 # test comment c
  - name: n6
    ipv4:
      address: 10.0.0.4/24
      gateway: 10.0.0.2
    vlanID: 101 # test comment e
  # test comment d
  networkInstances:
  - name: vpc-ran
  - name: vpc-internet # test comment f
`

var upfdeploymentEmpty = `apiVersion: workload.nephio.org/v1alpha1
kind: UPFDeployment
spec:
`

func TestNewFromYAML(t *testing.T) {
	cases := map[string]struct {
		input       []byte
		errExpected bool
	}{
		"TestNewFromYAMLNormal": {
			input:       []byte(upfdeployment),
			errExpected: false,
		},
		"TestNewFromYAMLNil": {
			input:       nil,
			errExpected: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := NewFromYAML[*nephiodeployv1alpha1.UPFDeployment](tc.input)
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
		input       *nephiodeployv1alpha1.UPFDeployment
		errExpected bool
	}{
		"TestNewFromGoStructNormal": {
			input: &nephiodeployv1alpha1.UPFDeployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: nephioreqv1alpha1.SchemeBuilder.GroupVersion.Identifier(),
					Kind:       nephioreqv1alpha1.InterfaceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "a",
				},
				Spec: nephiodeployv1alpha1.UPFDeploymentSpec{
					NFDeploymentSpec: nephiodeployv1alpha1.NFDeploymentSpec{
						Capacity: &nephioreqv1alpha1.CapacitySpec{
							MaxDownlinkThroughput: resource.MustParse("10G"),
							MaxUplinkThroughput:   resource.MustParse("10G"),
						},
						Interfaces: []nephiodeployv1alpha1.InterfaceConfig{
							{
								Name: "n3",
							},
							{
								Name: "n6",
							},
						},
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
			_, err := NewFromGoStruct[*nephiodeployv1alpha1.UPFDeployment](tc.input)

			if tc.errExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSetSpec(t *testing.T) {
	gatewayN3 := "10.0.0.1"
	gatewayN6 := "10.0.0.2"

	vlanN3 := uint16(100)
	vlanN6 := uint16(101)
	cases := map[string]struct {
		file string
		t    nephiodeployv1alpha1.UPFDeploymentSpec
	}{
		"Override": {
			file: upfdeployment,
			t: nephiodeployv1alpha1.UPFDeploymentSpec{
				NFDeploymentSpec: nephiodeployv1alpha1.NFDeploymentSpec{
					Capacity: &nephioreqv1alpha1.CapacitySpec{
						MaxDownlinkThroughput: resource.MustParse("10G"),
						MaxUplinkThroughput:   resource.MustParse("10G"),
					},
					Interfaces: []nephiodeployv1alpha1.InterfaceConfig{
						{
							Name: "n3",
							IPv4: &nephiodeployv1alpha1.IPv4{
								Address: "10.0.0.3/24",
								Gateway: &gatewayN3,
							},
							VLANID: &vlanN3,
						},
						{
							Name: "n6",
							IPv4: &nephiodeployv1alpha1.IPv4{
								Address: "10.0.0.4/24",
								Gateway: &gatewayN6,
							},
							VLANID: &vlanN6,
						},
					},
				},
			},
		},
		"Empty": {
			file: upfdeploymentEmpty,
			t: nephiodeployv1alpha1.UPFDeploymentSpec{
				NFDeploymentSpec: nephiodeployv1alpha1.NFDeploymentSpec{
					Capacity: &nephioreqv1alpha1.CapacitySpec{
						MaxDownlinkThroughput: resource.MustParse("10G"),
						MaxUplinkThroughput:   resource.MustParse("10G"),
					},
					Interfaces: []nephiodeployv1alpha1.InterfaceConfig{
						{
							Name: "n3",
							IPv4: &nephiodeployv1alpha1.IPv4{
								Address: "10.0.0.3/24",
								Gateway: &gatewayN3,
							},
							VLANID: &vlanN3,
						},
						{
							Name: "n6",
							IPv4: &nephiodeployv1alpha1.IPv4{
								Address: "10.0.0.4/24",
								Gateway: &gatewayN6,
							},
							VLANID: &vlanN6,
						},
					},
				},
			},
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML[*nephiodeployv1alpha1.UPFDeployment]([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			err = i.SetSpec(tc.t.NFDeploymentSpec)
			assert.NoError(t, err)

			got, err := i.GetGoStruct()
			assert.NoError(t, err)

			if diff := cmp.Diff(tc.t, got.Spec); diff != "" {
				t.Errorf("-want, +got:\n%s", diff)
			}
		})
	}
}

func TestGetGoStruct(t *testing.T) {
	cases := map[string]struct {
		file                    string
		wantInterfaceConfigName string
		capacity                nephioreqv1alpha1.Capacity
		wantNetworkInstances    []nephiodeployv1alpha1.NetworkInstance
	}{
		"Normal": {
			file:                    upfdeployment,
			wantInterfaceConfigName: "n3",
			wantNetworkInstances:    nil,
		},
		"Empty": {
			file:                    upfdeployment,
			wantInterfaceConfigName: "",
			wantNetworkInstances:    nil,
		},
	}

	for name, tc := range cases {
		i, err := NewFromYAML[*nephiodeployv1alpha1.UPFDeployment]([]byte(tc.file))
		if err != nil {
			t.Errorf("cannot unmarshal file: %s", err.Error())
		}

		t.Run(name, func(t *testing.T) {
			g, err := i.GetGoStruct()
			assert.NoError(t, err)

			gotNI := g.Spec.NetworkInstances
			if diff := cmp.Diff(tc.wantNetworkInstances, gotNI); diff != "" {
				t.Errorf("NetworkInstances: -want, +got:\n%s", diff)
			}

			gotInterfaceName := g.Spec.Interfaces[0].Name
			if diff := cmp.Diff(tc.wantInterfaceConfigName, gotInterfaceName); diff != "" {
				t.Errorf("NetworkInstances: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestGetKubeObject(t *testing.T) {
	i, err := NewFromYAML[*nephiodeployv1alpha1.UPFDeployment]([]byte(upfdeployment))
	if err != nil {
		t.Errorf("cannot unmarshal file: %s", err.Error())
	}

	cases := map[string]struct {
		wantKind string
		wantName string
	}{
		"ParseObject": {
			wantKind: "UPFDeployment",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			if diff := cmp.Diff(tc.wantKind, i.GetKind()); diff != "" {
				t.Errorf("TestGetKubeObject: -want, +got:\n%s", diff)
			}
		})
	}
}
