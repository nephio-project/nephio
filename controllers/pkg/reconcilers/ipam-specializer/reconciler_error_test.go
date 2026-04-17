package ipamspecializer

import (
	"context"
	"errors"
	"testing"

	porchv1alpha1 "github.com/nephio-project/porch/api/porch/v1alpha1"
	"github.com/kptdev/krm-functions-sdk/go/fn"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	ctrl "sigs.k8s.io/controller-runtime"
	kptfilelibv1 "github.com/nephio-project/nephio/krm-functions/lib/kptfile/v1"
	ipamv1alpha1 "github.com/nokia/k8s-ipam/apis/resource/ipam/v1alpha1"
)

type mockResourceListProcessor struct {
	err error
}

func (m *mockResourceListProcessor) Process(rl *fn.ResourceList) (bool, error) {
	return false, m.err
}

func TestReconcileKrmFnError(t *testing.T) {
	scheme := runtime.NewScheme()
	porchv1alpha1.AddToScheme(scheme)

	pr := &porchv1alpha1.PackageRevision{
		ObjectMeta: ctrl.ObjectMeta{
			Name:      "test-pr",
			Namespace: "default",
		},
		Status: porchv1alpha1.PackageRevisionStatus{
			Conditions: []porchv1alpha1.Condition{
				{
					Type:   kptfilelibv1.GetConditionType(&corev1.ObjectReference{Kind: ipamv1alpha1.IPClaimKind, APIVersion: ipamv1alpha1.SchemeBuilder.GroupVersion.Identifier()}) + ".test",
					Status: porchv1alpha1.ConditionTrue,
				},
			},
		},
	}

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
		Client:      fakeClient,
		For:         corev1.ObjectReference{Kind: ipamv1alpha1.IPClaimKind, APIVersion: ipamv1alpha1.SchemeBuilder.GroupVersion.Identifier()},
		porchClient: fakeClient,
		krmfn:       &mockResourceListProcessor{err: errors.New("mock process error")},
	}

	req := ctrl.Request{
		NamespacedName: client.ObjectKey{
			Name:      "test-pr",
			Namespace: "default",
		},
	}


	res, err := r.Reconcile(context.Background(), req)
	if err == nil {
		t.Fatalf("Expected error from Reconcile, got nil")
	}
	if err.Error() != "function run failed: mock process error" {
		t.Fatalf("Expected specific error message, got: %v", err)
	}
	if res.Requeue {
		t.Fatalf("Expected Requeue to be false, got true")
	}
}
