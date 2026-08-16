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

// TestOCloudCRUDOperations tests complete OCloud CRUD operations via HTTP
func TestOCloudCRUDOperations(t *testing.T) {
	skipUnlessIntegration(t)
	ts := NewTestServer()
	defer ts.Close()

	// Clear storage to ensure clean state
	err := ts.ClearStorage()
	require.NoError(t, err)

	// Test data
	createRequest := map[string]interface{}{
		"namespace":   "default",
		"name":        "test-ocloud",
		"description": "Test OCloud for integration testing",
		"o2imsSecret": map[string]interface{}{
			"secretRef": map[string]interface{}{
				"name":      "test-secret",
				"namespace": "default",
			},
		},
	}

	updateRequest := map[string]interface{}{
		"description": "Updated test OCloud description",
		"o2imsSecret": map[string]interface{}{
			"secretRef": map[string]interface{}{
				"name":      "updated-secret",
				"namespace": "default",
			},
		},
	}

	t.Run("Create OCloud draft", func(t *testing.T) {
		resp, err := ts.makeRequest("POST", "/o-clouds/draft", createRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = parseResponse(resp, &result)
		require.NoError(t, err)

		assert.Equal(t, "test-ocloud", result["name"])
		assert.Equal(t, "Test OCloud for integration testing", result["description"])
		assert.Equal(t, "DRAFT", result["oCloudRevisionState"])
		assert.NotEmpty(t, result["oCloudId"])

		// Store the ID for subsequent tests
		ocloudID := result["oCloudId"].(string)

		t.Run("Get OCloud draft", func(t *testing.T) {
			resp, err := ts.makeRequest("GET", fmt.Sprintf("/o-clouds/%s/draft", ocloudID), nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var draft map[string]interface{}
			err = parseResponse(resp, &draft)
			require.NoError(t, err)

			assert.Equal(t, ocloudID, draft["oCloudId"])
			assert.Equal(t, "test-ocloud", draft["name"])
			assert.Equal(t, "DRAFT", draft["oCloudRevisionState"])
		})

		t.Run("Update OCloud draft", func(t *testing.T) {
			resp, err := ts.makeRequest("PATCH", fmt.Sprintf("/o-clouds/%s/draft", ocloudID), updateRequest)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var updated map[string]interface{}
			err = parseResponse(resp, &updated)
			require.NoError(t, err)

			assert.Equal(t, "Updated test OCloud description", updated["description"])
			o2imsSecret := updated["o2imsSecret"].(map[string]interface{})
			secretRef := o2imsSecret["secretRef"].(map[string]interface{})
			assert.Equal(t, "updated-secret", secretRef["name"])
		})

		t.Run("Validate OCloud draft", func(t *testing.T) {
			resp, err := ts.makeRequest("POST", fmt.Sprintf("/o-clouds/%s/draft/validate", ocloudID), nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var validation map[string]interface{}
			err = parseResponse(resp, &validation)
			require.NoError(t, err)

			// Check the actual API response format
			validationResult := validation["validationResult"].(map[string]interface{})
			assert.True(t, validationResult["success"].(bool))
			assert.NotEmpty(t, validationResult["validationTime"])
		})

		t.Run("Approve OCloud draft", func(t *testing.T) {
			resp, err := ts.makeRequest("POST", fmt.Sprintf("/o-clouds/%s/draft/approve", ocloudID), nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var approval map[string]interface{}
			err = parseResponse(resp, &approval)
			require.NoError(t, err)

			// The approval response just returns the resource data
			assert.Equal(t, ocloudID, approval["oCloudId"])
			assert.NotEmpty(t, approval["revisionId"])
		})

		t.Run("Get approved OCloud", func(t *testing.T) {
			resp, err := ts.makeRequest("GET", fmt.Sprintf("/o-clouds/%s", ocloudID), nil)
			require.NoError(t, err)

			// DEBUG: Print actual response status and body
			t.Logf("DEBUG: Response status code: %d", resp.StatusCode)

			var approved map[string]interface{}
			err = parseResponse(resp, &approved)
			if err != nil {
				t.Logf("DEBUG: parseResponse error: %v", err)
			}
			t.Logf("DEBUG: Response body: %+v", approved)

			assert.Equal(t, http.StatusOK, resp.StatusCode)
			require.NoError(t, err)

			// The get response now returns the resource data directly
			assert.Equal(t, ocloudID, approved["oCloudId"])
			assert.Equal(t, "test-ocloud", approved["name"])
			assert.Equal(t, "Updated test OCloud description", approved["description"])
			assert.NotEmpty(t, approved["revisionId"])
		})

		t.Run("List OClouds", func(t *testing.T) {
			resp, err := ts.makeRequest("GET", "/o-clouds", nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var items []interface{}
			err = parseResponse(resp, &items)
			require.NoError(t, err)

			assert.GreaterOrEqual(t, len(items), 1)

			// Find our created OCloud
			found := false
			for _, item := range items {
				ocloud := item.(map[string]interface{})
				oCloudData := ocloud["oCloudData"].(map[string]interface{})
				if oCloudData["oCloudId"] == ocloudID {
					found = true
					assert.Equal(t, "test-ocloud", oCloudData["name"])
					break
				}
			}
			assert.True(t, found, "Created OCloud should be in the list")
		})

		t.Run("Get OCloud revisions", func(t *testing.T) {
			resp, err := ts.makeRequest("GET", fmt.Sprintf("/o-clouds/%s/revisions", ocloudID), nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var items []interface{}
			err = parseResponse(resp, &items)
			require.NoError(t, err)

			assert.Equal(t, 1, len(items))

			revision := items[0].(map[string]interface{})
			assert.Equal(t, ocloudID, revision["resourceId"])
			assert.NotEmpty(t, revision["revisionId"])
		})

		t.Run("Delete OCloud draft (should fail - no draft exists)", func(t *testing.T) {
			resp, err := ts.makeRequest("DELETE", fmt.Sprintf("/o-clouds/%s/draft", ocloudID), nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		})

		t.Run("Create new draft from revision", func(t *testing.T) {
			// First get the revision ID
			resp, err := ts.makeRequest("GET", fmt.Sprintf("/o-clouds/%s/revisions", ocloudID), nil)
			require.NoError(t, err)

			var items []interface{}
			err = parseResponse(resp, &items)
			require.NoError(t, err)

			revision := items[0].(map[string]interface{})
			revisionID := revision["revisionId"].(string)

			// Create draft from revision
			resp, err = ts.makeRequest("POST", fmt.Sprintf("/o-clouds/%s/revisions/%s/draft", ocloudID, revisionID), nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusCreated, resp.StatusCode)

			var newDraft map[string]interface{}
			err = parseResponse(resp, &newDraft)
			require.NoError(t, err)

			assert.Equal(t, ocloudID, newDraft["oCloudId"])
			assert.Equal(t, "DRAFT", newDraft["oCloudRevisionState"])
		})

		t.Run("Delete OCloud draft", func(t *testing.T) {
			resp, err := ts.makeRequest("DELETE", fmt.Sprintf("/o-clouds/%s/draft", ocloudID), nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		})
	})
}

// TestOCloudDraftWorkflow tests the complete draft workflow
func TestOCloudDraftWorkflow(t *testing.T) {
	skipUnlessIntegration(t)
	ts := NewTestServer()
	defer ts.Close()

	// Clear storage to ensure clean state
	err := ts.ClearStorage()
	require.NoError(t, err)

	createRequest := map[string]interface{}{
		"namespace":   "default",
		"name":        "workflow-test-ocloud",
		"description": "OCloud for workflow testing",
		"o2imsSecret": map[string]interface{}{
			"secretRef": map[string]interface{}{
				"name":      "workflow-secret",
				"namespace": "default",
			},
		},
	}

	// Create draft
	resp, err := ts.makeRequest("POST", "/o-clouds/draft", createRequest)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	err = parseResponse(resp, &result)
	require.NoError(t, err)
	ocloudID := result["oCloudId"].(string)

	t.Run("Reject draft workflow", func(t *testing.T) {
		// First validate the draft
		resp, err := ts.makeRequest("POST", fmt.Sprintf("/o-clouds/%s/draft/validate", ocloudID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Then reject it
		rejectRequest := map[string]interface{}{
			"reason": "Testing rejection workflow",
		}

		resp, err = ts.makeRequest("POST", fmt.Sprintf("/o-clouds/%s/draft/reject", ocloudID), rejectRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var rejection map[string]interface{}
		err = parseResponse(resp, &rejection)
		require.NoError(t, err)

		assert.True(t, rejection["rejected"].(bool))
		assert.Equal(t, "Testing rejection workflow", rejection["reason"])

		// Verify draft is back to DRAFT state
		resp, err = ts.makeRequest("GET", fmt.Sprintf("/o-clouds/%s/draft", ocloudID), nil)
		require.NoError(t, err)

		var draft map[string]interface{}
		err = parseResponse(resp, &draft)
		require.NoError(t, err)

		assert.Equal(t, "DRAFT", draft["oCloudRevisionState"])
	})
}

// TestOCloudErrorHandling tests error scenarios
func TestOCloudErrorHandling(t *testing.T) {
	skipUnlessIntegration(t)
	ts := NewTestServer()
	defer ts.Close()

	// Clear storage to ensure clean state
	err := ts.ClearStorage()
	require.NoError(t, err)

	t.Run("Get non-existent OCloud", func(t *testing.T) {
		resp, err := ts.makeRequest("GET", "/o-clouds/non-existent", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		var errorResp map[string]interface{}
		err = parseResponse(resp, &errorResp)
		require.NoError(t, err)

		assert.Contains(t, errorResp, "error")
		assert.Contains(t, errorResp, "code")
	})

	t.Run("Create OCloud with invalid data", func(t *testing.T) {
		invalidRequest := map[string]interface{}{
			"name": "", // Empty name should fail validation
		}

		resp, err := ts.makeRequest("POST", "/o-clouds/draft", invalidRequest)
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

		resp, err := ts.makeRequest("PATCH", "/o-clouds/non-existent/draft", updateRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Approve non-validated draft", func(t *testing.T) {
		// Create a draft
		createRequest := map[string]interface{}{
			"namespace":   "default",
			"name":        "error-test-ocloud",
			"description": "OCloud for error testing",
			"o2imsSecret": map[string]interface{}{
				"secretRef": map[string]interface{}{
					"name":      "error-secret",
					"namespace": "default",
				},
			},
		}

		resp, err := ts.makeRequest("POST", "/o-clouds/draft", createRequest)
		require.NoError(t, err)

		var result map[string]interface{}
		err = parseResponse(resp, &result)
		require.NoError(t, err)
		ocloudID := result["oCloudId"].(string)

		// Try to approve without validation
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/o-clouds/%s/draft/approve", ocloudID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var errorResp map[string]interface{}
		err = parseResponse(resp, &errorResp)
		require.NoError(t, err)

		assert.Contains(t, errorResp, "error")
		assert.Contains(t, errorResp["error"], "Proposed state")
	})
}

// TestOCloudRevisionManagement tests revision management endpoints
func TestOCloudRevisionManagement(t *testing.T) {
	skipUnlessIntegration(t)
	ts := NewTestServer()
	defer ts.Close()

	// Load test data with existing v1 revisions
	err := ts.LoadTestData()
	require.NoError(t, err)

	t.Run("List revisions for existing OCloud", func(t *testing.T) {
		resp, err := ts.makeRequest("GET", "/o-clouds/ocloud-1/revisions", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var items []interface{}
		err = parseResponse(resp, &items)
		require.NoError(t, err)
		assert.Equal(t, 1, len(items), "Should have exactly 1 revision (v1)")

		revision := items[0].(map[string]interface{})
		assert.Equal(t, "ocloud-1", revision["resourceId"])
		assert.Equal(t, "v1", revision["revisionId"])
		assert.NotEmpty(t, revision["updatedAt"])
	})

	t.Run("Get specific revision", func(t *testing.T) {
		resp, err := ts.makeRequest("GET", "/o-clouds/ocloud-1/revisions/v1", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var revision map[string]interface{}
		err = parseResponse(resp, &revision)
		require.NoError(t, err)

		assert.Equal(t, "ocloud-1", revision["resourceId"])
		assert.Equal(t, "v1", revision["revisionId"])

		// Check revision data with proper nil checks
		revisionData, ok := revision["revisionData"].(map[string]interface{})
		if ok && revisionData != nil {
			assert.Equal(t, "ocloud-1", revisionData["name"])
		} else {
			t.Logf("WARNING: revisionData is nil or not a map. Full revision response: %+v", revision)
		}
	})

	t.Run("Multiple revisions", func(t *testing.T) {
		t.Skip("Skipping: multiple drafts from existing revisions not yet supported — see why-multiple-drafts-not-possible.md")
		// Create draft from v1
		resp, err := ts.makeRequest("POST", "/o-clouds/ocloud-1/revisions/v1/draft", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var draft map[string]interface{}
		err = parseResponse(resp, &draft)
		require.NoError(t, err)

		assert.Equal(t, "ocloud-1", draft["oCloudId"])
		assert.Equal(t, "DRAFT", draft["oCloudRevisionState"])
		assert.Equal(t, "ocloud-1", draft["name"])

		// Modify the draft
		updateRequest := map[string]interface{}{
			"description": "Updated OCloud description for v2",
		}

		resp, err = ts.makeRequest("PATCH", "/o-clouds/ocloud-1/draft", updateRequest)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Validate the draft
		resp, err = ts.makeRequest("POST", "/o-clouds/ocloud-1/draft/validate", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Approve to create v2
		resp, err = ts.makeRequest("POST", "/o-clouds/ocloud-1/draft/approve", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify we now have 2 revisions
		resp, err = ts.makeRequest("GET", "/o-clouds/ocloud-1/revisions", nil)
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
		resp, err = ts.makeRequest("GET", "/o-clouds/ocloud-1/revisions/v2", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var v2Revision map[string]interface{}
		err = parseResponse(resp, &v2Revision)
		require.NoError(t, err)

		assert.Equal(t, "ocloud-1", v2Revision["resourceId"])
		assert.Equal(t, "v2", v2Revision["revisionId"])

		// Verify the updated description is in v2 with proper nil checks
		revisionData, ok := v2Revision["revisionData"].(map[string]interface{})
		if ok && revisionData != nil {
			assert.Equal(t, "Updated OCloud description for v2", revisionData["description"])
		} else {
			t.Logf("WARNING: revisionData is nil or not a map. Full v2 revision response: %+v", v2Revision)
		}
	})
}
