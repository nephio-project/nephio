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
		deploymentKubeObject, _ := NewFromGoStruct[appsv1.Deployment](deploymentReceived)

		s, _, err := deploymentKubeObject.NestedString([]string{"metadata", "name"}...)
		if err != nil {
			t.Errorf("unexpected error: %v\n", err)
		}
		if deploymentReceived.Name != s {
			t.Errorf("-want%s, +got:\n%s", deploymentReceived.Name, s)
		}
	}

	// test with nil input
	_, err := NewFromGoStruct[v1.Deployment](nil)
	if err == nil {
		t.Errorf("NewFromGoStruct(nil) doesn't return with an error")
	}
}

func compareKubeObjectWithExpectedYaml(t *testing.T, obj *fn.KubeObject, inputFile string) {
	actualYAML := strings.TrimSpace(obj.String())
	expectedFile := testlib.InsertBeforeExtension(inputFile, "_expected")
	expectedYAML := strings.TrimSpace(string(testhelpers.MustReadFile(t, expectedFile)))

	if actualYAML != expectedYAML {
		t.Errorf(`mismatch in expected and actual KubeObject YAML:
--- want: -----
%v
--- got: ----
%v
----------------`, expectedYAML, actualYAML)
		os.WriteFile(testlib.InsertBeforeExtension(inputFile, "_actual"), []byte(actualYAML), 0666)
	}

}

func TestSetNestedFieldKeepFormatting(t *testing.T) {
	testfiles := []string{"testdata/comments.yaml"}
	for _, inputFile := range testfiles {
		t.Run(inputFile, func(t *testing.T) {
			obj := testlib.MustParseKubeObject(t, inputFile)

			deploy, err := ToStruct[appsv1.Deployment](obj)
			if err != nil {
				t.Errorf("unexpected error in ToStruct[v1.Deployment]: %v", err)
			}
			deploy.Spec.Replicas = nil                                              // delete Replicas field if present
			deploy.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure // update field value
			err = SetNestedFieldKeepFormatting(&obj.SubObject, deploy.Spec, "spec")
			if err != nil {
				t.Errorf("unexpected error in SetNestedFieldKeepFormatting: %v", err)
			}

			compareKubeObjectWithExpectedYaml(t, obj, inputFile)
		})
	}
}

func TestSetSpec(t *testing.T) {
	testfiles := []string{"testdata/comments.yaml"}
	for _, inputFile := range testfiles {
		t.Run(inputFile, func(t *testing.T) {
			obj := testlib.MustParseKubeObject(t, inputFile)

			spec, err := GetSpec[appsv1.DeploymentSpec](obj)
			if err != nil {
				t.Errorf("unexpected error in GetSpec: %v", err)
			}
			spec.Replicas = nil                                              // delete Replicas field if present
			spec.Template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure // update field value
			err = SetSpec(obj, spec)
			if err != nil {
				t.Errorf("unexpected error in SetSpec: %v", err)
			}

			compareKubeObjectWithExpectedYaml(t, obj, inputFile)
		})
	}
}

func TestSetStatus(t *testing.T) {
	testfiles := []string{
		"testdata/status_comments.yaml",
		"testdata/empty_status.yaml",
	}
	for _, inputFile := range testfiles {
		t.Run(inputFile, func(t *testing.T) {
			obj := testlib.MustParseKubeObject(t, inputFile)

			status, err := GetStatus[appsv1.DeploymentStatus](obj)
			if err != nil {
				t.Errorf("unexpected error in GetStatus: %v", err)
			}
			status.AvailableReplicas = 0
			err = SetStatus(obj, status)
			if err != nil {
				t.Errorf("unexpected error in SetStatus: %v", err)
			}

			compareKubeObjectWithExpectedYaml(t, obj, inputFile)
		})
	}
}

func TestKubeObjectExtSetNestedFieldKeepFormatting(t *testing.T) {
	testfiles := []string{"testdata/comments.yaml"}
	for _, inputFile := range testfiles {
		t.Run(inputFile, func(t *testing.T) {
			obj := testlib.MustParseKubeObject(t, inputFile)

			koe, err := NewFromKubeObject[appsv1.Deployment](obj)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			deploy, err := koe.GetGoStruct()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			deploy.Spec.Replicas = nil                                              // delete Replicas field if present
			deploy.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure // update field value
			err = koe.SetNestedFieldKeepFormatting(deploy.Spec, "spec")
			if err != nil {
				t.Errorf("unexpected error in SetNestedFieldKeepFormatting: %v", err)
			}

			compareKubeObjectWithExpectedYaml(t, obj, inputFile)
		})
	}
}

func TestKubeObjectExtSetSpec(t *testing.T) {
	testfiles := []string{"testdata/comments.yaml"}
	for _, inputFile := range testfiles {
		t.Run(inputFile, func(t *testing.T) {
			obj := testlib.MustParseKubeObject(t, inputFile)

			koe, err := NewFromKubeObject[appsv1.Deployment](obj)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			deploy, err := koe.GetGoStruct()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			deploy.Spec.Replicas = nil                                              // delete Replicas field if present
			deploy.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure // update field value
			err = koe.SetSpec(&deploy.Spec)
			if err != nil {
				t.Errorf("unexpected error in SetSpec: %v", err)
			}

			compareKubeObjectWithExpectedYaml(t, obj, inputFile)
		})
	}
}

func TestKubeObjectExtSetStatus(t *testing.T) {
	testfiles := []string{
		"testdata/status_comments.yaml",
		"testdata/empty_status.yaml",
	}
	for _, inputFile := range testfiles {
		t.Run(inputFile, func(t *testing.T) {
			obj := testlib.MustParseKubeObject(t, inputFile)

			koe, err := NewFromKubeObject[appsv1.Deployment](obj)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			deploy, err := koe.GetGoStruct()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			deploy.Status.AvailableReplicas = 0
			err = koe.SetStatus(deploy.Status)
			if err != nil {
				t.Errorf("unexpected error in SetStatus: %v", err)
			}

			compareKubeObjectWithExpectedYaml(t, obj, inputFile)
		})
	}
}
