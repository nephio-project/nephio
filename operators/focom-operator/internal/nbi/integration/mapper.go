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
	"encoding/json"
	"time"

	focomv1alpha1 "github.com/nephio-project/nephio/operators/focom-operator/api/focom/v1alpha1"
	provisioningv1alpha1 "github.com/nephio-project/nephio/operators/focom-operator/api/provisioning/v1alpha1"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/models"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DefaultCRDMapper implements the CRDMapper interface
type DefaultCRDMapper struct{}

// NewDefaultCRDMapper creates a new default CRD mapper
func NewDefaultCRDMapper() *DefaultCRDMapper {
	return &DefaultCRDMapper{}
}

// OCloudDataToCR converts OCloudData to OCloud CR
func (m *DefaultCRDMapper) OCloudDataToCR(data *models.OCloudData) *focomv1alpha1.OCloud {
	return &focomv1alpha1.OCloud{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "focom.nephio.org/v1alpha1",
			Kind:       "OCloud",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      data.Name,
			Namespace: data.Namespace,
		},
		Spec: focomv1alpha1.OCloudSpec{
			O2imsSecret: focomv1alpha1.O2imsSecret{
				SecretRef: focomv1alpha1.SecretRef{
					Name:      data.O2IMSSecret.SecretRef.Name,
					Namespace: data.O2IMSSecret.SecretRef.Namespace,
				},
			},
		},
	}
}

// TemplateInfoDataToCR converts TemplateInfoData to TemplateInfo CR
func (m *DefaultCRDMapper) TemplateInfoDataToCR(data *models.TemplateInfoData) *provisioningv1alpha1.TemplateInfo {
	return &provisioningv1alpha1.TemplateInfo{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "provisioning.nephio.org/v1alpha1",
			Kind:       "TemplateInfo",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      data.Name,
			Namespace: data.Namespace,
		},
		Spec: provisioningv1alpha1.TemplateInfoSpec{
			TemplateName:            data.TemplateName,
			TemplateVersion:         data.TemplateVersion,
			TemplateParameterSchema: data.TemplateParameterSchema,
		},
	}
}

// FocomProvisioningRequestDataToCR converts FocomProvisioningRequestData to FocomProvisioningRequest CR
func (m *DefaultCRDMapper) FocomProvisioningRequestDataToCR(data *models.FocomProvisioningRequestData) *focomv1alpha1.FocomProvisioningRequest {
	// Convert template parameters to RawExtension
	templateParamsBytes, _ := json.Marshal(data.TemplateParameters)

	return &focomv1alpha1.FocomProvisioningRequest{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "focom.nephio.org/v1alpha1",
			Kind:       "FocomProvisioningRequest",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      data.Name,
			Namespace: data.Namespace,
		},
		Spec: focomv1alpha1.FocomProvisioningRequestSpec{
			OCloudId:           data.OCloudID,
			OCloudNamespace:    data.OCloudNamespace,
			Name:               data.Name,
			Description:        data.Description,
			TemplateName:       data.TemplateName,
			TemplateVersion:    data.TemplateVersion,
			TemplateParameters: runtime.RawExtension{Raw: templateParamsBytes},
		},
	}
}

// OCloudCRToData converts OCloud CR to OCloudData
func (m *DefaultCRDMapper) OCloudCRToData(cr *focomv1alpha1.OCloud) *models.OCloudData {
	return &models.OCloudData{
		BaseResource: models.BaseResource{
			ID:          cr.Name,
			Namespace:   cr.Namespace,
			Name:        cr.Name,
			Description: "", // Not available in CR
			State:       models.StateApproved,
			CreatedAt:   cr.CreationTimestamp.Time,
			UpdatedAt:   time.Now(),
			Metadata:    make(map[string]interface{}),
		},
		O2IMSSecret: models.O2IMSSecretRef{
			SecretRef: models.SecretReference{
				Name:      cr.Spec.O2imsSecret.SecretRef.Name,
				Namespace: cr.Spec.O2imsSecret.SecretRef.Namespace,
			},
		},
	}
}

// TemplateInfoCRToData converts TemplateInfo CR to TemplateInfoData
func (m *DefaultCRDMapper) TemplateInfoCRToData(cr *provisioningv1alpha1.TemplateInfo) *models.TemplateInfoData {
	return &models.TemplateInfoData{
		BaseResource: models.BaseResource{
			ID:          cr.Name,
			Namespace:   cr.Namespace,
			Name:        cr.Name,
			Description: "", // Not available in CR
			State:       models.StateApproved,
			CreatedAt:   cr.CreationTimestamp.Time,
			UpdatedAt:   time.Now(),
			Metadata:    make(map[string]interface{}),
		},
		TemplateName:            cr.Spec.TemplateName,
		TemplateVersion:         cr.Spec.TemplateVersion,
		TemplateParameterSchema: cr.Spec.TemplateParameterSchema,
	}
}

// FocomProvisioningRequestCRToData converts FocomProvisioningRequest CR to FocomProvisioningRequestData
func (m *DefaultCRDMapper) FocomProvisioningRequestCRToData(cr *focomv1alpha1.FocomProvisioningRequest) *models.FocomProvisioningRequestData {
	// Convert RawExtension to map
	var templateParams map[string]interface{}
	if cr.Spec.TemplateParameters.Raw != nil {
		if err := json.Unmarshal(cr.Spec.TemplateParameters.Raw, &templateParams); err != nil {
			// Log error but continue with empty template params
			templateParams = make(map[string]interface{})
		}
	}

	return &models.FocomProvisioningRequestData{
		BaseResource: models.BaseResource{
			ID:          cr.Name,
			Namespace:   cr.Namespace,
			Name:        cr.Spec.Name,
			Description: cr.Spec.Description,
			State:       models.StateApproved,
			CreatedAt:   cr.CreationTimestamp.Time,
			UpdatedAt:   time.Now(),
			Metadata:    make(map[string]interface{}),
		},
		OCloudID:           cr.Spec.OCloudId,
		OCloudNamespace:    cr.Spec.OCloudNamespace,
		TemplateName:       cr.Spec.TemplateName,
		TemplateVersion:    cr.Spec.TemplateVersion,
		TemplateParameters: templateParams,
	}
}

// FocomProvisioningRequestDataToO2IMS converts FocomProvisioningRequestData to O2IMSProvisioningRequest
func (m *DefaultCRDMapper) FocomProvisioningRequestDataToO2IMS(data *models.FocomProvisioningRequestData) *O2IMSProvisioningRequest {
	return &O2IMSProvisioningRequest{
		ID:                 data.ID,
		Name:               data.Name,
		Description:        data.Description,
		OCloudID:           data.OCloudID,
		TemplateName:       data.TemplateName,
		TemplateVersion:    data.TemplateVersion,
		TemplateParameters: data.TemplateParameters,
	}
}
