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

package storage

import (
	"time"

	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/models"
)

// Helper functions for creating test data - shared across test files

func createTestOCloudData(id string) *models.OCloudData {
	ocloud := models.NewOCloudData(
		"default",
		id,
		"Test OCloud",
		models.O2IMSSecretRef{
			SecretRef: models.SecretReference{
				Name:      "o2ims-secret",
				Namespace: "default",
			},
		},
	)
	// Override the auto-generated ID with the provided one for testing
	ocloud.ID = id
	return ocloud
}

func createTestTemplateInfoData(id string) *models.TemplateInfoData {
	template := models.NewTemplateInfoData(
		"default",
		id,
		"Test Template",
		"test-template",
		"v1.0.0",
		`{"type": "object"}`,
	)
	// Override the auto-generated ID with the provided one for testing
	template.ID = id
	return template
}

func createTestFocomProvisioningRequestData(id string) *models.FocomProvisioningRequestData {
	fpr := models.NewFocomProvisioningRequestData(
		"default",
		id,
		"Test FPR",
		"ocloud-1",
		"default",
		"test-template",
		"v1.0.0",
		map[string]interface{}{"param1": "value1"},
	)
	// Override the auto-generated ID with the provided one for testing
	fpr.ID = id
	return fpr
}

func createTestDraftResource(id string, resourceType ResourceType) *DraftResource {
	return &DraftResource{
		ResourceID:   id,
		ResourceType: resourceType,
		DraftData:    createTestOCloudData(id),
		State:        StateDraft,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func createTestRevisionResource(resourceID, revisionID string, resourceType ResourceType) *RevisionResource {
	return &RevisionResource{
		ResourceID:   resourceID,
		RevisionID:   revisionID,
		ResourceType: resourceType,
		RevisionData: createTestOCloudData(resourceID),
		ApprovedAt:   time.Now(),
	}
}
