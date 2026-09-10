package genericspecializer

import (
	"context"
	"testing"

	porchv1alpha1 "github.com/nephio-project/porch/api/porch/v1alpha1"
	"github.com/kptdev/krm-functions-sdk/go/fn"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	ctrl "sigs.k8s.io/controller-runtime"
	"k8s.io/client-go/tools/record"
)

type mockResourceListProcessor struct {
	err error
}

func (m *mockResourceListProcessor) Process(rl *fn.ResourceList) (bool, error) {
	return false, m.err
}

func TestNilStatusPanicGenericSpecializer(t *testing.T) {
	scheme := runtime.NewScheme()
	porchv1alpha1.AddToScheme(scheme)

	pr := &porchv1alpha1.PackageRevision{
		ObjectMeta: ctrl.ObjectMeta{
			Name:      "test-pr",
			Namespace: "default",
		},
		Status: porchv1alpha1.PackageRevisionStatus{
			Conditions: []porchv1alpha1.Condition{},
		},
	}

	// Kptfile without status
	kptfileYaml := `apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: test-pr
`

	prr := &porchv1alpha1.PackageRevisionResources{
		ObjectMeta: ctrl.ObjectMeta{
			Name:      "test-pr",
			Namespace: "default",
		},
		Spec: porchv1alpha1.PackageRevisionResourcesSpec{
			Resources: map[string]string{
				"Kptfile": kptfileYaml,
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pr, prr).Build()

	r := &reconciler{
		apiReader:   fakeClient,
		porchClient: fakeClient,
		recorder:    record.NewFakeRecorder(100),
	}

	req := ctrl.Request{
		NamespacedName: client.ObjectKey{
			Name:      "test-pr",
			Namespace: "default",
		},
	}

	// This should not panic
	_, err := r.Reconcile(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected nil error, got %v", err)
	}
}
