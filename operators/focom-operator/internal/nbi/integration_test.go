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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/gin-gonic/gin"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/handlers"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/models"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/storage"
	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/testfixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skipUnlessIntegration skips the test unless FOCOM_INTEGRATION_TESTS=true is set.
// Integration tests exercise individual resource CRUD, workflows, error handling, and
// revision management against Porch. Each test creates its own server and fixtures,
// so the full suite can take 30+ minutes.
func skipUnlessIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("FOCOM_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test — set FOCOM_INTEGRATION_TESTS=true to run")
	}
}

// TestServer provides a test server for HTTP integration tests
type TestServer struct {
	server  *httptest.Server
	storage storage.StorageInterface
	router  *gin.Engine
	config  *TestConfig
}

// NewTestServer creates a new test server with configuration from file or defaults
func NewTestServer() *TestServer {
	return NewTestServerFromConfig()
}

// NewTestServerWithConfig creates a new test server with the specified configuration
func NewTestServerWithConfig(config *TestConfig) *TestServer {
	gin.SetMode(gin.TestMode)

	// Create storage based on configuration
	var storageImpl storage.StorageInterface
	var err error

	switch config.Storage.Backend {
	case "porch":
		// Create Porch storage
		authMethod := "token"
		if config.Porch.UseKubeconfig {
			authMethod = "kubeconfig"
		}
		fmt.Printf("[TEST] Using Porch storage (namespace=%s, repository=%s, auth=%s)\n",
			config.Porch.Namespace, config.Porch.Repository, authMethod)
		porchConfig := &storage.PorchStorageConfig{
			UseKubeconfig:          config.Porch.UseKubeconfig,
			Kubeconfig:             config.Porch.Kubeconfig,
			KubernetesURL:          config.Porch.KubernetesURL,
			Token:                  config.Porch.Token,
			Namespace:              config.Porch.Namespace,
			Repository:             config.Porch.Repository,
			HTTPSVerify:            config.Porch.HTTPSVerify,
			PackageRevisionTimeout: 120 * time.Second, // Use 2 minutes for tests (default is 30s)
		}
		storageImpl, err = storage.NewPorchStorage(porchConfig)
		if err != nil {
			panic(fmt.Sprintf("failed to create Porch storage: %v", err))
		}
	case "memory":
		fallthrough
	default:
		// Create in-memory storage
		fmt.Printf("[TEST] Using in-memory storage\n")
		storageImpl = storage.NewInMemoryStorage()
	}

	// Setup router with default namespace
	router := handlers.SetupRouter(storageImpl, handlers.DefaultRouterConfig(), "focom-system")
	handlers.SetupAPIInfoEndpoint(router)

	// Create test server
	server := httptest.NewServer(router)

	return &TestServer{
		server:  server,
		storage: storageImpl,
		router:  router,
		config:  config,
	}
}

// NewTestServerFromConfig creates a test server by loading configuration
func NewTestServerFromConfig() *TestServer {
	config := LoadTestConfigOrDefault()
	return NewTestServerWithConfig(config)
}

// Close closes the test server
func (ts *TestServer) Close() {
	ts.server.Close()
}

// URL returns the base URL of the test server
func (ts *TestServer) URL() string {
	return ts.server.URL
}

// ClearStorage clears all data from storage
func (ts *TestServer) ClearStorage() error {
	// Clear the existing storage instead of replacing it
	if inMemStorage, ok := ts.storage.(*storage.InMemoryStorage); ok {
		inMemStorage.Clear()
		return nil
	}

	// For Porch storage, delete ALL PackageRevisions in the focom-resources repository
	if porchStorage, ok := ts.storage.(*storage.PorchStorage); ok {
		ctx := context.Background()

		fmt.Println("[CLEAR-DEBUG] Starting ClearStorage for Porch")

		// Get ALL PackageRevisions in the repository (regardless of lifecycle state)
		allPackages := ts.listAllPackageRevisionsFromPorch(ctx, porchStorage)
		fmt.Printf("[CLEAR-DEBUG] Found %d total packages to delete: %v\n", len(allPackages), allPackages)

		// Delete each package found
		// We need to try both DeleteDraft (for Draft/Proposed) and Delete (for Published)
		for _, pkgName := range allPackages {
			fmt.Printf("[CLEAR-DEBUG] Attempting to delete package: %s\n", pkgName)

			// Try DeleteDraft first (handles Draft and Proposed lifecycle states)
			// Use OCloud as the resource type - it doesn't matter for deletion by package name
			err := porchStorage.DeleteDraft(ctx, storage.ResourceTypeOCloud, pkgName)
			if err == nil {
				fmt.Printf("[CLEAR-DEBUG] Deleted %s as draft/proposed\n", pkgName)
				continue
			}

			// Try Delete (handles Published lifecycle state)
			err = porchStorage.Delete(ctx, storage.ResourceTypeOCloud, pkgName)
			if err == nil {
				fmt.Printf("[CLEAR-DEBUG] Deleted %s as published\n", pkgName)
				continue
			}

			// If both failed, log it but continue (might have been deleted already)
			fmt.Printf("[CLEAR-DEBUG] Could not delete %s (may already be deleted): %v\n", pkgName, err)
		}

		// Wait for all deletions to complete by polling Porch directly
		fmt.Println("[CLEAR-DEBUG] Waiting for deletions to complete...")
		maxWaitTime := 90 * time.Second
		pollInterval := 5 * time.Second
		deadline := time.Now().Add(maxWaitTime)

		for time.Now().Before(deadline) {
			// Check if any packages still exist
			remainingPackages := ts.listAllPackageRevisionsFromPorch(ctx, porchStorage)

			if len(remainingPackages) == 0 {
				fmt.Println("[CLEAR-DEBUG] All packages verified as deleted")
				break
			}

			fmt.Printf("[CLEAR-DEBUG] %d packages still exist: %v, waiting...\n", len(remainingPackages), remainingPackages)
			time.Sleep(pollInterval)
		}

		// Final verification
		finalPackages := ts.listAllPackageRevisionsFromPorch(ctx, porchStorage)
		if len(finalPackages) > 0 {
			fmt.Printf("[CLEAR-DEBUG] Warning: %d packages still exist after cleanup: %v\n", len(finalPackages), finalPackages)
		}

		fmt.Println("[CLEAR-DEBUG] ClearStorage complete")
	}

	return nil
}

// listAllPackageRevisionsFromPorch directly queries Porch API to list all PackageRevision names
// This includes Draft, Proposed, and Published packages
func (ts *TestServer) listAllPackageRevisionsFromPorch(ctx context.Context, porchStorage *storage.PorchStorage) []string {
	var packageNames []string
	seenNames := make(map[string]bool)

	// We need to make a direct API call to Porch to get ALL PackageRevisions
	// The storage layer's List() method filters by lifecycle state
	// We'll use reflection with unsafe to access the internal fields we need

	v := reflect.ValueOf(porchStorage).Elem()

	// Get the httpClient field using unsafe to access unexported field
	httpClientField := v.FieldByName("httpClient")
	if !httpClientField.IsValid() {
		fmt.Println("[CLEAR-DEBUG] Warning: Could not access httpClient field")
		return packageNames
	}
	httpClient := reflect.NewAt(httpClientField.Type(), unsafe.Pointer(httpClientField.UnsafeAddr())).Elem().Interface().(*http.Client)

	// Get the kubernetesURL field
	kubernetesURLField := v.FieldByName("kubernetesURL")
	if !kubernetesURLField.IsValid() {
		fmt.Println("[CLEAR-DEBUG] Warning: Could not access kubernetesURL field")
		return packageNames
	}
	kubernetesURL := reflect.NewAt(kubernetesURLField.Type(), unsafe.Pointer(kubernetesURLField.UnsafeAddr())).Elem().String()

	// Get the namespace field
	namespaceField := v.FieldByName("namespace")
	if !namespaceField.IsValid() {
		fmt.Println("[CLEAR-DEBUG] Warning: Could not access namespace field")
		return packageNames
	}
	namespace := reflect.NewAt(namespaceField.Type(), unsafe.Pointer(namespaceField.UnsafeAddr())).Elem().String()

	// Get the repository field - CRITICAL: we only want packages from focom-resources
	repositoryField := v.FieldByName("repository")
	if !repositoryField.IsValid() {
		fmt.Println("[CLEAR-DEBUG] Warning: Could not access repository field")
		return packageNames
	}
	repository := reflect.NewAt(repositoryField.Type(), unsafe.Pointer(repositoryField.UnsafeAddr())).Elem().String()

	// Get the token field
	tokenField := v.FieldByName("token")
	token := ""
	if tokenField.IsValid() {
		token = reflect.NewAt(tokenField.Type(), unsafe.Pointer(tokenField.UnsafeAddr())).Elem().String()
	}

	// Make direct API call to list ALL PackageRevisions
	apiPath := fmt.Sprintf("%s/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisions", kubernetesURL, namespace)

	req, err := http.NewRequestWithContext(ctx, "GET", apiPath, nil)
	if err != nil {
		fmt.Printf("[CLEAR-DEBUG] Error creating request: %v\n", err)
		return packageNames
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		fmt.Printf("[CLEAR-DEBUG] Error making request: %v\n", err)
		return packageNames
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("[CLEAR-DEBUG] Error response from Porch API: %d %s\n", resp.StatusCode, string(body))
		return packageNames
	}

	// Parse the response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("[CLEAR-DEBUG] Error decoding response: %v\n", err)
		return packageNames
	}

	// Extract package names from items
	items, ok := result["items"].([]interface{})
	if !ok {
		fmt.Println("[CLEAR-DEBUG] No items found in response")
		return packageNames
	}

	fmt.Printf("[CLEAR-DEBUG] Filtering packages for repository: %s\n", repository)

	for _, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		spec, ok := itemMap["spec"].(map[string]interface{})
		if !ok {
			continue
		}

		// CRITICAL: Only include packages from the focom-resources repository
		pkgRepository, ok := spec["repository"].(string)
		if !ok || pkgRepository != repository {
			continue // Skip packages from other repositories
		}

		pkgName, ok := spec["packageName"].(string)
		if !ok || pkgName == "" {
			continue
		}

		if !seenNames[pkgName] {
			packageNames = append(packageNames, pkgName)
			seenNames[pkgName] = true
		}
	}

	fmt.Printf("[CLEAR-DEBUG] Found %d unique packages in repository '%s' via direct API call\n", len(packageNames), repository)
	return packageNames
}

// LoadTestData loads test fixtures into the storage
func (ts *TestServer) LoadTestData() error {
	fmt.Println("[LOAD-DEBUG] ========== LoadTestData STARTED ==========")

	// Clear storage first to avoid conflicts
	fmt.Println("[LOAD-DEBUG] Calling ClearStorage()...")
	if err := ts.ClearStorage(); err != nil {
		return err
	}
	fmt.Println("[LOAD-DEBUG] ClearStorage() completed")

	// Give Porch extra time to fully process deletions before creating new resources
	// This prevents "package already exists" errors when tests run back-to-back
	fmt.Println("[LOAD-DEBUG] Waiting 10 seconds after cleanup before loading test data...")
	time.Sleep(10 * time.Second)
	fmt.Println("[LOAD-DEBUG] Wait complete, starting to create fixtures...")

	fixtures := testfixtures.NewTestFixtures()

	// Load approved resources
	fmt.Printf("[LOAD-DEBUG] Creating %d OCloud fixtures...\n", len(fixtures.OClouds))
	for i, ocloud := range fixtures.OClouds {
		if ocloud.State == models.StateApproved {
			fmt.Printf("[LOAD-DEBUG] Creating OCloud %d/%d: %s\n", i+1, len(fixtures.OClouds), ocloud.ID)
			if err := ts.storage.CreateRevision(context.Background(), storage.ResourceTypeOCloud, ocloud.ID, ocloud.RevisionID, ocloud); err != nil {
				return fmt.Errorf("failed to create OCloud revision %s: %w", ocloud.ID, err)
			}
			fmt.Printf("[LOAD-DEBUG] Successfully created OCloud: %s\n", ocloud.ID)
		}
	}

	fmt.Printf("[LOAD-DEBUG] Creating %d TemplateInfo fixtures...\n", len(fixtures.TemplateInfos))
	for i, template := range fixtures.TemplateInfos {
		if template.State == models.StateApproved {
			fmt.Printf("[LOAD-DEBUG] Creating TemplateInfo %d/%d: %s\n", i+1, len(fixtures.TemplateInfos), template.ID)
			if err := ts.storage.CreateRevision(context.Background(), storage.ResourceTypeTemplateInfo, template.ID, template.RevisionID, template); err != nil {
				return fmt.Errorf("failed to create TemplateInfo revision %s: %w", template.ID, err)
			}
			fmt.Printf("[LOAD-DEBUG] Successfully created TemplateInfo: %s\n", template.ID)
		}
	}

	fmt.Printf("[LOAD-DEBUG] Creating %d FPR fixtures...\n", len(fixtures.FocomProvisioningRequests))
	for i, fpr := range fixtures.FocomProvisioningRequests {
		if fpr.State == models.StateApproved {
			fmt.Printf("[LOAD-DEBUG] Creating FPR %d/%d: %s\n", i+1, len(fixtures.FocomProvisioningRequests), fpr.ID)
			if err := ts.storage.CreateRevision(context.Background(), storage.ResourceTypeFocomProvisioningRequest, fpr.ID, fpr.RevisionID, fpr); err != nil {
				return fmt.Errorf("failed to create FPR revision %s: %w", fpr.ID, err)
			}
			fmt.Printf("[LOAD-DEBUG] Successfully created FPR: %s\n", fpr.ID)
		}
	}

	// Load additional revision fixtures (for multiple revisions per resource)
	fmt.Printf("[LOAD-DEBUG] Creating %d RevisionResource fixtures...\n", len(fixtures.RevisionResources))
	for i, revision := range fixtures.RevisionResources {
		// Convert models.ResourceType to storage.ResourceType
		var storageResourceType storage.ResourceType
		switch revision.ResourceType {
		case models.ResourceTypeOCloud:
			storageResourceType = storage.ResourceTypeOCloud
		case models.ResourceTypeTemplateInfo:
			storageResourceType = storage.ResourceTypeTemplateInfo
		case models.ResourceTypeFocomProvisioningRequest:
			storageResourceType = storage.ResourceTypeFocomProvisioningRequest
		}

		fmt.Printf("[LOAD-DEBUG] Creating RevisionResource %d/%d: %s (revision: %s)\n", i+1, len(fixtures.RevisionResources), revision.ResourceID, revision.RevisionID)
		if err := ts.storage.CreateRevision(context.Background(), storageResourceType, revision.ResourceID, revision.RevisionID, revision.RevisionData); err != nil {
			return fmt.Errorf("failed to create revision %s/%s: %w", revision.ResourceID, revision.RevisionID, err)
		}
		fmt.Printf("[LOAD-DEBUG] Successfully created RevisionResource: %s/%s\n", revision.ResourceID, revision.RevisionID)
	}

	// Load draft resources
	fmt.Printf("[LOAD-DEBUG] Creating %d DraftResource fixtures...\n", len(fixtures.DraftResources))
	for i, draft := range fixtures.DraftResources {
		// Convert models.ResourceType to storage.ResourceType
		var storageResourceType storage.ResourceType
		switch draft.ResourceType {
		case models.ResourceTypeOCloud:
			storageResourceType = storage.ResourceTypeOCloud
		case models.ResourceTypeTemplateInfo:
			storageResourceType = storage.ResourceTypeTemplateInfo
		case models.ResourceTypeFocomProvisioningRequest:
			storageResourceType = storage.ResourceTypeFocomProvisioningRequest
		}

		fmt.Printf("[LOAD-DEBUG] Creating DraftResource %d/%d: %s\n", i+1, len(fixtures.DraftResources), draft.ResourceID)
		if err := ts.storage.CreateDraft(context.Background(), storageResourceType, draft.DraftData); err != nil {
			return fmt.Errorf("failed to create draft %s: %w", draft.ResourceID, err)
		}
		fmt.Printf("[LOAD-DEBUG] Successfully created DraftResource: %s\n", draft.ResourceID)

		// Update draft state if needed
		if draft.State != models.StateDraft {
			// Convert models.ResourceState to storage.ResourceState
			var storageState storage.ResourceState
			switch draft.State {
			case models.StateDraft:
				storageState = storage.StateDraft
			case models.StateValidated:
				storageState = storage.StateValidated
			case models.StateApproved:
				storageState = storage.StateApproved
			}

			fmt.Printf("[LOAD-DEBUG] Updating draft state for %s to %s\n", draft.ResourceID, draft.State)
			if err := ts.storage.UpdateDraftState(context.Background(), storageResourceType, draft.ResourceID, storageState); err != nil {
				return fmt.Errorf("failed to update draft state for %s: %w", draft.ResourceID, err)
			}
		}
	}

	fmt.Println("[LOAD-DEBUG] ========== LoadTestData COMPLETED ==========")
	return nil
}

// makeRequest makes an HTTP request to the test server
func (ts *TestServer) makeRequest(method, path string, body interface{}) (*http.Response, error) {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, ts.URL()+path, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Use longer timeout for Porch storage (async operations can take time)
	// Approval operations can take 30+ seconds due to async processing
	timeout := 10 * time.Second
	if ts.config != nil && ts.config.Storage.Backend == "porch" {
		timeout = 90 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	return client.Do(req)
}

// parseResponse parses the response body into the given interface
func parseResponse(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(v)
}

// TestAPIInfo tests the API info endpoints
func TestAPIInfo(t *testing.T) {
	ts := NewTestServer()
	defer ts.Close()

	tests := []struct {
		name     string
		path     string
		expected int
	}{
		{"Root endpoint", "/", http.StatusOK},
		{"API info endpoint", "/api/info", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := ts.makeRequest("GET", tt.path, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, resp.StatusCode)

			var info handlers.APIInfo
			err = parseResponse(resp, &info)
			require.NoError(t, err)
			assert.Equal(t, "FOCOM REST NBI", info.Name)
			assert.Equal(t, "v1alpha1", info.Version)
		})
	}
}

// TestHealthEndpoints tests the health check endpoints
func TestHealthEndpoints(t *testing.T) {
	ts := NewTestServer()
	defer ts.Close()

	tests := []struct {
		name     string
		path     string
		expected int
	}{
		{"Liveness check", "/health/live", http.StatusOK},
		{"Readiness check", "/health/ready", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := ts.makeRequest("GET", tt.path, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, resp.StatusCode)

			var health map[string]interface{}
			err = parseResponse(resp, &health)
			require.NoError(t, err)
			assert.Contains(t, health, "status")
			assert.Contains(t, health, "service")
		})
	}
}

// TestMetricsEndpoint tests the metrics endpoint
func TestMetricsEndpoint(t *testing.T) {
	ts := NewTestServer()
	defer ts.Close()

	resp, err := ts.makeRequest("GET", "/metrics", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var metrics map[string]interface{}
	err = parseResponse(resp, &metrics)
	require.NoError(t, err)
	assert.Contains(t, metrics, "service")
	assert.Equal(t, "focom-nbi", metrics["service"])
}

// TestCORSHeaders tests CORS headers are properly set
func TestCORSHeaders(t *testing.T) {
	ts := NewTestServer()
	defer ts.Close()

	// Test OPTIONS request
	req, err := http.NewRequest("OPTIONS", ts.URL()+"/o-clouds", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:3000")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Contains(t, resp.Header.Get("Access-Control-Allow-Methods"), "GET")
	assert.Contains(t, resp.Header.Get("Access-Control-Allow-Methods"), "POST")
}

// TestRequestIDHeader tests that request ID headers are properly handled
func TestRequestIDHeader(t *testing.T) {
	ts := NewTestServer()
	defer ts.Close()

	// Test with provided request ID
	req, err := http.NewRequest("GET", ts.URL()+"/", nil)
	require.NoError(t, err)
	req.Header.Set("X-Request-ID", "test-request-123")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)

	assert.Equal(t, "test-request-123", resp.Header.Get("X-Request-ID"))

	// Test without provided request ID (should generate one)
	resp2, err := ts.makeRequest("GET", "/", nil)
	require.NoError(t, err)

	requestID := resp2.Header.Get("X-Request-ID")
	assert.NotEmpty(t, requestID)
	assert.NotEqual(t, "test-request-123", requestID)
}
