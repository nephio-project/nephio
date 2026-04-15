package ipamspecializer

import (
	"testing"

	porchv1alpha1 "github.com/nephio-project/porch/api/porch/v1alpha1"
)

func TestSyncDeletedResources(t *testing.T) {
	prr := &porchv1alpha1.PackageRevisionResources{
		Spec: porchv1alpha1.PackageRevisionResourcesSpec{
			Resources: map[string]string{
				"file1.yaml": "content1",
				"file2.yaml": "content2",
				"file3.yaml": "content3",
			},
		},
	}

	originalPaths := map[string]struct{}{
		"file1.yaml": {},
		"file2.yaml": {},
		"file3.yaml": {},
	}

	// file2 was deleted by KRM function
	survivingPaths := map[string]struct{}{
		"file1.yaml": {},
		"file3.yaml": {},
	}

	syncDeletedResources(prr, originalPaths, survivingPaths)

	if _, exists := prr.Spec.Resources["file2.yaml"]; exists {
		t.Errorf("Expected file2.yaml to be deleted, but it still exists")
	}
	if _, exists := prr.Spec.Resources["file1.yaml"]; !exists {
		t.Errorf("Expected file1.yaml to be retained")
	}
	if _, exists := prr.Spec.Resources["file3.yaml"]; !exists {
		t.Errorf("Expected file3.yaml to be retained")
	}
}
