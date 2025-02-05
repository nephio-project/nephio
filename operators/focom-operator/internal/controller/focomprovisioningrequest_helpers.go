package controller

import (
	"context"
	"encoding/json"
	"fmt"

	focomv1alpha1 "github.com/dekstroza/focom-operator/api/focom/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

/***********************************************
    HELPER METHODS
***********************************************/

func (r *FocomProvisioningRequestReconciler) updateStatus(
	f *focomv1alpha1.FocomProvisioningRequest,
	phase, msg string,
) {
	f.Status.Phase = phase
	f.Status.Message = msg
	now := metav1.Now()
	f.Status.LastUpdated = &now
}

/********** DELETE REMOTE (UNSTRUCTURED) ***********/
func (r *FocomProvisioningRequestReconciler) deleteRemoteResource(
	ctx context.Context,
	fpr *focomv1alpha1.FocomProvisioningRequest,
) error {
	if fpr.Status.RemoteName == "" {
		return nil
	}

	remoteCl, err := r.buildRemoteClient(ctx, fpr)
	if err != nil {
		return fmt.Errorf("buildRemoteClient error: %w", err)
	}

	remoteObj := &unstructured.Unstructured{}
	remoteObj.SetAPIVersion("o2ims.provisioning.oran.org/v1alpha1")
	remoteObj.SetKind("ProvisioningRequest")
	remoteObj.SetName(fpr.Status.RemoteName)

	if err := remoteCl.Delete(ctx, remoteObj); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("delete remote CR: %w", err)
	}
	return nil
}

/********** DECODE templateParameters ***********/

// decodeRawParameters converts the raw JSON in fpr.Spec.TemplateParameters to map[string]interface{}.
// If empty, returns empty map.
func decodeRawParameters(raw runtime.RawExtension) (map[string]interface{}, error) {
	if len(raw.Raw) == 0 {
		return map[string]interface{}{}, nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw.Raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

/********** CREATE REMOTE ***********/
func (r *FocomProvisioningRequestReconciler) createRemoteProvisioningRequest(
	ctx context.Context,
	remoteCl client.Client,
	fpr *focomv1alpha1.FocomProvisioningRequest,
) (string, error) {

	// If local CR name is empty, generate a unique one
	remoteName := fpr.Name
	if remoteName == "" {
		remoteName = string(uuid.NewUUID())
	}

	// Decode user-provided templateParameters from runtime.RawExtension
	templateParams, err := decodeRawParameters(fpr.Spec.TemplateParameters)
	if err != nil {
		return "", fmt.Errorf("failed to parse templateParameters: %w", err)
	}

	// Build an unstructured for the remote ProvisioningRequest
	remoteObj := &unstructured.Unstructured{}
	remoteObj.SetAPIVersion("o2ims.provisioning.oran.org/v1alpha1")
	remoteObj.SetKind("ProvisioningRequest")
	remoteObj.SetName(remoteName) // cluster-scoped from your CRD

	// Build the .spec map
	spec := map[string]interface{}{
		"name":               fpr.Spec.Name,
		"description":        fpr.Spec.Description,
		"templateName":       fpr.Spec.TemplateName,
		"templateVersion":    fpr.Spec.TemplateVersion,
		"templateParameters": templateParams,
	}
	remoteObj.Object["spec"] = spec

	// Create the remote resource
	if err := remoteCl.Create(ctx, remoteObj); err != nil {
		return "", err
	}
	return remoteObj.GetName(), nil
}

/********** POLL REMOTE (UNSTRUCTURED) ***********/
func (r *FocomProvisioningRequestReconciler) pollRemoteProvisioningRequest(
	ctx context.Context,
	remoteCl client.Client,
	fpr *focomv1alpha1.FocomProvisioningRequest,
) (done bool, phase string, msg string, err error) {

	remoteName := fpr.Status.RemoteName
	remoteObj := &unstructured.Unstructured{}
	remoteObj.SetAPIVersion("o2ims.provisioning.oran.org/v1alpha1")
	remoteObj.SetKind("ProvisioningRequest")
	remoteObj.SetName(remoteName)

	if err := remoteCl.Get(ctx, types.NamespacedName{Name: remoteName}, remoteObj); err != nil {
		if k8serrors.IsNotFound(err) {
			return true, "failed", "Remote resource not found", nil
		}
		return false, "failed", "", fmt.Errorf("error fetching remote CR: %w", err)
	}

	// Extract status.provisioningStatus fields from unstructured
	statusMap, _, _ := unstructured.NestedMap(remoteObj.Object, "status", "provisioningStatus")
	if statusMap == nil {
		// no status yet
		return false, "provisioning", "Remote CR has no status yet", nil
	}
	state, _, _ := unstructured.NestedString(statusMap, "provisioningState")
	message, _, _ := unstructured.NestedString(statusMap, "provisioningMessage")

	switch state {
	case "fulfilled":
		return true, "Fulfilled", message, nil
	case "failed":
		return true, "Failed", message, nil
	case "deleting", "progressing":
		// keep requeueing
		return false, state, message, nil
	default:
		// unknown => keep requeueing
		return false, "provisioning", fmt.Sprintf("Unknown remote state: %s", state), nil
	}
}
