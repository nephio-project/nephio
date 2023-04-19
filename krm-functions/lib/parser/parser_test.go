package parser

import (
	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
	"testing"
)

type objParser struct {
	p Parser[v1.Deployment]
}

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
		deploymentKubeObjectParser := NewFromKubeObject[v1.Deployment](deploymentKubeObject)
		deploymentKubeObjectParserObject := &objParser{
			p: deploymentKubeObjectParser,
		}
		if deploymentKubeObjectParser.GetKubeObject() != deploymentKubeObjectParserObject.p.GetKubeObject() {
			t.Errorf("TestNewFromKubeObject: -want%s, +got:\n%s", deploymentKubeObjectParserObject.p.GetKubeObject(), deploymentKubeObjectParser.GetKubeObject())
		}
		deploymentGoStruct, _ := deploymentKubeObjectParser.GetGoStruct()
		if deploymentGoStruct.Name != deploymentKubeObjectParserObject.p.GetStringValue([]string{"metadata", "name"}...) {
			t.Errorf("TestNewFromKubeObject: -want%s, +got:\n%s", deploymentKubeObjectParserObject.p.GetKubeObject(), deploymentKubeObjectParser.GetKubeObject())
		}
		if deploymentKubeObjectParserObject.p.GetStringValue([]string{"metadata", "name"}...) != tt.input.name {
			t.Errorf("TestNewFromKubeObject: -want:%s, +got:%s", tt.input.name, deploymentKubeObjectParserObject.p.GetStringValue([]string{"metadata", "name"}...))
		}
		if deploymentKubeObjectParserObject.p.GetIntValue([]string{"spec", "replicas"}...) != int(tt.input.replicas) {
			t.Errorf("TestNewFromKubeObject: -want:%v, +got:%v", tt.input.replicas, deploymentKubeObjectParserObject.p.GetIntValue([]string{"spec", "replicas"}...))
		}
		if deploymentKubeObjectParserObject.p.GetBoolValue([]string{"spec", "paused"}...) != tt.input.paused {
			t.Errorf("TestNewFromKubeObject: -want:%t, +got:%t", tt.input.paused, deploymentKubeObjectParserObject.p.GetBoolValue([]string{"spec", "paused"}...))
		}
		if len(deploymentKubeObjectParserObject.p.GetStringMap([]string{"metadata", "name"}...)) != 0 {
			t.Errorf("TestNewFromKubeObject: -want:%v, +got:%v", 0, len(deploymentKubeObjectParserObject.p.GetStringMap([]string{"metadata", "name"}...)))
		}
		if len(deploymentKubeObjectParserObject.p.GetStringMap([]string{"spec", "selector", "matchLabels"}...)) != len(tt.input.selector) {
			t.Errorf("TestNewFromKubeObject: -want:%v, +got:%v", len(tt.input.selector), len(deploymentKubeObjectParserObject.p.GetStringMap([]string{"spec", "selector", "matchLabels"}...)))
		}
		if deploymentKubeObjectParserObject.p.GetStringValue([]string{"metadata", "namespace"}...) != tt.input.namespace {
			t.Errorf("TestNewFromKubeObject: -want:%s, +got:%s", tt.input.namespace, deploymentKubeObjectParserObject.p.GetStringValue([]string{"metadata", "namespace"}...))
		}

		err = deploymentKubeObjectParserObject.p.SetNestedString(tt.input.overwrittenNamespace, []string{"metadata", "namespace"}...)
		if err != nil {
			t.Errorf("SetNestedString error: %s", err)
		}
		if deploymentKubeObjectParserObject.p.GetStringValue([]string{"metadata", "namespace"}...) != tt.input.overwrittenNamespace {
			t.Errorf("TestNewFromKubeObject: -want:%s, +got:%s", tt.input.overwrittenNamespace, deploymentKubeObjectParserObject.p.GetStringValue([]string{"metadata", "namespace"}...))
		}

		err = deploymentKubeObjectParserObject.p.SetNestedInt(int(tt.input.overwrittenReplicas), []string{"spec", "replicas"}...)
		if err != nil {
			t.Errorf("SetNestedInt error: %s", err)
		}
		if deploymentKubeObjectParserObject.p.GetIntValue([]string{"spec", "replicas"}...) != int(tt.input.overwrittenReplicas) {
			t.Errorf("TestNewFromKubeObject: -want:%v, +got:%v", tt.input.overwrittenReplicas, deploymentKubeObjectParserObject.p.GetIntValue([]string{"spec", "replicas"}...))
		}

		err = deploymentKubeObjectParserObject.p.SetNestedBool(tt.input.overwrittenPaused, []string{"spec", "paused"}...)
		if err != nil {
			t.Errorf("SetNestedBool error: %s", err)
		}
		if deploymentKubeObjectParserObject.p.GetBoolValue([]string{"spec", "paused"}...) != tt.input.overwrittenPaused {
			t.Errorf("TestNewFromKubeObject: -want:%t, +got:%t", tt.input.overwrittenPaused, deploymentKubeObjectParserObject.p.GetBoolValue([]string{"spec", "paused"}...))
		}

		err = deploymentKubeObjectParserObject.p.SetNestedMap(tt.input.overwrittenSelector, []string{"spec", "selector", "matchLabels"}...)
		if err != nil {
			t.Errorf("SetNestedBool error: %s", err)
		}
		if len(deploymentKubeObjectParserObject.p.GetStringMap([]string{"spec", "selector", "matchLabels"}...)) != len(tt.input.overwrittenSelector) {
			t.Errorf("TestNewFromKubeObject: -want:%v, +got:%v", 2, len(deploymentKubeObjectParserObject.p.GetStringMap([]string{"spec", "selector", "matchLabels"}...)))
		}

		err = deploymentKubeObjectParserObject.p.DeleteNestedField([]string{"spec", "selector", "matchLabels"}...)
		if err != nil {
			t.Errorf("SetNestedBool error: %s", err)
		}
		if len(deploymentKubeObjectParserObject.p.GetStringMap([]string{"spec", "selector", "matchLabels"}...)) != 0 {
			t.Errorf("TestNewFromKubeObject: -want:%v, +got:%v", 0, len(deploymentKubeObjectParserObject.p.GetStringMap([]string{"spec", "selector", "matchLabels"}...)))
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
		deploymentKubeObjectParserObject := &objParser{
			p: deploymentKubeObjectParser,
		}
		if deploymentKubeObjectParser.GetKubeObject() != deploymentKubeObjectParserObject.p.GetKubeObject() {
			t.Errorf("TestNewFromKubeObject: -want%s, +got:\n%s", deploymentKubeObjectParserObject.p.GetKubeObject(), deploymentKubeObjectParser.GetKubeObject())
		}
		if deploymentKubeObjectParserObject.p.GetStringValue([]string{"metadata", "name"}...) != tt.input.name {
			t.Errorf("TestNewFromKubeObject: -want:%s, +got:%s", tt.input.name, deploymentKubeObjectParserObject.p.GetStringValue([]string{"metadata", "name"}...))
		}
		if deploymentKubeObjectParserObject.p.GetIntValue([]string{"spec", "replicas"}...) != int(tt.input.replicas) {
			t.Errorf("TestNewFromKubeObject: -want:%v, +got:%v", tt.input.replicas, deploymentKubeObjectParserObject.p.GetIntValue([]string{"spec", "replicas"}...))
		}
		if deploymentKubeObjectParserObject.p.GetBoolValue([]string{"spec", "paused"}...) != tt.input.paused {
			t.Errorf("TestNewFromKubeObject: -want:%t, +got:%t", tt.input.paused, deploymentKubeObjectParserObject.p.GetBoolValue([]string{"spec", "paused"}...))
		}
		if len(deploymentKubeObjectParserObject.p.GetStringMap([]string{"metadata", "name"}...)) != 0 {
			t.Errorf("TestNewFromKubeObject: -want:%v, +got:%v", 0, len(deploymentKubeObjectParserObject.p.GetStringMap([]string{"metadata", "name"}...)))
		}
		if len(deploymentKubeObjectParserObject.p.GetStringMap([]string{"spec", "selector", "matchLabels"}...)) != len(tt.input.selector) {
			t.Errorf("TestNewFromKubeObject: -want:%v, +got:%v", len(tt.input.selector), len(deploymentKubeObjectParserObject.p.GetStringMap([]string{"spec", "selector", "matchLabels"}...)))
		}
		if deploymentKubeObjectParserObject.p.GetStringValue([]string{"metadata", "namespace"}...) != tt.input.namespace {
			t.Errorf("TestNewFromKubeObject: -want:%s, +got:%s", tt.input.namespace, deploymentKubeObjectParserObject.p.GetStringValue([]string{"metadata", "namespace"}...))
		}

		err = deploymentKubeObjectParserObject.p.SetNestedString(tt.input.overwrittenNamespace, []string{"metadata", "namespace"}...)
		if err != nil {
			t.Errorf("SetNestedString error: %s", err)
		}
		if deploymentKubeObjectParserObject.p.GetStringValue([]string{"metadata", "namespace"}...) != tt.input.overwrittenNamespace {
			t.Errorf("TestNewFromKubeObject: -want:%s, +got:%s", tt.input.overwrittenNamespace, deploymentKubeObjectParserObject.p.GetStringValue([]string{"metadata", "namespace"}...))
		}

		err = deploymentKubeObjectParserObject.p.SetNestedInt(int(tt.input.overwrittenReplicas), []string{"spec", "replicas"}...)
		if err != nil {
			t.Errorf("SetNestedInt error: %s", err)
		}
		if deploymentKubeObjectParserObject.p.GetIntValue([]string{"spec", "replicas"}...) != int(tt.input.overwrittenReplicas) {
			t.Errorf("TestNewFromKubeObject: -want:%v, +got:%v", tt.input.overwrittenReplicas, deploymentKubeObjectParserObject.p.GetIntValue([]string{"spec", "replicas"}...))
		}

		err = deploymentKubeObjectParserObject.p.SetNestedBool(tt.input.overwrittenPaused, []string{"spec", "paused"}...)
		if err != nil {
			t.Errorf("SetNestedBool error: %s", err)
		}
		if deploymentKubeObjectParserObject.p.GetBoolValue([]string{"spec", "paused"}...) != tt.input.overwrittenPaused {
			t.Errorf("TestNewFromKubeObject: -want:%t, +got:%t", tt.input.overwrittenPaused, deploymentKubeObjectParserObject.p.GetBoolValue([]string{"spec", "paused"}...))
		}

		err = deploymentKubeObjectParserObject.p.SetNestedMap(tt.input.overwrittenSelector, []string{"spec", "selector", "matchLabels"}...)
		if err != nil {
			t.Errorf("SetNestedBool error: %s", err)
		}
		if len(deploymentKubeObjectParserObject.p.GetStringMap([]string{"spec", "selector", "matchLabels"}...)) != len(tt.input.overwrittenSelector) {
			t.Errorf("TestNewFromKubeObject: -want:%v, +got:%v", 2, len(deploymentKubeObjectParserObject.p.GetStringMap([]string{"spec", "selector", "matchLabels"}...)))
		}

		err = deploymentKubeObjectParserObject.p.DeleteNestedField([]string{"spec", "selector", "matchLabels"}...)
		if err != nil {
			t.Errorf("SetNestedBool error: %s", err)
		}
		if len(deploymentKubeObjectParserObject.p.GetStringMap([]string{"spec", "selector", "matchLabels"}...)) != 0 {
			t.Errorf("TestNewFromKubeObject: -want:%v, +got:%v", 0, len(deploymentKubeObjectParserObject.p.GetStringMap([]string{"spec", "selector", "matchLabels"}...)))
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
		deploymentKubeObjectParserObject := &objParser{
			p: deploymentKubeObjectParser,
		}
		if deploymentKubeObjectParser.GetKubeObject() != deploymentKubeObjectParserObject.p.GetKubeObject() {
			t.Errorf("TestNewFromKubeObject: -want%s, +got:\n%s", deploymentKubeObjectParserObject.p.GetKubeObject(), deploymentKubeObjectParser.GetKubeObject())
		}
		if deploymentKubeObjectParserObject.p.GetStringValue([]string{"metadata", "name"}...) != tt.input.name {
			t.Errorf("TestNewFromKubeObject: -want:%s, +got:%s", tt.input.name, deploymentKubeObjectParserObject.p.GetStringValue([]string{"metadata", "name"}...))
		}
		if deploymentKubeObjectParserObject.p.GetIntValue([]string{"spec", "replicas"}...) != int(tt.input.replicas) {
			t.Errorf("TestNewFromKubeObject: -want:%v, +got:%v", tt.input.replicas, deploymentKubeObjectParserObject.p.GetIntValue([]string{"spec", "replicas"}...))
		}
		if deploymentKubeObjectParserObject.p.GetBoolValue([]string{"spec", "paused"}...) != tt.input.paused {
			t.Errorf("TestNewFromKubeObject: -want:%t, +got:%t", tt.input.paused, deploymentKubeObjectParserObject.p.GetBoolValue([]string{"spec", "paused"}...))
		}
		if len(deploymentKubeObjectParserObject.p.GetStringMap([]string{"metadata", "name"}...)) != 0 {
			t.Errorf("TestNewFromKubeObject: -want:%v, +got:%v", 0, len(deploymentKubeObjectParserObject.p.GetStringMap([]string{"metadata", "name"}...)))
		}
		if len(deploymentKubeObjectParserObject.p.GetStringMap([]string{"spec", "selector", "matchLabels"}...)) != len(tt.input.selector) {
			t.Errorf("TestNewFromKubeObject: -want:%v, +got:%v", len(tt.input.selector), len(deploymentKubeObjectParserObject.p.GetStringMap([]string{"spec", "selector", "matchLabels"}...)))
		}
		if deploymentKubeObjectParserObject.p.GetStringValue([]string{"metadata", "namespace"}...) != tt.input.namespace {
			t.Errorf("TestNewFromKubeObject: -want:%s, +got:%s", tt.input.namespace, deploymentKubeObjectParserObject.p.GetStringValue([]string{"metadata", "namespace"}...))
		}

		err := deploymentKubeObjectParserObject.p.SetNestedString(tt.input.overwrittenNamespace, []string{"metadata", "namespace"}...)
		if err != nil {
			t.Errorf("SetNestedString error: %s", err)
		}
		if deploymentKubeObjectParserObject.p.GetStringValue([]string{"metadata", "namespace"}...) != tt.input.overwrittenNamespace {
			t.Errorf("TestNewFromKubeObject: -want:%s, +got:%s", tt.input.overwrittenNamespace, deploymentKubeObjectParserObject.p.GetStringValue([]string{"metadata", "namespace"}...))
		}

		err = deploymentKubeObjectParserObject.p.SetNestedInt(int(tt.input.overwrittenReplicas), []string{"spec", "replicas"}...)
		if err != nil {
			t.Errorf("SetNestedInt error: %s", err)
		}
		if deploymentKubeObjectParserObject.p.GetIntValue([]string{"spec", "replicas"}...) != int(tt.input.overwrittenReplicas) {
			t.Errorf("TestNewFromKubeObject: -want:%v, +got:%v", tt.input.overwrittenReplicas, deploymentKubeObjectParserObject.p.GetIntValue([]string{"spec", "replicas"}...))
		}

		err = deploymentKubeObjectParserObject.p.SetNestedBool(tt.input.overwrittenPaused, []string{"spec", "paused"}...)
		if err != nil {
			t.Errorf("SetNestedBool error: %s", err)
		}
		if deploymentKubeObjectParserObject.p.GetBoolValue([]string{"spec", "paused"}...) != tt.input.overwrittenPaused {
			t.Errorf("TestNewFromKubeObject: -want:%t, +got:%t", tt.input.overwrittenPaused, deploymentKubeObjectParserObject.p.GetBoolValue([]string{"spec", "paused"}...))
		}

		err = deploymentKubeObjectParserObject.p.SetNestedMap(tt.input.overwrittenSelector, []string{"spec", "selector", "matchLabels"}...)
		if err != nil {
			t.Errorf("SetNestedBool error: %s", err)
		}
		if len(deploymentKubeObjectParserObject.p.GetStringMap([]string{"spec", "selector", "matchLabels"}...)) != len(tt.input.overwrittenSelector) {
			t.Errorf("TestNewFromKubeObject: -want:%v, +got:%v", 2, len(deploymentKubeObjectParserObject.p.GetStringMap([]string{"spec", "selector", "matchLabels"}...)))
		}

		err = deploymentKubeObjectParserObject.p.DeleteNestedField([]string{"spec", "selector", "matchLabels"}...)
		if err != nil {
			t.Errorf("SetNestedBool error: %s", err)
		}
		if len(deploymentKubeObjectParserObject.p.GetStringMap([]string{"spec", "selector", "matchLabels"}...)) != 0 {
			t.Errorf("TestNewFromKubeObject: -want:%v, +got:%v", 0, len(deploymentKubeObjectParserObject.p.GetStringMap([]string{"spec", "selector", "matchLabels"}...)))
		}
	}
}
