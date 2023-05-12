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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn/testhelpers"
	testlib "github.com/nephio-project/nephio/krm-functions/lib/test"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func TestNewFromKubeObject(t *testing.T) {
	type object struct {
		namespace            string
		overwrittenNamespace string
		config               string
		name                 string
		gv                   string
		kind                 string
		replicas             int32
		overwrittenReplicas  int32
		paused               bool
		overwrittenPaused    bool
		selector             map[string]string
		overwrittenSelector  map[string]string
	}
	testItems := []struct {
		input object
	}{
		{
			input: object{
				gv:                   "apps/v1",
				kind:                 "Deployment",
				name:                 "a",
				namespace:            "b",
				overwrittenNamespace: "new",
				config:               "c",
				replicas:             3,
				overwrittenReplicas:  10,
				paused:               true,
				overwrittenPaused:    false,
				selector:             map[string]string{"install": "output"},
				overwrittenSelector:  map[string]string{"install": "large", "network": "CONF"},
			},
		},
		{
			input: object{
				gv:                   "apps/v1",
				kind:                 "Deployment",
				name:                 "d",
				namespace:            "e",
				overwrittenNamespace: "old",
				config:               "f",
				replicas:             10,
				overwrittenReplicas:  3,
				paused:               false,
				overwrittenPaused:    true,
				selector:             map[string]string{"flavor": "large"},
				overwrittenSelector:  map[string]string{"flavor": "large", "network": "VLAN"},
			},
		},
		{
			input: object{
				gv:                   "apps/v1",
				kind:                 "Deployment",
				name:                 "",
				namespace:            "",
				overwrittenNamespace: "",
				config:               "",
				replicas:             0,
				overwrittenReplicas:  0,
				paused:               false,
				overwrittenPaused:    true,
				selector:             map[string]string{"flavor": "large"},
				overwrittenSelector:  map[string]string{"flavor": "large", "network": "VLAN"},
			},
		},
	}
	for _, tt := range testItems {
		deploymentReceived := appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				APIVersion: tt.input.gv,
				Kind:       tt.input.kind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      tt.input.name,
				Namespace: tt.input.namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &tt.input.replicas,
				Paused:   tt.input.paused,
				Selector: &metav1.LabelSelector{
					MatchLabels: tt.input.selector,
				},
			},
		}
		b, err := yaml.Marshal(deploymentReceived)
		if err != nil {
			t.Errorf("YAML Marshal error: %s", err)
		}
		deploymentKubeObject, _ := fn.ParseKubeObject(b)
		deploymentKubeObjectParser, _ := NewFromKubeObject[appsv1.Deployment](deploymentKubeObject)
		if deploymentKubeObjectParser.SubObject != deploymentKubeObject.SubObject {
			t.Errorf("-want%s, +got:\n%s", deploymentKubeObjectParser.String(), deploymentKubeObject.String())
		}
		deploymentGoStruct, _ := deploymentKubeObjectParser.GetGoStruct()
		s, _, err := deploymentKubeObjectParser.NestedString([]string{"metadata", "name"}...)
		if err != nil {
			t.Errorf("unexpected error: %v\n", err)
		}
		if deploymentGoStruct.Name != s {
			t.Errorf("-want%s, +got:\n%s", deploymentGoStruct.Name, s)
		}
	}
}

func TestNewFromYaml(t *testing.T) {
	type object struct {
		namespace            string
		overwrittenNamespace string
		config               string
		name                 string
		gv                   string
		kind                 string
		replicas             int32
		overwrittenReplicas  int32
		paused               bool
		overwrittenPaused    bool
		selector             map[string]string
		overwrittenSelector  map[string]string
	}
	testItems := []struct {
		input object
	}{
		{
			input: object{
				gv:                   "apps/v1",
				kind:                 "Deployment",
				name:                 "a",
				namespace:            "b",
				overwrittenNamespace: "new",
				config:               "c",
				replicas:             3,
				overwrittenReplicas:  10,
				paused:               true,
				overwrittenPaused:    false,
				selector:             map[string]string{"install": "output"},
				overwrittenSelector:  map[string]string{"install": "large", "network": "CONF"},
			},
		},
		{
			input: object{
				gv:                   "apps/v1",
				kind:                 "Deployment",
				name:                 "d",
				namespace:            "e",
				overwrittenNamespace: "old",
				config:               "f",
				replicas:             10,
				overwrittenReplicas:  3,
				paused:               false,
				overwrittenPaused:    true,
				selector:             map[string]string{"flavor": "large"},
				overwrittenSelector:  map[string]string{"flavor": "large", "network": "VLAN"},
			},
		},
		{
			input: object{
				gv:                   "apps/v1",
				kind:                 "Deployment",
				name:                 "",
				namespace:            "",
				overwrittenNamespace: "",
				config:               "",
				replicas:             0,
				overwrittenReplicas:  0,
				paused:               false,
				overwrittenPaused:    true,
				selector:             map[string]string{"flavor": "large"},
				overwrittenSelector:  map[string]string{"flavor": "large", "network": "VLAN"},
			},
		},
	}
	for _, tt := range testItems {
		deploymentReceived := appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				APIVersion: tt.input.gv,
				Kind:       tt.input.kind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      tt.input.name,
				Namespace: tt.input.namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &tt.input.replicas,
				Paused:   tt.input.paused,
				Selector: &metav1.LabelSelector{
					MatchLabels: tt.input.selector,
				},
			},
		}
		b, err := yaml.Marshal(deploymentReceived)
		if err != nil {
			t.Errorf("YAML Marshal error: %s", err)
		}
		deploymentKubeObjectParser, _ := NewFromYaml[appsv1.Deployment](b)

		if deploymentKubeObjectParser.String() != string(b) {
			t.Errorf("-want%s, +got:\n%s", string(b), deploymentKubeObjectParser.String())
		}
		deploymentGoStruct, _ := deploymentKubeObjectParser.GetGoStruct()
		s, _, err := deploymentKubeObjectParser.NestedString([]string{"metadata", "name"}...)
		if err != nil {
			t.Errorf("unexpected error: %v\n", err)
		}
		if deploymentGoStruct.Name != s {
			t.Errorf("-want%s, +got:\n%s", deploymentGoStruct.Name, s)
		}
	}
}

func TestNewFromGoStruct(t *testing.T) {
	type object struct {
		namespace            string
		overwrittenNamespace string
		config               string
		name                 string
		gv                   string
		kind                 string
		replicas             int32
		overwrittenReplicas  int32
		paused               bool
		overwrittenPaused    bool
		selector             map[string]string
		overwrittenSelector  map[string]string
	}
	testItems := []struct {
		input object
	}{
		{
			input: object{
				gv:                   "apps/v1",
				kind:                 "Deployment",
				name:                 "a",
				namespace:            "b",
				overwrittenNamespace: "new",
				config:               "c",
				replicas:             3,
				overwrittenReplicas:  10,
				paused:               true,
				overwrittenPaused:    false,
				selector:             map[string]string{"install": "output"},
				overwrittenSelector:  map[string]string{"install": "large", "network": "CONF"},
			},
		},
		{
			input: object{
				gv:                   "apps/v1",
				kind:                 "Deployment",
				name:                 "d",
				namespace:            "e",
				overwrittenNamespace: "old",
				config:               "f",
				replicas:             10,
				overwrittenReplicas:  3,
				paused:               false,
				overwrittenPaused:    true,
				selector:             map[string]string{"flavor": "large"},
				overwrittenSelector:  map[string]string{"flavor": "large", "network": "VLAN"},
			},
		},
		{
			input: object{
				gv:                   "apps/v1",
				kind:                 "Deployment",
				name:                 "",
				namespace:            "",
				overwrittenNamespace: "",
				config:               "",
				replicas:             0,
				overwrittenReplicas:  0,
				paused:               false,
				overwrittenPaused:    true,
				selector:             map[string]string{"flavor": "large"},
				overwrittenSelector:  map[string]string{"flavor": "large", "network": "VLAN"},
			},
		},
	}
	for _, tt := range testItems {
		deploymentReceived := appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				APIVersion: tt.input.gv,
				Kind:       tt.input.kind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      tt.input.name,
				Namespace: tt.input.namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &tt.input.replicas,
				Paused:   tt.input.paused,
				Selector: &metav1.LabelSelector{
					MatchLabels: tt.input.selector,
				},
			},
		}
		deploymentKubeObject, _ := NewFromGoStruct(&deploymentReceived)

		s, _, err := deploymentKubeObject.NestedString([]string{"metadata", "name"}...)
		if err != nil {
			t.Errorf("unexpected error: %v\n", err)
		}
		if deploymentReceived.Name != s {
			t.Errorf("-want%s, +got:\n%s", deploymentReceived.Name, s)
		}
	}
}

func compareKubeObjectWithExpectedYaml(t *testing.T, obj *fn.KubeObject, expectedFile string) {
	actualYAML := strings.TrimSpace(obj.String())
	expectedYAML := strings.TrimSpace(string(testhelpers.MustReadFile(t, expectedFile)))

	if actualYAML != expectedYAML {
		// assemble the full path of the _actual.yaml file
		ext := filepath.Ext(expectedFile)
		base, _ := strings.CutSuffix(expectedFile, ext)
		base, _ = strings.CutSuffix(base, "_expected")
		actualFile := base + "_actual" + ext
		// write actual output into the file
		os.WriteFile(actualFile, []byte(actualYAML), 0666)
		t.Errorf(`mismatch in expected and actual KubeObject YAML:
  - find expected YAML in %v
  - find actual YAML in   %v`, expectedFile, actualFile)
	}

}

type deploymentTestcase struct {
	inputFile    string
	expectedFile string
	transform    func(*appsv1.Deployment)
}

// various transformations of appsv1.Deployments used in tests

func noop(deploy *appsv1.Deployment) {
}

func setSpecFields(deploy *appsv1.Deployment) {
	deploy.Spec.Strategy.Type = appsv1.RollingUpdateDeploymentStrategyType  // "create new" field
	deploy.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure // update field value
	deploy.Spec.Replicas = nil                                              // "delete" field if present

}

func setStatusFields(deploy *appsv1.Deployment) {
	deploy.Status.AvailableReplicas = 0 // "delete"
	deploy.Status.Replicas = 3          // "update"
}

func setAllFields(deploy *appsv1.Deployment) {
	setSpecFields(deploy)
	setStatusFields(deploy)
}

func changeList(deploy *appsv1.Deployment) {
	deploy.Spec.Template.Spec.Containers = []corev1.Container{
		deploy.Spec.Template.Spec.Containers[1],
		{
			Name:  "injected-by-test",
			Image: "noop:1",
			Ports: []corev1.ContainerPort{
				{
					Name:          "test-port",
					ContainerPort: 8080,
					Protocol:      "tcp",
				},
			},
		},
		deploy.Spec.Template.Spec.Containers[0],
	}
	deploy.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort = 1111

}

var formattingTestDataDir = "testdata/formatting/"

func TestSetNestedFieldKeepFormatting(t *testing.T) {
	testcases := []deploymentTestcase{
		{
			inputFile:    formattingTestDataDir + "deployment_full.yaml",
			expectedFile: formattingTestDataDir + "deployment_full__noop_expected.yaml",
			transform:    noop,
		},
		{
			inputFile:    formattingTestDataDir + "deployment_full.yaml",
			expectedFile: formattingTestDataDir + "deployment_full__change_spec_fields_expected.yaml",
			transform:    setSpecFields,
		},
		{
			inputFile:    formattingTestDataDir + "deployment_no_status.yaml",
			expectedFile: formattingTestDataDir + "deployment_no_status__change_spec_fields_expected.yaml",
			transform:    setSpecFields,
		},
		{
			inputFile:    formattingTestDataDir + "deployment_full.yaml",
			expectedFile: formattingTestDataDir + "deployment_full__change_status_fields_expected.yaml",
			transform:    setStatusFields,
		},
		{
			inputFile:    formattingTestDataDir + "deployment_no_status.yaml",
			expectedFile: formattingTestDataDir + "deployment_no_status__change_status_fields_expected.yaml",
			transform:    setStatusFields,
		},
		{
			inputFile:    formattingTestDataDir + "deployment_full.yaml",
			expectedFile: formattingTestDataDir + "deployment_full__change_all_fields_expected.yaml",
			transform:    setAllFields,
		},
		{
			inputFile:    formattingTestDataDir + "deployment_full.yaml",
			expectedFile: formattingTestDataDir + "deployment_full__change_list_expected.yaml",
			transform:    changeList,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.expectedFile, func(t *testing.T) {
			var err error
			obj := testlib.MustParseKubeObject(t, tc.inputFile)
			var deploy appsv1.Deployment
			if err := obj.As(&deploy); err != nil {
				t.Fatalf("couldn't convert object to Deployment: %v", err)
			}

			tc.transform(&deploy)

			err = setNestedFieldKeepFormatting(obj, deploy.Spec, "spec")
			if err != nil {
				t.Errorf("unexpected error in SetNestedFieldKeepFormatting: %v", err)
			}
			err = setNestedFieldKeepFormatting(obj, deploy.Status, "status")
			if err != nil {
				t.Errorf("unexpected error in SetNestedFieldKeepFormatting: %v", err)
			}

			compareKubeObjectWithExpectedYaml(t, obj, tc.expectedFile)
		})
	}
}

func TestSetFromTypedObject(t *testing.T) {
	testcases := []deploymentTestcase{
		{
			inputFile:    formattingTestDataDir + "deployment_full.yaml",
			expectedFile: formattingTestDataDir + "deployment_full__noop_expected.yaml",
			transform:    noop,
		},
		{
			inputFile:    formattingTestDataDir + "deployment_full.yaml",
			expectedFile: formattingTestDataDir + "deployment_full__change_spec_fields_expected.yaml",
			transform:    setSpecFields,
		},
		{
			inputFile:    formattingTestDataDir + "deployment_no_status.yaml",
			expectedFile: formattingTestDataDir + "deployment_no_status__change_spec_fields_expected.yaml",
			transform:    setSpecFields,
		},
		{
			inputFile:    formattingTestDataDir + "deployment_full.yaml",
			expectedFile: formattingTestDataDir + "deployment_full__change_status_fields_expected.yaml",
			transform:    setStatusFields,
		},
		{
			inputFile:    formattingTestDataDir + "deployment_no_status.yaml",
			expectedFile: formattingTestDataDir + "deployment_no_status__change_status_fields_expected.yaml",
			transform:    setStatusFields,
		},
		{
			inputFile:    formattingTestDataDir + "deployment_full.yaml",
			expectedFile: formattingTestDataDir + "deployment_full__change_all_fields_expected.yaml",
			transform:    setAllFields,
		},
		{
			inputFile:    formattingTestDataDir + "deployment_full.yaml",
			expectedFile: formattingTestDataDir + "deployment_full__change_list_expected.yaml",
			transform:    changeList,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.expectedFile, func(t *testing.T) {
			obj := testlib.MustParseKubeObject(t, tc.inputFile)
			koe, err := NewFromKubeObject[appsv1.Deployment](obj)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			deploy, err := koe.GetGoStruct()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			tc.transform(deploy)

			err = koe.SetFromTypedObject(deploy)
			if err != nil {
				t.Errorf("unexpected error in SetFromTypedObject: %v", err)
			}

			compareKubeObjectWithExpectedYaml(t, &koe.KubeObject, tc.expectedFile)
		})
	}
}

func TestKubeObjectExtSetSpec(t *testing.T) {
	testcases := []deploymentTestcase{
		{
			inputFile:    formattingTestDataDir + "deployment_full.yaml",
			expectedFile: formattingTestDataDir + "setspec__deployment_full__change_spec_fields_expected.yaml",
			transform:    setSpecFields,
		},
		{
			inputFile:    formattingTestDataDir + "deployment_no_status.yaml",
			expectedFile: formattingTestDataDir + "setspec__deployment_no_status__change_spec_fields_expected.yaml",
			transform:    setSpecFields,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.expectedFile, func(t *testing.T) {
			obj := testlib.MustParseKubeObject(t, tc.inputFile)
			koe, err := NewFromKubeObject[appsv1.Deployment](obj)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			deploy, err := koe.GetGoStruct()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			tc.transform(deploy)

			err = koe.SetSpec(deploy)
			if err != nil {
				t.Errorf("unexpected error in SetSpec: %v", err)
			}

			compareKubeObjectWithExpectedYaml(t, &koe.KubeObject, tc.expectedFile)
		})
	}
}

func TestKubeObjectExtSetStatus(t *testing.T) {
	testcases := []deploymentTestcase{
		{
			inputFile:    formattingTestDataDir + "deployment_full.yaml",
			expectedFile: formattingTestDataDir + "setstatus__deployment_full__change_status_fields_expected.yaml",
			transform:    setStatusFields,
		},
		{
			inputFile:    formattingTestDataDir + "deployment_no_status.yaml",
			expectedFile: formattingTestDataDir + "setstatus__deployment_no_status__change_status_fields_expected.yaml",
			transform:    setStatusFields,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.expectedFile, func(t *testing.T) {
			obj := testlib.MustParseKubeObject(t, tc.inputFile)
			koe, err := NewFromKubeObject[appsv1.Deployment](obj)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			deploy, err := koe.GetGoStruct()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			tc.transform(deploy)

			err = koe.SetStatus(deploy)
			if err != nil {
				t.Errorf("unexpected error in SetStatus: %v", err)
			}
			compareKubeObjectWithExpectedYaml(t, &koe.KubeObject, tc.expectedFile)
		})
	}

}

func TestNewFromGoStructWithPointerType(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("NewFromGoStruct did NOT panic")
		}
	}()

	deploy := &appsv1.Deployment{}
	_, _ = NewFromGoStruct(&deploy)
}

func TestNewFromKubeObjectWithPointerType(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("NewFromKubeObject did NOT panic")
		}
	}()

	_, _ = NewFromKubeObject[*appsv1.Deployment](nil)
}

func TestSetSpecWithMissingSpecField(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("SetSpec did NOT panic")
		}
	}()

	type NoSpecOrStatus struct{}
	var val NoSpecOrStatus

	koe, err := NewFromGoStruct(&val)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	_ = koe.SetSpec(&val)
}

func TestSetStatusWithMissingStatusField(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("SetStatus did NOT panic")
		}
	}()

	type NoSpecOrStatus struct{}
	var val NoSpecOrStatus

	koe, err := NewFromGoStruct(&val)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	_ = koe.SetStatus(&val)
}
