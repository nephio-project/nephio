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
	"testing"

	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// PERMANENT UNIT TESTS FOR PORCH STORAGE
// ============================================================================
//
// These tests focus on testing pure logic functions without HTTP mocking.
// They provide high code coverage for:
// - YAML creation and parsing
// - Data type conversions
// - Helper functions
// - Error handling
// - Edge cases
//
// Complex workflows (CreateDraft, ValidateDraft, etc.) are covered by
// integration tests that run against a real Porch instance.

// ============================================================================
// YAML CREATION TESTS
// ============================================================================

func TestCreateResourceYAML_OCloud_AllFields(t *testing.T) {
	storage := &PorchStorage{}

	ocloud := &OCloudData{
		BaseResource: BaseResource{
			ID:          "ocloud-001",
			Namespace:   "test-ns",
			Name:        "Production OCloud",
			Description: "Main production O-Cloud instance",
			State:       StateValidated,
		},
		O2IMSSecret: O2IMSSecretRef{
			SecretRef: SecretReference{
				Name:      "ocloud-secret",
				Namespace: "test-ns",
			},
		},
	}

	yaml, err := storage.createResourceYAML(ocloud, ResourceTypeOCloud)
	require.NoError(t, err)
	assert.NotEmpty(t, yaml)

	// Verify structure
	assert.Contains(t, yaml, "apiVersion: focom.nephio.org/v1alpha1")
	assert.Contains(t, yaml, "kind: OCloud")
	assert.Contains(t, yaml, "name: ocloud-001")
	assert.Contains(t, yaml, "namespace: test-ns")

	// Verify annotations (name and description stored here)
	assert.Contains(t, yaml, "focom.nephio.org/display-name: Production OCloud")
	assert.Contains(t, yaml, "focom.nephio.org/description: Main production O-Cloud instance")

	// Verify spec
	assert.Contains(t, yaml, "o2imsSecret:")
	assert.Contains(t, yaml, "name: ocloud-secret")

	// Verify state is NOT in YAML (not part of CRD spec)
	assert.NotContains(t, yaml, "state:")
	assert.NotContains(t, yaml, "VALIDATED")
}

func TestCreateResourceYAML_OCloud_MinimalFields(t *testing.T) {
	storage := &PorchStorage{}

	ocloud := &OCloudData{
		BaseResource: BaseResource{
			ID:        "ocloud-minimal",
			Namespace: "default",
		},
		O2IMSSecret: O2IMSSecretRef{},
	}

	yaml, err := storage.createResourceYAML(ocloud, ResourceTypeOCloud)
	require.NoError(t, err)
	assert.NotEmpty(t, yaml)

	assert.Contains(t, yaml, "name: ocloud-minimal")
	assert.Contains(t, yaml, "namespace: default")
	// Should not have annotations if name/description are empty
	assert.NotContains(t, yaml, "annotations:")
}

func TestCreateResourceYAML_TemplateInfo_AllFields(t *testing.T) {
	storage := &PorchStorage{}

	template := &TemplateInfoData{
		BaseResource: BaseResource{
			ID:          "template-001",
			Namespace:   "templates",
			Name:        "5G Core Template",
			Description: "Template for 5G Core deployment",
			State:       StateApproved,
		},
		TemplateName:            "5g-core-v1",
		TemplateVersion:         "v1.2.3",
		TemplateParameterSchema: `{"type":"object","properties":{"replicas":{"type":"integer"}}}`,
	}

	yaml, err := storage.createResourceYAML(template, ResourceTypeTemplateInfo)
	require.NoError(t, err)
	assert.NotEmpty(t, yaml)

	// Verify correct API group
	assert.Contains(t, yaml, "apiVersion: provisioning.oran.org/v1alpha1")
	assert.Contains(t, yaml, "kind: TemplateInfo")
	assert.Contains(t, yaml, "name: template-001")

	// Verify annotations
	assert.Contains(t, yaml, "focom.nephio.org/display-name: 5G Core Template")
	assert.Contains(t, yaml, "focom.nephio.org/description: Template for 5G Core deployment")

	// Verify spec
	assert.Contains(t, yaml, "templateName: 5g-core-v1")
	assert.Contains(t, yaml, "templateVersion: v1.2.3")
	assert.Contains(t, yaml, "templateParameterSchema:")
}

func TestCreateResourceYAML_FPR_AllFields(t *testing.T) {
	storage := &PorchStorage{}

	fpr := &FocomProvisioningRequestData{
		BaseResource: BaseResource{
			ID:          "fpr-001",
			Namespace:   "default",
			Name:        "Deploy 5G Core",
			Description: "Provision 5G core network",
			State:       StateDraft,
		},
		OCloudID:        "ocloud-001",
		OCloudNamespace: "default",
		TemplateName:    "5g-core-v1",
		TemplateVersion: "v1.2.3",
		TemplateParameters: map[string]interface{}{
			"replicas": 3,
			"region":   "us-west",
		},
	}

	yaml, err := storage.createResourceYAML(fpr, ResourceTypeFocomProvisioningRequest)
	require.NoError(t, err)
	assert.NotEmpty(t, yaml)

	assert.Contains(t, yaml, "apiVersion: focom.nephio.org/v1alpha1")
	assert.Contains(t, yaml, "kind: FocomProvisioningRequest")
	assert.Contains(t, yaml, "oCloudId: ocloud-001")
	assert.Contains(t, yaml, "templateName: 5g-core-v1")

	// FPR has name and description in spec
	assert.Contains(t, yaml, "name: Deploy 5G Core")
	assert.Contains(t, yaml, "description: Provision 5G core network")
}

// ============================================================================
// YAML PARSING TESTS
// ============================================================================

func TestParseResourceYAML_OCloud_WithAnnotations(t *testing.T) {
	storage := &PorchStorage{}

	yaml := `
apiVersion: focom.nephio.org/v1alpha1
kind: OCloud
metadata:
  name: ocloud-001
  namespace: production
  annotations:
    focom.nephio.org/display-name: Production OCloud
    focom.nephio.org/description: Main production instance
spec:
  o2imsSecret:
    secretRef:
      name: prod-secret
      namespace: production
`

	result, err := storage.parseResourceYAML(yaml, ResourceTypeOCloud)
	require.NoError(t, err)
	require.NotNil(t, result)

	ocloud, ok := result.(*OCloudData)
	require.True(t, ok)

	assert.Equal(t, "ocloud-001", ocloud.ID)
	assert.Equal(t, "production", ocloud.Namespace)
	assert.Equal(t, "Production OCloud", ocloud.Name)
	assert.Equal(t, "Main production instance", ocloud.Description)
	assert.Equal(t, "prod-secret", ocloud.O2IMSSecret.SecretRef.Name)
	assert.Equal(t, "production", ocloud.O2IMSSecret.SecretRef.Namespace)

	// State should be empty (set by caller based on PackageRevision)
	assert.Equal(t, ResourceState(""), ocloud.State)
}

func TestParseResourceYAML_OCloud_NoAnnotations(t *testing.T) {
	storage := &PorchStorage{}

	yaml := `
apiVersion: focom.nephio.org/v1alpha1
kind: OCloud
metadata:
  name: ocloud-simple
  namespace: default
spec:
  o2imsSecret:
    secretRef:
      name: simple-secret
      namespace: default
`

	result, err := storage.parseResourceYAML(yaml, ResourceTypeOCloud)
	require.NoError(t, err)

	ocloud := result.(*OCloudData)
	assert.Equal(t, "ocloud-simple", ocloud.ID)
	// Name defaults to ID when no display-name annotation
	assert.Equal(t, "ocloud-simple", ocloud.Name)
	assert.Equal(t, "", ocloud.Description)
}

func TestParseResourceYAML_TemplateInfo_Complete(t *testing.T) {
	storage := &PorchStorage{}

	yaml := `
apiVersion: provisioning.oran.org/v1alpha1
kind: TemplateInfo
metadata:
  name: template-001
  namespace: templates
  annotations:
    focom.nephio.org/display-name: 5G Template
    focom.nephio.org/description: Template for 5G deployment
spec:
  templateName: 5g-core
  templateVersion: v1.0.0
  templateParameterSchema: '{"type":"object"}'
`

	result, err := storage.parseResourceYAML(yaml, ResourceTypeTemplateInfo)
	require.NoError(t, err)

	template := result.(*TemplateInfoData)
	assert.Equal(t, "template-001", template.ID)
	assert.Equal(t, "templates", template.Namespace)
	assert.Equal(t, "5G Template", template.Name)
	assert.Equal(t, "Template for 5G deployment", template.Description)
	assert.Equal(t, "5g-core", template.TemplateName)
	assert.Equal(t, "v1.0.0", template.TemplateVersion)
	assert.Equal(t, `{"type":"object"}`, template.TemplateParameterSchema)
}

func TestParseResourceYAML_FPR_Complete(t *testing.T) {
	storage := &PorchStorage{}

	yaml := `
apiVersion: focom.nephio.org/v1alpha1
kind: FocomProvisioningRequest
metadata:
  name: fpr-001
  namespace: default
spec:
  name: Deploy Core
  description: Deploy 5G core
  oCloudId: ocloud-001
  oCloudNamespace: default
  templateName: 5g-core
  templateVersion: v1.0.0
  templateParameters:
    replicas: 3
`

	result, err := storage.parseResourceYAML(yaml, ResourceTypeFocomProvisioningRequest)
	require.NoError(t, err)

	fpr := result.(*FocomProvisioningRequestData)
	assert.Equal(t, "fpr-001", fpr.ID)
	assert.Equal(t, "Deploy Core", fpr.Name)
	assert.Equal(t, "Deploy 5G core", fpr.Description)
	assert.Equal(t, "ocloud-001", fpr.OCloudID)
	assert.Equal(t, "5g-core", fpr.TemplateName)
}

func TestParseResourceYAML_InvalidYAML_Permanent(t *testing.T) {
	storage := &PorchStorage{}

	yaml := `invalid: yaml: {`

	result, err := storage.parseResourceYAML(yaml, ResourceTypeOCloud)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to unmarshal YAML")
}

func TestParseResourceYAML_MissingSpec_Permanent(t *testing.T) {
	storage := &PorchStorage{}

	yaml := `
apiVersion: focom.nephio.org/v1alpha1
kind: OCloud
metadata:
  name: ocloud-001
`

	result, err := storage.parseResourceYAML(yaml, ResourceTypeOCloud)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "missing or invalid spec section")
}

// ============================================================================
// ROUND-TRIP TESTS
// ============================================================================

func TestRoundTrip_OCloud_PreservesData(t *testing.T) {
	storage := &PorchStorage{}

	original := &OCloudData{
		BaseResource: BaseResource{
			ID:          "ocloud-rt",
			Namespace:   "test",
			Name:        "RoundTrip Test",
			Description: "Testing round-trip",
		},
		O2IMSSecret: O2IMSSecretRef{
			SecretRef: SecretReference{
				Name:      "rt-secret",
				Namespace: "test",
			},
		},
	}

	// Create YAML
	yaml, err := storage.createResourceYAML(original, ResourceTypeOCloud)
	require.NoError(t, err)

	// Parse back
	result, err := storage.parseResourceYAML(yaml, ResourceTypeOCloud)
	require.NoError(t, err)

	parsed := result.(*OCloudData)
	assert.Equal(t, original.ID, parsed.ID)
	assert.Equal(t, original.Namespace, parsed.Namespace)
	assert.Equal(t, original.Name, parsed.Name)
	assert.Equal(t, original.Description, parsed.Description)
	assert.Equal(t, original.O2IMSSecret.SecretRef.Name, parsed.O2IMSSecret.SecretRef.Name)
}

func TestRoundTrip_TemplateInfo_PreservesData(t *testing.T) {
	storage := &PorchStorage{}

	original := &TemplateInfoData{
		BaseResource: BaseResource{
			ID:          "template-rt",
			Namespace:   "test",
			Name:        "RT Template",
			Description: "Round-trip template",
		},
		TemplateName:            "rt-template",
		TemplateVersion:         "v2.0.0",
		TemplateParameterSchema: `{"type":"object"}`,
	}

	yaml, err := storage.createResourceYAML(original, ResourceTypeTemplateInfo)
	require.NoError(t, err)

	result, err := storage.parseResourceYAML(yaml, ResourceTypeTemplateInfo)
	require.NoError(t, err)

	parsed := result.(*TemplateInfoData)
	assert.Equal(t, original.ID, parsed.ID)
	assert.Equal(t, original.Name, parsed.Name)
	assert.Equal(t, original.TemplateName, parsed.TemplateName)
	assert.Equal(t, original.TemplateVersion, parsed.TemplateVersion)
}

// ============================================================================
// HELPER FUNCTION TESTS
// ============================================================================

func TestExtractResourceID_OCloud(t *testing.T) {
	storage := &PorchStorage{}

	tests := []struct {
		name     string
		resource interface{}
		expected string
		wantErr  bool
	}{
		{
			name:     "OCloud pointer",
			resource: &OCloudData{BaseResource: BaseResource{ID: "ocloud-123"}},
			expected: "ocloud-123",
			wantErr:  false,
		},
		{
			name:     "OCloud value",
			resource: OCloudData{BaseResource: BaseResource{ID: "ocloud-456"}},
			expected: "ocloud-456",
			wantErr:  false,
		},
		{
			name:     "TemplateInfo pointer",
			resource: &TemplateInfoData{BaseResource: BaseResource{ID: "template-789"}},
			expected: "template-789",
			wantErr:  false,
		},
		{
			name:     "FPR pointer",
			resource: &FocomProvisioningRequestData{BaseResource: BaseResource{ID: "fpr-abc"}},
			expected: "fpr-abc",
			wantErr:  false,
		},
		{
			name:     "Models OCloud",
			resource: &models.OCloudData{BaseResource: models.BaseResource{ID: "model-123"}},
			expected: "model-123",
			wantErr:  false,
		},
		{
			name:     "Unsupported type",
			resource: "invalid",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := storage.extractResourceID(tt.resource)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, id)
			}
		})
	}
}

func TestExtractRevisionID_Success(t *testing.T) {
	storage := &PorchStorage{}

	tests := []struct {
		name     string
		resource interface{}
		expected string
	}{
		{
			name:     "OCloud with revision",
			resource: &OCloudData{BaseResource: BaseResource{RevisionID: "rev-123"}},
			expected: "rev-123",
		},
		{
			name:     "TemplateInfo with revision",
			resource: &TemplateInfoData{BaseResource: BaseResource{RevisionID: "rev-456"}},
			expected: "rev-456",
		},
		{
			name:     "Models OCloud with revision",
			resource: &models.OCloudData{BaseResource: models.BaseResource{RevisionID: "rev-789"}},
			expected: "rev-789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			revID, err := storage.extractRevisionID(tt.resource)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, revID)
		})
	}
}

func TestGenerateWorkspaceName_Permanent(t *testing.T) {
	storage := &PorchStorage{}

	tests := []struct {
		name       string
		resourceID string
	}{
		{"simple ID", "ocloud-001"},
		{"with dashes", "my-ocloud-test"},
		{"with numbers", "ocloud123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspace := storage.generateWorkspaceName(tt.resourceID)

			// Should start with "draft-"
			assert.Contains(t, workspace, "draft-")
			// Should contain the resource ID
			assert.Contains(t, workspace, tt.resourceID)
			// Should have a timestamp suffix
			assert.Regexp(t, `^draft-.+-\d+$`, workspace)
		})
	}
}

func TestSetResourceState(t *testing.T) {
	storage := &PorchStorage{}

	tests := []struct {
		name          string
		lifecycle     string
		expectedState ResourceState
	}{
		{"Draft lifecycle", PorchLifecycleDraft, StateDraft},
		{"Proposed lifecycle", PorchLifecycleProposed, StateValidated},
		{"Published lifecycle", PorchLifecyclePublished, StateApproved},
		{"Unknown lifecycle", "Unknown", StateDraft},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ocloud := &OCloudData{}
			storage.setResourceState(ocloud, tt.lifecycle)
			assert.Equal(t, tt.expectedState, ocloud.State)
		})
	}
}

func TestSetResourceRevisionID(t *testing.T) {
	storage := &PorchStorage{}

	tests := []struct {
		name       string
		resource   interface{}
		revisionID string
	}{
		{
			name:       "OCloud",
			resource:   &OCloudData{},
			revisionID: "rev-123",
		},
		{
			name:       "TemplateInfo",
			resource:   &TemplateInfoData{},
			revisionID: "rev-456",
		},
		{
			name:       "FPR",
			resource:   &FocomProvisioningRequestData{},
			revisionID: "rev-789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage.setResourceRevisionID(tt.resource, tt.revisionID)

			// Extract and verify
			revID, err := storage.extractRevisionID(tt.resource)
			assert.NoError(t, err)
			assert.Equal(t, tt.revisionID, revID)
		})
	}
}

// ============================================================================
// TYPE CONVERSION TESTS
// ============================================================================

func TestConvertToModelsType_OCloud(t *testing.T) {
	storage := &OCloudData{
		BaseResource: BaseResource{
			ID:          "ocloud-001",
			RevisionID:  "rev-123",
			Namespace:   "default",
			Name:        "Test OCloud",
			Description: "Test description",
			State:       StateApproved,
		},
		O2IMSSecret: O2IMSSecretRef{
			SecretRef: SecretReference{
				Name:      "secret-1",
				Namespace: "default",
			},
		},
	}

	result, err := convertToModelsType(storage)
	require.NoError(t, err)
	require.NotNil(t, result)

	model, ok := result.(*models.OCloudData)
	require.True(t, ok)

	assert.Equal(t, storage.ID, model.ID)
	assert.Equal(t, storage.RevisionID, model.RevisionID)
	assert.Equal(t, storage.Name, model.Name)
	assert.Equal(t, models.ResourceState(storage.State), model.State)
	assert.Equal(t, storage.O2IMSSecret.SecretRef.Name, model.O2IMSSecret.SecretRef.Name)
}

func TestConvertToModelsType_TemplateInfo(t *testing.T) {
	storage := &TemplateInfoData{
		BaseResource: BaseResource{
			ID:        "template-001",
			Namespace: "templates",
			Name:      "Test Template",
			State:     StateApproved,
		},
		TemplateName:            "test-template",
		TemplateVersion:         "v1.0.0",
		TemplateParameterSchema: `{"type":"object"}`,
	}

	result, err := convertToModelsType(storage)
	require.NoError(t, err)

	model := result.(*models.TemplateInfoData)
	assert.Equal(t, storage.ID, model.ID)
	assert.Equal(t, storage.TemplateName, model.TemplateName)
	assert.Equal(t, storage.TemplateVersion, model.TemplateVersion)
}

func TestConvertToStorageType_ModelsToStorage(t *testing.T) {
	storage := &PorchStorage{}

	model := &models.OCloudData{
		BaseResource: models.BaseResource{
			ID:        "ocloud-001",
			Namespace: "default",
			Name:      "Model OCloud",
		},
		O2IMSSecret: models.O2IMSSecretRef{
			SecretRef: models.SecretReference{
				Name:      "model-secret",
				Namespace: "default",
			},
		},
	}

	result := storage.convertToStorageType(model, ResourceTypeOCloud)
	require.NotNil(t, result)

	storageData, ok := result.(*OCloudData)
	require.True(t, ok)

	assert.Equal(t, model.ID, storageData.ID)
	assert.Equal(t, model.Name, storageData.Name)
	assert.Equal(t, model.O2IMSSecret.SecretRef.Name, storageData.O2IMSSecret.SecretRef.Name)
}

// ============================================================================
// KPTFILE CREATION TESTS
// ============================================================================

func TestCreateKptfile_OCloud(t *testing.T) {
	storage := &PorchStorage{}

	kptfile, err := storage.createKptfile("ocloud-001", ResourceTypeOCloud)
	require.NoError(t, err)
	assert.NotEmpty(t, kptfile)

	assert.Contains(t, kptfile, "apiVersion: kpt.dev/v1")
	assert.Contains(t, kptfile, "kind: Kptfile")
	assert.Contains(t, kptfile, "name: ocloud-001")
	assert.Contains(t, kptfile, "description: FOCOM ocloud resource")
}

func TestCreateKptfile_TemplateInfo(t *testing.T) {
	storage := &PorchStorage{}

	kptfile, err := storage.createKptfile("template-001", ResourceTypeTemplateInfo)
	require.NoError(t, err)

	assert.Contains(t, kptfile, "name: template-001")
	assert.Contains(t, kptfile, "description: FOCOM templateinfo resource")
}

func TestCreateKptfile_FPR(t *testing.T) {
	storage := &PorchStorage{}

	kptfile, err := storage.createKptfile("fpr-001", ResourceTypeFocomProvisioningRequest)
	require.NoError(t, err)

	assert.Contains(t, kptfile, "name: fpr-001")
	assert.Contains(t, kptfile, "description: FOCOM focomprovisioningrequest resource")
}

// ============================================================================
// EDGE CASE TESTS
// ============================================================================

func TestCreateResourceYAML_EmptySecretRef(t *testing.T) {
	storage := &PorchStorage{}

	ocloud := &OCloudData{
		BaseResource: BaseResource{
			ID:        "ocloud-empty",
			Namespace: "default",
		},
		O2IMSSecret: O2IMSSecretRef{
			SecretRef: SecretReference{
				Name:      "",
				Namespace: "",
			},
		},
	}

	yaml, err := storage.createResourceYAML(ocloud, ResourceTypeOCloud)
	require.NoError(t, err)

	// Should still create valid YAML with empty secret fields
	assert.Contains(t, yaml, "o2imsSecret:")
	assert.Contains(t, yaml, `name: ""`)
}

func TestParseResourceYAML_EmptyTemplateParameters(t *testing.T) {
	storage := &PorchStorage{}

	yaml := `
apiVersion: focom.nephio.org/v1alpha1
kind: FocomProvisioningRequest
metadata:
  name: fpr-empty
  namespace: default
spec:
  oCloudId: ocloud-001
  oCloudNamespace: default
  templateName: test
  templateVersion: v1.0.0
  templateParameters: {}
`

	result, err := storage.parseResourceYAML(yaml, ResourceTypeFocomProvisioningRequest)
	require.NoError(t, err)

	fpr := result.(*FocomProvisioningRequestData)
	assert.NotNil(t, fpr.TemplateParameters)
	assert.Empty(t, fpr.TemplateParameters)
}

func TestExtractResourceID_NilResource(t *testing.T) {
	storage := &PorchStorage{}

	id, err := storage.extractResourceID(nil)
	assert.Error(t, err)
	assert.Empty(t, id)
	assert.Contains(t, err.Error(), "resource is nil")
}
