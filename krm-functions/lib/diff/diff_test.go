package diff

import (
	"testing"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func TestDiffWithSameSpec(t *testing.T) {
	inventory := New()
	type object struct {
		apiVersion string
		kind       string
		name       string
		dummy      string
		spec       string
	}

	test1 := []struct {
		input object
	}{
		{
			input: object{
				apiVersion: "a",
				kind:       "b",
				name:       "c",
				spec:       "test1",
				dummy:      "samsung",
			},
		},
		{
			input: object{
				apiVersion: "d",
				kind:       "e",
				name:       "f",
				spec:       "test2",
				dummy:      "samsung",
			},
		},
	}
	for _, tt := range test1 {
		ipa := &nadv1.NetworkAttachmentDefinition{
			TypeMeta: metav1.TypeMeta{
				APIVersion: tt.input.apiVersion,
				Kind:       tt.input.kind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: tt.input.name,
			},
			Spec: nadv1.NetworkAttachmentDefinitionSpec{
				Config: tt.input.spec,
			},
		}
		byteStream, _ := yaml.Marshal(ipa)
		kubeObjectMade, _ := fn.ParseKubeObject(byteStream)
		inventory.AddExistingResource(&corev1.ObjectReference{
			APIVersion:      tt.input.apiVersion,
			Kind:            tt.input.kind,
			Name:            tt.input.name,
			Namespace:       tt.input.dummy,
			FieldPath:       tt.input.dummy,
			ResourceVersion: tt.input.apiVersion,
		}, kubeObjectMade)

		inventory.AddNewResource(&corev1.ObjectReference{
			APIVersion:      tt.input.apiVersion,
			Kind:            tt.input.kind,
			Name:            tt.input.name,
			Namespace:       tt.input.dummy,
			FieldPath:       tt.input.dummy,
			ResourceVersion: tt.input.apiVersion,
		}, kubeObjectMade)
	}
	diffList, _ := inventory.Diff()
	if len(diffList.CreateObjs) != 0 {
		t.Errorf("TestGetCondition: -want nothing, +got:\n%v", diffList.CreateObjs)
	}
	if len(diffList.DeleteObjs) != 0 {
		t.Errorf("TestGetCondition: -want nothing, +got:\n%v", diffList.DeleteObjs)
	}
	if len(diffList.UpdateObjs) != 0 {
		t.Errorf("TestGetCondition: -want nothing, +got:\n%v", diffList.UpdateObjs)
	}
}

func TestDiffWithSpecToDelete(t *testing.T) {
	inventory := New()
	type object struct {
		apiVersion string
		kind       string
		name       string
		dummy      string
		spec       string
	}

	test1 := []struct {
		input object
	}{
		{
			input: object{
				apiVersion: "a",
				kind:       "b",
				name:       "c",
				dummy:      "samsung",
			},
		},
	}
	for _, tt := range test1 {
		currentGVKN := &corev1.ObjectReference{
			APIVersion:      tt.input.apiVersion,
			Kind:            tt.input.kind,
			Name:            tt.input.name,
			Namespace:       tt.input.dummy,
			FieldPath:       tt.input.dummy,
			ResourceVersion: tt.input.apiVersion,
		}
		ipa := &nadv1.NetworkAttachmentDefinition{
			TypeMeta: metav1.TypeMeta{
				APIVersion: tt.input.apiVersion,
				Kind:       tt.input.kind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: tt.input.name,
			},
			Spec: nadv1.NetworkAttachmentDefinitionSpec{
				Config: "type1",
			},
		}
		byteStream, _ := yaml.Marshal(ipa)
		kubeObjectMade, _ := fn.ParseKubeObject(byteStream)
		inventory.AddExistingResource(currentGVKN, kubeObjectMade)

		ipa = &nadv1.NetworkAttachmentDefinition{
			TypeMeta: metav1.TypeMeta{
				APIVersion: tt.input.apiVersion,
				Kind:       tt.input.kind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: tt.input.name,
			},
			Spec: nadv1.NetworkAttachmentDefinitionSpec{
				Config: "type2",
			},
		}
		byteStream, _ = yaml.Marshal(ipa)
		kubeObjectMade, _ = fn.ParseKubeObject(byteStream)
		//inventory.AddNewResource(currentGVKN, kubeObjectMade)
	}
	diffList, _ := inventory.Diff()
	if len(diffList.CreateObjs) != 0 {
		t.Errorf("TestGetCondition: -want nothing, +got:\n%v", diffList.CreateObjs)
	}
	if len(diffList.DeleteObjs) != 1 {
		t.Errorf("TestGetCondition: -want nothing, +got:\n%v", diffList.DeleteObjs)
	}
	if len(diffList.UpdateObjs) != 0 {
		t.Errorf("TestGetCondition: -want nothing, +got:\n%v", diffList.UpdateObjs)
	}
}

// TODO: need to update with more conditions
func TestDiffWithSpecToAdd(t *testing.T) {
	inventory := New()
	type object struct {
		apiVersion string
		kind       string
		name       string
		dummy      string
		spec       string
	}

	test1 := []struct {
		input object
	}{
		{
			input: object{
				apiVersion: "a",
				kind:       "b",
				name:       "c",
				dummy:      "samsung",
			},
		},
	}
	for _, tt := range test1 {
		currentGVKN := &corev1.ObjectReference{
			APIVersion:      tt.input.apiVersion,
			Kind:            tt.input.kind,
			Name:            tt.input.name,
			Namespace:       tt.input.dummy,
			FieldPath:       tt.input.dummy,
			ResourceVersion: tt.input.apiVersion,
		}
		ipa := &nadv1.NetworkAttachmentDefinition{
			TypeMeta: metav1.TypeMeta{
				APIVersion: tt.input.apiVersion,
				Kind:       tt.input.kind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: tt.input.name,
			},
			Spec: nadv1.NetworkAttachmentDefinitionSpec{
				Config: "type1",
			},
		}
		byteStream, _ := yaml.Marshal(ipa)
		kubeObjectMade, _ := fn.ParseKubeObject(byteStream)
		//inventory.AddExistingResource(currentGVKN, kubeObjectMade)

		ipa = &nadv1.NetworkAttachmentDefinition{
			TypeMeta: metav1.TypeMeta{
				APIVersion: tt.input.apiVersion,
				Kind:       tt.input.kind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: tt.input.name,
			},
			Spec: nadv1.NetworkAttachmentDefinitionSpec{
				Config: "type2",
			},
		}
		byteStream, _ = yaml.Marshal(ipa)
		kubeObjectMade, _ = fn.ParseKubeObject(byteStream)
		inventory.AddNewResource(currentGVKN, kubeObjectMade)
	}
	diffList, _ := inventory.Diff()
	if len(diffList.CreateObjs) != 1 {
		t.Errorf("TestGetCondition: -want nothing, +got:\n%v", diffList.CreateObjs)
	}
	if len(diffList.DeleteObjs) != 0 {
		t.Errorf("TestGetCondition: -want nothing, +got:\n%v", diffList.DeleteObjs)
	}
	if len(diffList.UpdateObjs) != 0 {
		t.Errorf("TestGetCondition: -want nothing, +got:\n%v", diffList.UpdateObjs)
	}
}

// TODO: need to update with more conditions
func TestDiffWithSpecToUpdate(t *testing.T) {
	inventory := New()
	type object struct {
		apiVersion string
		kind       string
		name       string
		dummy      string
		spec       string
	}

	test1 := []struct {
		input object
	}{
		{
			input: object{
				apiVersion: "a",
				kind:       "b",
				name:       "c",
				spec:       "test1",
				dummy:      "samsung",
			},
		},
		{
			input: object{
				apiVersion: "d",
				kind:       "e",
				name:       "f",
				spec:       "test2",
				dummy:      "samsung",
			},
		},
	}
	test2 := []struct {
		input object
	}{
		{
			input: object{
				apiVersion: "a",
				kind:       "b",
				name:       "c",
				spec:       "test3",
				dummy:      "samsung",
			},
		},
		{
			input: object{
				apiVersion: "d",
				kind:       "e",
				name:       "f",
				spec:       "test4",
				dummy:      "samsung",
			},
		},
	}
	for _, tt := range test1 {
		ipa := &nadv1.NetworkAttachmentDefinition{
			TypeMeta: metav1.TypeMeta{
				APIVersion: tt.input.apiVersion,
				Kind:       tt.input.kind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: tt.input.name,
			},
			Spec: nadv1.NetworkAttachmentDefinitionSpec{
				Config: tt.input.spec,
			},
		}
		byteStream, _ := yaml.Marshal(ipa)
		kubeObjectMade, _ := fn.ParseKubeObject(byteStream)

		inventory.AddNewResource(&corev1.ObjectReference{
			APIVersion:      tt.input.apiVersion,
			Kind:            tt.input.kind,
			Name:            tt.input.name,
			Namespace:       tt.input.dummy,
			FieldPath:       tt.input.dummy,
			ResourceVersion: tt.input.apiVersion,
		}, kubeObjectMade)
	}
	for _, tt := range test2 {
		ipa := &nadv1.NetworkAttachmentDefinition{
			TypeMeta: metav1.TypeMeta{
				APIVersion: tt.input.apiVersion,
				Kind:       tt.input.kind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: tt.input.name,
			},
			Spec: nadv1.NetworkAttachmentDefinitionSpec{
				Config: tt.input.spec,
			},
		}
		byteStream, _ := yaml.Marshal(ipa)
		kubeObjectMade, _ := fn.ParseKubeObject(byteStream)

		inventory.AddExistingResource(&corev1.ObjectReference{
			APIVersion:      tt.input.apiVersion,
			Kind:            tt.input.kind,
			Name:            tt.input.name,
			Namespace:       tt.input.dummy,
			FieldPath:       tt.input.dummy,
			ResourceVersion: tt.input.apiVersion,
		}, kubeObjectMade)
	}
	diffList, _ := inventory.Diff()
	if len(diffList.CreateObjs) != 0 {
		t.Errorf("TestGetCondition: -want nothing, +got:\n%v", diffList.CreateObjs)
	}
	if len(diffList.DeleteObjs) != 2 {
		t.Errorf("TestGetCondition: -want nothing, +got:\n%v", diffList.DeleteObjs)
	}
	if len(diffList.UpdateObjs) != 0 {
		t.Errorf("TestGetCondition: -want nothing, +got:\n%v", diffList.UpdateObjs)
	}
}
