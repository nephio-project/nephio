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
	infrav1alpha1 "github.com/nephio-project/nephio-controller-poc/apis/infra/v1alpha1"
)

var cluster = `apiVersion: infra.nephio.org/v1alpha1
kind: ClusterContext
metadata:
  name: clusterA
  annotations:
    config.kubernetes.io/local-config: "true"
spec:
  cniConfig:
    cniType: macvlan
    masterInterface: eth1
  siteCode: edge1
`

func TestParseObject(t *testing.T) {
	kf, err := New(cluster)
	if err != nil {
		t.Errorf("cannot unmarshal file: %s", err.Error())
	}

	cases := map[string]struct {
		wantKind string
		wantName string
	}{
		"ParseObject": {
			wantKind: "ClusterContext",
			wantName: "clusterA",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			o, err := kf.ParseKubeObject()
			if err != nil {
				t.Errorf("cannot parse object: %s", err.Error())
			}

			if diff := cmp.Diff(tc.wantKind, o.GetKind()); diff != "" {
				t.Errorf("TestParseObjectKind: -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantName, o.GetName()); diff != "" {
				t.Errorf("TestParseObjectName: -want, +got:\n%s", diff)
			}
		})
	}
}

func TestClusterContext(t *testing.T) {
	x, err := New(cluster)
	if err != nil {
		t.Errorf("cannot unmarshal file: %s", err.Error())
	}
	c := x.GetClusterContext()

	cases := map[string]struct {
		fn   func(*infrav1alpha1.ClusterContext) string
		want string
	}{
		"siteCode": {
			fn: func(*infrav1alpha1.ClusterContext) string {
				return *c.Spec.SiteCode
			},
			want: "edge1",
		},
		"cniType": {
			fn: func(*infrav1alpha1.ClusterContext) string {
				return string(c.Spec.CNIConfig.CNIType)
			},
			want: "macvlan",
		},
		"AttachementType": {
			fn: func(*infrav1alpha1.ClusterContext) string {
				return string(c.Spec.CNIConfig.MasterInterface)
			},
			want: "eth1",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := tc.fn(c)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("TestClusterContext: -want, +got:\n%s", diff)
			}
		})
	}
}
