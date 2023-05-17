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
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAddFinalizer(t *testing.T) {
	finalizer := "fin"
	funalizer := "fun"

	type args struct {
		o         metav1.Object
		finalizer string
	}

	cases := map[string]struct {
		args args
		want []string
	}{
		"NoExistingFinalizers": {
			args: args{
				o:         &corev1.Pod{},
				finalizer: finalizer,
			},
			want: []string{finalizer},
		},
		"FinalizerAlreadyExists": {
			args: args{
				o: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Finalizers: []string{finalizer},
					},
				},
				finalizer: finalizer,
			},
			want: []string{finalizer},
		},
		"AnotherFinalizerExists": {
			args: args{
				o: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Finalizers: []string{funalizer},
					},
				},
				finalizer: finalizer,
			},
			want: []string{funalizer, finalizer},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			AddFinalizer(tc.args.o, tc.args.finalizer)

			got := tc.args.o.GetFinalizers()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("tc.args.o.GetFinalizers(...): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestRemoveFinalizer(t *testing.T) {
	finalizer := "fin"
	funalizer := "fun"

	type args struct {
		o         metav1.Object
		finalizer string
	}

	cases := map[string]struct {
		args args
		want []string
	}{
		"NoExistingFinalizers": {
			args: args{
				o:         &corev1.Pod{},
				finalizer: finalizer,
			},
			want: nil,
		},
		"FinalizerExists": {
			args: args{
				o: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Finalizers: []string{finalizer},
					},
				},
				finalizer: finalizer,
			},
			want: []string{},
		},
		"AnotherFinalizerExists": {
			args: args{
				o: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Finalizers: []string{finalizer, funalizer},
					},
				},
				finalizer: finalizer,
			},
			want: []string{funalizer},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			RemoveFinalizer(tc.args.o, tc.args.finalizer)

			got := tc.args.o.GetFinalizers()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("tc.args.o.GetFinalizers(...): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestFinalizerExists(t *testing.T) {
	finalizer := "fin"
	funalizer := "fun"

	type args struct {
		o         metav1.Object
		finalizer string
	}

	cases := map[string]struct {
		args args
		want bool
	}{
		"NoExistingFinalizers": {
			args: args{
				o:         &corev1.Pod{},
				finalizer: finalizer,
			},
			want: false,
		},
		"FinalizerExists": {
			args: args{
				o: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Finalizers: []string{finalizer},
					},
				},
				finalizer: finalizer,
			},
			want: true,
		},
		"AnotherFinalizerExists": {
			args: args{
				o: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Finalizers: []string{funalizer},
					},
				},
				finalizer: finalizer,
			},
			want: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if diff := cmp.Diff(tc.want, FinalizerExists(tc.args.o, tc.args.finalizer)); diff != "" {
				t.Errorf("tc.args.o.GetFinalizers(...): -want, +got:\n%s", diff)
			}
		})
	}
}
