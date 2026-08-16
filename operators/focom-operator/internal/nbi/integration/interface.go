/*
Copyright 2026 The Nephio Authors.

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

package integration

import (
	"context"

	focomv1alpha1 "github.com/nephio-project/nephio/operators/focom-operator/api/focom/v1alpha1"
	provisioningv1alpha1 "github.com/nephio-project/nephio/operators/focom-operator/api/provisioning/v1alpha1"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/models"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// OperatorIntegration defines the interface for integrating with the existing FOCOM operator
type OperatorIntegration interface {
	// Kubernetes operations using existing operator client (stages 1 & 2)
	CreateOCloudCR(ctx context.Context, ocloud *models.OCloudData) error
	CreateTemplateInfoCR(ctx context.Context, templateInfo *models.TemplateInfoData) error
	CreateFocomProvisioningRequestCR(ctx context.Context, fpr *models.FocomProvisioningRequestData) error

	// Generic Kubernetes client access
	GetKubernetesClient() client.Client

	// O2IMS operations (stage 3)
	CreateO2IMSProvisioningRequest(ctx context.Context, request *O2IMSProvisioningRequest) error
	UpdateO2IMSProvisioningRequest(ctx context.Context, id string, request *O2IMSProvisioningRequest) error
	DeleteO2IMSProvisioningRequest(ctx context.Context, id string) error
	GetO2IMSProvisioningStatus(ctx context.Context, id string) (*O2IMSProvisioningStatus, error)

	// Health and connectivity
	HealthCheck(ctx context.Context) error
}

// O2IMSProvisioningRequest represents a provisioning request for O2IMS
type O2IMSProvisioningRequest struct {
	ID                 string                 `json:"id"`
	Name               string                 `json:"name"`
	Description        string                 `json:"description"`
	OCloudID           string                 `json:"oCloudId"`
	TemplateName       string                 `json:"templateName"`
	TemplateVersion    string                 `json:"templateVersion"`
	TemplateParameters map[string]interface{} `json:"templateParameters"`
}

// O2IMSProvisioningStatus represents the status of a provisioning request in O2IMS
type O2IMSProvisioningStatus struct {
	ID                        string                `json:"id"`
	Phase                     string                `json:"phase"`
	Message                   string                `json:"message"`
	ClusterRegistrationStatus string                `json:"clusterRegistrationStatus,omitempty"`
	ProvisionedResources      []ProvisionedResource `json:"provisionedResources,omitempty"`
}

// ProvisionedResource represents a resource provisioned by O2IMS
type ProvisionedResource struct {
	Type string `json:"type"`
	Name string `json:"name"`
	ID   string `json:"id"`
}

// O2IMSEndpoint represents an O2IMS server endpoint configuration
type O2IMSEndpoint struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// CRDMapper defines methods for converting between internal models and CRDs
type CRDMapper interface {
	// Convert internal models to CRDs
	OCloudDataToCR(data *models.OCloudData) *focomv1alpha1.OCloud
	TemplateInfoDataToCR(data *models.TemplateInfoData) *provisioningv1alpha1.TemplateInfo
	FocomProvisioningRequestDataToCR(data *models.FocomProvisioningRequestData) *focomv1alpha1.FocomProvisioningRequest

	// Convert CRDs to internal models
	OCloudCRToData(cr *focomv1alpha1.OCloud) *models.OCloudData
	TemplateInfoCRToData(cr *provisioningv1alpha1.TemplateInfo) *models.TemplateInfoData
	FocomProvisioningRequestCRToData(cr *focomv1alpha1.FocomProvisioningRequest) *models.FocomProvisioningRequestData

	// Convert internal models to O2IMS requests
	FocomProvisioningRequestDataToO2IMS(data *models.FocomProvisioningRequestData) *O2IMSProvisioningRequest
}
