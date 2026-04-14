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

package testfixtures

import (
	"encoding/json"
	"time"

	focomv1alpha1 "github.com/nephio-project/nephio/operators/focom-operator/api/focom/v1alpha1"
	provisioningv1alpha1 "github.com/nephio-project/nephio/operators/focom-operator/api/provisioning/v1alpha1"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/integration"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/models"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/storage"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// TestFixtures provides comprehensive test data for NBI testing
type TestFixtures struct {
	OClouds                     []*models.OCloudData
	TemplateInfos               []*models.TemplateInfoData
	FocomProvisioningRequests   []*models.FocomProvisioningRequestData
	DraftResources              []*models.DraftStorage
	RevisionResources           []*models.RevisionStorage
	O2IMSProvisioningRequests   []*integration.O2IMSProvisioningRequest
	O2IMSProvisioningStatuses   []*integration.O2IMSProvisioningStatus
	O2IMSEndpoints              []*integration.O2IMSEndpoint
	OCloudCRs                   []*focomv1alpha1.OCloud
	TemplateInfoCRs             []*provisioningv1alpha1.TemplateInfo
	FocomProvisioningRequestCRs []*focomv1alpha1.FocomProvisioningRequest
}

// NewTestFixtures creates a new set of comprehensive test fixtures
func NewTestFixtures() *TestFixtures {
	fixtures := &TestFixtures{}

	// Create base test data
	fixtures.createOCloudFixtures()
	fixtures.createTemplateInfoFixtures()
	fixtures.createFocomProvisioningRequestFixtures()
	fixtures.createDraftFixtures()
	fixtures.createRevisionFixtures()
	fixtures.createO2IMSFixtures()
	fixtures.createCRFixtures()

	return fixtures
}

func (f *TestFixtures) createOCloudFixtures() {
	now := time.Now()

	f.OClouds = []*models.OCloudData{
		{
			BaseResource: models.BaseResource{
				ID:          "ocloud-1",
				RevisionID:  "v1",
				Namespace:   "default",
				Name:        "ocloud-1",
				Description: "Primary OCloud for testing",
				State:       models.StateApproved,
				CreatedAt:   now.Add(-24 * time.Hour),
				UpdatedAt:   now.Add(-1 * time.Hour),
				Metadata: map[string]interface{}{
					"environment": "test",
					"region":      "us-west-1",
				},
			},
			O2IMSSecret: models.O2IMSSecretRef{
				SecretRef: models.SecretReference{
					Name:      "ocloud-1-secret",
					Namespace: "default",
				},
			},
		},
		{
			BaseResource: models.BaseResource{
				ID:          "ocloud-2",
				RevisionID:  "v1",
				Namespace:   "default",
				Name:        "ocloud-2",
				Description: "Production OCloud",
				State:       models.StateApproved,
				CreatedAt:   now.Add(-48 * time.Hour),
				UpdatedAt:   now.Add(-2 * time.Hour),
				Metadata: map[string]interface{}{
					"environment": "production",
					"region":      "us-east-1",
				},
			},
			O2IMSSecret: models.O2IMSSecretRef{
				SecretRef: models.SecretReference{
					Name:      "ocloud-2-secret",
					Namespace: "default",
				},
			},
		},
		{
			BaseResource: models.BaseResource{
				ID:          "ocloud-draft",
				Namespace:   "default",
				Name:        "ocloud-draft",
				Description: "Draft OCloud for testing",
				State:       models.StateDraft,
				CreatedAt:   now.Add(-1 * time.Hour),
				UpdatedAt:   now.Add(-30 * time.Minute),
				Metadata:    map[string]interface{}{},
			},
			O2IMSSecret: models.O2IMSSecretRef{
				SecretRef: models.SecretReference{
					Name:      "ocloud-draft-secret",
					Namespace: "default",
				},
			},
		},
	}
}

func (f *TestFixtures) createTemplateInfoFixtures() {
	now := time.Now()

	f.TemplateInfos = []*models.TemplateInfoData{
		{
			BaseResource: models.BaseResource{
				ID:          "template-1",
				RevisionID:  "v1",
				Namespace:   "default",
				Name:        "template-1",
				Description: "Basic cluster template",
				State:       models.StateApproved,
				CreatedAt:   now.Add(-24 * time.Hour),
				UpdatedAt:   now.Add(-1 * time.Hour),
				Metadata: map[string]interface{}{
					"category":   "cluster",
					"complexity": "basic",
				},
			},
			TemplateName:    "basic-cluster",
			TemplateVersion: "v1.0.0",
			TemplateParameterSchema: `{
				"type": "object",
				"properties": {
					"clusterName": {"type": "string"},
					"nodeCount": {"type": "integer", "minimum": 1, "maximum": 10}
				},
				"required": ["clusterName", "nodeCount"]
			}`,
		},
		{
			BaseResource: models.BaseResource{
				ID:          "template-2",
				RevisionID:  "v1",
				Namespace:   "default",
				Name:        "template-2",
				Description: "Advanced cluster template with networking",
				State:       models.StateApproved,
				CreatedAt:   now.Add(-48 * time.Hour),
				UpdatedAt:   now.Add(-2 * time.Hour),
				Metadata: map[string]interface{}{
					"category":   "cluster",
					"complexity": "advanced",
				},
			},
			TemplateName:    "advanced-cluster",
			TemplateVersion: "v2.1.0",
			TemplateParameterSchema: `{
				"type": "object",
				"properties": {
					"clusterName": {"type": "string"},
					"nodeCount": {"type": "integer", "minimum": 3, "maximum": 50},
					"networkConfig": {
						"type": "object",
						"properties": {
							"cidr": {"type": "string"},
							"enableLoadBalancer": {"type": "boolean"}
						}
					}
				},
				"required": ["clusterName", "nodeCount", "networkConfig"]
			}`,
		},
		{
			BaseResource: models.BaseResource{
				ID:          "template-validated",
				Namespace:   "default",
				Name:        "template-validated",
				Description: "Validated template awaiting approval",
				State:       models.StateValidated,
				CreatedAt:   now.Add(-2 * time.Hour),
				UpdatedAt:   now.Add(-30 * time.Minute),
				Metadata:    map[string]interface{}{},
			},
			TemplateName:            "validated-template",
			TemplateVersion:         "v1.0.0-beta",
			TemplateParameterSchema: `{"type": "object"}`,
		},
	}
}

func (f *TestFixtures) createFocomProvisioningRequestFixtures() {
	now := time.Now()

	f.FocomProvisioningRequests = []*models.FocomProvisioningRequestData{
		{
			BaseResource: models.BaseResource{
				ID:          "fpr-1",
				RevisionID:  "v1",
				Namespace:   "default",
				Name:        "fpr-1",
				Description: "Test cluster provisioning request",
				State:       models.StateApproved,
				CreatedAt:   now.Add(-12 * time.Hour),
				UpdatedAt:   now.Add(-1 * time.Hour),
				Metadata: map[string]interface{}{
					"priority":  "high",
					"requestor": "test-user",
				},
			},
			OCloudID:        "ocloud-1",
			OCloudNamespace: "default",
			TemplateName:    "basic-cluster",
			TemplateVersion: "v1.0.0",
			TemplateParameters: map[string]interface{}{
				"clusterName": "test-cluster-1",
				"nodeCount":   3,
			},
		},
		{
			BaseResource: models.BaseResource{
				ID:          "fpr-2",
				RevisionID:  "v1",
				Namespace:   "default",
				Name:        "fpr-2",
				Description: "Production cluster provisioning request",
				State:       models.StateApproved,
				CreatedAt:   now.Add(-24 * time.Hour),
				UpdatedAt:   now.Add(-2 * time.Hour),
				Metadata: map[string]interface{}{
					"priority":  "critical",
					"requestor": "prod-admin",
				},
			},
			OCloudID:        "ocloud-2",
			OCloudNamespace: "default",
			TemplateName:    "advanced-cluster",
			TemplateVersion: "v2.1.0",
			TemplateParameters: map[string]interface{}{
				"clusterName": "prod-cluster-1",
				"nodeCount":   10,
				"networkConfig": map[string]interface{}{
					"cidr":               "10.0.0.0/16",
					"enableLoadBalancer": true,
				},
			},
		},
		{
			BaseResource: models.BaseResource{
				ID:          "fpr-draft",
				Namespace:   "default",
				Name:        "fpr-draft",
				Description: "Draft provisioning request",
				State:       models.StateDraft,
				CreatedAt:   now.Add(-1 * time.Hour),
				UpdatedAt:   now.Add(-15 * time.Minute),
				Metadata:    map[string]interface{}{},
			},
			OCloudID:        "ocloud-1",
			OCloudNamespace: "default",
			TemplateName:    "basic-cluster",
			TemplateVersion: "v1.0.0",
			TemplateParameters: map[string]interface{}{
				"clusterName": "draft-cluster",
				"nodeCount":   1,
			},
		},
	}
}

func (f *TestFixtures) createDraftFixtures() {
	now := time.Now()

	f.DraftResources = []*models.DraftStorage{
		{
			ResourceID:   "ocloud-draft",
			ResourceType: models.ResourceTypeOCloud,
			DraftData:    f.OClouds[2], // ocloud-draft
			State:        models.StateDraft,
			CreatedAt:    now.Add(-1 * time.Hour),
			UpdatedAt:    now.Add(-30 * time.Minute),
		},
		{
			ResourceID:   "template-validated",
			ResourceType: models.ResourceTypeTemplateInfo,
			DraftData:    f.TemplateInfos[2], // template-validated
			State:        models.StateValidated,
			CreatedAt:    now.Add(-2 * time.Hour),
			UpdatedAt:    now.Add(-30 * time.Minute),
		},
		{
			ResourceID:   "fpr-draft",
			ResourceType: models.ResourceTypeFocomProvisioningRequest,
			DraftData:    f.FocomProvisioningRequests[2], // fpr-draft
			State:        models.StateDraft,
			CreatedAt:    now.Add(-1 * time.Hour),
			UpdatedAt:    now.Add(-15 * time.Minute),
		},
	}
}

func (f *TestFixtures) createRevisionFixtures() {
	// NOTE: RevisionResources should only contain ADDITIONAL revisions beyond v1
	// The main fixtures (OClouds, TemplateInfos, FocomProvisioningRequests) already create v1
	// This array is for creating v2, v3, etc. when testing revision history

	// Currently empty - tests will create additional revisions via API when needed
	// This avoids "package already exists" errors from trying to create v1 twice
	f.RevisionResources = []*models.RevisionStorage{}
}

func (f *TestFixtures) createO2IMSFixtures() {
	f.O2IMSProvisioningRequests = []*integration.O2IMSProvisioningRequest{
		{
			ID:              "fpr-1",
			Name:            "fpr-1",
			Description:     "Test cluster provisioning request",
			OCloudID:        "ocloud-1",
			TemplateName:    "basic-cluster",
			TemplateVersion: "v1.0.0",
			TemplateParameters: map[string]interface{}{
				"clusterName": "test-cluster-1",
				"nodeCount":   3,
			},
		},
		{
			ID:              "fpr-2",
			Name:            "fpr-2",
			Description:     "Production cluster provisioning request",
			OCloudID:        "ocloud-2",
			TemplateName:    "advanced-cluster",
			TemplateVersion: "v2.1.0",
			TemplateParameters: map[string]interface{}{
				"clusterName": "prod-cluster-1",
				"nodeCount":   10,
				"networkConfig": map[string]interface{}{
					"cidr":               "10.0.0.0/16",
					"enableLoadBalancer": true,
				},
			},
		},
	}

	f.O2IMSProvisioningStatuses = []*integration.O2IMSProvisioningStatus{
		{
			ID:                        "fpr-1",
			Phase:                     "Provisioned",
			Message:                   "Cluster successfully provisioned",
			ClusterRegistrationStatus: "Registered",
			ProvisionedResources: []integration.ProvisionedResource{
				{
					Type: "Cluster",
					Name: "test-cluster-1",
					ID:   "cluster-test-1",
				},
				{
					Type: "LoadBalancer",
					Name: "test-cluster-1-lb",
					ID:   "lb-test-1",
				},
			},
		},
		{
			ID:                        "fpr-2",
			Phase:                     "Provisioning",
			Message:                   "Cluster provisioning in progress",
			ClusterRegistrationStatus: "Pending",
			ProvisionedResources: []integration.ProvisionedResource{
				{
					Type: "Cluster",
					Name: "prod-cluster-1",
					ID:   "cluster-prod-1",
				},
			},
		},
	}

	f.O2IMSEndpoints = []*integration.O2IMSEndpoint{
		{
			Name:     "ocloud-1-endpoint",
			URL:      "https://ocloud-1.o2ims.example.com",
			Username: "admin",
			Password: "secret123",
		},
		{
			Name:     "ocloud-2-endpoint",
			URL:      "https://ocloud-2.o2ims.example.com",
			Username: "admin",
			Password: "secret456",
		},
	}
}

func (f *TestFixtures) createCRFixtures() {
	now := metav1.Now()

	// Create OCloud CRs
	f.OCloudCRs = []*focomv1alpha1.OCloud{
		{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "focom.nephio.org/v1alpha1",
				Kind:       "OCloud",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:              "ocloud-1",
				Namespace:         "default",
				CreationTimestamp: now,
			},
			Spec: focomv1alpha1.OCloudSpec{
				O2imsSecret: focomv1alpha1.O2imsSecret{
					SecretRef: focomv1alpha1.SecretRef{
						Name:      "ocloud-1-secret",
						Namespace: "default",
					},
				},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "focom.nephio.org/v1alpha1",
				Kind:       "OCloud",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:              "ocloud-2",
				Namespace:         "default",
				CreationTimestamp: now,
			},
			Spec: focomv1alpha1.OCloudSpec{
				O2imsSecret: focomv1alpha1.O2imsSecret{
					SecretRef: focomv1alpha1.SecretRef{
						Name:      "ocloud-2-secret",
						Namespace: "default",
					},
				},
			},
		},
	}

	// Create TemplateInfo CRs
	f.TemplateInfoCRs = []*provisioningv1alpha1.TemplateInfo{
		{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "provisioning.nephio.org/v1alpha1",
				Kind:       "TemplateInfo",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:              "template-1",
				Namespace:         "default",
				CreationTimestamp: now,
			},
			Spec: provisioningv1alpha1.TemplateInfoSpec{
				TemplateName:            "basic-cluster",
				TemplateVersion:         "v1.0.0",
				TemplateParameterSchema: f.TemplateInfos[0].TemplateParameterSchema,
			},
		},
		{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "provisioning.nephio.org/v1alpha1",
				Kind:       "TemplateInfo",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:              "template-2",
				Namespace:         "default",
				CreationTimestamp: now,
			},
			Spec: provisioningv1alpha1.TemplateInfoSpec{
				TemplateName:            "advanced-cluster",
				TemplateVersion:         "v2.1.0",
				TemplateParameterSchema: f.TemplateInfos[1].TemplateParameterSchema,
			},
		},
	}

	// Create FocomProvisioningRequest CRs
	templateParams1, _ := json.Marshal(f.FocomProvisioningRequests[0].TemplateParameters)
	templateParams2, _ := json.Marshal(f.FocomProvisioningRequests[1].TemplateParameters)

	f.FocomProvisioningRequestCRs = []*focomv1alpha1.FocomProvisioningRequest{
		{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "focom.nephio.org/v1alpha1",
				Kind:       "FocomProvisioningRequest",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:              "fpr-1",
				Namespace:         "default",
				CreationTimestamp: now,
			},
			Spec: focomv1alpha1.FocomProvisioningRequestSpec{
				OCloudId:           "ocloud-1",
				OCloudNamespace:    "default",
				Name:               "fpr-1",
				Description:        "Test cluster provisioning request",
				TemplateName:       "basic-cluster",
				TemplateVersion:    "v1.0.0",
				TemplateParameters: runtime.RawExtension{Raw: templateParams1},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "focom.nephio.org/v1alpha1",
				Kind:       "FocomProvisioningRequest",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:              "fpr-2",
				Namespace:         "default",
				CreationTimestamp: now,
			},
			Spec: focomv1alpha1.FocomProvisioningRequestSpec{
				OCloudId:           "ocloud-2",
				OCloudNamespace:    "default",
				Name:               "fpr-2",
				Description:        "Production cluster provisioning request",
				TemplateName:       "advanced-cluster",
				TemplateVersion:    "v2.1.0",
				TemplateParameters: runtime.RawExtension{Raw: templateParams2},
			},
		},
	}
}

// GetOCloudByID returns an OCloud by ID
func (f *TestFixtures) GetOCloudByID(id string) *models.OCloudData {
	for _, ocloud := range f.OClouds {
		if ocloud.ID == id {
			return ocloud
		}
	}
	return nil
}

// GetTemplateInfoByID returns a TemplateInfo by ID
func (f *TestFixtures) GetTemplateInfoByID(id string) *models.TemplateInfoData {
	for _, template := range f.TemplateInfos {
		if template.ID == id {
			return template
		}
	}
	return nil
}

// GetFocomProvisioningRequestByID returns a FocomProvisioningRequest by ID
func (f *TestFixtures) GetFocomProvisioningRequestByID(id string) *models.FocomProvisioningRequestData {
	for _, fpr := range f.FocomProvisioningRequests {
		if fpr.ID == id {
			return fpr
		}
	}
	return nil
}

// GetDraftByResourceID returns a draft by resource ID
func (f *TestFixtures) GetDraftByResourceID(resourceID string) *models.DraftStorage {
	for _, draft := range f.DraftResources {
		if draft.ResourceID == resourceID {
			return draft
		}
	}
	return nil
}

// GetRevisionsByResourceID returns all revisions for a resource ID
func (f *TestFixtures) GetRevisionsByResourceID(resourceID string) []*models.RevisionStorage {
	var revisions []*models.RevisionStorage
	for _, revision := range f.RevisionResources {
		if revision.ResourceID == resourceID {
			revisions = append(revisions, revision)
		}
	}
	return revisions
}

// GetApprovedResources returns all approved resources of a given type
func (f *TestFixtures) GetApprovedResources(resourceType storage.ResourceType) []interface{} {
	var resources []interface{}

	switch resourceType {
	case storage.ResourceTypeOCloud:
		for _, ocloud := range f.OClouds {
			if ocloud.State == models.StateApproved {
				resources = append(resources, ocloud)
			}
		}
	case storage.ResourceTypeTemplateInfo:
		for _, template := range f.TemplateInfos {
			if template.State == models.StateApproved {
				resources = append(resources, template)
			}
		}
	case storage.ResourceTypeFocomProvisioningRequest:
		for _, fpr := range f.FocomProvisioningRequests {
			if fpr.State == models.StateApproved {
				resources = append(resources, fpr)
			}
		}
	}

	return resources
}
