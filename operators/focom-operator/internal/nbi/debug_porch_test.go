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
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDebugPorchCreate is a debug test to see what error we get when creating a draft
func TestDebugPorchCreate(t *testing.T) {
	skipUnlessIntegration(t)
	ts := NewTestServer()
	defer ts.Close()

	// Test data
	createRequest := map[string]interface{}{
		"namespace":   "default",
		"name":        "debug-ocloud",
		"description": "Debug test",
		"o2imsSecret": map[string]interface{}{
			"secretRef": map[string]interface{}{
				"name":      "test-secret",
				"namespace": "default",
			},
		},
	}

	// Create OCloud draft
	resp, err := ts.makeRequest("POST", "/o-clouds/draft", createRequest)
	require.NoError(t, err)

	// Read response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Response Body: %s\n", string(body))

	if resp.StatusCode != http.StatusCreated {
		var errorResp map[string]interface{}
		json.Unmarshal(body, &errorResp)
		fmt.Printf("Error Response: %+v\n", errorResp)
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}
}
