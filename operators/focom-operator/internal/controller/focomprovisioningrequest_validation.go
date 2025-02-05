package controller

import (
	"context"
	"fmt"

	focomv1alpha1 "github.com/dekstroza/focom-operator/api/focom/v1alpha1"
	provisioningv1alpha1 "github.com/dekstroza/focom-operator/api/provisioning/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

// validateTemplateAlignment checks if there's a matching TemplateInfo resource
// for the FocomProvisioningRequest (by templateName, templateVersion).
// Returns nil if alignment is good, or an error with details if not.
func (r *FocomProvisioningRequestReconciler) validateTemplateAlignment(
	ctx context.Context,
	fpr *focomv1alpha1.FocomProvisioningRequest,
) error {
	// Construct the name for TemplateInfo.
	// This naming strategy can vary, but commonly: "<templateName>-<templateVersion>"
	tplInfoName := fmt.Sprintf("%s-%s", fpr.Spec.TemplateName, fpr.Spec.TemplateVersion)

	// Fetch TemplateInfo from the same namespace as the FPR (or a special "catalog" namespace if that's your design)
	var tplInfo provisioningv1alpha1.TemplateInfo
	err := r.Get(ctx, types.NamespacedName{
		Name:      tplInfoName,
		Namespace: fpr.Namespace,
	}, &tplInfo)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("TemplateInfo %q not found", tplInfoName)
		}
		return fmt.Errorf("failed to get TemplateInfo: %v", err)
	}

	// Check that the TemplateInfo's spec matches the FPR's spec
	if tplInfo.Spec.TemplateName != fpr.Spec.TemplateName ||
		tplInfo.Spec.TemplateVersion != fpr.Spec.TemplateVersion {
		return fmt.Errorf("mismatch: TemplateInfo has (%s/%s), request has (%s/%s)",
			tplInfo.Spec.TemplateName, tplInfo.Spec.TemplateVersion,
			fpr.Spec.TemplateName, fpr.Spec.TemplateVersion,
		)
	}

	// For now, if we got here, the basic alignment is valid.
	return nil
}
