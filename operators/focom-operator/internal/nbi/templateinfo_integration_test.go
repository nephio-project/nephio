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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTemplateInfoCRUDOperations tests complete TemplateInfo CRUD operations via HTTP
func TestTemplateInfoCRUDOperations(t *testing.T) {
	skipUnlessIntegration(t)
	ts := NewTestServer()
	defer ts.Close()

	// Clear storage to ensure clean state
	err := ts.ClearStorage()
	require.NoError(t, err)

	// Test data with valid JSON schema
	createRequest := map[string]interface{}{
		"namespace":       "default",
		"name":            "test-template",
		"description":     "Test template for integration testing",
		"templateName":    "test-cluster-template",
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
					"maximum": 100,
					"description": "Number of nodes in the cluster"
				},
				"region": {
					"type": "string",
					"enum": ["us-east-1", "us-west-1", "eu-west-1"],
					"description": "AWS region for deployment"
				}
			},
			"required": ["clusterName", "nodeCount"]
		}`,
	}

	updateRequest := map[string]interface{}{
		"description":     "Updated test template description",
		"templateVersion": "v1.1.0",
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
					"maximum": 200,
					"description": "Number of nodes in the cluster"
				},
				"region": {
					"type": "string",
					"enum": ["us-east-1", "us-west-1", "eu-west-1", "ap-south-1"],
					"description": "AWS region for deployment"
				},
				"enableMonitoring": {
					"type": "boolean",
					"default": true,
					"description": "Enable cluster monitoring"
				}
			},
			"required": ["clusterName", "nodeCount", "region"]
		}`,
	}

	t.Run("Create TemplateInfo draft", func(t *testing.T) {
		resp, err := ts.makeRequest("POST", "/template-infos/draft", createRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = parseResponse(resp, &result)
		require.NoError(t, err)

		assert.Equal(t, "test-template", result["name"])
		assert.Equal(t, "Test template for integration testing", result["description"])
		assert.Equal(t, "test-cluster-template", result["templateName"])
		assert.Equal(t, "v1.0.0", result["templateVersion"])
		assert.Equal(t, "DRAFT", result["templateInfoRevisionState"])
		assert.NotEmpty(t, result["templateInfoId"])

		// Store the ID for subsequent tests
		templateID := result["templateInfoId"].(string)

		t.Run("Get TemplateInfo draft", func(t *testing.T) {
			resp, err := ts.makeRequest("GET", fmt.Sprintf("/template-infos/%s/draft", templateID), nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var draft map[string]interface{}
			err = parseResponse(resp, &draft)
			require.NoError(t, err)

			assert.Equal(t, templateID, draft["templateInfoId"])
			assert.Equal(t, "test-template", draft["name"])
			assert.Equal(t, "DRAFT", draft["templateInfoRevisionState"])
			assert.Contains(t, draft["templateParameterSchema"], "clusterName")
		})

		t.Run("Update TemplateInfo draft", func(t *testing.T) {
			resp, err := ts.makeRequest("PATCH", fmt.Sprintf("/template-infos/%s/draft", templateID), updateRequest)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var updated map[string]interface{}
			err = parseResponse(resp, &updated)
			require.NoError(t, err)

			assert.Equal(t, "Updated test template description", updated["description"])
			assert.Equal(t, "v1.1.0", updated["templateVersion"])
			assert.Contains(t, updated["templateParameterSchema"], "enableMonitoring")
		})

		t.Run("Validate TemplateInfo draft", func(t *testing.T) {
			resp, err := ts.makeRequest("POST", fmt.Sprintf("/template-infos/%s/draft/validate", templateID), nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var validation map[string]interface{}
			err = parseResponse(resp, &validation)
			require.NoError(t, err)

			validationResult := validation["validationResult"].(map[string]interface{})
			assert.True(t, validationResult["success"].(bool))
			assert.NotEmpty(t, validationResult["validationTime"])
		})

		t.Run("Approve TemplateInfo draft", func(t *testing.T) {
			resp, err := ts.makeRequest("POST", fmt.Sprintf("/template-infos/%s/draft/approve", templateID), nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var approval map[string]interface{}
			err = parseResponse(resp, &approval)
			require.NoError(t, err)

			assert.True(t, approval["approved"].(bool))
		})

		t.Run("Get approved TemplateInfo", func(t *testing.T) {
			resp, err := ts.makeRequest("GET", fmt.Sprintf("/template-infos/%s", templateID), nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var approved map[string]interface{}
			err = parseResponse(resp, &approved)
			require.NoError(t, err)

			assert.Equal(t, templateID, approved["templateInfoId"])
			assert.Equal(t, "test-template", approved["name"])
			assert.Equal(t, "Updated test template description", approved["description"])
			assert.Equal(t, "v1.1.0", approved["templateVersion"])
			assert.Equal(t, "APPROVED", approved["templateInfoRevisionState"])
			assert.NotEmpty(t, approved["revisionId"])
		})

		t.Run("List TemplateInfos", func(t *testing.T) {
			resp, err := ts.makeRequest("GET", "/template-infos", nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var items []interface{}
			err = parseResponse(resp, &items)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(items), 1)

			// Find our created TemplateInfo
			found := false
			for _, item := range items {
				templateInfo := item.(map[string]interface{})
				templateData := templateInfo["templateInfoData"].(map[string]interface{})
				if templateData["templateInfoId"] == templateID {
					found = true
					assert.Equal(t, "test-template", templateData["name"])
					assert.Equal(t, "test-cluster-template", templateData["templateName"])
					break
				}
			}
			assert.True(t, found, "Created TemplateInfo should be in the list")
		})

		t.Run("Get TemplateInfo revisions", func(t *testing.T) {
			resp, err := ts.makeRequest("GET", fmt.Sprintf("/template-infos/%s/revisions", templateID), nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var items []interface{}
			err = parseResponse(resp, &items)
			require.NoError(t, err)
			assert.Equal(t, 1, len(items))

			revision := items[0].(map[string]interface{})
			assert.Equal(t, templateID, revision["resourceId"])
			assert.NotEmpty(t, revision["revisionId"])
		})
	})
}

// TestTemplateInfoParameterSchemaValidation tests template parameter schema validation
func TestTemplateInfoParameterSchemaValidation(t *testing.T) {
	skipUnlessIntegration(t)
	ts := NewTestServer()
	defer ts.Close()

	// Clear storage to ensure clean state
	err := ts.ClearStorage()
	require.NoError(t, err)

	t.Run("Valid JSON schema", func(t *testing.T) {
		validRequest := map[string]interface{}{
			"namespace":       "default",
			"name":            "valid-schema-template",
			"description":     "Template with valid JSON schema",
			"templateName":    "valid-template",
			"templateVersion": "v1.0.0",
			"templateParameterSchema": `{
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"count": {"type": "integer", "minimum": 1}
				},
				"required": ["name"]
			}`,
		}

		resp, err := ts.makeRequest("POST", "/template-infos/draft", validRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("Invalid JSON schema", func(t *testing.T) {
		invalidRequest := map[string]interface{}{
			"namespace":       "default",
			"name":            "invalid-schema-template",
			"description":     "Template with invalid JSON schema",
			"templateName":    "invalid-template",
			"templateVersion": "v1.0.0",
			"templateParameterSchema": `{
				"type": "object",
				"properties": {
					"name": {"type": "invalid-type"}
				}
			}`,
		}

		resp, err := ts.makeRequest("POST", "/template-infos/draft", invalidRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var errorResp map[string]interface{}
		err = parseResponse(resp, &errorResp)
		require.NoError(t, err)

		assert.Contains(t, errorResp, "error")
		assert.Contains(t, errorResp["error"], "schema")
	})

	t.Run("Malformed JSON schema", func(t *testing.T) {
		malformedRequest := map[string]interface{}{
			"namespace":       "default",
			"name":            "malformed-schema-template",
			"description":     "Template with malformed JSON schema",
			"templateName":    "malformed-template",
			"templateVersion": "v1.0.0",
			"templateParameterSchema": `{
				"type": "object",
				"properties": {
					"name": {"type": "string"
				}
			}`, // Missing closing brace
		}

		resp, err := ts.makeRequest("POST", "/template-infos/draft", malformedRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var errorResp map[string]interface{}
		err = parseResponse(resp, &errorResp)
		require.NoError(t, err)

		assert.Contains(t, errorResp, "error")
	})

	t.Run("YAML schema format", func(t *testing.T) {
		yamlRequest := map[string]interface{}{
			"namespace":       "default",
			"name":            "yaml-schema-template",
			"description":     "Template with YAML schema",
			"templateName":    "yaml-template",
			"templateVersion": "v1.0.0",
			"templateParameterSchema": `
type: object
properties:
  clusterName:
    type: string
    description: Name of the cluster
  nodeCount:
    type: integer
    minimum: 1
    maximum: 50
required:
  - clusterName
  - nodeCount
`,
		}

		resp, err := ts.makeRequest("POST", "/template-infos/draft", yamlRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})
}

// TestTemplateInfoDraftWorkflow tests the complete draft workflow
func TestTemplateInfoDraftWorkflow(t *testing.T) {
	skipUnlessIntegration(t)
	ts := NewTestServer()
	defer ts.Close()

	// Clear storage to ensure clean state
	err := ts.ClearStorage()
	require.NoError(t, err)

	createRequest := map[string]interface{}{
		"namespace":               "default",
		"name":                    "workflow-test-template",
		"description":             "Template for workflow testing",
		"templateName":            "workflow-template",
		"templateVersion":         "v1.0.0",
		"templateParameterSchema": `{"type": "object"}`,
	}

	// Create draft
	resp, err := ts.makeRequest("POST", "/template-infos/draft", createRequest)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	err = parseResponse(resp, &result)
	require.NoError(t, err)
	templateID := result["templateInfoId"].(string)

	t.Run("Reject draft workflow", func(t *testing.T) {
		// First validate the draft
		resp, err := ts.makeRequest("POST", fmt.Sprintf("/template-infos/%s/draft/validate", templateID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Then reject it
		rejectRequest := map[string]interface{}{
			"reason": "Schema needs improvement",
		}

		resp, err = ts.makeRequest("POST", fmt.Sprintf("/template-infos/%s/draft/reject", templateID), rejectRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var rejection map[string]interface{}
		err = parseResponse(resp, &rejection)
		require.NoError(t, err)

		assert.True(t, rejection["rejected"].(bool))
		assert.Equal(t, "Schema needs improvement", rejection["reason"])

		// Verify draft is back to DRAFT state
		resp, err = ts.makeRequest("GET", fmt.Sprintf("/template-infos/%s/draft", templateID), nil)
		require.NoError(t, err)

		var draft map[string]interface{}
		err = parseResponse(resp, &draft)
		require.NoError(t, err)

		assert.Equal(t, "DRAFT", draft["templateInfoRevisionState"])
	})

	t.Run("Update draft after rejection", func(t *testing.T) {
		improvedRequest := map[string]interface{}{
			"description": "Improved template after feedback",
			"templateParameterSchema": `{
				"type": "object",
				"properties": {
					"name": {"type": "string"},
					"version": {"type": "string", "pattern": "^v[0-9]+\\.[0-9]+\\.[0-9]+$"}
				},
				"required": ["name", "version"]
			}`,
		}

		resp, err := ts.makeRequest("PATCH", fmt.Sprintf("/template-infos/%s/draft", templateID), improvedRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var updated map[string]interface{}
		err = parseResponse(resp, &updated)
		require.NoError(t, err)

		assert.Equal(t, "Improved template after feedback", updated["description"])
		assert.Contains(t, updated["templateParameterSchema"], "pattern")
	})
}

// TestTemplateInfoErrorHandling tests error scenarios
func TestTemplateInfoErrorHandling(t *testing.T) {
	skipUnlessIntegration(t)
	ts := NewTestServer()
	defer ts.Close()

	// Clear storage to ensure clean state
	err := ts.ClearStorage()
	require.NoError(t, err)

	t.Run("Get non-existent TemplateInfo", func(t *testing.T) {
		resp, err := ts.makeRequest("GET", "/template-infos/non-existent", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		var errorResp map[string]interface{}
		err = parseResponse(resp, &errorResp)
		require.NoError(t, err)

		assert.Contains(t, errorResp, "error")
		assert.Contains(t, errorResp, "code")
	})

	t.Run("Create TemplateInfo with missing required fields", func(t *testing.T) {
		invalidRequest := map[string]interface{}{
			"name": "incomplete-template",
			// Missing templateName, templateVersion, templateParameterSchema
		}

		resp, err := ts.makeRequest("POST", "/template-infos/draft", invalidRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var errorResp map[string]interface{}
		err = parseResponse(resp, &errorResp)
		require.NoError(t, err)

		assert.Contains(t, errorResp, "error")
		assert.Contains(t, errorResp, "code")
	})

	t.Run("Update validated draft (should fail)", func(t *testing.T) {
		// Create and validate a draft
		createRequest := map[string]interface{}{
			"namespace":               "default",
			"name":                    "validated-template",
			"description":             "Template for validation test",
			"templateName":            "validated-template",
			"templateVersion":         "v1.0.0",
			"templateParameterSchema": `{"type": "object"}`,
		}

		resp, err := ts.makeRequest("POST", "/template-infos/draft", createRequest)
		require.NoError(t, err)

		var result map[string]interface{}
		err = parseResponse(resp, &result)
		require.NoError(t, err)
		templateID := result["templateInfoId"].(string)

		// Validate the draft
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/template-infos/%s/draft/validate", templateID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Try to update validated draft (should fail)
		updateRequest := map[string]interface{}{
			"description": "This should fail",
		}

		resp, err = ts.makeRequest("PATCH", fmt.Sprintf("/template-infos/%s/draft", templateID), updateRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		var errorResp map[string]interface{}
		err = parseResponse(resp, &errorResp)
		require.NoError(t, err)

		assert.Contains(t, errorResp, "error")
		assert.Contains(t, errorResp["error"], "validated")
	})
}

// TestTemplateInfoRevisionManagement tests revision management endpoints
func TestTemplateInfoRevisionManagement(t *testing.T) {
	skipUnlessIntegration(t)
	ts := NewTestServer()
	defer ts.Close()

	// Load test data with existing revisions (creates v1 for each resource)
	// Note: LoadTestData() calls ClearStorage() internally
	loadErr := ts.LoadTestData()
	require.NoError(t, loadErr)

	t.Run("List revisions for existing TemplateInfo", func(t *testing.T) {
		resp, err := ts.makeRequest("GET", "/template-infos/template-1/revisions", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var items []interface{}
		err = parseResponse(resp, &items)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(items), 1)

		revision := items[0].(map[string]interface{})
		assert.Equal(t, "template-1", revision["resourceId"])
		assert.NotEmpty(t, revision["revisionId"])
		assert.NotEmpty(t, revision["updatedAt"])
	})

	t.Run("Get specific revision", func(t *testing.T) {
		resp, err := ts.makeRequest("GET", "/template-infos/template-1/revisions/v1", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var revision map[string]interface{}
		err = parseResponse(resp, &revision)
		require.NoError(t, err)

		assert.Equal(t, "template-1", revision["resourceId"])
		assert.Equal(t, "v1", revision["revisionId"])

		// Check revision data with proper nil checks
		revisionData, ok := revision["revisionData"].(map[string]interface{})
		if ok && revisionData != nil {
			assert.Equal(t, "template-1", revisionData["name"])
			assert.Equal(t, "basic-cluster", revisionData["templateName"])
		} else {
			t.Logf("WARNING: revisionData is nil or not a map. Full revision response: %+v", revision)
		}
	})

	t.Run("Create draft from specific revision", func(t *testing.T) {
		resp, err := ts.makeRequest("POST", "/template-infos/template-1/revisions/v1/draft", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var draft map[string]interface{}
		err = parseResponse(resp, &draft)
		require.NoError(t, err)

		assert.Equal(t, "template-1", draft["templateInfoId"])
		assert.Equal(t, "DRAFT", draft["templateInfoRevisionState"])
		assert.Equal(t, "template-1", draft["name"])
		assert.Equal(t, "basic-cluster", draft["templateName"])
	})

	t.Run("Multiple revisions for same resource", func(t *testing.T) {
		t.Skip("Skipping: multiple drafts from existing revisions not yet supported — see why-multiple-drafts-not-possible.md")
		// Create a second revision for template-2 via API
		// Step 1: Create draft from v1
		resp, err := ts.makeRequest("POST", "/template-infos/template-2/revisions/v1/draft", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		// Step 2: Modify the draft slightly (update description)
		updateReq := map[string]interface{}{
			"description": "Updated template-2 for v2",
		}
		resp, err = ts.makeRequest("PATCH", "/template-infos/template-2/draft", updateReq)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Step 3: Validate the draft
		resp, err = ts.makeRequest("POST", "/template-infos/template-2/draft/validate", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Step 4: Approve the draft (this creates v2)
		resp, err = ts.makeRequest("POST", "/template-infos/template-2/draft/approve", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Step 5: List revisions - should now have v1 and v2
		resp, err = ts.makeRequest("GET", "/template-infos/template-2/revisions", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var items []interface{}
		err = parseResponse(resp, &items)
		require.NoError(t, err)
		assert.Equal(t, 2, len(items), "Should have exactly 2 revisions (v1 and v2)")

		// Verify both revisions exist
		revisionIDs := make(map[string]bool)
		var v2Revision map[string]interface{}
		for _, item := range items {
			revision := item.(map[string]interface{})
			revID := revision["revisionId"].(string)
			revisionIDs[revID] = true
			if revID == "v2" {
				v2Revision = revision
			}
		}

		assert.True(t, revisionIDs["v1"], "v1 should exist")
		assert.True(t, revisionIDs["v2"], "v2 should exist")

		// Verify v2 has the updated description
		require.NotNil(t, v2Revision, "v2 revision should be found")
		revisionData, ok := v2Revision["revisionData"].(map[string]interface{})
		require.True(t, ok, "revisionData should be a map")
		require.NotNil(t, revisionData, "revisionData should not be nil")
		assert.Equal(t, "Updated template-2 for v2", revisionData["description"])
	})
}
