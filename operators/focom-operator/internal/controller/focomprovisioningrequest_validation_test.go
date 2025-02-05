package controller

import (
	"context"
	"testing"

	focomv1alpha1 "github.com/dekstroza/focom-operator/api/focom/v1alpha1"
	provisioningv1alpha1 "github.com/dekstroza/focom-operator/api/provisioning/v1alpha1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestValidateTemplateAlignment(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = focomv1alpha1.AddToScheme(scheme)
	_ = provisioningv1alpha1.AddToScheme(scheme)

	// Create a TemplateInfo matching name: "my-template-1.0" in "default"
	tplInfo := provisioningv1alpha1.TemplateInfo{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-template-1.0",
			Namespace: "default",
		},
		Spec: provisioningv1alpha1.TemplateInfoSpec{
			TemplateName:    "my-template",
			TemplateVersion: "1.0",
		},
	}

	// Create an FPR referencing the same template
	fpr := focomv1alpha1.FocomProvisioningRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sample-fpr",
			Namespace: "default",
		},
		Spec: focomv1alpha1.FocomProvisioningRequestSpec{
			TemplateName:    "my-template",
			TemplateVersion: "1.0",
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&tplInfo, &fpr).
		Build()

	r := &FocomProvisioningRequestReconciler{
		Client: client,
	}

	ctx := context.Background()
	err := r.validateTemplateAlignment(ctx, &fpr)
	require.NoError(t, err, "Expected no error, because templateName/version match")

	// Now modify the FPR to mismatch
	fpr.Spec.TemplateVersion = "2.0"
	err = r.validateTemplateAlignment(ctx, &fpr)
	require.Error(t, err, "Expected error, because mismatch with TemplateInfo")
}
