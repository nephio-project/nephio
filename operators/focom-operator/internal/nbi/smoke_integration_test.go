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

// skipUnlessSmoke skips the test unless FOCOM_SMOKE_TESTS=true is set.
// Smoke tests exercise the critical path with a single setup/teardown cycle
// and should complete in under 6 minutes against Porch.
func skipUnlessSmoke(t *testing.T) {
	t.Helper()
	if os.Getenv("FOCOM_SMOKE_TESTS") != "true" {
		t.Skip("Skipping smoke test — set FOCOM_SMOKE_TESTS=true to run")
	}
}

// TestSmokeWorkflow exercises the critical path for all three resource types
// using a single test server and one setup/teardown cycle.
// Target runtime: < 6 minutes against Porch.
func TestSmokeWorkflow(t *testing.T) {
	skipUnlessSmoke(t)

	ts := NewTestServer()
	defer ts.Close()

	// Single cleanup at the start — this is the most expensive operation
	err := ts.ClearStorage()
	require.NoError(t, err)

	var ocloudID, templateID, fprID string

	// ── OCloud: create → update → validate → approve ──

	t.Run("OCloud draft lifecycle", func(t *testing.T) {
		resp, err := ts.makeRequest("POST", "/o-clouds/draft", map[string]interface{}{
			"namespace":   "default",
			"name":        "smoke-ocloud",
			"description": "Smoke test OCloud",
			"o2imsSecret": map[string]interface{}{
				"secretRef": map[string]interface{}{
					"name":      "smoke-secret",
					"namespace": "default",
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = parseResponse(resp, &result)
		require.NoError(t, err)
		ocloudID = result["oCloudId"].(string)
		assert.Equal(t, "DRAFT", result["oCloudRevisionState"])

		// Get draft
		resp, err = ts.makeRequest("GET", fmt.Sprintf("/o-clouds/%s/draft", ocloudID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Update draft
		resp, err = ts.makeRequest("PATCH", fmt.Sprintf("/o-clouds/%s/draft", ocloudID), map[string]interface{}{
			"description": "Updated smoke OCloud",
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Validate
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/o-clouds/%s/draft/validate", ocloudID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Approve
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/o-clouds/%s/draft/approve", ocloudID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify approved
		resp, err = ts.makeRequest("GET", fmt.Sprintf("/o-clouds/%s", ocloudID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var approved map[string]interface{}
		err = parseResponse(resp, &approved)
		require.NoError(t, err)
		assert.NotEmpty(t, approved["revisionId"])
	})

	// ── TemplateInfo: create → validate → approve ──

	t.Run("TemplateInfo draft lifecycle", func(t *testing.T) {
		resp, err := ts.makeRequest("POST", "/template-infos/draft", map[string]interface{}{
			"namespace":               "default",
			"name":                    "smoke-template",
			"description":             "Smoke test template",
			"templateName":            "smoke-cluster",
			"templateVersion":         "v1.0.0",
			"templateParameterSchema": `{"type":"object","properties":{"clusterName":{"type":"string"},"nodeCount":{"type":"integer"}},"required":["clusterName"]}`,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = parseResponse(resp, &result)
		require.NoError(t, err)
		templateID = result["templateInfoId"].(string)

		// Validate
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/template-infos/%s/draft/validate", templateID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Approve
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/template-infos/%s/draft/approve", templateID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify approved
		resp, err = ts.makeRequest("GET", fmt.Sprintf("/template-infos/%s", templateID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// ── FPR: create → validate → approve (with dependency check) ──

	t.Run("FPR draft lifecycle with dependencies", func(t *testing.T) {
		resp, err := ts.makeRequest("POST", "/focom-provisioning-requests/draft", map[string]interface{}{
			"namespace":       "default",
			"name":            "smoke-fpr",
			"description":     "Smoke test FPR",
			"oCloudId":        ocloudID,
			"oCloudNamespace": "default",
			"templateName":    "smoke-cluster",
			"templateVersion": "v1.0.0",
			"templateParameters": map[string]interface{}{
				"clusterName": "smoke-cluster-1",
				"nodeCount":   3,
			},
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = parseResponse(resp, &result)
		require.NoError(t, err)
		fprID = result["provisioningRequestId"].(string)

		// Validate
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/focom-provisioning-requests/%s/draft/validate", fprID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Approve
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/focom-provisioning-requests/%s/draft/approve", fprID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify approved
		resp, err = ts.makeRequest("GET", fmt.Sprintf("/focom-provisioning-requests/%s", fprID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// ── List all resource types ──

	t.Run("List all resources", func(t *testing.T) {
		for _, endpoint := range []string{"/o-clouds", "/template-infos", "/focom-provisioning-requests"} {
			resp, err := ts.makeRequest("GET", endpoint, nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			var items []interface{}
			err = parseResponse(resp, &items)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(items), 1, "Expected at least 1 item for %s", endpoint)
		}
	})

	// ── Revision management ──

	t.Run("OCloud revisions and create-draft-from-revision", func(t *testing.T) {
		// List revisions
		resp, err := ts.makeRequest("GET", fmt.Sprintf("/o-clouds/%s/revisions", ocloudID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var revisions []interface{}
		err = parseResponse(resp, &revisions)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(revisions), 1)

		revisionID := revisions[0].(map[string]interface{})["revisionId"].(string)

		// Get specific revision
		resp, err = ts.makeRequest("GET", fmt.Sprintf("/o-clouds/%s/revisions/%s", ocloudID, revisionID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Create draft from revision
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/o-clouds/%s/revisions/%s/draft", ocloudID, revisionID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		// Delete the draft (cleanup for next potential run)
		resp, err = ts.makeRequest("DELETE", fmt.Sprintf("/o-clouds/%s/draft", ocloudID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	// ── Reject workflow ──

	t.Run("TemplateInfo reject workflow", func(t *testing.T) {
		// Create a new draft
		resp, err := ts.makeRequest("POST", "/template-infos/draft", map[string]interface{}{
			"namespace":               "default",
			"name":                    "smoke-reject-template",
			"description":             "Template for reject test",
			"templateName":            "smoke-reject-tmpl",
			"templateVersion":         "v1.0.0",
			"templateParameterSchema": `{"type":"object"}`,
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		err = parseResponse(resp, &result)
		require.NoError(t, err)
		rejectID := result["templateInfoId"].(string)

		// Validate
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/template-infos/%s/draft/validate", rejectID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Reject
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/template-infos/%s/draft/reject", rejectID), map[string]interface{}{
			"reason": "Smoke test rejection",
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify back to DRAFT
		resp, err = ts.makeRequest("GET", fmt.Sprintf("/template-infos/%s/draft", rejectID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var draft map[string]interface{}
		err = parseResponse(resp, &draft)
		require.NoError(t, err)
		assert.Equal(t, "DRAFT", draft["templateInfoRevisionState"])
	})

	// ── Error handling spot checks ──

	t.Run("Error handling", func(t *testing.T) {
		// 404 on non-existent resource
		resp, err := ts.makeRequest("GET", "/o-clouds/does-not-exist", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		// 400 on invalid create
		resp, err = ts.makeRequest("POST", "/o-clouds/draft", map[string]interface{}{
			"name": "",
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		// FPR with missing dependency should fail approval
		resp, err = ts.makeRequest("POST", "/focom-provisioning-requests/draft", map[string]interface{}{
			"namespace":       "default",
			"name":            "smoke-bad-dep-fpr",
			"description":     "FPR with bad dependency",
			"oCloudId":        "non-existent-ocloud",
			"oCloudNamespace": "default",
			"templateName":    "smoke-cluster",
			"templateVersion": "v1.0.0",
			"templateParameters": map[string]interface{}{
				"clusterName": "bad-cluster",
				"nodeCount":   1,
			},
		})
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var badFPR map[string]interface{}
		err = parseResponse(resp, &badFPR)
		require.NoError(t, err)
		badFPRID := badFPR["provisioningRequestId"].(string)

		// Validate (succeeds — validation doesn't check deps)
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/focom-provisioning-requests/%s/draft/validate", badFPRID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Approve should fail due to missing OCloud
		resp, err = ts.makeRequest("POST", fmt.Sprintf("/focom-provisioning-requests/%s/draft/approve", badFPRID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var errorResp map[string]interface{}
		err = parseResponse(resp, &errorResp)
		require.NoError(t, err)
		assert.Contains(t, errorResp["error"], "OCloud")
	})

	// ── Health / info endpoints ──

	t.Run("Health and info endpoints", func(t *testing.T) {
		for _, path := range []string{"/", "/api/info", "/health/live", "/health/ready", "/metrics"} {
			resp, err := ts.makeRequest("GET", path, nil)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 for %s", path)
		}
	})
}
