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

package openshift

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetClusterName(t *testing.T) {
	cases := map[string]struct {
		secret *corev1.Secret
		want   string
	}{
		"Nil": {
			secret: &corev1.Secret{},
			want:   "",
		},
		"None": {
			secret: &corev1.Secret{},
			want:   "",
		},
		"OpenShift": {
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "a-kubeconfig",
					Labels: map[string]string{
						"hive.openshift.io/cluster-deployment-name": "ca-montreal",
						"hive.openshift.io/secret-type":             "kubeconfig",
					},
				},
				Type: corev1.SecretType("cluster.x-k8s.io/secret"),
			},
			want: "ca-montreal",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := OpenShift{
				Secret: tc.secret,
			}
			got := c.GetClusterName()

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("-want, +got:\n%s", diff)
			}
		})
	}
}
