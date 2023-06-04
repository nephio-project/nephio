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

package kptrl

import (
	"errors"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

var resList = []byte(`
apiVersion: config.kubernetes.io/v1
kind: ResourceList
items:
- apiVersion: a.a/v1
  kind: A
  metadata:
    name: a
- apiVersion: b.b/v1
  kind: B
  metadata:
    name: b
`)

var objA = []byte(`
apiVersion: a.a/v1
kind: A
metadata:
  name: a
  labels:
    a: a
`)

var objB = []byte(`
apiVersion: b.b/v1
kind: B
metadata:
  name: b
  labels:
    b: b
`)

var objC = []byte(`
apiVersion: c.c/v1
kind: C
metadata:
  name: c
  labels:
    c: c
`)

func TestAddResults(t *testing.T) {

	rl, err := fn.ParseResourceList(resList)
	if err != nil {
		t.Errorf("cannot parse resourceList: %s", err.Error())
	}
	r := &ResourceList{
		*rl,
	}

	cases := map[string]struct {
	}{
		"AddResult": {},
	}

	for name := range cases {
		t.Run(name, func(t *testing.T) {
			tObj, err := fn.ParseKubeObject(objA)
			if err != nil {
				t.Errorf("cannot parse test obj: %s", err.Error())
			}

			r.AddResult(errors.New("dummy error"), tObj)
			results := r.GetResults()
			found := false
			for _, result := range results {
				if strings.Contains(result.Error(), "dummy error") {
					found = true
				}
			}
			if !found {
				t.Errorf("TestAddResults: result not found:\n")
			}

		})
	}
}

func TestGetObject(t *testing.T) {
	rl, err := fn.ParseResourceList(resList)
	if err != nil {
		t.Errorf("cannot parse resourceList: %s", err.Error())
	}
	r := &ResourceList{
		*rl,
	}

	cases := map[string]struct {
		t    []byte
		want *corev1.ObjectReference
	}{
		"GetObjectExist": {
			t: objA,
			want: &corev1.ObjectReference{
				APIVersion: "a.a/v1",
				Kind:       "A",
				Name:       "a",
			},
		},
		"GetObjectNotExist": {
			t:    objC,
			want: nil,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tObj, err := fn.ParseKubeObject(tc.t)
			if err != nil {
				t.Errorf("cannot parse test obj: %s", err.Error())
			}
			got := r.GetObjects(tObj)

			if got == nil {
				if tc.want != nil {
					t.Errorf("TestGetObject: -want: %v, +got:%v\n", tc.want, got)
				}
			} else {
				objRef := &corev1.ObjectReference{
					APIVersion: got[0].GetAPIVersion(),
					Kind:       got[0].GetKind(),
					Name:       got[0].GetName(),
				}
				if diff := cmp.Diff(tc.want, objRef); diff != "" {
					t.Errorf("TestParseObjectKind: -want, +got:\n%s", diff)
				}
			}

		})
	}
}

func TestGetObjects(t *testing.T) {
	rl, err := fn.ParseResourceList(resList)
	if err != nil {
		t.Errorf("cannot parse resourceList: %s", err.Error())
	}
	r := &ResourceList{
		*rl,
	}

	cases := map[string]struct {
		wantLen         int
		wantAPIVersions []string
	}{
		"GetObjects": {
			wantLen: 2,
			wantAPIVersions: []string{
				"a.a/v1",
				"b.b/v1",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			objs := r.GetAllObjects()

			if len(objs) != tc.wantLen {
				t.Errorf("TestGetObjects: -want %d, +got: %d\n", tc.wantLen, len(objs))
			}

			for i, o := range objs {
				if o.GetAPIVersion() != tc.wantAPIVersions[i] {
					t.Errorf("TestGetObjects: -want %s, +got: %s\n", o.GetAPIVersion(), tc.wantAPIVersions[i])
				}
			}
		})
	}
}

func TestSetObject(t *testing.T) {
	rl, err := fn.ParseResourceList(resList)
	if err != nil {
		t.Errorf("cannot parse resourceList: %s", err.Error())
	}
	r := &ResourceList{
		*rl,
	}

	cases := map[string]struct {
		t    []byte
		want *corev1.ObjectReference
	}{
		"SetObjectNew": {
			t: objC,
		},
		"SetObjectOverwrite": {
			t: objA,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tObj, err := fn.ParseKubeObject(tc.t)
			if err != nil {
				t.Errorf("cannot parse test obj: %s", err.Error())
			}

			r.SetObject(tObj)
			got := r.GetObjects(tObj)

			if got == nil {
				t.Errorf("TestGetObject: -want: %v, +got:%v\n", tObj, got)
			} else {
				if diff := cmp.Diff(tObj.GetLabels(), got[0].GetLabels()); diff != "" {
					t.Errorf("TestParseObjectKind: -want, +got:\n%s", diff)
				}
			}
		})
	}
}

func TestDeleteObject(t *testing.T) {
	rl, err := fn.ParseResourceList(resList)
	if err != nil {
		t.Errorf("cannot parse resourceList: %s", err.Error())
	}
	r := &ResourceList{
		*rl,
	}

	cases := map[string]struct {
		t    []byte
		want *corev1.ObjectReference
	}{
		"DeleteObj": {
			t: objB,
		},
		"DeleteNonExistingObject": {
			t: objC,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			tObj, err := fn.ParseKubeObject(tc.t)
			if err != nil {
				t.Errorf("cannot parse test obj: %s", err.Error())
			}

			r.DeleteObject(tObj)
			got := r.GetObjects(tObj)

			if got != nil {
				t.Errorf("TestDeleteObject: -want: nil, +got:%v\n", got)
			}
		})
	}
}

func TestGetResourceList(t *testing.T) {
	cases := map[string]struct {
		t    map[string]string
		want fn.ResourceList
	}{
		"Normal": {
			t: map[string]string{
				"a.yaml": string(objA),
				"b.yaml": string(objB),
				"c.yaml": string(objC)},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			rl, err := GetResourceList(tc.t)
			if err != nil {
				assert.NoError(t, err)
			}
			if len(rl.Items) != len(tc.t) {
				t.Errorf("ResouceList: -want: nil, +got:%v\n", rl.Items)
			}
		})
	}
}
