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
	"fmt"

	focomv1alpha1 "github.com/nephio-project/nephio/operators/focom-operator/api/focom/v1alpha1"
	provisioningv1alpha1 "github.com/nephio-project/nephio/operators/focom-operator/api/provisioning/v1alpha1"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/models"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// OperatorIntegrationImpl implements the OperatorIntegration interface
type OperatorIntegrationImpl struct {
	client client.Client
	mapper CRDMapper
}

// NewOperatorIntegration creates a new operator integration instance
func NewOperatorIntegration(client client.Client, mapper CRDMapper) OperatorIntegration {
	if mapper == nil {
		mapper = NewDefaultCRDMapper()
	}

	return &OperatorIntegrationImpl{
		client: client,
		mapper: mapper,
	}
}

// CreateOCloudCR creates or updates an OCloud custom resource
func (o *OperatorIntegrationImpl) CreateOCloudCR(ctx context.Context, ocloud *models.OCloudData) error {
	logger := log.FromContext(ctx)

	cr := o.mapper.OCloudDataToCR(ocloud)

	// Try to get existing CR first
	existingCR := &focomv1alpha1.OCloud{}
	err := o.client.Get(ctx, client.ObjectKey{Name: cr.Name, Namespace: cr.Namespace}, existingCR)

	if err != nil {
		// CR doesn't exist, create it
		logger.Info("Creating OCloud CR", "name", cr.Name, "namespace", cr.Namespace)
		if err := o.client.Create(ctx, cr); err != nil {
			return fmt.Errorf("failed to create OCloud CR: %w", err)
		}
		logger.Info("Successfully created OCloud CR", "name", cr.Name, "namespace", cr.Namespace)
	} else {
		// CR exists, update it
		logger.Info("Updating existing OCloud CR", "name", cr.Name, "namespace", cr.Namespace)
		existingCR.Spec = cr.Spec
		if err := o.client.Update(ctx, existingCR); err != nil {
			return fmt.Errorf("failed to update OCloud CR: %w", err)
		}
		logger.Info("Successfully updated OCloud CR", "name", cr.Name, "namespace", cr.Namespace)
	}

	return nil
}

// CreateTemplateInfoCR creates or updates a TemplateInfo custom resource
func (o *OperatorIntegrationImpl) CreateTemplateInfoCR(ctx context.Context, templateInfo *models.TemplateInfoData) error {
	logger := log.FromContext(ctx)

	cr := o.mapper.TemplateInfoDataToCR(templateInfo)

	// Try to get existing CR first
	existingCR := &provisioningv1alpha1.TemplateInfo{}
	err := o.client.Get(ctx, client.ObjectKey{Name: cr.Name, Namespace: cr.Namespace}, existingCR)

	if err != nil {
		// CR doesn't exist, create it
		logger.Info("Creating TemplateInfo CR", "name", cr.Name, "namespace", cr.Namespace)
		if err := o.client.Create(ctx, cr); err != nil {
			return fmt.Errorf("failed to create TemplateInfo CR: %w", err)
		}
		logger.Info("Successfully created TemplateInfo CR", "name", cr.Name, "namespace", cr.Namespace)
	} else {
		// CR exists, update it
		logger.Info("Updating existing TemplateInfo CR", "name", cr.Name, "namespace", cr.Namespace)
		existingCR.Spec = cr.Spec
		if err := o.client.Update(ctx, existingCR); err != nil {
			return fmt.Errorf("failed to update TemplateInfo CR: %w", err)
		}
		logger.Info("Successfully updated TemplateInfo CR", "name", cr.Name, "namespace", cr.Namespace)
	}

	return nil
}

// CreateFocomProvisioningRequestCR creates or updates a FocomProvisioningRequest custom resource
func (o *OperatorIntegrationImpl) CreateFocomProvisioningRequestCR(ctx context.Context, fpr *models.FocomProvisioningRequestData) error {
	logger := log.FromContext(ctx)

	cr := o.mapper.FocomProvisioningRequestDataToCR(fpr)

	// Try to get existing CR first
	existingCR := &focomv1alpha1.FocomProvisioningRequest{}
	err := o.client.Get(ctx, client.ObjectKey{Name: cr.Name, Namespace: cr.Namespace}, existingCR)

	if err != nil {
		// CR doesn't exist, create it
		logger.Info("Creating FocomProvisioningRequest CR", "name", cr.Name, "namespace", cr.Namespace)
		if err := o.client.Create(ctx, cr); err != nil {
			return fmt.Errorf("failed to create FocomProvisioningRequest CR: %w", err)
		}
		logger.Info("Successfully created FocomProvisioningRequest CR", "name", cr.Name, "namespace", cr.Namespace)
	} else {
		// CR exists, update it
		logger.Info("Updating existing FocomProvisioningRequest CR", "name", cr.Name, "namespace", cr.Namespace)
		existingCR.Spec = cr.Spec
		if err := o.client.Update(ctx, existingCR); err != nil {
			return fmt.Errorf("failed to update FocomProvisioningRequest CR: %w", err)
		}
		logger.Info("Successfully updated FocomProvisioningRequest CR", "name", cr.Name, "namespace", cr.Namespace)
	}

	return nil
}

// GetKubernetesClient returns the Kubernetes client
func (o *OperatorIntegrationImpl) GetKubernetesClient() client.Client {
	return o.client
}

// CreateO2IMSProvisioningRequest creates a provisioning request in O2IMS (Stage 3)
func (o *OperatorIntegrationImpl) CreateO2IMSProvisioningRequest(ctx context.Context, request *O2IMSProvisioningRequest) error {
	// TODO: Implement O2IMS REST API integration for Stage 3
	return fmt.Errorf("O2IMS integration not implemented yet (Stage 3 feature)")
}

// UpdateO2IMSProvisioningRequest updates a provisioning request in O2IMS (Stage 3)
func (o *OperatorIntegrationImpl) UpdateO2IMSProvisioningRequest(ctx context.Context, id string, request *O2IMSProvisioningRequest) error {
	// TODO: Implement O2IMS REST API integration for Stage 3
	return fmt.Errorf("O2IMS integration not implemented yet (Stage 3 feature)")
}

// DeleteO2IMSProvisioningRequest deletes a provisioning request in O2IMS (Stage 3)
func (o *OperatorIntegrationImpl) DeleteO2IMSProvisioningRequest(ctx context.Context, id string) error {
	// TODO: Implement O2IMS REST API integration for Stage 3
	return fmt.Errorf("O2IMS integration not implemented yet (Stage 3 feature)")
}

// GetO2IMSProvisioningStatus gets the status of a provisioning request from O2IMS (Stage 3)
func (o *OperatorIntegrationImpl) GetO2IMSProvisioningStatus(ctx context.Context, id string) (*O2IMSProvisioningStatus, error) {
	// TODO: Implement O2IMS REST API integration for Stage 3
	return nil, fmt.Errorf("O2IMS integration not implemented yet (Stage 3 feature)")
}

// HealthCheck performs a health check on the integration
func (o *OperatorIntegrationImpl) HealthCheck(ctx context.Context) error {
	// For Stage 1, just check if the Kubernetes client is available
	if o.client == nil {
		return fmt.Errorf("kubernetes client is not available")
	}

	// TODO: Add more comprehensive health checks for different stages
	return nil
}
