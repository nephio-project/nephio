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

package condkptsdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestNewInventory(t *testing.T) {
	cases := map[string]struct {
		input       *Config
		errExpected bool
	}{
		"Normal": {
			input: &Config{
				For: corev1.ObjectReference{APIVersion: "a", Kind: "a"},
				Owns: map[corev1.ObjectReference]ResourceKind{
					{APIVersion: "b", Kind: "b"}: ChildRemote,
				},
				Watch: map[corev1.ObjectReference]WatchCallbackFn{
					{APIVersion: "c", Kind: "c"}: nil,
				},
				GenerateResourceFn: GenerateResourceFnNop,
			},
			errExpected: false,
		},
		"NoFor": {
			input: &Config{
				GenerateResourceFn: GenerateResourceFnNop,
			},
			errExpected: true,
		},
		"GenerateResourceFn": {
			input: &Config{
				For: corev1.ObjectReference{APIVersion: "a", Kind: "a"},
			},
			errExpected: true,
		},
		"DuplicateGVK1": {
			input: &Config{
				For: corev1.ObjectReference{APIVersion: "a", Kind: "a"},
				Owns: map[corev1.ObjectReference]ResourceKind{
					{APIVersion: "b", Kind: "b"}: ChildRemote,
				},
				Watch: map[corev1.ObjectReference]WatchCallbackFn{
					{APIVersion: "b", Kind: "b"}: nil,
				},
				GenerateResourceFn: GenerateResourceFnNop,
			},
			errExpected: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := newInventory(tc.input)

			if tc.errExpected {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
