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
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
)

func TestIgnore(t *testing.T) {
	errBoom := errors.New("boom")

	type args struct {
		is  ErrorIs
		err error
	}
	cases := map[string]struct {
		args args
		want error
	}{
		"IgnoreError": {
			args: args{
				is:  func(err error) bool { return true },
				err: errBoom,
			},
			want: nil,
		},
		"PropagateError": {
			args: args{
				is:  func(err error) bool { return false },
				err: errBoom,
			},
			want: errBoom,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := Ignore(tc.args.is, tc.args.err)
			if diff := cmp.Diff(tc.want, got, EquateErrors()); diff != "" {
				t.Errorf("Ignore(...): -want error, +got error:\n%s", diff)
			}
		})
	}
}

func TestIgnoreAny(t *testing.T) {
	errBoom := errors.New("boom")

	type args struct {
		is  []ErrorIs
		err error
	}
	cases := map[string]struct {
		args args
		want error
	}{
		"IgnoreError": {
			args: args{
				is:  []ErrorIs{func(err error) bool { return true }},
				err: errBoom,
			},
			want: nil,
		},
		"IgnoreErrorArr": {
			args: args{
				is: []ErrorIs{
					func(err error) bool { return true },
					func(err error) bool { return false },
				},
				err: errBoom,
			},
			want: nil,
		},
		"PropagateError": {
			args: args{
				is:  []ErrorIs{func(err error) bool { return false }},
				err: errBoom,
			},
			want: errBoom,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := IgnoreAny(tc.args.err, tc.args.is...)
			if diff := cmp.Diff(tc.want, got, EquateErrors()); diff != "" {
				t.Errorf("Ignore(...): -want error, +got error:\n%s", diff)
			}
		})
	}
}

func EquateErrors() cmp.Option {
	return cmp.Comparer(func(a, b error) bool {
		if a == nil || b == nil {
			return a == nil && b == nil
		}

		av := reflect.ValueOf(a)
		bv := reflect.ValueOf(b)
		if av.Type() != bv.Type() {
			return false
		}

		return a.Error() == b.Error()
	})
}
