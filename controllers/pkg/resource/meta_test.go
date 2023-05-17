/*
Copyright 2022 Nokia.

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

package resource

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestWasDeleted(t *testing.T) {
	now := metav1.Now()

	cases := map[string]struct {
		o    metav1.Object
		want bool
	}{
		"ObjectWasDeleted": {
			o:    &corev1.Pod{ObjectMeta: metav1.ObjectMeta{DeletionTimestamp: &now}},
			want: true,
		},
		"ObjectWasNotDeleted": {
			o:    &corev1.Pod{ObjectMeta: metav1.ObjectMeta{DeletionTimestamp: nil}},
			want: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := WasDeleted(tc.o)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("WasDeleted(...): -want, +got:\n%s", diff)
			}
		})
	}
}
