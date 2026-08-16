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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDebugAPIResponses helps debug what the API is actually returning
func TestDebugAPIResponses(t *testing.T) {
	skipUnlessIntegration(t)
	ts := NewTestServer()
	defer ts.Close()

	// Test creating an OCloud draft
	createRequest := `{
		"namespace": "default",
		"name": "debug-ocloud",
		"description": "Debug OCloud",
		"o2imsSecret": {
			"secretRef": {
				"name": "debug-secret",
				"namespace": "default"
			}
		}
	}`

	resp, err := http.Post(ts.URL()+"/o-clouds/draft", "application/json", strings.NewReader(createRequest))
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	fmt.Printf("Create OCloud Draft Response:\n")
	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Body: %s\n\n", string(body))

	if resp.StatusCode == 201 {
		// Parse the response to get the actual ID
		var createResp map[string]interface{}
		err = json.Unmarshal(body, &createResp)
		require.NoError(t, err)

		ocloudID := createResp["oCloudId"].(string)
		fmt.Printf("Extracted OCloud ID: %s\n\n", ocloudID)

		// Try to validate the draft using the correct ID
		validateResp, err := http.Post(ts.URL()+"/o-clouds/"+ocloudID+"/draft/validate", "application/json", nil)
		require.NoError(t, err)
		defer validateResp.Body.Close()

		validateBody, err := io.ReadAll(validateResp.Body)
		require.NoError(t, err)

		fmt.Printf("Validate OCloud Draft Response:\n")
		fmt.Printf("Status: %d\n", validateResp.StatusCode)
		fmt.Printf("Body: %s\n\n", string(validateBody))

		if validateResp.StatusCode == 200 {
			// Try to approve the draft
			approveResp, err := http.Post(ts.URL()+"/o-clouds/"+ocloudID+"/draft/approve", "application/json", nil)
			require.NoError(t, err)
			defer approveResp.Body.Close()

			approveBody, err := io.ReadAll(approveResp.Body)
			require.NoError(t, err)

			fmt.Printf("Approve OCloud Draft Response:\n")
			fmt.Printf("Status: %d\n", approveResp.StatusCode)
			fmt.Printf("Body: %s\n\n", string(approveBody))

			if approveResp.StatusCode == 200 {
				// Try to get the approved resource
				getResp, err := http.Get(ts.URL() + "/o-clouds/" + ocloudID)
				require.NoError(t, err)
				defer getResp.Body.Close()

				getBody, err := io.ReadAll(getResp.Body)
				require.NoError(t, err)

				fmt.Printf("Get Approved OCloud Response:\n")
				fmt.Printf("Status: %d\n", getResp.StatusCode)
				fmt.Printf("Body: %s\n\n", string(getBody))

				// Try to list all OClouds
				listResp, err := http.Get(ts.URL() + "/o-clouds")
				require.NoError(t, err)
				defer listResp.Body.Close()

				listBody, err := io.ReadAll(listResp.Body)
				require.NoError(t, err)

				fmt.Printf("List OClouds Response:\n")
				fmt.Printf("Status: %d\n", listResp.StatusCode)
				fmt.Printf("Body: %s\n\n", string(listBody))

				// Try to create a new draft from revision
				revResp, err := http.Get(ts.URL() + "/o-clouds/" + ocloudID + "/revisions")
				require.NoError(t, err)
				defer revResp.Body.Close()

				revBody, err := io.ReadAll(revResp.Body)
				require.NoError(t, err)

				fmt.Printf("Get Revisions Response:\n")
				fmt.Printf("Status: %d\n", revResp.StatusCode)
				fmt.Printf("Body: %s\n\n", string(revBody))

				// Try to delete a non-existent draft
				delReq, _ := http.NewRequest("DELETE", ts.URL()+"/o-clouds/"+ocloudID+"/draft", nil)
				delResp, err := http.DefaultClient.Do(delReq)
				require.NoError(t, err)
				defer delResp.Body.Close()

				delBody, err := io.ReadAll(delResp.Body)
				require.NoError(t, err)

				fmt.Printf("Delete Draft Response:\n")
				fmt.Printf("Status: %d\n", delResp.StatusCode)
				fmt.Printf("Body: %s\n\n", string(delBody))
			}
		}
	}
}
