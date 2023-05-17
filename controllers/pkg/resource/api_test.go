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

package resource

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type object struct {
	runtime.Object
	metav1.ObjectMeta
}

func (o *object) DeepCopyObject() runtime.Object {
	return &object{ObjectMeta: *o.ObjectMeta.DeepCopy()}
}

func TestAPIPatchingApplicator(t *testing.T) {
	errBoom := errors.New("boom")
	desired := &object{}
	desired.SetName("desired")

	type args struct {
		ctx context.Context
		o   client.Object
		ao  []ApplyOption
	}

	type want struct {
		o   client.Object
		err error
	}

	cases := map[string]struct {
		reason string
		c      client.Client
		args   args
		want   want
	}{
		"GetError": {
			reason: "An error should be returned if we can't get the object",
			c:      &MockClient{MockGet: NewMockGetFn(errBoom)},
			args: args{
				o: &object{},
			},
			want: want{
				o:   &object{},
				err: errors.Wrap(errBoom, "cannot get object"),
			},
		},
		"CreateError": {
			reason: "No error should be returned if we successfully create a new object",
			c: &MockClient{
				MockGet:    NewMockGetFn(kerrors.NewNotFound(schema.GroupResource{}, "")),
				MockCreate: NewMockCreateFn(errBoom),
			},
			args: args{
				o: &object{},
			},
			want: want{
				o:   &object{},
				err: errors.Wrap(errBoom, "cannot create object"),
			},
		},
		"ApplyOptionError": {
			reason: "Any errors from an apply option should be returned",
			c:      &MockClient{MockGet: NewMockGetFn(nil)},
			args: args{
				o:  &object{},
				ao: []ApplyOption{func(_ context.Context, _, _ runtime.Object) error { return errBoom }},
			},
			want: want{
				o:   &object{},
				err: errBoom,
			},
		},
		"PatchError": {
			reason: "An error should be returned if we can't patch the object",
			c: &MockClient{
				MockGet:   NewMockGetFn(nil),
				MockPatch: NewMockPatchFn(errBoom),
			},
			args: args{
				o: &object{},
			},
			want: want{
				o:   &object{},
				err: errors.Wrap(errBoom, "cannot patch object"),
			},
		},
		"Created": {
			reason: "No error should be returned if we successfully create a new object",
			c: &MockClient{
				MockGet: NewMockGetFn(kerrors.NewNotFound(schema.GroupResource{}, "")),
				MockCreate: NewMockCreateFn(nil, func(o client.Object) error {
					*o.(*object) = *desired
					return nil
				}),
			},
			args: args{
				o: desired,
			},
			want: want{
				o: desired,
			},
		},
		"Patched": {
			reason: "No error should be returned if we successfully patch an existing object",
			c: &MockClient{
				MockGet: NewMockGetFn(nil),
				MockPatch: NewMockPatchFn(nil, func(o client.Object) error {
					*o.(*object) = *desired
					return nil
				}),
			},
			args: args{
				o: desired,
			},
			want: want{
				o: desired,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			a := NewAPIPatchingApplicator(tc.c)
			err := a.Apply(tc.args.ctx, tc.args.o, tc.args.ao...)
			if diff := cmp.Diff(tc.want.err, err, EquateErrors()); diff != "" {
				t.Errorf("\n%s\nApply(...): -want error, +got error\n%s\n", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.o, tc.args.o); diff != "" {
				t.Errorf("\n%s\nApply(...): -want, +got\n%s\n", tc.reason, diff)
			}
		})
	}
}

func TestAPIRemoveFinalizer(t *testing.T) {
	finalizer := "veryfinal"

	type args struct {
		ctx context.Context
		obj Object
	}

	type want struct {
		err error
		obj Object
	}

	errBoom := errors.New("boom")

	cases := map[string]struct {
		client client.Client
		args   args
		want   want
	}{
		"UpdateError": {
			client: &MockClient{MockUpdate: NewMockUpdateFn(errBoom)},
			args: args{
				ctx: context.Background(),
				obj: &object{ObjectMeta: metav1.ObjectMeta{Finalizers: []string{finalizer}}},
			},
			want: want{
				err: errors.Wrap(errBoom, errUpdateObject),
				obj: &object{ObjectMeta: metav1.ObjectMeta{Finalizers: []string{}}},
			},
		},
		"Successful": {
			client: &MockClient{MockUpdate: NewMockUpdateFn(nil)},
			args: args{
				ctx: context.Background(),
				obj: &object{ObjectMeta: metav1.ObjectMeta{Finalizers: []string{finalizer}}},
			},
			want: want{
				err: nil,
				obj: &object{ObjectMeta: metav1.ObjectMeta{Finalizers: []string{}}},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			api := NewAPIFinalizer(tc.client, finalizer)
			err := api.RemoveFinalizer(tc.args.ctx, tc.args.obj)
			if diff := cmp.Diff(tc.want.err, err, EquateErrors()); diff != "" {
				t.Errorf("api.RemoveFinalizer(...): -want error, +got error:\n%s", diff)
			}
		})
	}
}

func TestAPIFinalizerAdder(t *testing.T) {
	finalizer := "veryfinal"

	type args struct {
		ctx context.Context
		obj Object
	}

	type want struct {
		err error
		obj Object
	}

	errBoom := errors.New("boom")

	cases := map[string]struct {
		client client.Client
		args   args
		want   want
	}{
		"UpdateError": {
			client: &MockClient{MockUpdate: NewMockUpdateFn(errBoom)},
			args: args{
				ctx: context.Background(),
				obj: &object{ObjectMeta: metav1.ObjectMeta{Finalizers: []string{}}},
			},
			want: want{
				err: errors.Wrap(errBoom, errUpdateObject),
				obj: &object{ObjectMeta: metav1.ObjectMeta{Finalizers: []string{finalizer}}},
			},
		},
		"Successful": {
			client: &MockClient{MockUpdate: NewMockUpdateFn(nil)},
			args: args{
				ctx: context.Background(),
				obj: &object{ObjectMeta: metav1.ObjectMeta{Finalizers: []string{}}},
			},
			want: want{
				err: nil,
				obj: &object{ObjectMeta: metav1.ObjectMeta{Finalizers: []string{finalizer}}},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			api := NewAPIFinalizer(tc.client, finalizer)
			err := api.AddFinalizer(tc.args.ctx, tc.args.obj)
			if diff := cmp.Diff(tc.want.err, err, EquateErrors()); diff != "" {
				t.Errorf("api.Initialize(...): -want error, +got error:\n%s", diff)
			}
		})
	}
}
