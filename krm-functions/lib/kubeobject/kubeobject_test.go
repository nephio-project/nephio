package parser

import (
	"testing"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	v1 "k8s.io/api/apps/v1"
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
		deploymentReceived := v1.Deployment{
			TypeMeta: metav1.TypeMeta{
				APIVersion: tt.input.gv,
				Kind:       tt.input.kind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      tt.input.name,
				Namespace: tt.input.namespace,
			},
			Spec: v1.DeploymentSpec{
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
		deploymentKubeObjectParser := NewFromKubeObject[v1.Deployment](*deploymentKubeObject)
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
		deploymentReceived := v1.Deployment{
			TypeMeta: metav1.TypeMeta{
				APIVersion: tt.input.gv,
				Kind:       tt.input.kind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      tt.input.name,
				Namespace: tt.input.namespace,
			},
			Spec: v1.DeploymentSpec{
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
		deploymentKubeObjectParser, _ := NewFromYaml[v1.Deployment](b)
		
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
		deploymentReceived := v1.Deployment{
			TypeMeta: metav1.TypeMeta{
				APIVersion: tt.input.gv,
				Kind:       tt.input.kind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      tt.input.name,
				Namespace: tt.input.namespace,
			},
			Spec: v1.DeploymentSpec{
				Replicas: &tt.input.replicas,
				Paused:   tt.input.paused,
				Selector: &metav1.LabelSelector{
					MatchLabels: tt.input.selector,
				},
			},
		}
		deploymentKubeObjectParser, _ := NewFromGoStruct[v1.Deployment](deploymentReceived)
		
		s, _, err := deploymentKubeObjectParser.NestedString([]string{"metadata", "name"}...)
		if err != nil {
			t.Errorf("unexpected error: %v\n", err)
		}
		if deploymentReceived.Name != s {
			t.Errorf("-want%s, +got:\n%s", deploymentReceived.Name, s)
		}
	}
}
