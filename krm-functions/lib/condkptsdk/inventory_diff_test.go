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

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func TestDiffWithSameSpec(t *testing.T) {
	inv, err := newInventory(&Config{
		For:                corev1.ObjectReference{APIVersion: "a", Kind: "a"},
		GenerateResourceFn: GenerateResourceFnNop,
	})
	if err != nil {
		assert.NoError(t, err)
	}
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
		// set for exisiting resource
		inv.set(&gvkKindCtx{gvkKind: forGVKKind}, []corev1.ObjectReference{
			{APIVersion: "a", Kind: "a", Name: "a"},
		}, kubeObjectMade, false)
		// set own existing resource
		inv.set(&gvkKindCtx{gvkKind: ownGVKKind}, []corev1.ObjectReference{
			{APIVersion: "a", Kind: "a", Name: "a"},
			{APIVersion: tt.input.apiVersion, Kind: tt.input.kind, Name: tt.input.name},
		}, kubeObjectMade, false)
		// set own new resource
		inv.set(&gvkKindCtx{gvkKind: ownGVKKind}, []corev1.ObjectReference{
			{APIVersion: "a", Kind: "a", Name: "a"},
			{APIVersion: tt.input.apiVersion, Kind: tt.input.kind, Name: tt.input.name},
		}, kubeObjectMade, true)

	}
	diffList, _ := inv.diff()
	if len(diffList) == 0 {
		t.Error("expected a diff")
	}
	for _, diff := range diffList {
		if len(diff.createObjs) != 0 {
			t.Errorf("TestDiffWithSpecToDelete: -want 0, +got:\n%v", len(diff.createObjs))
		}
		if len(diff.deleteObjs) != 0 {
			t.Errorf("TestDiffWithSpecToDelete: -want 0, +got:\n%v", len(diff.deleteObjs))
		}
		if len(diff.updateObjs) != 0 {
			t.Errorf("TestDiffWithSpecToUpdate: -want 0, +got:\n%v", len(diff.updateObjs))
		}
	}
}

func TestDiffWithSpecToUpdate(t *testing.T) {
	inv, err := newInventory(&Config{
		For:                corev1.ObjectReference{APIVersion: "a", Kind: "a"},
		GenerateResourceFn: GenerateResourceFnNop,
	})
	if err != nil {
		assert.NoError(t, err)
	}
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
		// set for exisiting resource
		inv.set(&gvkKindCtx{gvkKind: forGVKKind}, []corev1.ObjectReference{
			{APIVersion: "a", Kind: "a", Name: "a"},
		}, kubeObjectMade, false)
		// set own exisiting resource
		inv.set(&gvkKindCtx{gvkKind: ownGVKKind}, []corev1.ObjectReference{
			{APIVersion: "a", Kind: "a", Name: "a"},
			{APIVersion: tt.input.apiVersion, Kind: tt.input.kind, Name: tt.input.name},
		}, kubeObjectMade, false)

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
		// set own new resource
		inv.set(&gvkKindCtx{gvkKind: ownGVKKind}, []corev1.ObjectReference{
			{APIVersion: "a", Kind: "a", Name: "a"},
			{APIVersion: tt.input.apiVersion, Kind: tt.input.kind, Name: tt.input.name},
		}, kubeObjectMade, true)
	}
	diffList, _ := inv.diff()
	if len(diffList) == 0 {
		t.Error("expected a diff")
	}
	for _, diff := range diffList {
		if len(diff.createObjs) != 0 {
			t.Errorf("TestDiffWithSpecToDelete: -want 0, +got:\n%v", len(diff.createObjs))
		}
		if len(diff.deleteObjs) != 0 {
			t.Errorf("TestDiffWithSpecToDelete: -want 0, +got:\n%v", len(diff.deleteObjs))
		}
		if len(diff.updateObjs) != 2 {
			t.Errorf("TestDiffWithSpecToUpdate: -want 2, +got:\n%v", len(diff.updateObjs))
		}
	}
}

func TestDiffWithSpecToAdd(t *testing.T) {
	inv, err := newInventory(&Config{
		For:                corev1.ObjectReference{APIVersion: "a", Kind: "a"},
		GenerateResourceFn: GenerateResourceFnNop,
	})
	if err != nil {
		assert.NoError(t, err)
	}
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
		// set for exisiting resource
		inv.set(&gvkKindCtx{gvkKind: forGVKKind}, []corev1.ObjectReference{
			{APIVersion: "a", Kind: "a", Name: "a"},
		}, kubeObjectMade, false)
		// set own exisiting resource
		byteStream, _ = yaml.Marshal(ipa)
		kubeObjectMade, _ = fn.ParseKubeObject(byteStream)
		// set own new resource
		inv.set(&gvkKindCtx{gvkKind: ownGVKKind}, []corev1.ObjectReference{
			{APIVersion: "a", Kind: "a", Name: "a"},
			{APIVersion: tt.input.apiVersion, Kind: tt.input.kind, Name: tt.input.name},
		}, kubeObjectMade, true)
	}
	diffList, _ := inv.diff()
	if len(diffList) == 0 {
		t.Error("expected a diff")
	}
	for _, diff := range diffList {
		if len(diff.createObjs) != 2 {
			t.Errorf("TestDiffWithSpecToDelete: -want 2, +got:\n%v", len(diff.createObjs))
		}
		if len(diff.deleteObjs) != 0 {
			t.Errorf("TestDiffWithSpecToDelete: -want 0, +got:\n%v", len(diff.deleteObjs))
		}
		if len(diff.updateObjs) != 0 {
			t.Errorf("TestDiffWithSpecToUpdate: -want 0, +got:\n%v", len(diff.updateObjs))
		}
	}
}

func TestDiffWithSpecToDelete(t *testing.T) {
	inv, err := newInventory(&Config{
		For:                corev1.ObjectReference{APIVersion: "a", Kind: "a"},
		GenerateResourceFn: GenerateResourceFnNop,
	})
	if err != nil {
		assert.NoError(t, err)
	}
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
		// set for exisiting resource
		inv.set(&gvkKindCtx{gvkKind: forGVKKind}, []corev1.ObjectReference{
			{APIVersion: "a", Kind: "a", Name: "a"},
		}, kubeObjectMade, false)
		byteStream, _ = yaml.Marshal(ipa)
		kubeObjectMade, _ = fn.ParseKubeObject(byteStream)
		// set own new resource
		inv.set(&gvkKindCtx{gvkKind: ownGVKKind}, []corev1.ObjectReference{
			{APIVersion: "a", Kind: "a", Name: "a"},
			{APIVersion: tt.input.apiVersion, Kind: tt.input.kind, Name: tt.input.name},
		}, kubeObjectMade, false)
	}
	diffList, _ := inv.diff()
	if len(diffList) == 0 {
		t.Error("expected a diff")
	}
	for _, diff := range diffList {
		if len(diff.createObjs) != 0 {
			t.Errorf("TestDiffWithSpecToDelete: -want 0, +got:\n%v", len(diff.createObjs))
		}
		if len(diff.deleteObjs) != 2 {
			t.Errorf("TestDiffWithSpecToDelete: -want 2, +got:\n%v", len(diff.deleteObjs))
		}
		if len(diff.updateObjs) != 0 {
			t.Errorf("TestDiffWithSpecToUpdate: -want 0, +got:\n%v", len(diff.updateObjs))
		}
	}
}

/*
func TestDiffWithSpecToAddCondition(t *testing.T) {
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
*/
