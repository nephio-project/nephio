package fnruntime

import (
	"testing"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	kptv1 "github.com/GoogleContainerTools/kpt/pkg/api/kptfile/v1"
	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func TestDiffWithSameSpec(t *testing.T) {
	inventory := NewUpstreamInventory()
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
				dummy:      "sample",
			},
		},
		{
			input: object{
				apiVersion: "d",
				kind:       "e",
				name:       "f",
				spec:       "test2",
				dummy:      "sample",
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
		t.Errorf("TestDiffWithSameSpec: -want 0, +got:\n%v", len(diffList.CreateObjs))
	}
	if len(diffList.DeleteObjs) != 0 {
		t.Errorf("TestDiffWithSameSpec: -want 0, +got:\n%v", len(diffList.DeleteObjs))
	}
	if len(diffList.UpdateObjs) != 0 {
		t.Errorf("TestDiffWithSameSpec: -want 0, +got:\n%v", len(diffList.UpdateObjs))
	}
}

func TestDiffWithSpecToDelete(t *testing.T) {
	inventory := NewUpstreamInventory()
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
				dummy:      "sample",
				spec:       "test1",
			},
		},
		{
			input: object{
				apiVersion: "d",
				kind:       "e",
				name:       "f",
				spec:       "test2",
				dummy:      "sample",
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
				Config: tt.input.spec,
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
	}
	diffList, _ := inventory.Diff()
	if len(diffList.CreateObjs) != 0 {
		t.Errorf("TestDiffWithSpecToDelete: -want 0, +got:\n%v", len(diffList.CreateObjs))
	}
	if len(diffList.DeleteObjs) != 2 {
		t.Errorf("TestDiffWithSpecToDelete: -want 2, +got:\n%v", len(diffList.DeleteObjs))
	}
	if len(diffList.UpdateObjs) != 0 {
		t.Errorf("TestDiffWithSpecToDelete: -want 0, +got:\n%v", len(diffList.UpdateObjs))
	}
}

func TestDiffWithSpecToAdd(t *testing.T) {
	inventory := NewUpstreamInventory()
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
				dummy:      "sample",
				spec:       "test1",
			},
		},
		{
			input: object{
				apiVersion: "d",
				kind:       "e",
				name:       "f",
				spec:       "test2",
				dummy:      "sample",
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
				Config: tt.input.spec,
			},
		}
		byteStream, _ := yaml.Marshal(ipa)
		kubeObjectMade, _ := fn.ParseKubeObject(byteStream)

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
	if len(diffList.CreateObjs) != 2 {
		t.Errorf("TestDiffWithSpecToAdd: -want 2, +got:\n%v", len(diffList.CreateObjs))
	}
	if len(diffList.DeleteObjs) != 0 {
		t.Errorf("TestDiffWithSpecToAdd: -want 0, +got:\n%v", len(diffList.DeleteObjs))
	}
	if len(diffList.UpdateObjs) != 0 {
		t.Errorf("TestDiffWithSpecToAdd: -want 0, +got:\n%v", len(diffList.UpdateObjs))
	}
}

func TestDiffWithSpecToUpdate(t *testing.T) {
	inventory := NewUpstreamInventory()
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
				dummy:      "sample",
			},
		},
		{
			input: object{
				apiVersion: "d",
				kind:       "e",
				name:       "f",
				spec:       "test2",
				dummy:      "sample",
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
				dummy:      "sample",
			},
		},
		{
			input: object{
				apiVersion: "d",
				kind:       "e",
				name:       "f",
				spec:       "test4",
				dummy:      "sample",
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
		t.Errorf("TestDiffWithSpecToUpdate: -want 0, +got:\n%v", len(diffList.CreateObjs))
	}
	if len(diffList.DeleteObjs) != 0 {
		t.Errorf("TestDiffWithSpecToUpdate: -want 0, +got:\n%v", len(diffList.DeleteObjs))
	}
	if len(diffList.UpdateObjs) != 2 {
		t.Errorf("TestDiffWithSpecToUpdate: -want 2, +got:\n%v", len(diffList.UpdateObjs))
	}
	if len(diffList.UpdateConditions) != 2 {
		t.Errorf("TestDiffWithSpecToUpdate: -want 2, +got:\n%v", len(diffList.UpdateConditions))
	}
}

func TestDiffWithSpecToAddCondition(t *testing.T) {
	inventory := NewUpstreamInventory()
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
				dummy:      "sample",
				spec:       "test1",
			},
		},
		{
			input: object{
				apiVersion: "d",
				kind:       "e",
				name:       "f",
				spec:       "test2",
				dummy:      "sample",
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
				Config: tt.input.spec,
			},
		}
		byteStream, _ := yaml.Marshal(ipa)
		kubeObjectMade, _ := fn.ParseKubeObject(byteStream)

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
	if len(diffList.CreateConditions) != 2 {
		t.Errorf("TestDiffWithSpecToAddCondition: -want 2, +got:\n%v", len(diffList.CreateConditions))
	}
}

func TestDiffWithSpecToDeleteCondition(t *testing.T) {
	inventory := NewUpstreamInventory()
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
				dummy:      "sample",
				spec:       "test1",
			},
		},
		{
			input: object{
				apiVersion: "d",
				kind:       "e",
				name:       "f",
				spec:       "test2",
				dummy:      "sample",
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
				Config: tt.input.spec,
			},
		}
		byteStream, _ := yaml.Marshal(ipa)
		kubeObjectMade, _ := fn.ParseKubeObject(byteStream)
		inventory.AddExistingCondition(currentGVKN, &kptv1.Condition{
			Type:    currentGVKN.APIVersion + currentGVKN.Kind + currentGVKN.Name,
			Status:  kptv1.ConditionFalse,
			Reason:  tt.input.dummy,
			Message: tt.input.dummy,
		})
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
	}
	diffList, _ := inventory.Diff()
	if len(diffList.DeleteConditions) != 2 {
		t.Errorf("TestDiffWithSpecToDeleteCondition: -want 0, +got:\n%v", len(diffList.DeleteConditions))
	}
}
