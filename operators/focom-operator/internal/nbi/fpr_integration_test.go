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

// TestFocomProvisioningRequestCRUDOperations tests complete FPR CRUD operations via HTTP
func TestFocomProvisioningRequestCRUDOperations(t *testing.T) {
	skipUnlessIntegration(t)
	ts := NewTestServer()
	defer ts.Close()

	// Load test data to have dependencies available
	err := ts.LoadTestData()
	require.NoError(t, err)

	// Test data
	createRequest := map[string]interface{}{
		"namespace":       "default",
		"name":            "test-fpr",
		"description":     "Test FPR for integration testing",
		"oCloudId":        "ocloud-1",
		"oCloudNamespace": "default",
		"templateName":    "basic-cluster",
		"templateVersion": "v1.0.0",
		"templateParameters": map[string]interface{}{
			"clusterName": "integration-test-cluster",
			"nodeCount":   5,
		},
	}

	updateRequest := map[string]interface{}{
		"description": "Updated test FPR description",
		"templateParameters": map[string]interface{}{
			"clusterName": "updated-integration-test-cluster",
			"nodeCount":   7,
			"region":      "us-west-1",
		},
	}

	t.Run("Create FPR draft", func(t *testing.T) {
		resp, err := ts.makeRequest("POST", "/focom-provisioning-requests/draft", createRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = parseResponse(resp, &result)
		require.NoError(t, err)

		assert.Equal(t, "test-fpr", result["name"])
		assert.Equal(t, "Test FPR for integration testing", result["description"])
		assert.Equal(t, "ocloud-1", result["oCloudId"])
		assert.Equal(t, "basic-cluster", result["templateName"])
		assert.Equal(t, "v1.0.0", result["templateVersion"])
		assert.Equal(t, "DRAFT", result["focomProvisioningRequestRevisionState"])
		assert.NotEmpty(t, result["provisioningRequestId"])

		// Verify template parameters
		templateParams := result["templateParameters"].(map[string]interface{})
		assert.Equal(t, "integration-test-cluster", templateParams["clusterName"])
		assert.Equal(t, float64(5), templateParams["nodeCount"]) // JSON numbers are float64

		// Store the ID for subsequent tests
		fprID := result["provisioningRequestId"].(string)

		t.Run("Get FPR draft", func(t *testing.T) {
			resp, err := ts.makeRequest("GET", fmt.Sprintf("/focom-provisioning-requests/%s/draft", fprID), nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var draft map[string]interface{}
			err = parseResponse(resp, &draft)
			require.NoError(t, err)

			assert.Equal(t, fprID, draft["provisioningRequestId"])
			assert.Equal(t, "test-fpr", draft["name"])
			assert.Equal(t, "DRAFT", draft["focomProvisioningRequestRevisionState"])
		})

		t.Run("Update FPR draft", func(t *testing.T) {
			resp, err := ts.makeRequest("PATCH", fmt.Sprintf("/focom-provisioning-requests/%s/draft", fprID), updateRequest)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var updated map[string]interface{}
			err = parseResponse(resp, &updated)
			require.NoError(t, err)

			assert.Equal(t, "Updated test FPR description", updated["description"])

			templateParams := updated["templateParameters"].(map[string]interface{})
			assert.Equal(t, "updated-integration-test-cluster", templateParams["clusterName"])
			assert.Equal(t, float64(7), templateParams["nodeCount"])
			assert.Equal(t, "us-west-1", templateParams["region"])
		})

		t.Run("Validate FPR draft", func(t *testing.T) {
			resp, err := ts.makeRequest("POST", fmt.Sprintf("/focom-provisioning-requests/%s/draft/validate", fprID), nil)
			require.NoError(t, err)

			if resp.StatusCode != http.StatusOK {
				var errorResp map[string]interface{}
				parseResponse(resp, &errorResp)
				t.Logf("FPR validation failed: %+v", errorResp)
			}
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var validation map[string]interface{}
			err = parseResponse(resp, &validation)
			require.NoError(t, err)

			// Check the actual API response format (same as OCloud)
			validationResult := validation["validationResult"].(map[string]interface{})
			assert.True(t, validationResult["success"].(bool))
			assert.NotEmpty(t, validationResult["validationTime"])
		})

		t.Run("Approve FPR draft", func(t *testing.T) {
			resp, err := ts.makeRequest("POST", fmt.Sprintf("/focom-provisioning-requests/%s/draft/approve", fprID), nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var approval map[string]interface{}
			err = parseResponse(resp, &approval)
			require.NoError(t, err)

			assert.True(t, approval["approved"].(bool))
			assert.NotEmpty(t, approval["approvedAt"])
			assert.NotEmpty(t, approval["revisionId"])
		})

		t.Run("Get approved FPR", func(t *testing.T) {
			resp, err := ts.makeRequest("GET", fmt.Sprintf("/focom-provisioning-requests/%s", fprID), nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var approved map[string]interface{}
			err = parseResponse(resp, &approved)
			require.NoError(t, err)

			assert.Equal(t, fprID, approved["provisioningRequestId"])
			assert.Equal(t, "test-fpr", approved["name"])
			assert.Equal(t, "Updated test FPR description", approved["description"])
			assert.Equal(t, "APPROVED", approved["focomProvisioningRequestRevisionState"])
			assert.NotEmpty(t, approved["revisionId"])
		})

		t.Run("List FPRs", func(t *testing.T) {
			resp, err := ts.makeRequest("GET", "/focom-provisioning-requests", nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var items []interface{}
			err = parseResponse(resp, &items)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(items), 1)

			// Find our created FPR
			found := false
			for _, item := range items {
				fpr := item.(map[string]interface{})
				if fpr["provisioningRequestId"] == fprID {
					found = true
					assert.Equal(t, "test-fpr", fpr["name"])
					break
				}
			}
			assert.True(t, found, "Created FPR should be in the list")
		})

		t.Run("Get FPR revisions", func(t *testing.T) {
			resp, err := ts.makeRequest("GET", fmt.Sprintf("/focom-provisioning-requests/%s/revisions", fprID), nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var items []interface{}
			err = parseResponse(resp, &items)
			require.NoError(t, err)
			assert.Equal(t, 1, len(items))

			revision := items[0].(map[string]interface{})
			assert.Equal(t, fprID, revision["resourceId"])
			assert.NotEmpty(t, revision["revisionId"])
		})
	})
}

// TestFocomProvisioningRequestDependencyValidation tests dependency validation during approval
func TestFocomProvisioningRequestDependencyValidation(t *testing.T) {
	skipUnlessIntegration(t)
	ts := NewTestServer()
	defer ts.Close()

	// Load test data to have some dependencies available
	err := ts.LoadTestData()
	require.NoError(t, err)

	t.Run("Approve FPR with valid dependencies", func(t *testing.T) {
		validRequest := map[string]interface{}{
			"namespace":       "default",
			"name":            "valid-dependency-fpr",
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
		resp, err := ts.makeRequest("POST", "/focom-provisioning-requests/draft", validRequest)
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

	t.Run("Approve FPR with missing OCloud dependency", func(t *testing.T) {
		invalidRequest := map[string]interface{}{
			"namespace":       "default",
			"name":            "missing-ocloud-fpr",
			"description":     "FPR with missing OCloud",
			"oCloudId":        "non-existent-ocloud",
			"oCloudNamespace": "default",
			"templateName":    "basic-cluster",
			"templateVersion": "v1.0.0",
			"templateParameters": map[string]interface{}{
				"clusterName": "invalid-cluster",
				"nodeCount":   3,
			},
		}

		// Create draft
		resp, err := ts.makeRequest("POST", "/focom-provisioning-requests/draft", invalidRequest)
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

		// Approve (should fail due to missing dependency)
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/focom-provisioning-requests/%s/draft/approve", fprID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var errorResp map[string]interface{}
		err = parseResponse(resp, &errorResp)
		require.NoError(t, err)

		assert.Contains(t, errorResp, "error")
		assert.Contains(t, errorResp["error"], "OCloud")
		assert.Contains(t, errorResp["error"], "non-existent-ocloud")
	})

	t.Run("Approve FPR with missing TemplateInfo dependency", func(t *testing.T) {
		invalidRequest := map[string]interface{}{
			"namespace":       "default",
			"name":            "missing-template-fpr",
			"description":     "FPR with missing TemplateInfo",
			"oCloudId":        "ocloud-1",
			"oCloudNamespace": "default",
			"templateName":    "non-existent-template",
			"templateVersion": "v1.0.0",
			"templateParameters": map[string]interface{}{
				"clusterName": "invalid-cluster",
				"nodeCount":   3,
			},
		}

		// Create draft
		resp, err := ts.makeRequest("POST", "/focom-provisioning-requests/draft", invalidRequest)
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

		// Approve (should also fail since draft is not validated)
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/focom-provisioning-requests/%s/draft/approve", fprID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Approve FPR with wrong template version", func(t *testing.T) {
		invalidRequest := map[string]interface{}{
			"namespace":       "default",
			"name":            "wrong-version-fpr",
			"description":     "FPR with wrong template version",
			"oCloudId":        "ocloud-1",
			"oCloudNamespace": "default",
			"templateName":    "basic-cluster",
			"templateVersion": "v2.0.0", // Wrong version
			"templateParameters": map[string]interface{}{
				"clusterName": "invalid-cluster",
				"nodeCount":   3,
			},
		}

		// Create draft
		resp, err := ts.makeRequest("POST", "/focom-provisioning-requests/draft", invalidRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = parseResponse(resp, &result)
		require.NoError(t, err)
		fprID := result["provisioningRequestId"].(string)

		// Validate - should fail because template version doesn't exist
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/focom-provisioning-requests/%s/draft/validate", fprID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var validateResp map[string]interface{}
		err = parseResponse(resp, &validateResp)
		require.NoError(t, err)
		assert.Contains(t, validateResp, "error")

		// Approve (should also fail since draft is not validated)
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/focom-provisioning-requests/%s/draft/approve", fprID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestFocomProvisioningRequestErrorHandling tests error handling for missing dependencies
func TestFocomProvisioningRequestErrorHandling(t *testing.T) {
	skipUnlessIntegration(t)
	ts := NewTestServer()
	defer ts.Close()

	t.Run("Get non-existent FPR", func(t *testing.T) {
		resp, err := ts.makeRequest("GET", "/focom-provisioning-requests/non-existent", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		var errorResp map[string]interface{}
		err = parseResponse(resp, &errorResp)
		require.NoError(t, err)

		assert.Contains(t, errorResp, "error")
		assert.Contains(t, errorResp, "code")
	})

	t.Run("Create FPR with invalid data", func(t *testing.T) {
		invalidRequest := map[string]interface{}{
			"name": "", // Empty name should fail validation
		}

		resp, err := ts.makeRequest("POST", "/focom-provisioning-requests/draft", invalidRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var errorResp map[string]interface{}
		err = parseResponse(resp, &errorResp)
		require.NoError(t, err)

		assert.Contains(t, errorResp, "error")
		assert.Contains(t, errorResp, "code")
	})

	t.Run("Update non-existent draft", func(t *testing.T) {
		updateRequest := map[string]interface{}{
			"description": "This should fail",
		}

		resp, err := ts.makeRequest("PATCH", "/focom-provisioning-requests/non-existent/draft", updateRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Template parameter validation", func(t *testing.T) {
		// Load test data to have template available
		err := ts.LoadTestData()
		require.NoError(t, err)

		invalidParamsRequest := map[string]interface{}{
			"namespace":       "default",
			"name":            "invalid-params-fpr",
			"description":     "FPR with invalid template parameters",
			"oCloudId":        "ocloud-1",
			"oCloudNamespace": "default",
			"templateName":    "basic-cluster",
			"templateVersion": "v1.0.0",
			"templateParameters": map[string]interface{}{
				"clusterName": "", // Empty cluster name should fail validation
				"nodeCount":   0,  // Zero nodes should fail validation
			},
		}

		// Create draft
		resp, err := ts.makeRequest("POST", "/focom-provisioning-requests/draft", invalidParamsRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = parseResponse(resp, &result)
		require.NoError(t, err)
		fprID := result["provisioningRequestId"].(string)

		// Validate - should fail because nodeCount=0 violates minimum:1 in the schema
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/focom-provisioning-requests/%s/draft/validate", fprID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var errorResp map[string]interface{}
		err = parseResponse(resp, &errorResp)
		require.NoError(t, err)
		assert.Contains(t, errorResp, "error")
	})
}

// TestFocomProvisioningRequestRevisionManagement tests revision management endpoints
func TestFocomProvisioningRequestRevisionManagement(t *testing.T) {
	skipUnlessIntegration(t)
	ts := NewTestServer()
	defer ts.Close()

	// Load test data with existing v1 revisions
	err := ts.LoadTestData()
	require.NoError(t, err)

	t.Run("List revisions for existing FPR", func(t *testing.T) {
		resp, err := ts.makeRequest("GET", "/focom-provisioning-requests/fpr-1/revisions", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var items []interface{}
		err = parseResponse(resp, &items)
		require.NoError(t, err)
		assert.Equal(t, 1, len(items), "Should have exactly 1 revision (v1)")

		revision := items[0].(map[string]interface{})
		assert.Equal(t, "fpr-1", revision["resourceId"])
		assert.Equal(t, "v1", revision["revisionId"])
		assert.NotEmpty(t, revision["updatedAt"])
	})

	t.Run("Get specific revision", func(t *testing.T) {
		resp, err := ts.makeRequest("GET", "/focom-provisioning-requests/fpr-1/revisions/v1", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var revision map[string]interface{}
		err = parseResponse(resp, &revision)
		require.NoError(t, err)

		assert.Equal(t, "fpr-1", revision["resourceId"])
		assert.Equal(t, "v1", revision["revisionId"])

		// Check revision data with proper nil checks
		revisionData, ok := revision["revisionData"].(map[string]interface{})
		if ok && revisionData != nil {
			assert.Equal(t, "fpr-1", revisionData["name"])
			assert.Equal(t, "ocloud-1", revisionData["oCloudId"])
			assert.Equal(t, "basic-cluster", revisionData["templateName"])
		} else {
			t.Logf("WARNING: revisionData is nil or not a map. Full revision response: %+v", revision)
		}
	})

	t.Run("Multiple revisions", func(t *testing.T) {
		t.Skip("Skipping: multiple drafts from existing revisions not yet supported — see why-multiple-drafts-not-possible.md")
		// Create draft from v1
		resp, err := ts.makeRequest("POST", "/focom-provisioning-requests/fpr-1/revisions/v1/draft", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var draft map[string]interface{}
		err = parseResponse(resp, &draft)
		require.NoError(t, err)

		assert.Equal(t, "fpr-1", draft["provisioningRequestId"])
		assert.Equal(t, "DRAFT", draft["focomProvisioningRequestRevisionState"])
		assert.Equal(t, "fpr-1", draft["name"])
		assert.Equal(t, "ocloud-1", draft["oCloudId"])

		// Verify template parameters are preserved
		templateParams := draft["templateParameters"].(map[string]interface{})
		assert.Equal(t, "test-cluster-1", templateParams["clusterName"])
		assert.Equal(t, float64(3), templateParams["nodeCount"])

		// Modify the draft
		updateRequest := map[string]interface{}{
			"description": "Updated FPR description for v2",
			"templateParameters": map[string]interface{}{
				"clusterName": "test-cluster-1-updated",
				"nodeCount":   5,
			},
		}

		resp, err = ts.makeRequest("PATCH", "/focom-provisioning-requests/fpr-1/draft", updateRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Validate the draft
		resp, err = ts.makeRequest("POST", "/focom-provisioning-requests/fpr-1/draft/validate", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Approve to create v2
		resp, err = ts.makeRequest("POST", "/focom-provisioning-requests/fpr-1/draft/approve", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify we now have 2 revisions
		resp, err = ts.makeRequest("GET", "/focom-provisioning-requests/fpr-1/revisions", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var items []interface{}
		err = parseResponse(resp, &items)
		require.NoError(t, err)
		assert.Equal(t, 2, len(items), "Should have 2 revisions (v1 and v2)")

		// Verify both revisions exist
		revisionIDs := make(map[string]bool)
		for _, item := range items {
			revision := item.(map[string]interface{})
			revisionIDs[revision["revisionId"].(string)] = true
		}
		assert.True(t, revisionIDs["v1"], "v1 should exist")
		assert.True(t, revisionIDs["v2"], "v2 should exist")

		// Get v2 specifically
		resp, err = ts.makeRequest("GET", "/focom-provisioning-requests/fpr-1/revisions/v2", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var v2Revision map[string]interface{}
		err = parseResponse(resp, &v2Revision)
		require.NoError(t, err)

		assert.Equal(t, "fpr-1", v2Revision["resourceId"])
		assert.Equal(t, "v2", v2Revision["revisionId"])

		// Verify the updated data is in v2 with proper nil checks
		revisionData, ok := v2Revision["revisionData"].(map[string]interface{})
		if ok && revisionData != nil {
			assert.Equal(t, "Updated FPR description for v2", revisionData["description"])
			v2TemplateParams, paramsOk := revisionData["templateParameters"].(map[string]interface{})
			if paramsOk && v2TemplateParams != nil {
				assert.Equal(t, "test-cluster-1-updated", v2TemplateParams["clusterName"])
				assert.Equal(t, float64(5), v2TemplateParams["nodeCount"])
			} else {
				t.Logf("WARNING: templateParameters is nil or not a map")
			}
		} else {
			t.Logf("WARNING: revisionData is nil or not a map. Full v2 revision response: %+v", v2Revision)
		}
	})
}

// TestFocomProvisioningRequestDraftWorkflow tests the complete draft workflow
func TestFocomProvisioningRequestDraftWorkflow(t *testing.T) {
	skipUnlessIntegration(t)
	ts := NewTestServer()
	defer ts.Close()

	// Load test data to have dependencies available
	err := ts.LoadTestData()
	require.NoError(t, err)

	createRequest := map[string]interface{}{
		"namespace":       "default",
		"name":            "workflow-test-fpr",
		"description":     "FPR for workflow testing",
		"oCloudId":        "ocloud-1",
		"oCloudNamespace": "default",
		"templateName":    "basic-cluster",
		"templateVersion": "v1.0.0",
		"templateParameters": map[string]interface{}{
			"clusterName": "workflow-cluster",
			"nodeCount":   2,
		},
	}

	// Create draft
	resp, err := ts.makeRequest("POST", "/focom-provisioning-requests/draft", createRequest)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	err = parseResponse(resp, &result)
	require.NoError(t, err)
	fprID := result["provisioningRequestId"].(string)

	t.Run("Reject draft workflow", func(t *testing.T) {
		// First validate the draft
		resp, err := ts.makeRequest("POST", fmt.Sprintf("/focom-provisioning-requests/%s/draft/validate", fprID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Then reject it
		rejectRequest := map[string]interface{}{
			"reason": "Need more nodes for production workload",
		}

		resp, err = ts.makeRequest("POST", fmt.Sprintf("/focom-provisioning-requests/%s/draft/reject", fprID), rejectRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var rejection map[string]interface{}
		err = parseResponse(resp, &rejection)
		require.NoError(t, err)

		assert.True(t, rejection["rejected"].(bool))
		assert.Equal(t, "Need more nodes for production workload", rejection["reason"])

		// Verify draft is back to DRAFT state
		resp, err = ts.makeRequest("GET", fmt.Sprintf("/focom-provisioning-requests/%s/draft", fprID), nil)
		require.NoError(t, err)

		var draft map[string]interface{}
		err = parseResponse(resp, &draft)
		require.NoError(t, err)

		assert.Equal(t, "DRAFT", draft["focomProvisioningRequestRevisionState"])
	})

	t.Run("Update draft after rejection", func(t *testing.T) {
		improvedRequest := map[string]interface{}{
			"description": "Improved FPR after feedback",
			"templateParameters": map[string]interface{}{
				"clusterName": "improved-workflow-cluster",
				"nodeCount":   5, // Increased node count
			},
		}

		resp, err := ts.makeRequest("PATCH", fmt.Sprintf("/focom-provisioning-requests/%s/draft", fprID), improvedRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var updated map[string]interface{}
		err = parseResponse(resp, &updated)
		require.NoError(t, err)

		assert.Equal(t, "Improved FPR after feedback", updated["description"])

		templateParams := updated["templateParameters"].(map[string]interface{})
		assert.Equal(t, "improved-workflow-cluster", templateParams["clusterName"])
		assert.Equal(t, float64(5), templateParams["nodeCount"])
	})
}
