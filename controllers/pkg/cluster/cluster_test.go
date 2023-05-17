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

package cluster

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetClusterClient(t *testing.T) {
	cases := map[string]struct {
		secret *corev1.Secret
		want   bool
	}{
		"None": {
			secret: &corev1.Secret{},
			want:   false,
		},
		"Capi": {
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "a-kubeconfig",
				},
				Type: corev1.SecretType("cluster.x-k8s.io/secret"),
			},
			want: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c := Cluster{}
			_, ok := c.GetClusterClient(tc.secret)

			if tc.want != ok {
				t.Errorf("want: %t, got: %t", tc.want, ok)
			}
		})
	}
}
