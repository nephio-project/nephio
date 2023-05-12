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

package kubeobject

import (
	"testing"

	tlib "github.com/nephio-project/nephio/krm-functions/lib/test"
	appsv1 "k8s.io/api/apps/v1"
)

func TestFilterByType(t *testing.T) {
	_ = appsv1.AddToScheme(TheScheme)

	objs := tlib.MustParseKubeObjects(t, "testdata/lists/resources.yaml")

	deploys, rest, err := FilterByType[appsv1.Deployment](objs)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}
	if len(deploys) != 2 {
		t.Errorf("wrong number of Deployments were found in the list: got %v, expected 2", len(deploys))
	}
	if len(rest) != 1 {
		t.Errorf("wrong number of KubeObjects were left in the list: got %v, expected 1", len(rest))
	}
}

func TestGetSingleton(t *testing.T) {
	_ = appsv1.AddToScheme(TheScheme)

	objs := tlib.MustParseKubeObjects(t, "testdata/lists/resources.yaml")

	ds, err := GetSingleton[appsv1.DaemonSet](objs)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if ds == nil {
		t.Errorf("singleton wasn't found")
	}

	_, err = GetSingleton[appsv1.Deployment](objs)
	if err == nil {
		t.Errorf("GetSingleton should return with an error if multiple objects of the given type exists, but it hasn't")
	}

}
