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

package nbi

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skipUnlessStability skips the test unless FOCOM_STABILITY_TESTS=true is set.
// These tests involve many create-validate-approve cycles against Porch and can take 30+ minutes.
func skipUnlessStability(t *testing.T) {
	t.Helper()
	if os.Getenv("FOCOM_STABILITY_TESTS") != "true" {
		t.Skip("Skipping stability test — set FOCOM_STABILITY_TESTS=true to run")
	}
}

// TestCompleteResourceCreationOrder tests the complete resource creation order (OCloud → TemplateInfo → FPR)
func TestCompleteResourceCreationOrder(t *testing.T) {
	skipUnlessStability(t)
	ts := NewTestServer()
	defer ts.Close()

	// Clear any existing resources before starting
	err := ts.ClearStorage()
	require.NoError(t, err)

	var ocloudID, templateID, fprID string

	t.Run("Step 1: Create and approve OCloud", func(t *testing.T) {
		ocloudRequest := map[string]interface{}{
			"namespace":   "default",
			"name":        "integration-ocloud",
			"description": "OCloud for cross-resource integration testing",
			"o2imsSecret": map[string]interface{}{
				"secretRef": map[string]interface{}{
					"name":      "integration-ocloud-secret",
					"namespace": "default",
				},
			},
		}

		// Create OCloud draft
		resp, err := ts.makeRequest("POST", "/o-clouds/draft", ocloudRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = parseResponse(resp, &result)
		require.NoError(t, err)
		ocloudID = result["oCloudId"].(string)

		// Validate OCloud
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/o-clouds/%s/draft/validate", ocloudID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Approve OCloud
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/o-clouds/%s/draft/approve", ocloudID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify OCloud is approved
		resp, err = ts.makeRequest("GET", fmt.Sprintf("/o-clouds/%s", ocloudID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var approved map[string]interface{}
		err = parseResponse(resp, &approved)
		require.NoError(t, err)
		assert.Equal(t, "APPROVED", approved["oCloudRevisionState"])
	})

	t.Run("Step 2: Create and approve TemplateInfo", func(t *testing.T) {
		templateRequest := map[string]interface{}{
			"namespace":       "default",
			"name":            "integration-template",
			"description":     "Template for cross-resource integration testing",
			"templateName":    "integration-cluster-template",
			"templateVersion": "v1.0.0",
			"templateParameterSchema": `{
				"type": "object",
				"properties": {
					"clusterName": {
						"type": "string",
						"description": "Name of the cluster"
					},
					"nodeCount": {
						"type": "integer",
						"minimum": 1,
						"maximum": 20,
						"description": "Number of nodes"
					},
					"environment": {
						"type": "string",
						"enum": ["dev", "staging", "prod"],
						"description": "Environment type"
					}
				},
				"required": ["clusterName", "nodeCount", "environment"]
			}`,
		}

		// Create TemplateInfo draft
		resp, err := ts.makeRequest("POST", "/template-infos/draft", templateRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = parseResponse(resp, &result)
		require.NoError(t, err)
		templateID = result["templateInfoId"].(string)

		// Validate TemplateInfo
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/template-infos/%s/draft/validate", templateID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Approve TemplateInfo
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/template-infos/%s/draft/approve", templateID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify TemplateInfo is approved
		resp, err = ts.makeRequest("GET", fmt.Sprintf("/template-infos/%s", templateID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var approved map[string]interface{}
		err = parseResponse(resp, &approved)
		require.NoError(t, err)
		assert.Equal(t, "APPROVED", approved["templateInfoRevisionState"])
	})

	t.Run("Step 3: Create and approve FocomProvisioningRequest", func(t *testing.T) {
		fprRequest := map[string]interface{}{
			"namespace":       "default",
			"name":            "integration-fpr",
			"description":     "FPR for cross-resource integration testing",
			"oCloudId":        ocloudID,
			"oCloudNamespace": "default",
			"templateName":    "integration-cluster-template",
			"templateVersion": "v1.0.0",
			"templateParameters": map[string]interface{}{
				"clusterName": "integration-test-cluster",
				"nodeCount":   3,
				"environment": "dev",
			},
		}

		// Create FPR draft
		resp, err := ts.makeRequest("POST", "/focom-provisioning-requests/draft", fprRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = parseResponse(resp, &result)
		require.NoError(t, err)
		fprID = result["provisioningRequestId"].(string)

		// Validate FPR
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/focom-provisioning-requests/%s/draft/validate", fprID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Approve FPR (should succeed because dependencies exist)
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/focom-provisioning-requests/%s/draft/approve", fprID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify FPR is approved
		resp, err = ts.makeRequest("GET", fmt.Sprintf("/focom-provisioning-requests/%s", fprID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var approved map[string]interface{}
		err = parseResponse(resp, &approved)
		require.NoError(t, err)
		assert.Equal(t, "APPROVED", approved["focomProvisioningRequestRevisionState"])
	})

	t.Run("Step 4: Verify all resources are listed", func(t *testing.T) {
		// Check OCloud list
		resp, err := ts.makeRequest("GET", "/o-clouds", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var ocloudItems []interface{}
		err = parseResponse(resp, &ocloudItems)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(ocloudItems), 1)

		// Check TemplateInfo list
		resp, err = ts.makeRequest("GET", "/template-infos", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var templateItems []interface{}
		err = parseResponse(resp, &templateItems)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(templateItems), 1)

		// Check FPR list
		resp, err = ts.makeRequest("GET", "/focom-provisioning-requests", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var fprItems []interface{}
		err = parseResponse(resp, &fprItems)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, len(fprItems), 1)
	})
}

// TestDependencyValidationAcrossResourceTypes tests dependency validation across resource types
func TestDependencyValidationAcrossResourceTypes(t *testing.T) {
	skipUnlessStability(t)
	ts := NewTestServer()
	defer ts.Close()

	// Load some test data for existing dependencies
	err := ts.LoadTestData()
	require.NoError(t, err)

	t.Run("FPR cannot reference non-existent OCloud", func(t *testing.T) {
		fprRequest := map[string]interface{}{
			"namespace":       "default",
			"name":            "invalid-ocloud-fpr",
			"description":     "FPR referencing non-existent OCloud",
			"oCloudId":        "non-existent-ocloud-id",
			"oCloudNamespace": "default",
			"templateName":    "basic-cluster", // This exists in test data
			"templateVersion": "v1.0.0",
			"templateParameters": map[string]interface{}{
				"clusterName": "invalid-cluster",
				"nodeCount":   3,
			},
		}

		// Create draft
		resp, err := ts.makeRequest("POST", "/focom-provisioning-requests/draft", fprRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = parseResponse(resp, &result)
		require.NoError(t, err)
		fprID := result["provisioningRequestId"].(string)

		// Validate
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/focom-provisioning-requests/%s/draft/validate", fprID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Try to approve (should fail)
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/focom-provisioning-requests/%s/draft/approve", fprID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var errorResp map[string]interface{}
		err = parseResponse(resp, &errorResp)
		require.NoError(t, err)

		assert.Contains(t, errorResp, "error")
		assert.Contains(t, errorResp["error"], "OCloud")
		assert.Contains(t, errorResp["error"], "non-existent-ocloud-id")
	})

	t.Run("FPR cannot reference non-existent TemplateInfo", func(t *testing.T) {
		fprRequest := map[string]interface{}{
			"namespace":       "default",
			"name":            "invalid-template-fpr",
			"description":     "FPR referencing non-existent TemplateInfo",
			"oCloudId":        "ocloud-1", // This exists in test data
			"oCloudNamespace": "default",
			"templateName":    "non-existent-template",
			"templateVersion": "v1.0.0",
			"templateParameters": map[string]interface{}{
				"clusterName": "invalid-cluster",
				"nodeCount":   3,
			},
		}

		// Create draft
		resp, err := ts.makeRequest("POST", "/focom-provisioning-requests/draft", fprRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = parseResponse(resp, &result)
		require.NoError(t, err)
		fprID := result["provisioningRequestId"].(string)

		// Validate - should fail because TemplateInfo doesn't exist
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/focom-provisioning-requests/%s/draft/validate", fprID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var validateResp map[string]interface{}
		err = parseResponse(resp, &validateResp)
		require.NoError(t, err)
		assert.Contains(t, validateResp, "error")

		// Try to approve (should also fail since draft is not validated)
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/focom-provisioning-requests/%s/draft/approve", fprID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("FPR with valid dependencies can be approved", func(t *testing.T) {
		fprRequest := map[string]interface{}{
			"namespace":       "default",
			"name":            "valid-dependencies-fpr",
			"description":     "FPR with valid dependencies",
			"oCloudId":        "ocloud-1", // Exists in test data
			"oCloudNamespace": "default",
			"templateName":    "basic-cluster", // Exists in test data
			"templateVersion": "v1.0.0",
			"templateParameters": map[string]interface{}{
				"clusterName": "valid-cluster",
				"nodeCount":   3,
			},
		}

		// Create draft
		resp, err := ts.makeRequest("POST", "/focom-provisioning-requests/draft", fprRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = parseResponse(resp, &result)
		require.NoError(t, err)
		fprID := result["provisioningRequestId"].(string)

		// Validate
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/focom-provisioning-requests/%s/draft/validate", fprID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Approve (should succeed)
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/focom-provisioning-requests/%s/draft/approve", fprID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var approval map[string]interface{}
		err = parseResponse(resp, &approval)
		require.NoError(t, err)

		assert.True(t, approval["approved"].(bool))
	})
}

// TestDeletionPreventionForReferencedResources tests deletion prevention for referenced resources
func TestDeletionPreventionForReferencedResources(t *testing.T) {
	skipUnlessStability(t)
	ts := NewTestServer()
	defer ts.Close()

	// Load test data which includes FPRs that reference OClouds and TemplateInfos
	err := ts.LoadTestData()
	require.NoError(t, err)

	t.Run("Cannot delete OCloud referenced by approved FPR", func(t *testing.T) {
		// Try to delete ocloud-1 which is referenced by fpr-1 in test data
		resp, err := ts.makeRequest("DELETE", "/o-clouds/ocloud-1", nil)
		require.NoError(t, err)

		// Should return 409 Conflict due to existing references
		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		var errorResp map[string]interface{}
		err = parseResponse(resp, &errorResp)
		require.NoError(t, err)

		assert.Contains(t, errorResp, "error")
		assert.Contains(t, errorResp["error"], "referenced")
		assert.Contains(t, errorResp["error"], "FocomProvisioningRequest")
	})

	t.Run("Cannot delete TemplateInfo referenced by approved FPR", func(t *testing.T) {
		// Try to delete template-1 which is referenced by fpr-1 in test data
		resp, err := ts.makeRequest("DELETE", "/template-infos/template-1", nil)
		require.NoError(t, err)

		// Should return 409 Conflict due to existing references
		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		var errorResp map[string]interface{}
		err = parseResponse(resp, &errorResp)
		require.NoError(t, err)

		assert.Contains(t, errorResp, "error")
		assert.Contains(t, errorResp["error"], "referenced")
		assert.Contains(t, errorResp["error"], "FocomProvisioningRequest")
	})

	t.Run("Can delete unreferenced OCloud", func(t *testing.T) {
		// Create a new OCloud that won't be referenced
		ocloudRequest := map[string]interface{}{
			"namespace":   "default",
			"name":        "unreferenced-ocloud",
			"description": "OCloud that won't be referenced",
			"o2imsSecret": map[string]interface{}{
				"secretRef": map[string]interface{}{
					"name":      "unreferenced-secret",
					"namespace": "default",
				},
			},
		}

		// Create and approve the OCloud
		resp, err := ts.makeRequest("POST", "/o-clouds/draft", ocloudRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = parseResponse(resp, &result)
		require.NoError(t, err)
		ocloudID := result["oCloudId"].(string)

		// Validate and approve
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/o-clouds/%s/draft/validate", ocloudID), nil)
		require.NoError(t, err)

		resp, err = ts.makeRequest("POST", fmt.Sprintf("/o-clouds/%s/draft/approve", ocloudID), nil)
		require.NoError(t, err)

		// Now delete it (should succeed)
		resp, err = ts.makeRequest("DELETE", fmt.Sprintf("/o-clouds/%s", ocloudID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	})

	t.Run("Can delete unreferenced TemplateInfo", func(t *testing.T) {
		// Create a new TemplateInfo that won't be referenced
		templateRequest := map[string]interface{}{
			"namespace":               "default",
			"name":                    "unreferenced-template",
			"description":             "Template that won't be referenced",
			"templateName":            "unreferenced-template",
			"templateVersion":         "v1.0.0",
			"templateParameterSchema": `{"type": "object"}`,
		}

		// Create and approve the TemplateInfo
		resp, err := ts.makeRequest("POST", "/template-infos/draft", templateRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = parseResponse(resp, &result)
		require.NoError(t, err)
		templateID := result["templateInfoId"].(string)

		// Validate and approve
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/template-infos/%s/draft/validate", templateID), nil)
		require.NoError(t, err)

		resp, err = ts.makeRequest("POST", fmt.Sprintf("/template-infos/%s/draft/approve", templateID), nil)
		require.NoError(t, err)

		// Now delete it (should succeed)
		resp, err = ts.makeRequest("DELETE", fmt.Sprintf("/template-infos/%s", templateID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	})

	t.Run("Can delete FPR (triggers decommissioning)", func(t *testing.T) {
		// Create a new FPR that we can delete
		fprRequest := map[string]interface{}{
			"namespace":       "default",
			"name":            "deletable-fpr",
			"description":     "FPR that can be deleted",
			"oCloudId":        "ocloud-1", // Use existing OCloud
			"oCloudNamespace": "default",
			"templateName":    "basic-cluster", // Use existing template
			"templateVersion": "v1.0.0",
			"templateParameters": map[string]interface{}{
				"clusterName": "deletable-cluster",
				"nodeCount":   2,
			},
		}

		// Create and approve the FPR
		resp, err := ts.makeRequest("POST", "/focom-provisioning-requests/draft", fprRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = parseResponse(resp, &result)
		require.NoError(t, err)
		fprID := result["provisioningRequestId"].(string)

		// Validate and approve
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/focom-provisioning-requests/%s/draft/validate", fprID), nil)
		require.NoError(t, err)

		resp, err = ts.makeRequest("POST", fmt.Sprintf("/focom-provisioning-requests/%s/draft/approve", fprID), nil)
		require.NoError(t, err)

		// Now delete it (should succeed and trigger decommissioning)
		resp, err = ts.makeRequest("DELETE", fmt.Sprintf("/focom-provisioning-requests/%s", fprID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusAccepted, resp.StatusCode)

		var deleteResp map[string]interface{}
		err = parseResponse(resp, &deleteResp)
		require.NoError(t, err)

		assert.Contains(t, deleteResp, "message")
		assert.Contains(t, deleteResp["message"], "decommissioning")
	})
}

// TestCrossResourceWorkflowErrorMessages tests clear error messages for dependency violations
func TestCrossResourceWorkflowErrorMessages(t *testing.T) {
	skipUnlessStability(t)
	ts := NewTestServer()
	defer ts.Close()

	// Clear storage and load test data once at the beginning
	// (Don't reload in each subtest to avoid "already exists" errors)
	err := ts.LoadTestData()
	require.NoError(t, err)

	t.Run("Clear error message for missing OCloud", func(t *testing.T) {
		fprRequest := map[string]interface{}{
			"namespace":       "default",
			"name":            "missing-ocloud-test",
			"description":     "Test for missing OCloud error",
			"oCloudId":        "clearly-missing-ocloud",
			"oCloudNamespace": "default",
			"templateName":    "some-template",
			"templateVersion": "v1.0.0",
			"templateParameters": map[string]interface{}{
				"param": "value",
			},
		}

		// Create draft
		resp, err := ts.makeRequest("POST", "/focom-provisioning-requests/draft", fprRequest)
		require.NoError(t, err)

		var result map[string]interface{}
		err = parseResponse(resp, &result)
		require.NoError(t, err)
		fprID := result["provisioningRequestId"].(string)

		// Validate
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/focom-provisioning-requests/%s/draft/validate", fprID), nil)
		require.NoError(t, err)

		// Try to approve
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/focom-provisioning-requests/%s/draft/approve", fprID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var errorResp map[string]interface{}
		err = parseResponse(resp, &errorResp)
		require.NoError(t, err)

		assert.Contains(t, errorResp, "error")
		assert.Contains(t, errorResp, "code")
		assert.Contains(t, errorResp, "details")

		// Error message should be clear and specific
		errorMsg := errorResp["error"].(string)
		assert.Contains(t, errorMsg, "OCloud")
		assert.Contains(t, errorMsg, "clearly-missing-ocloud")
		assert.Contains(t, errorMsg, "not found")
	})

	t.Run("Clear error message for missing TemplateInfo", func(t *testing.T) {
		// Test data already loaded at test start

		fprRequest := map[string]interface{}{
			"namespace":       "default",
			"name":            "missing-template-test",
			"description":     "Test for missing TemplateInfo error",
			"oCloudId":        "ocloud-1", // Exists
			"oCloudNamespace": "default",
			"templateName":    "clearly-missing-template",
			"templateVersion": "v1.0.0",
			"templateParameters": map[string]interface{}{
				"param": "value",
			},
		}

		// Create draft
		resp, err := ts.makeRequest("POST", "/focom-provisioning-requests/draft", fprRequest)
		require.NoError(t, err)

		var result map[string]interface{}
		err = parseResponse(resp, &result)
		require.NoError(t, err)
		fprID := result["provisioningRequestId"].(string)

		// Validate
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/focom-provisioning-requests/%s/draft/validate", fprID), nil)
		require.NoError(t, err)

		// Try to approve
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/focom-provisioning-requests/%s/draft/approve", fprID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var errorResp map[string]interface{}
		err = parseResponse(resp, &errorResp)
		require.NoError(t, err)

		// Error message should be clear and specific
		errorMsg := errorResp["error"].(string)
		assert.Contains(t, errorMsg, "TemplateInfo")
		assert.Contains(t, errorMsg, "clearly-missing-template")
		assert.Contains(t, errorMsg, "v1.0.0")
		assert.Contains(t, errorMsg, "not found")
	})

	t.Run("Clear error message for deletion prevention", func(t *testing.T) {
		// Test data already loaded at test start

		// Try to delete referenced OCloud
		resp, err := ts.makeRequest("DELETE", "/o-clouds/ocloud-1", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		var errorResp map[string]interface{}
		err = parseResponse(resp, &errorResp)
		require.NoError(t, err)

		// Error message should be clear about what's preventing deletion
		errorMsg := errorResp["error"].(string)
		assert.Contains(t, errorMsg, "cannot be deleted")
		assert.Contains(t, errorMsg, "referenced by")
		assert.Contains(t, errorMsg, "FocomProvisioningRequest")

		// Should include details about which resources are referencing it
		if details, ok := errorResp["details"]; ok {
			detailsStr := details.(string)
			assert.Contains(t, detailsStr, "fpr-1") // From test data
		}
	})
}
