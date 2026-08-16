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
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/models"
	"gopkg.in/yaml.v3"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Verify PorchStorage implements StorageInterface at compile time
var _ StorageInterface = (*PorchStorage)(nil)

// Porch lifecycle states
const (
	PorchLifecycleDraft     = "Draft"
	PorchLifecycleProposed  = "Proposed"
	PorchLifecyclePublished = "Published"
)

// PorchStorage implements StorageInterface using Nephio Porch via REST API
type PorchStorage struct {
	httpClient             *http.Client
	kubernetesURL          string        // e.g., "https://kubernetes.default.svc"
	token                  string        // Service account token
	namespace              string        // Namespace for PackageRevisions (usually "default")
	repository             string        // Porch repository name (e.g., "focom-resources")
	packageRevisionTimeout time.Duration // Timeout for waiting for PackageRevision operations (default: 30s)
}

// PorchStorageConfig holds configuration for Porch storage
type PorchStorageConfig struct {
	KubernetesURL          string        // Kubernetes API server URL (optional, will auto-detect)
	Token                  string        // Service account token (optional, will auto-detect)
	Namespace              string        // Namespace for PackageRevisions (usually "default")
	Repository             string        // Porch repository name (e.g., "focom-resources")
	HTTPSVerify            bool          // Whether to verify HTTPS certificates (default: false for dev)
	UseKubeconfig          bool          // Use kubeconfig for authentication (handles exec, certs, etc.)
	Kubeconfig             string        // Path to kubeconfig file (optional, defaults to KUBECONFIG env or ~/.kube/config)
	PackageRevisionTimeout time.Duration // Timeout for waiting for PackageRevision operations (optional, default: 30s)
}

// NewPorchStorage creates a new PorchStorage instance with auto-detection of Kubernetes configuration
func NewPorchStorage(config *PorchStorageConfig) (*PorchStorage, error) {
	// Validate required configuration
	if config.Namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	if config.Repository == "" {
		return nil, fmt.Errorf("repository is required")
	}

	// If UseKubeconfig is true, use client-go to load config and create authenticated client
	if config.UseKubeconfig {
		return newPorchStorageFromKubeconfig(config)
	}

	// Otherwise, use token-based authentication (original method)
	return newPorchStorageWithToken(config)
}

// newPorchStorageFromKubeconfig creates PorchStorage using kubeconfig (supports exec, certs, etc.)
func newPorchStorageFromKubeconfig(config *PorchStorageConfig) (*PorchStorage, error) {
	// Determine kubeconfig path
	kubeconfigPath := config.Kubeconfig
	if kubeconfigPath == "" {
		kubeconfigPath = os.Getenv("KUBECONFIG")
		if kubeconfigPath == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get home directory: %w", err)
			}
			kubeconfigPath = filepath.Join(homeDir, ".kube", "config")
		}
	}

	// Load kubeconfig
	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	// Create HTTP client with kubeconfig's transport (handles auth automatically)
	httpClient, err := rest.HTTPClientFor(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client from kubeconfig: %w", err)
	}

	// Set timeout
	httpClient.Timeout = 30 * time.Second

	// Set default PackageRevision timeout if not specified
	prTimeout := config.PackageRevisionTimeout
	if prTimeout == 0 {
		prTimeout = 30 * time.Second
	}

	return &PorchStorage{
		httpClient:             httpClient,
		kubernetesURL:          restConfig.Host,
		token:                  restConfig.BearerToken, // May be empty if using exec/cert auth
		namespace:              config.Namespace,
		repository:             config.Repository,
		packageRevisionTimeout: prTimeout,
	}, nil
}

// newPorchStorageWithToken creates PorchStorage using token-based authentication (original method)
func newPorchStorageWithToken(config *PorchStorageConfig) (*PorchStorage, error) {
	// 1. Get Kubernetes API URL (env var or default)
	kubernetesURL := config.KubernetesURL
	if kubernetesURL == "" {
		kubernetesURL = os.Getenv("KUBERNETES_BASE_URL")
		if kubernetesURL == "" {
			kubernetesURL = "https://kubernetes.default.svc"
		}
	}

	// 2. Get service account token (env var, file, or kubeconfig)
	token := config.Token
	if token == "" {
		var err error
		token, err = resolveToken()
		if err != nil {
			return nil, fmt.Errorf("failed to resolve authentication token: %w", err)
		}
	}

	// 3. Create HTTP client with timeout and TLS config
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: !config.HTTPSVerify, // #nosec G402 -- configurable TLS verification for dev environments
			},
		},
	}

	// Set default PackageRevision timeout if not specified
	prTimeout := config.PackageRevisionTimeout
	if prTimeout == 0 {
		prTimeout = 30 * time.Second
	}

	return &PorchStorage{
		httpClient:             httpClient,
		kubernetesURL:          kubernetesURL,
		token:                  token,
		namespace:              config.Namespace,
		repository:             config.Repository,
		packageRevisionTimeout: prTimeout,
	}, nil
}

// resolveToken attempts to resolve the authentication token from multiple sources
func resolveToken() (string, error) {
	// Try TOKEN environment variable (can be token string or file path)
	tokenEnv := os.Getenv("TOKEN")
	if tokenEnv != "" {
		// Check if it's a file path or token string
		cleanTokenPath := filepath.Clean(tokenEnv)
		if _, err := os.Stat(cleanTokenPath); err == nil {
			// It's a file path, read it
			tokenBytes, err := os.ReadFile(cleanTokenPath) // #nosec G304 -- token path from trusted env var
			if err != nil {
				return "", fmt.Errorf("failed to read token from file %s: %w", tokenEnv, err)
			}
			return strings.TrimSpace(string(tokenBytes)), nil
		}
		// It's a token string
		return tokenEnv, nil
	}

	// Try default in-cluster token path
	tokenPath := "/var/run/secrets/kubernetes.io/serviceaccount/token" // #nosec G101 -- standard k8s service account token path
	if tokenBytes, err := os.ReadFile(tokenPath); err == nil {
		return strings.TrimSpace(string(tokenBytes)), nil
	}

	// Try to read from kubeconfig as fallback
	token, err := readTokenFromKubeconfig()
	if err != nil {
		return "", fmt.Errorf("failed to read token from environment, file, or kubeconfig: %w", err)
	}

	return token, nil
}

// readTokenFromKubeconfig reads token from kubeconfig file
func readTokenFromKubeconfig() (string, error) {
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		kubeconfigPath = filepath.Join(homeDir, ".kube", "config")
	}

	// Read and parse kubeconfig
	data, err := os.ReadFile(kubeconfigPath) // #nosec G304 G703 -- kubeconfig path from trusted source
	if err != nil {
		return "", fmt.Errorf("failed to read kubeconfig: %w", err)
	}

	// Parse YAML to extract token
	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return "", fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	// Extract token from current user
	users, ok := config["users"].([]interface{})
	if !ok || len(users) == 0 {
		return "", fmt.Errorf("no users found in kubeconfig")
	}

	user, ok := users[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid user format in kubeconfig")
	}

	userInfo, ok := user["user"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid user info in kubeconfig")
	}

	if token, ok := userInfo["token"].(string); ok && token != "" {
		return token, nil
	}

	return "", fmt.Errorf("no token found in kubeconfig")
}

// HealthCheck verifies connectivity to Porch
func (s *PorchStorage) HealthCheck(ctx context.Context) error {
	// Create a context with 5-second timeout for health check
	healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Try to list PackageRevisions with limit=1 to verify connectivity
	path := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisions?limit=1", s.namespace)
	resp, err := s.makeRequest(healthCtx, http.MethodGet, path, nil)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "Porch is not accessible", err)
	}
	defer resp.Body.Close()

	// Check for authentication/authorization errors
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return NewStorageError(ErrorCodeStorageFailure, "authentication failed", nil)
	}

	// Check for successful response
	if resp.StatusCode != http.StatusOK {
		return NewStorageError(ErrorCodeStorageFailure,
			fmt.Sprintf("unexpected status code: %d", resp.StatusCode), nil)
	}

	return nil
}

// makeRequest creates and executes an HTTP request to Kubernetes API
func (s *PorchStorage) makeRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	url := fmt.Sprintf("%s%s", s.kubernetesURL, path)
	var req *http.Request

	if len(reqBody) > 0 {
		req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(reqBody))
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token))

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// For non-2xx responses, read and recreate the body so callers can inspect it
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	return resp, nil
}

// parseResponse reads and parses HTTP response
func (s *PorchStorage) parseResponse(resp *http.Response, expectedStatus int, result interface{}) error {
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != expectedStatus {
		return s.handleHTTPError(resp.StatusCode, bodyBytes)
	}

	if result != nil && len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// handleHTTPError maps HTTP status codes to StorageError codes
func (s *PorchStorage) handleHTTPError(statusCode int, body []byte) error {
	switch statusCode {
	case 401, 403:
		return NewStorageError(ErrorCodeStorageFailure, "unauthorized", nil)
	case 404:
		return NewStorageError(ErrorCodeNotFound, "resource not found", ErrResourceNotFound)
	case 409:
		return NewStorageError(ErrorCodeAlreadyExists, "resource already exists", ErrResourceExists)
	case 500:
		return NewStorageError(ErrorCodeStorageFailure, "k8s API server error", nil)
	default:
		return NewStorageError(ErrorCodeStorageFailure,
			fmt.Sprintf("unexpected status %d: %s", statusCode, string(body)), nil)
	}
}

// createKptfile generates a Kptfile in YAML format for a given resource
func (s *PorchStorage) createKptfile(resourceID string, resourceType ResourceType) (string, error) {
	kptfile := map[string]interface{}{
		"apiVersion": "kpt.dev/v1",
		"kind":       "Kptfile",
		"metadata": map[string]interface{}{
			"name": resourceID,
		},
		"info": map[string]interface{}{
			"description": fmt.Sprintf("FOCOM %s resource", resourceType),
		},
	}

	yamlBytes, err := yaml.Marshal(kptfile)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Kptfile: %w", err)
	}

	return string(yamlBytes), nil
}

// createResourceYAML converts a Go struct to Kubernetes-style YAML
func (s *PorchStorage) createResourceYAML(resource interface{}, resourceType ResourceType) (string, error) {
	// Extract resource ID and namespace - these are used for metadata
	var resourceID, namespace string

	switch r := resource.(type) {
	case *OCloudData:
		resourceID = r.ID
		namespace = r.Namespace
	case OCloudData:
		resourceID = r.ID
		namespace = r.Namespace
	case *TemplateInfoData:
		resourceID = r.ID
		namespace = r.Namespace
	case TemplateInfoData:
		resourceID = r.ID
		namespace = r.Namespace
	case *FocomProvisioningRequestData:
		resourceID = r.ID
		namespace = r.Namespace
	case FocomProvisioningRequestData:
		resourceID = r.ID
		namespace = r.Namespace
	default:
		return "", fmt.Errorf("unsupported resource type: %T", resource)
	}

	// Create Kubernetes-style YAML structure matching the CRD spec
	// Note: Internal fields (id, state) are NOT in the CRD spec
	var k8sResource map[string]interface{}

	switch r := resource.(type) {
	case *OCloudData:
		// OCloud CRD spec only has o2imsSecret
		// Store name and description in annotations since they're not in the CRD spec
		annotations := map[string]interface{}{}
		if r.Name != "" {
			annotations["focom.nephio.org/display-name"] = r.Name
		}
		if r.Description != "" {
			annotations["focom.nephio.org/description"] = r.Description
		}
		metadata := map[string]interface{}{
			"name":      resourceID,
			"namespace": namespace,
		}
		if len(annotations) > 0 {
			metadata["annotations"] = annotations
		}
		k8sResource = map[string]interface{}{
			"apiVersion": "focom.nephio.org/v1alpha1",
			"kind":       "OCloud",
			"metadata":   metadata,
			"spec": map[string]interface{}{
				"o2imsSecret": r.O2IMSSecret,
			},
		}
	case OCloudData:
		annotations := map[string]interface{}{}
		if r.Name != "" {
			annotations["focom.nephio.org/display-name"] = r.Name
		}
		if r.Description != "" {
			annotations["focom.nephio.org/description"] = r.Description
		}
		metadata := map[string]interface{}{
			"name":      resourceID,
			"namespace": namespace,
		}
		if len(annotations) > 0 {
			metadata["annotations"] = annotations
		}
		k8sResource = map[string]interface{}{
			"apiVersion": "focom.nephio.org/v1alpha1",
			"kind":       "OCloud",
			"metadata":   metadata,
			"spec": map[string]interface{}{
				"o2imsSecret": r.O2IMSSecret,
			},
		}
	case *TemplateInfoData:
		// TemplateInfo CRD spec has templateName, templateVersion, templateParameterSchema
		// Store name and description in annotations
		annotations := map[string]interface{}{}
		if r.Name != "" {
			annotations["focom.nephio.org/display-name"] = r.Name
		}
		if r.Description != "" {
			annotations["focom.nephio.org/description"] = r.Description
		}
		metadata := map[string]interface{}{
			"name":      resourceID,
			"namespace": namespace,
		}
		if len(annotations) > 0 {
			metadata["annotations"] = annotations
		}
		k8sResource = map[string]interface{}{
			"apiVersion": "provisioning.oran.org/v1alpha1",
			"kind":       "TemplateInfo",
			"metadata":   metadata,
			"spec": map[string]interface{}{
				"templateName":            r.TemplateName,
				"templateVersion":         r.TemplateVersion,
				"templateParameterSchema": r.TemplateParameterSchema,
			},
		}
	case TemplateInfoData:
		annotations := map[string]interface{}{}
		if r.Name != "" {
			annotations["focom.nephio.org/display-name"] = r.Name
		}
		if r.Description != "" {
			annotations["focom.nephio.org/description"] = r.Description
		}
		metadata := map[string]interface{}{
			"name":      resourceID,
			"namespace": namespace,
		}
		if len(annotations) > 0 {
			metadata["annotations"] = annotations
		}
		k8sResource = map[string]interface{}{
			"apiVersion": "provisioning.oran.org/v1alpha1",
			"kind":       "TemplateInfo",
			"metadata":   metadata,
			"spec": map[string]interface{}{
				"templateName":            r.TemplateName,
				"templateVersion":         r.TemplateVersion,
				"templateParameterSchema": r.TemplateParameterSchema,
			},
		}
	case *FocomProvisioningRequestData:
		// FPR CRD spec has required and optional fields
		spec := map[string]interface{}{
			"oCloudId":           r.OCloudID,
			"oCloudNamespace":    r.OCloudNamespace,
			"templateName":       r.TemplateName,
			"templateVersion":    r.TemplateVersion,
			"templateParameters": r.TemplateParameters,
		}
		// Add optional fields if present
		if r.Name != "" {
			spec["name"] = r.Name
		}
		if r.Description != "" {
			spec["description"] = r.Description
		}
		k8sResource = map[string]interface{}{
			"apiVersion": "focom.nephio.org/v1alpha1",
			"kind":       "FocomProvisioningRequest",
			"metadata": map[string]interface{}{
				"name":      resourceID,
				"namespace": namespace,
			},
			"spec": spec,
		}
	case FocomProvisioningRequestData:
		spec := map[string]interface{}{
			"oCloudId":           r.OCloudID,
			"oCloudNamespace":    r.OCloudNamespace,
			"templateName":       r.TemplateName,
			"templateVersion":    r.TemplateVersion,
			"templateParameters": r.TemplateParameters,
		}
		if r.Name != "" {
			spec["name"] = r.Name
		}
		if r.Description != "" {
			spec["description"] = r.Description
		}
		k8sResource = map[string]interface{}{
			"apiVersion": "focom.nephio.org/v1alpha1",
			"kind":       "FocomProvisioningRequest",
			"metadata": map[string]interface{}{
				"name":      resourceID,
				"namespace": namespace,
			},
			"spec": spec,
		}
	default:
		return "", fmt.Errorf("unsupported resource type: %T", resource)
	}

	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(k8sResource)
	if err != nil {
		return "", fmt.Errorf("failed to marshal resource to YAML: %w", err)
	}

	return string(yamlBytes), nil
}

// convertToModelsType converts storage types to models types for handlers
func convertToModelsType(resource interface{}) (interface{}, error) {
	switch r := resource.(type) {
	case *OCloudData:
		return &models.OCloudData{
			BaseResource: models.BaseResource{
				ID:          r.ID,
				RevisionID:  r.RevisionID,
				Namespace:   r.Namespace,
				Name:        r.Name,
				Description: r.Description,
				State:       models.ResourceState(r.State),
				CreatedAt:   r.CreatedAt,
				UpdatedAt:   r.UpdatedAt,
				Metadata:    r.Metadata,
			},
			O2IMSSecret: models.O2IMSSecretRef{
				SecretRef: models.SecretReference{
					Name:      r.O2IMSSecret.SecretRef.Name,
					Namespace: r.O2IMSSecret.SecretRef.Namespace,
				},
			},
		}, nil
	case *TemplateInfoData:
		return &models.TemplateInfoData{
			BaseResource: models.BaseResource{
				ID:          r.ID,
				RevisionID:  r.RevisionID,
				Namespace:   r.Namespace,
				Name:        r.Name,
				Description: r.Description,
				State:       models.ResourceState(r.State),
				CreatedAt:   r.CreatedAt,
				UpdatedAt:   r.UpdatedAt,
				Metadata:    r.Metadata,
			},
			TemplateName:            r.TemplateName,
			TemplateVersion:         r.TemplateVersion,
			TemplateParameterSchema: r.TemplateParameterSchema,
		}, nil
	case *FocomProvisioningRequestData:
		return &models.FocomProvisioningRequestData{
			BaseResource: models.BaseResource{
				ID:          r.ID,
				RevisionID:  r.RevisionID,
				Namespace:   r.Namespace,
				Name:        r.Name,
				Description: r.Description,
				State:       models.ResourceState(r.State),
				CreatedAt:   r.CreatedAt,
				UpdatedAt:   r.UpdatedAt,
				Metadata:    r.Metadata,
			},
			OCloudID:           r.OCloudID,
			OCloudNamespace:    r.OCloudNamespace,
			TemplateName:       r.TemplateName,
			TemplateVersion:    r.TemplateVersion,
			TemplateParameters: r.TemplateParameters,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported resource type for conversion: %T", resource)
	}
}

// parseResourceYAML converts Kubernetes-style YAML to a Go struct
func (s *PorchStorage) parseResourceYAML(yamlContent string, resourceType ResourceType) (interface{}, error) {
	// Parse YAML to generic map
	var k8sResource map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlContent), &k8sResource); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	// Extract spec section
	spec, ok := k8sResource["spec"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing or invalid spec section in YAML")
	}

	// Extract metadata section
	metadata, _ := k8sResource["metadata"].(map[string]interface{})
	namespace, _ := metadata["namespace"].(string)
	// ID comes from metadata.name, not spec
	id, _ := metadata["name"].(string)

	// Extract name and description from annotations (if present)
	var displayName, description string
	if annotations, ok := metadata["annotations"].(map[string]interface{}); ok {
		displayName, _ = annotations["focom.nephio.org/display-name"].(string)
		description, _ = annotations["focom.nephio.org/description"].(string)
	}
	// Default to ID if no display name
	if displayName == "" {
		displayName = id
	}

	// Note: state is NOT in the CRD spec
	// State will be set based on the PackageRevision lifecycle by the caller

	// Create resource based on type
	switch resourceType {
	case ResourceTypeOCloud:
		ocloud := &OCloudData{
			BaseResource: BaseResource{
				ID:          id,
				Namespace:   namespace,
				Name:        displayName,
				Description: description,
				State:       "", // Will be set by caller based on PackageRevision lifecycle
				Metadata:    nil,
			},
		}
		// Extract O2IMSSecret (handle both camelCase and lowercase from YAML)
		if o2imsSecret, ok := spec["o2imsSecret"].(map[string]interface{}); ok {
			// Try camelCase first, then lowercase
			var secretRef map[string]interface{}
			var ok bool
			if secretRef, ok = o2imsSecret["secretRef"].(map[string]interface{}); !ok {
				secretRef, ok = o2imsSecret["secretref"].(map[string]interface{})
			}
			if ok {
				name, _ := secretRef["name"].(string)
				ns, _ := secretRef["namespace"].(string)
				ocloud.O2IMSSecret = O2IMSSecretRef{
					SecretRef: SecretReference{
						Name:      name,
						Namespace: ns,
					},
				}
			}
		}
		return ocloud, nil

	case ResourceTypeTemplateInfo:
		templateInfo := &TemplateInfoData{
			BaseResource: BaseResource{
				ID:          id,
				Namespace:   namespace,
				Name:        displayName,
				Description: description,
				State:       "", // Will be set by caller
				Metadata:    nil,
			},
			TemplateName:            spec["templateName"].(string),
			TemplateVersion:         spec["templateVersion"].(string),
			TemplateParameterSchema: spec["templateParameterSchema"].(string),
		}
		return templateInfo, nil

	case ResourceTypeFocomProvisioningRequest:
		// FPR has optional name and description in the CRD spec
		fprName, _ := spec["name"].(string)
		fprDescription, _ := spec["description"].(string)
		if fprName == "" {
			fprName = id // Default to ID if not specified
		}

		fpr := &FocomProvisioningRequestData{
			BaseResource: BaseResource{
				ID:          id,
				Namespace:   namespace,
				Name:        fprName,
				Description: fprDescription,
				State:       "", // Will be set by caller
				Metadata:    nil,
			},
			OCloudID:           spec["oCloudId"].(string),
			OCloudNamespace:    spec["oCloudNamespace"].(string),
			TemplateName:       spec["templateName"].(string),
			TemplateVersion:    spec["templateVersion"].(string),
			TemplateParameters: spec["templateParameters"].(map[string]interface{}),
		}
		return fpr, nil

	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

// extractResourceID extracts the ID from a resource
// This function handles both storage package types and models package types
func (s *PorchStorage) extractResourceID(resource interface{}) (string, error) {
	if resource == nil {
		return "", fmt.Errorf("resource is nil")
	}

	switch r := resource.(type) {
	// Models package types (used by handlers)
	case *models.OCloudData:
		if r == nil {
			return "", fmt.Errorf("resource pointer is nil")
		}
		return r.ID, nil
	case models.OCloudData:
		return r.ID, nil
	case *models.TemplateInfoData:
		if r == nil {
			return "", fmt.Errorf("resource pointer is nil")
		}
		return r.ID, nil
	case models.TemplateInfoData:
		return r.ID, nil
	case *models.FocomProvisioningRequestData:
		if r == nil {
			return "", fmt.Errorf("resource pointer is nil")
		}
		return r.ID, nil
	case models.FocomProvisioningRequestData:
		return r.ID, nil

	// Storage package types (for backward compatibility)
	case *OCloudData:
		if r == nil {
			return "", fmt.Errorf("resource pointer is nil")
		}
		return r.ID, nil
	case OCloudData:
		return r.ID, nil
	case *TemplateInfoData:
		if r == nil {
			return "", fmt.Errorf("resource pointer is nil")
		}
		return r.ID, nil
	case TemplateInfoData:
		return r.ID, nil
	case *FocomProvisioningRequestData:
		if r == nil {
			return "", fmt.Errorf("resource pointer is nil")
		}
		return r.ID, nil
	case FocomProvisioningRequestData:
		return r.ID, nil
	default:
		return "", fmt.Errorf("unsupported resource type: %T", resource)
	}
}

// extractRevisionID extracts the RevisionID from a resource
func (s *PorchStorage) extractRevisionID(resource interface{}) (string, error) {
	if resource == nil {
		return "", fmt.Errorf("resource is nil")
	}

	switch r := resource.(type) {
	// Models package types (used by handlers)
	case *models.OCloudData:
		if r == nil {
			return "", fmt.Errorf("resource pointer is nil")
		}
		return r.RevisionID, nil
	case models.OCloudData:
		return r.RevisionID, nil
	case *models.TemplateInfoData:
		if r == nil {
			return "", fmt.Errorf("resource pointer is nil")
		}
		return r.RevisionID, nil
	case models.TemplateInfoData:
		return r.RevisionID, nil
	case *models.FocomProvisioningRequestData:
		if r == nil {
			return "", fmt.Errorf("resource pointer is nil")
		}
		return r.RevisionID, nil
	case models.FocomProvisioningRequestData:
		return r.RevisionID, nil

	// Storage package types (for backward compatibility)
	case *OCloudData:
		if r == nil {
			return "", fmt.Errorf("resource pointer is nil")
		}
		return r.RevisionID, nil
	case OCloudData:
		return r.RevisionID, nil
	case *TemplateInfoData:
		if r == nil {
			return "", fmt.Errorf("resource pointer is nil")
		}
		return r.RevisionID, nil
	case TemplateInfoData:
		return r.RevisionID, nil
	case *FocomProvisioningRequestData:
		if r == nil {
			return "", fmt.Errorf("resource pointer is nil")
		}
		return r.RevisionID, nil
	case FocomProvisioningRequestData:
		return r.RevisionID, nil
	default:
		return "", fmt.Errorf("unsupported resource type: %T", resource)
	}
}

// setResourceState sets the state field of a resource based on Porch lifecycle
func (s *PorchStorage) setResourceState(resource interface{}, porchLifecycle string) {
	var state ResourceState
	switch porchLifecycle {
	case PorchLifecycleDraft:
		state = StateDraft
	case PorchLifecycleProposed:
		state = StateValidated
	case PorchLifecyclePublished:
		state = StateApproved
	default:
		state = StateDraft
	}

	// Set the state on the resource
	switch r := resource.(type) {
	case *OCloudData:
		r.State = state
	case *TemplateInfoData:
		r.State = state
	case *FocomProvisioningRequestData:
		r.State = state
	}
}

// setResourceRevisionID sets the revisionID field of a resource
func (s *PorchStorage) setResourceRevisionID(resource interface{}, revisionID string) {
	// Set the revisionID on the resource
	switch r := resource.(type) {
	case *OCloudData:
		r.RevisionID = revisionID
	case *TemplateInfoData:
		r.RevisionID = revisionID
	case *FocomProvisioningRequestData:
		r.RevisionID = revisionID
	}
}

// generateWorkspaceName generates a unique workspace name for a draft
// Ensures the workspace name stays within Kubernetes 63 character limit
func (s *PorchStorage) generateWorkspaceName(resourceID string) string {
	timestamp := time.Now().Unix()
	// Format: "draft-{resourceID}-{timestamp}"
	// Timestamp is 10 digits, "draft-" is 6 chars, hyphen is 1 char
	// So we need: 6 + resourceID + 1 + 10 = 17 + resourceID <= 63
	// Therefore resourceID must be <= 46 characters

	maxResourceIDLen := 46
	truncatedID := resourceID
	if len(resourceID) > maxResourceIDLen {
		truncatedID = resourceID[:maxResourceIDLen]
		// Ensure we don't end with a hyphen after truncation
		truncatedID = strings.TrimRight(truncatedID, "-")
	}

	return fmt.Sprintf("draft-%s-%d", truncatedID, timestamp)
}

// convertToStorageType converts models package types to storage package types
func (s *PorchStorage) convertToStorageType(resource interface{}, resourceType ResourceType) interface{} {
	switch r := resource.(type) {
	case *models.OCloudData:
		return &OCloudData{
			BaseResource: BaseResource{
				ID:          r.ID,
				RevisionID:  r.RevisionID,
				Namespace:   r.Namespace,
				Name:        r.Name,
				Description: r.Description,
				State:       ResourceState(r.State),
				CreatedAt:   r.CreatedAt,
				UpdatedAt:   r.UpdatedAt,
				Metadata:    r.Metadata,
			},
			O2IMSSecret: O2IMSSecretRef{
				SecretRef: SecretReference{
					Name:      r.O2IMSSecret.SecretRef.Name,
					Namespace: r.O2IMSSecret.SecretRef.Namespace,
				},
			},
		}
	case *models.TemplateInfoData:
		return &TemplateInfoData{
			BaseResource: BaseResource{
				ID:          r.ID,
				RevisionID:  r.RevisionID,
				Namespace:   r.Namespace,
				Name:        r.Name,
				Description: r.Description,
				State:       ResourceState(r.State),
				CreatedAt:   r.CreatedAt,
				UpdatedAt:   r.UpdatedAt,
				Metadata:    r.Metadata,
			},
			TemplateName:            r.TemplateName,
			TemplateVersion:         r.TemplateVersion,
			TemplateParameterSchema: r.TemplateParameterSchema,
		}
	case *models.FocomProvisioningRequestData:
		return &FocomProvisioningRequestData{
			BaseResource: BaseResource{
				ID:          r.ID,
				RevisionID:  r.RevisionID,
				Namespace:   r.Namespace,
				Name:        r.Name,
				Description: r.Description,
				State:       ResourceState(r.State),
				CreatedAt:   r.CreatedAt,
				UpdatedAt:   r.UpdatedAt,
				Metadata:    r.Metadata,
			},
			OCloudID:           r.OCloudID,
			OCloudNamespace:    r.OCloudNamespace,
			TemplateName:       r.TemplateName,
			TemplateVersion:    r.TemplateVersion,
			TemplateParameters: r.TemplateParameters,
		}
	default:
		// Already a storage type or unknown type, return as-is
		return resource
	}
}

// CreateDraft creates a new draft resource in Porch
func (s *PorchStorage) CreateDraft(ctx context.Context, resourceType ResourceType, draft interface{}) error {
	// Convert models types to storage types if needed
	draft = s.convertToStorageType(draft, resourceType)

	// Extract resource ID
	resourceID, err := s.extractResourceID(draft)
	if err != nil {
		return NewStorageError(ErrorCodeInvalidID, "failed to extract resource ID", err)
	}

	// Check if draft already exists
	existing, err := s.findPackageRevision(ctx, resourceID, PorchLifecycleDraft)
	if err != nil {
		return err
	}
	if existing != nil {
		return NewStorageError(ErrorCodeAlreadyExists,
			fmt.Sprintf("draft for resource %s already exists", resourceID), ErrResourceExists)
	}

	// Generate workspace name
	workspaceName := s.generateWorkspaceName(resourceID)

	// Create PackageRevision request body
	prRequest := map[string]interface{}{
		"apiVersion": "porch.kpt.dev/v1alpha1",
		"kind":       "PackageRevision",
		"spec": map[string]interface{}{
			"packageName":   resourceID,
			"repository":    s.repository,
			"lifecycle":     PorchLifecycleDraft,
			"workspaceName": workspaceName,
		},
	}

	// Make POST request to create PackageRevision
	path := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisions", s.namespace)
	resp, err := s.makeRequest(ctx, http.MethodPost, path, prRequest)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to create PackageRevision", err)
	}

	// Accept both 201 (immediate success) and 500 (async processing)
	// Porch uses asynchronous admission webhooks that may return 500 initially
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusInternalServerError {
		bodyBytes, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return s.handleHTTPError(resp.StatusCode, bodyBytes)
	}
	_ = resp.Body.Close()

	// Wait for PackageRevision to exist (handles async creation)
	prName, err := s.waitForPackageRevision(ctx, resourceID, PorchLifecycleDraft, s.packageRevisionTimeout)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "PackageRevision not created", err)
	}

	// Create package contents (Kptfile + resource YAML)
	kptfile, err := s.createKptfile(resourceID, resourceType)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to create Kptfile", err)
	}

	resourceYAML, err := s.createResourceYAML(draft, resourceType)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to create resource YAML", err)
	}

	// Determine resource filename based on type
	var resourceFilename string
	switch resourceType {
	case ResourceTypeOCloud:
		resourceFilename = "ocloud.yaml"
	case ResourceTypeTemplateInfo:
		resourceFilename = "templateinfo.yaml"
	case ResourceTypeFocomProvisioningRequest:
		resourceFilename = "focomprovisioningrequest.yaml"
	default:
		return NewStorageError(ErrorCodeStorageFailure,
			fmt.Sprintf("unsupported resource type: %s", resourceType), nil)
	}

	// Update PackageRevisionResources with package contents
	// Retry with backoff since PackageRevisionResources might not be immediately available
	prrPath := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisionresources/%s",
		s.namespace, prName)

	maxRetries := 10
	retryDelay := 500 * time.Millisecond
	var prrResp *http.Response
	var existingPRR map[string]interface{}

	for i := 0; i < maxRetries; i++ {
		// First, GET the existing PackageRevisionResources to merge with our changes
		getResp, getErr := s.makeRequest(ctx, http.MethodGet, prrPath, nil)
		if getErr == nil && getResp.StatusCode == http.StatusOK {
			// Parse existing resources
			if parseErr := s.parseResponse(getResp, http.StatusOK, &existingPRR); parseErr == nil {
				// Extract existing resources
				spec, ok := existingPRR["spec"].(map[string]interface{})
				if ok {
					resources, ok := spec["resources"].(map[string]interface{})
					if !ok {
						resources = make(map[string]interface{})
					}

					// Merge our files with existing resources
					resources["Kptfile"] = kptfile
					resources[resourceFilename] = resourceYAML

					// Create PUT request with merged resources
					prrRequest := map[string]interface{}{
						"apiVersion": "porch.kpt.dev/v1alpha1",
						"kind":       "PackageRevisionResources",
						"metadata":   existingPRR["metadata"],
						"spec": map[string]interface{}{
							"packageName": resourceID,
							"repository":  s.repository,
							"resources":   resources,
						},
					}

					// PUT the updated resources
					prrResp, err = s.makeRequest(ctx, http.MethodPut, prrPath, prrRequest)
					if err == nil && prrResp.StatusCode == http.StatusOK {
						break
					}
				}
			}
		}

		// If GET failed or PUT failed, the resource might not be ready yet, retry
		time.Sleep(retryDelay)
	}

	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to update package contents after retries", err)
	}
	if prrResp == nil || prrResp.StatusCode != http.StatusOK {
		return NewStorageError(ErrorCodeStorageFailure, "failed to update package contents: resource not ready", nil)
	}

	// Parse response
	if err := s.parseResponse(prrResp, http.StatusOK, nil); err != nil {
		return err
	}

	return nil
}

// GetDraft retrieves a draft resource from Porch
func (s *PorchStorage) GetDraft(ctx context.Context, resourceType ResourceType, id string) (interface{}, error) {
	// Find draft PackageRevision (search both Draft and Proposed states)
	pr, err := s.findPackageRevision(ctx, id, PorchLifecycleDraft, PorchLifecycleProposed)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, NewStorageError(ErrorCodeNotFound,
			fmt.Sprintf("draft for resource %s not found", id), ErrResourceNotFound)
	}

	// Extract PackageRevision name
	metadata, ok := pr["metadata"].(map[string]interface{})
	if !ok {
		return nil, NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevision: missing metadata", nil)
	}
	prName, ok := metadata["name"].(string)
	if !ok {
		return nil, NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevision: missing name", nil)
	}

	// Retrieve PackageRevisionResources to get package contents
	prrPath := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisionresources/%s",
		s.namespace, prName)
	resp, err := s.makeRequest(ctx, http.MethodGet, prrPath, nil)
	if err != nil {
		return nil, NewStorageError(ErrorCodeStorageFailure, "failed to retrieve package contents", err)
	}

	// Parse response
	var prr map[string]interface{}
	if err := s.parseResponse(resp, http.StatusOK, &prr); err != nil {
		return nil, err
	}

	// Extract resources from spec
	spec, ok := prr["spec"].(map[string]interface{})
	if !ok {
		return nil, NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevisionResources: missing spec", nil)
	}
	resources, ok := spec["resources"].(map[string]interface{})
	if !ok {
		return nil, NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevisionResources: missing resources", nil)
	}

	// Determine resource filename based on type
	var resourceFilename string
	switch resourceType {
	case ResourceTypeOCloud:
		resourceFilename = "ocloud.yaml"
	case ResourceTypeTemplateInfo:
		resourceFilename = "templateinfo.yaml"
	case ResourceTypeFocomProvisioningRequest:
		resourceFilename = "focomprovisioningrequest.yaml"
	default:
		return nil, NewStorageError(ErrorCodeStorageFailure,
			fmt.Sprintf("unsupported resource type: %s", resourceType), nil)
	}

	// Get resource YAML content
	resourceYAML, ok := resources[resourceFilename].(string)
	if !ok {
		return nil, NewStorageError(ErrorCodeStorageFailure,
			fmt.Sprintf("resource file %s not found in package", resourceFilename), nil)
	}

	// Parse YAML to Go struct
	resource, err := s.parseResourceYAML(resourceYAML, resourceType)
	if err != nil {
		return nil, NewStorageError(ErrorCodeStorageFailure, "failed to parse resource YAML", err)
	}

	// Set state based on PackageRevision lifecycle
	prSpec, _ := pr["spec"].(map[string]interface{})
	lifecycle, _ := prSpec["lifecycle"].(string)
	s.setResourceState(resource, lifecycle)

	// Convert storage types to models types for the handler
	modelResource, err := convertToModelsType(resource)
	if err != nil {
		return nil, NewStorageError(ErrorCodeStorageFailure, "failed to convert resource type", err)
	}

	return modelResource, nil
}

// UpdateDraft updates an existing draft resource in Porch
func (s *PorchStorage) UpdateDraft(ctx context.Context, resourceType ResourceType, id string, draft interface{}) error {
	// Convert models types to storage types if needed
	draft = s.convertToStorageType(draft, resourceType)

	// Find existing draft PackageRevision (search for both Draft and Proposed)
	pr, err := s.findPackageRevision(ctx, id, PorchLifecycleDraft, PorchLifecycleProposed)
	if err != nil {
		return err
	}
	if pr == nil {
		return NewStorageError(ErrorCodeNotFound,
			fmt.Sprintf("draft for resource %s not found", id), ErrResourceNotFound)
	}

	// Extract spec to check lifecycle state
	spec, ok := pr["spec"].(map[string]interface{})
	if !ok {
		return NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevision: missing spec", nil)
	}

	// Verify draft is in Draft state (not Proposed)
	lifecycle, _ := spec["lifecycle"].(string)
	if lifecycle != PorchLifecycleDraft {
		return NewStorageError(ErrorCodeInvalidState,
			fmt.Sprintf("cannot update draft in %s state, must be in Draft state", lifecycle), nil)
	}

	// Extract PackageRevision name
	metadata, ok := pr["metadata"].(map[string]interface{})
	if !ok {
		return NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevision: missing metadata", nil)
	}
	prName, ok := metadata["name"].(string)
	if !ok {
		return NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevision: missing name", nil)
	}

	// Create updated package contents (Kptfile + resource YAML)
	kptfile, err := s.createKptfile(id, resourceType)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to create Kptfile", err)
	}

	resourceYAML, err := s.createResourceYAML(draft, resourceType)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to create resource YAML", err)
	}

	// Determine resource filename based on type
	var resourceFilename string
	switch resourceType {
	case ResourceTypeOCloud:
		resourceFilename = "ocloud.yaml"
	case ResourceTypeTemplateInfo:
		resourceFilename = "templateinfo.yaml"
	case ResourceTypeFocomProvisioningRequest:
		resourceFilename = "focomprovisioningrequest.yaml"
	default:
		return NewStorageError(ErrorCodeStorageFailure,
			fmt.Sprintf("unsupported resource type: %s", resourceType), nil)
	}

	// Update PackageRevisionResources with new package contents
	// Use GET-merge-PUT pattern to preserve existing resources
	prrPath := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisionresources/%s",
		s.namespace, prName)

	// GET existing PackageRevisionResources
	getResp, getErr := s.makeRequest(ctx, http.MethodGet, prrPath, nil)
	if getErr != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to get package contents", getErr)
	}

	var existingPRR map[string]interface{}
	if parseErr := s.parseResponse(getResp, http.StatusOK, &existingPRR); parseErr != nil {
		return parseErr
	}

	// Extract existing resources
	var spec2 map[string]interface{}
	var ok2 bool
	spec2, ok2 = existingPRR["spec"].(map[string]interface{})
	if !ok2 {
		return NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevisionResources: missing spec", nil)
	}
	resources, ok3 := spec2["resources"].(map[string]interface{})
	if !ok3 {
		resources = make(map[string]interface{})
	}

	// Merge our files with existing resources
	resources["Kptfile"] = kptfile
	resources[resourceFilename] = resourceYAML

	// Create PUT request with merged resources
	prrRequest := map[string]interface{}{
		"apiVersion": "porch.kpt.dev/v1alpha1",
		"kind":       "PackageRevisionResources",
		"metadata":   existingPRR["metadata"],
		"spec": map[string]interface{}{
			"packageName": id,
			"repository":  s.repository,
			"resources":   resources,
		},
	}

	// PUT the updated resources
	resp, err := s.makeRequest(ctx, http.MethodPut, prrPath, prrRequest)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to update package contents", err)
	}

	// Parse response
	if err := s.parseResponse(resp, http.StatusOK, nil); err != nil {
		return err
	}

	return nil
}

// DeleteDraft deletes a draft resource from Porch
func (s *PorchStorage) DeleteDraft(ctx context.Context, resourceType ResourceType, id string) error {
	// Find existing PackageRevision (search all lifecycle states)
	pr, err := s.findPackageRevision(ctx, id, PorchLifecycleDraft, PorchLifecycleProposed, PorchLifecyclePublished)
	if err != nil {
		return err
	}
	if pr == nil {
		return NewStorageError(ErrorCodeNotFound,
			fmt.Sprintf("draft for resource %s not found", id), ErrResourceNotFound)
	}

	// Extract spec to check lifecycle state
	spec, ok := pr["spec"].(map[string]interface{})
	if !ok {
		return NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevision: missing spec", nil)
	}

	// Verify PackageRevision is Draft or Proposed (not Published)
	lifecycle, _ := spec["lifecycle"].(string)
	if lifecycle == PorchLifecyclePublished {
		return NewStorageError(ErrorCodeInvalidState,
			"cannot delete Published PackageRevision, use Delete() instead", nil)
	}

	// Extract PackageRevision name
	metadata, ok := pr["metadata"].(map[string]interface{})
	if !ok {
		return NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevision: missing metadata", nil)
	}
	prName, ok := metadata["name"].(string)
	if !ok {
		return NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevision: missing name", nil)
	}

	// Delete the PackageRevision
	deletePath := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisions/%s",
		s.namespace, prName)
	resp, err := s.makeRequest(ctx, http.MethodDelete, deletePath, nil)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to delete PackageRevision", err)
	}

	// Parse response (DELETE typically returns 200 OK or 204 No Content)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return s.handleHTTPError(resp.StatusCode, bodyBytes)
	}
	_ = resp.Body.Close()

	return nil
}

// ValidateDraft transitions a draft from Draft to Proposed state
func (s *PorchStorage) ValidateDraft(ctx context.Context, resourceType ResourceType, id string) error {
	// Find existing draft PackageRevision
	pr, err := s.findPackageRevision(ctx, id, PorchLifecycleDraft, PorchLifecycleProposed)
	if err != nil {
		return err
	}
	if pr == nil {
		return NewStorageError(ErrorCodeNotFound,
			fmt.Sprintf("draft for resource %s not found", id), ErrResourceNotFound)
	}

	// Extract spec to check lifecycle state
	spec, ok := pr["spec"].(map[string]interface{})
	if !ok {
		return NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevision: missing spec", nil)
	}

	// Verify draft is in Draft state (not already Proposed)
	lifecycle, _ := spec["lifecycle"].(string)
	if lifecycle != PorchLifecycleDraft {
		return NewStorageError(ErrorCodeInvalidState,
			fmt.Sprintf("cannot validate draft in %s state, must be in Draft state", lifecycle), nil)
	}

	// Extract PackageRevision name and namespace
	metadata, ok := pr["metadata"].(map[string]interface{})
	if !ok {
		return NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevision: missing metadata", nil)
	}
	prName, ok := metadata["name"].(string)
	if !ok {
		return NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevision: missing name", nil)
	}

	// Step 1: Update the resource state in the YAML file
	// Get the current draft to update its state
	draft, err := s.GetDraft(ctx, resourceType, id)
	if err != nil {
		return err
	}

	// Update the state field to VALIDATED
	// GetDraft returns models types, so we check for both models and storage types
	switch d := draft.(type) {
	case *models.OCloudData:
		d.State = models.StateValidated
		if err := s.UpdateDraft(ctx, resourceType, id, d); err != nil {
			return err
		}
	case *models.TemplateInfoData:
		d.State = models.StateValidated
		if err := s.UpdateDraft(ctx, resourceType, id, d); err != nil {
			return err
		}
	case *models.FocomProvisioningRequestData:
		d.State = models.StateValidated
		if err := s.UpdateDraft(ctx, resourceType, id, d); err != nil {
			return err
		}
	case *OCloudData:
		d.State = StateValidated
		if err := s.UpdateDraft(ctx, resourceType, id, d); err != nil {
			return err
		}
	case *TemplateInfoData:
		d.State = StateValidated
		if err := s.UpdateDraft(ctx, resourceType, id, d); err != nil {
			return err
		}
	case *FocomProvisioningRequestData:
		d.State = StateValidated
		if err := s.UpdateDraft(ctx, resourceType, id, d); err != nil {
			return err
		}
	}

	// Step 2: Re-fetch the PackageRevision to get the latest version with all tasks
	pr, err = s.findPackageRevision(ctx, id, PorchLifecycleDraft, PorchLifecycleProposed)
	if err != nil {
		return err
	}
	if pr == nil {
		return NewStorageError(ErrorCodeNotFound,
			fmt.Sprintf("draft for resource %s not found after update", id), ErrResourceNotFound)
	}

	// Extract the updated spec and metadata
	spec, ok = pr["spec"].(map[string]interface{})
	if !ok {
		return NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevision: missing spec", nil)
	}
	metadata, ok = pr["metadata"].(map[string]interface{})
	if !ok {
		return NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevision: missing metadata", nil)
	}

	// Step 3: Update Porch lifecycle to Proposed
	spec["lifecycle"] = PorchLifecycleProposed

	// Create update request body
	updateRequest := map[string]interface{}{
		"apiVersion": "porch.kpt.dev/v1alpha1",
		"kind":       "PackageRevision",
		"metadata":   metadata,
		"spec":       spec,
	}

	// Make PUT request to update PackageRevision
	updatePath := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisions/%s",
		s.namespace, prName)
	resp, err := s.makeRequest(ctx, http.MethodPut, updatePath, updateRequest)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to update PackageRevision lifecycle", err)
	}

	// Expect 200 OK - Porch returns success immediately
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return s.handleHTTPError(resp.StatusCode, bodyBytes)
	}
	_ = resp.Body.Close()

	// Wait a moment for the lifecycle change to be processed
	time.Sleep(500 * time.Millisecond)

	// Verify the lifecycle was updated by checking the PackageRevision
	updatedPR, err := s.findPackageRevision(ctx, id, PorchLifecycleProposed)
	if err != nil {
		return err
	}
	if updatedPR == nil {
		return NewStorageError(ErrorCodeStorageFailure, "PackageRevision lifecycle not updated to Proposed", nil)
	}

	return nil
}

// ApproveDraft transitions a draft from Proposed to Published state
func (s *PorchStorage) ApproveDraft(ctx context.Context, resourceType ResourceType, id string) error {
	// Find existing draft PackageRevision
	pr, err := s.findPackageRevision(ctx, id, PorchLifecycleDraft, PorchLifecycleProposed)
	if err != nil {
		return err
	}
	if pr == nil {
		return NewStorageError(ErrorCodeNotFound,
			fmt.Sprintf("draft for resource %s not found", id), ErrResourceNotFound)
	}

	// Extract spec to check lifecycle state
	spec, ok := pr["spec"].(map[string]interface{})
	if !ok {
		return NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevision: missing spec", nil)
	}

	// Verify draft is in Proposed state (not Draft)
	lifecycle, _ := spec["lifecycle"].(string)
	if lifecycle != PorchLifecycleProposed {
		return NewStorageError(ErrorCodeInvalidState,
			fmt.Sprintf("cannot approve draft in %s state, must be in Proposed state", lifecycle), nil)
	}

	// Extract PackageRevision name
	metadata, ok := pr["metadata"].(map[string]interface{})
	if !ok {
		return NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevision: missing metadata", nil)
	}
	prName, ok := metadata["name"].(string)
	if !ok {
		return NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevision: missing name", nil)
	}

	// Step 1: Get the current PackageRevision from the approval endpoint
	// This ensures we have the latest version with all tasks
	approvalPath := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisions/%s/approval",
		s.namespace, prName)
	resp, err := s.makeRequest(ctx, http.MethodGet, approvalPath, nil)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to get PackageRevision from approval endpoint", err)
	}

	var approvalPR map[string]interface{}
	if err := s.parseResponse(resp, http.StatusOK, &approvalPR); err != nil {
		return err
	}

	// Extract the spec from the approval endpoint response
	approvalSpec, ok := approvalPR["spec"].(map[string]interface{})
	if !ok {
		return NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevision from approval endpoint: missing spec", nil)
	}

	// Step 2: Generate next revision ID
	nextRevision, err := s.generateNextRevisionID(ctx, id)
	if err != nil {
		return err
	}

	// Note: We do NOT update the resource content here because:
	// 1. State and revisionId are not part of the CRD spec (we removed them from YAML)
	// 2. Porch doesn't allow content updates when PackageRevision is in Proposed state
	// 3. The state is derived from the PackageRevision lifecycle, not stored in the resource

	// Step 3: Update the spec with Published lifecycle and revision
	// Parse revision number from "v1" format to integer for new Porch API
	var revNum int
	if _, err := fmt.Sscanf(nextRevision, "v%d", &revNum); err != nil {
		return NewStorageError(ErrorCodeStorageFailure, fmt.Sprintf("failed to parse revision number from %s", nextRevision), err)
	}

	approvalSpec["lifecycle"] = PorchLifecyclePublished
	approvalSpec["revision"] = revNum // Use integer instead of string for new Porch API
	// Note: Keep workspaceName - Porch will handle it

	// Create the approval request with the full PackageRevision object
	approvalRequest := map[string]interface{}{
		"apiVersion": "porch.kpt.dev/v1alpha1",
		"kind":       "PackageRevision",
		"metadata":   approvalPR["metadata"],
		"spec":       approvalSpec,
	}

	// Step 4: Make PUT request to the approval endpoint
	resp, err = s.makeRequest(ctx, http.MethodPut, approvalPath, approvalRequest)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, fmt.Sprintf("failed to approve PackageRevision: %v", err), err)
	}

	// Expect 200 OK - Porch returns success immediately, then processes asynchronously
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		errMsg := fmt.Sprintf("approval endpoint returned %d: %s", resp.StatusCode, string(bodyBytes))
		return NewStorageError(ErrorCodeStorageFailure, errMsg, nil)
	}
	_ = resp.Body.Close()

	// Step 5: Poll for the PackageRevision to be Published (approval can be async)
	// Use the same polling approach as waitForPackageRevision
	// Porch can take 30+ seconds to process approval via admission webhooks
	deadline := time.Now().Add(60 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	pollCount := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			pollCount++

			if time.Now().After(deadline) {
				return NewStorageError(ErrorCodeStorageFailure,
					fmt.Sprintf("timeout waiting for PackageRevision %s to be Published after 60s", id), nil)
			}

			// Try to find the Published PackageRevision
			updatedPR, err := s.findPackageRevision(ctx, id, PorchLifecyclePublished)
			if err != nil {
				// If it's not a "not found" error, return it
				if !isNotFoundError(err) {
					return err
				}
				// Otherwise continue polling
				continue
			}

			if updatedPR != nil {
				return nil
			}
		}
	}
}

// generateNextRevisionID calculates the next revision ID for a resource
func (s *PorchStorage) generateNextRevisionID(ctx context.Context, packageName string) (string, error) {
	// List all Published PackageRevisions for this resource
	path := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisions", s.namespace)
	resp, err := s.makeRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return "", NewStorageError(ErrorCodeStorageFailure, "failed to list PackageRevisions", err)
	}

	// Parse response
	var prList map[string]interface{}
	if err := s.parseResponse(resp, http.StatusOK, &prList); err != nil {
		return "", err
	}

	// Find highest revision number for this package
	highestRevision := 0
	items, ok := prList["items"].([]interface{})
	if ok {
		for _, item := range items {
			pr, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			spec, ok := pr["spec"].(map[string]interface{})
			if !ok {
				continue
			}

			pkgName, _ := spec["packageName"].(string)
			repo, _ := spec["repository"].(string)
			lifecycle, _ := spec["lifecycle"].(string)

			// Revision is now an integer in new Porch API
			var revNum int
			switch v := spec["revision"].(type) {
			case int:
				revNum = v
			case float64:
				revNum = int(v)
			case string:
				_, _ = fmt.Sscanf(v, "v%d", &revNum)
			}

			// Only consider Published revisions for this package
			if pkgName == packageName && repo == s.repository && lifecycle == PorchLifecyclePublished && revNum > 0 {
				if revNum > highestRevision {
					highestRevision = revNum
				}
			}
		}
	}

	// Generate next revision (v1, v2, v3, etc.)
	nextRevision := fmt.Sprintf("v%d", highestRevision+1)
	return nextRevision, nil
}

// RejectDraft transitions a draft from Proposed back to Draft state
func (s *PorchStorage) RejectDraft(ctx context.Context, resourceType ResourceType, id string) error {
	// Find existing draft PackageRevision
	pr, err := s.findPackageRevision(ctx, id, PorchLifecycleDraft, PorchLifecycleProposed)
	if err != nil {
		return err
	}
	if pr == nil {
		return NewStorageError(ErrorCodeNotFound,
			fmt.Sprintf("draft for resource %s not found", id), ErrResourceNotFound)
	}

	// Extract spec to check lifecycle state
	spec, ok := pr["spec"].(map[string]interface{})
	if !ok {
		return NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevision: missing spec", nil)
	}

	// Verify draft is in Proposed state (not Draft)
	lifecycle, _ := spec["lifecycle"].(string)
	if lifecycle != PorchLifecycleProposed {
		return NewStorageError(ErrorCodeInvalidState,
			fmt.Sprintf("cannot reject draft in %s state, must be in Proposed state", lifecycle), nil)
	}

	// Extract PackageRevision name
	metadata, ok := pr["metadata"].(map[string]interface{})
	if !ok {
		return NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevision: missing metadata", nil)
	}
	prName, ok := metadata["name"].(string)
	if !ok {
		return NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevision: missing name", nil)
	}

	// Update lifecycle back to Draft
	spec["lifecycle"] = PorchLifecycleDraft

	// Create update request body
	updateRequest := map[string]interface{}{
		"apiVersion": "porch.kpt.dev/v1alpha1",
		"kind":       "PackageRevision",
		"metadata":   metadata,
		"spec":       spec,
	}

	// Make PUT request to update PackageRevision
	updatePath := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisions/%s",
		s.namespace, prName)
	resp, err := s.makeRequest(ctx, http.MethodPut, updatePath, updateRequest)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to update PackageRevision lifecycle", err)
	}

	// Parse response
	if err := s.parseResponse(resp, http.StatusOK, nil); err != nil {
		return err
	}

	return nil
}

// Create creates an approved (Published) resource directly in Porch
func (s *PorchStorage) Create(ctx context.Context, resourceType ResourceType, resource interface{}) error {
	// Extract resource ID
	resourceID, err := s.extractResourceID(resource)
	if err != nil {
		return NewStorageError(ErrorCodeInvalidID, "failed to extract resource ID", err)
	}

	// Check if resource already exists (any lifecycle state)
	existing, err := s.findPackageRevision(ctx, resourceID, PorchLifecycleDraft, PorchLifecycleProposed, PorchLifecyclePublished)
	if err != nil {
		return err
	}
	if existing != nil {
		return NewStorageError(ErrorCodeAlreadyExists,
			fmt.Sprintf("resource %s already exists", resourceID), ErrResourceExists)
	}

	// Create PackageRevision request body with lifecycle=Published and revision=v1
	prRequest := map[string]interface{}{
		"apiVersion": "porch.kpt.dev/v1alpha1",
		"kind":       "PackageRevision",
		"spec": map[string]interface{}{
			"packageName": resourceID,
			"repository":  s.repository,
			"lifecycle":   PorchLifecyclePublished,
			"revision":    "v1",
		},
	}

	// Make POST request to create PackageRevision
	path := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisions", s.namespace)
	resp, err := s.makeRequest(ctx, http.MethodPost, path, prRequest)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to create PackageRevision", err)
	}

	// Accept both 201 (immediate success) and 500 (async processing)
	// Porch uses asynchronous admission webhooks that may return 500 initially
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusInternalServerError {
		bodyBytes, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return s.handleHTTPError(resp.StatusCode, bodyBytes)
	}
	_ = resp.Body.Close()

	// Wait for PackageRevision to exist (handles async creation)
	prName, err := s.waitForPackageRevision(ctx, resourceID, PorchLifecyclePublished, s.packageRevisionTimeout)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "PackageRevision not created", err)
	}

	// Create package contents (Kptfile + resource YAML)
	kptfile, err := s.createKptfile(resourceID, resourceType)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to create Kptfile", err)
	}

	resourceYAML, err := s.createResourceYAML(resource, resourceType)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to create resource YAML", err)
	}

	// Determine resource filename based on type
	var resourceFilename string
	switch resourceType {
	case ResourceTypeOCloud:
		resourceFilename = "ocloud.yaml"
	case ResourceTypeTemplateInfo:
		resourceFilename = "templateinfo.yaml"
	case ResourceTypeFocomProvisioningRequest:
		resourceFilename = "focomprovisioningrequest.yaml"
	default:
		return NewStorageError(ErrorCodeStorageFailure,
			fmt.Sprintf("unsupported resource type: %s", resourceType), nil)
	}

	// Update PackageRevisionResources with package contents
	// Retry with backoff since PackageRevisionResources might not be immediately available
	prrPath := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisionresources/%s",
		s.namespace, prName)

	maxRetries := 10
	retryDelay := 500 * time.Millisecond
	var prrResp *http.Response
	var existingPRR map[string]interface{}

	for i := 0; i < maxRetries; i++ {
		// First, GET the existing PackageRevisionResources to merge with our changes
		getResp, getErr := s.makeRequest(ctx, http.MethodGet, prrPath, nil)
		if getErr == nil && getResp.StatusCode == http.StatusOK {
			// Parse existing resources
			if parseErr := s.parseResponse(getResp, http.StatusOK, &existingPRR); parseErr == nil {
				// Extract existing resources
				spec, ok := existingPRR["spec"].(map[string]interface{})
				if ok {
					resources, ok := spec["resources"].(map[string]interface{})
					if !ok {
						resources = make(map[string]interface{})
					}

					// Merge our files with existing resources
					resources["Kptfile"] = kptfile
					resources[resourceFilename] = resourceYAML

					// Create PUT request with merged resources
					prrRequest := map[string]interface{}{
						"apiVersion": "porch.kpt.dev/v1alpha1",
						"kind":       "PackageRevisionResources",
						"metadata":   existingPRR["metadata"],
						"spec": map[string]interface{}{
							"packageName": resourceID,
							"repository":  s.repository,
							"resources":   resources,
						},
					}

					// PUT the updated resources
					prrResp, err = s.makeRequest(ctx, http.MethodPut, prrPath, prrRequest)
					if err == nil && prrResp.StatusCode == http.StatusOK {
						break
					}
				}
			}
		}

		// If GET failed or PUT failed, the resource might not be ready yet, retry
		time.Sleep(retryDelay)
	}

	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to update package contents after retries", err)
	}
	if prrResp == nil || prrResp.StatusCode != http.StatusOK {
		return NewStorageError(ErrorCodeStorageFailure, "failed to update package contents: resource not ready", nil)
	}

	// Parse response
	if err := s.parseResponse(prrResp, http.StatusOK, nil); err != nil {
		return err
	}

	return nil
}

// Get retrieves the latest Published resource from Porch
func (s *PorchStorage) Get(ctx context.Context, resourceType ResourceType, id string) (interface{}, error) {
	// Find latest Published PackageRevision
	pr, err := s.findLatestPublishedPackageRevision(ctx, id)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return nil, NewStorageError(ErrorCodeNotFound,
			fmt.Sprintf("resource %s not found", id), ErrResourceNotFound)
	}

	// Extract PackageRevision name
	metadata, ok := pr["metadata"].(map[string]interface{})
	if !ok {
		return nil, NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevision: missing metadata", nil)
	}
	prName, ok := metadata["name"].(string)
	if !ok {
		return nil, NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevision: missing name", nil)
	}

	// Retrieve PackageRevisionResources to get package contents
	prrPath := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisionresources/%s",
		s.namespace, prName)
	resp, err := s.makeRequest(ctx, http.MethodGet, prrPath, nil)
	if err != nil {
		return nil, NewStorageError(ErrorCodeStorageFailure, "failed to retrieve package contents", err)
	}

	// Parse response
	var prr map[string]interface{}
	if err := s.parseResponse(resp, http.StatusOK, &prr); err != nil {
		return nil, err
	}

	// Extract resources from spec
	spec, ok := prr["spec"].(map[string]interface{})
	if !ok {
		return nil, NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevisionResources: missing spec", nil)
	}
	resources, ok := spec["resources"].(map[string]interface{})
	if !ok {
		return nil, NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevisionResources: missing resources", nil)
	}

	// Determine resource filename based on type
	var resourceFilename string
	switch resourceType {
	case ResourceTypeOCloud:
		resourceFilename = "ocloud.yaml"
	case ResourceTypeTemplateInfo:
		resourceFilename = "templateinfo.yaml"
	case ResourceTypeFocomProvisioningRequest:
		resourceFilename = "focomprovisioningrequest.yaml"
	default:
		return nil, NewStorageError(ErrorCodeStorageFailure,
			fmt.Sprintf("unsupported resource type: %s", resourceType), nil)
	}

	// Get resource YAML content
	resourceYAML, ok := resources[resourceFilename].(string)
	if !ok {
		return nil, NewStorageError(ErrorCodeStorageFailure,
			fmt.Sprintf("resource file %s not found in package", resourceFilename), nil)
	}

	// Parse YAML to Go struct
	resource, err := s.parseResourceYAML(resourceYAML, resourceType)
	if err != nil {
		return nil, NewStorageError(ErrorCodeStorageFailure, "failed to parse resource YAML", err)
	}

	// Set the revisionId and state from the PackageRevision metadata
	// These are internal API fields not stored in the CRD YAML
	prSpec, _ := pr["spec"].(map[string]interface{})

	// Revision is now an integer in new Porch API, convert to string for internal use
	var revisionID string
	switch v := prSpec["revision"].(type) {
	case int:
		revisionID = fmt.Sprintf("v%d", v)
	case float64: // JSON numbers are float64
		revisionID = fmt.Sprintf("v%d", int(v))
	case string:
		revisionID = v // Support old API format
	}

	// Set revisionId and state based on resource type
	switch r := resource.(type) {
	case *OCloudData:
		r.RevisionID = revisionID
		r.State = StateApproved // Published PackageRevisions are approved
	case *TemplateInfoData:
		r.RevisionID = revisionID
		r.State = StateApproved
	case *FocomProvisioningRequestData:
		r.RevisionID = revisionID
		r.State = StateApproved
	}

	// Convert storage types to models types for the handler
	modelResource, err := convertToModelsType(resource)
	if err != nil {
		return nil, NewStorageError(ErrorCodeStorageFailure, "failed to convert resource type", err)
	}

	return modelResource, nil
}

// findLatestPublishedPackageRevision finds the latest Published PackageRevision for a resource
func (s *PorchStorage) findLatestPublishedPackageRevision(ctx context.Context, packageName string) (map[string]interface{}, error) {
	// List all PackageRevisions
	path := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisions", s.namespace)
	resp, err := s.makeRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, NewStorageError(ErrorCodeStorageFailure, "failed to list PackageRevisions", err)
	}

	// Parse response
	var prList map[string]interface{}
	if err := s.parseResponse(resp, http.StatusOK, &prList); err != nil {
		return nil, err
	}

	// Find latest Published PackageRevision for this package
	var latestPR map[string]interface{}
	highestRevision := 0

	items, ok := prList["items"].([]interface{})
	if ok {
		for _, item := range items {
			pr, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			spec, ok := pr["spec"].(map[string]interface{})
			if !ok {
				continue
			}

			pkgName, _ := spec["packageName"].(string)
			repo, _ := spec["repository"].(string)
			lifecycle, _ := spec["lifecycle"].(string)

			// Revision is now an integer in new Porch API, but might be string in old API
			var revNum int
			switch v := spec["revision"].(type) {
			case int:
				revNum = v
			case float64: // JSON numbers are float64
				revNum = int(v)
			case string:
				// Support old API format "v1", "v2", etc.
				_, _ = fmt.Sscanf(v, "v%d", &revNum)
			}

			// Only consider Published revisions for this package
			if pkgName == packageName && repo == s.repository && lifecycle == PorchLifecyclePublished && revNum > 0 {
				if revNum > highestRevision {
					highestRevision = revNum
					latestPR = pr
				}
			}
		}
	}

	return latestPR, nil
}

// Update updates an approved resource by creating a draft, applying changes, validating, and approving
func (s *PorchStorage) Update(ctx context.Context, resourceType ResourceType, id string, resource interface{}) error {
	// Step 1: Get current Published revision
	currentResource, err := s.Get(ctx, resourceType, id)
	if err != nil {
		return err
	}
	if currentResource == nil {
		return NewStorageError(ErrorCodeNotFound,
			fmt.Sprintf("resource %s not found", id), ErrResourceNotFound)
	}

	// Step 2: Check if draft already exists, delete it if it does
	existingDraft, err := s.findPackageRevision(ctx, id, PorchLifecycleDraft, PorchLifecycleProposed)
	if err != nil {
		return err
	}
	if existingDraft != nil {
		// Delete existing draft
		if err := s.DeleteDraft(ctx, resourceType, id); err != nil {
			return err
		}
	}

	// Step 3: Create new Draft
	if err := s.CreateDraft(ctx, resourceType, resource); err != nil {
		return err
	}

	// Step 4: Validate Draft (Draft → Proposed)
	if err := s.ValidateDraft(ctx, resourceType, id); err != nil {
		// Cleanup: delete draft if validation fails
		_ = s.DeleteDraft(ctx, resourceType, id)
		return err
	}

	// Step 5: Approve Draft (Proposed → Published with next revision)
	if err := s.ApproveDraft(ctx, resourceType, id); err != nil {
		// Cleanup: delete draft if approval fails
		_ = s.DeleteDraft(ctx, resourceType, id)
		return err
	}

	return nil
}

// Delete removes all PackageRevisions (Draft and Published) for a resource
func (s *PorchStorage) Delete(ctx context.Context, resourceType ResourceType, id string) error {
	// List all PackageRevisions for this resource (all lifecycle states)
	path := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisions", s.namespace)
	resp, err := s.makeRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to list PackageRevisions", err)
	}

	// Parse response
	var prList map[string]interface{}
	if err := s.parseResponse(resp, http.StatusOK, &prList); err != nil {
		return err
	}

	// Find all PackageRevisions for this resource and categorize by lifecycle
	var toDelete []map[string]interface{}
	items, ok := prList["items"].([]interface{})
	if ok {
		for _, item := range items {
			pr, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			spec, ok := pr["spec"].(map[string]interface{})
			if !ok {
				continue
			}

			pkgName, _ := spec["packageName"].(string)
			repo, _ := spec["repository"].(string)

			if pkgName == id && repo == s.repository {
				toDelete = append(toDelete, pr)
			}
		}
	}

	// If no PackageRevisions found, that's okay (idempotent delete)
	if len(toDelete) == 0 {
		return nil
	}

	// Delete each PackageRevision
	// For Published PackageRevisions, we need to first propose deletion, then delete
	for _, pr := range toDelete {
		metadata := pr["metadata"].(map[string]interface{})
		prName := metadata["name"].(string)
		spec := pr["spec"].(map[string]interface{})
		lifecycle, _ := spec["lifecycle"].(string)

		// If Published, first propose deletion
		if lifecycle == PorchLifecyclePublished {
			// Update lifecycle to DeletionProposed
			spec["lifecycle"] = "DeletionProposed"

			updatePath := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisions/%s",
				s.namespace, prName)
			updateResp, err := s.makeRequest(ctx, http.MethodPut, updatePath, pr)
			if err != nil {
				return NewStorageError(ErrorCodeStorageFailure,
					fmt.Sprintf("failed to propose deletion for PackageRevision %s", prName), err)
			}
			if updateResp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(updateResp.Body)
				_ = updateResp.Body.Close()
				return s.handleHTTPError(updateResp.StatusCode, bodyBytes)
			}
			_ = updateResp.Body.Close()

			// Small delay to allow the update to be processed
			time.Sleep(100 * time.Millisecond)
		}

		// Now delete the PackageRevision
		deletePath := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisions/%s",
			s.namespace, prName)
		deleteResp, err := s.makeRequest(ctx, http.MethodDelete, deletePath, nil)
		if err != nil {
			return NewStorageError(ErrorCodeStorageFailure,
				fmt.Sprintf("failed to delete PackageRevision %s", prName), err)
		}

		// Handle both 200 OK and 204 No Content
		if deleteResp.StatusCode != http.StatusOK && deleteResp.StatusCode != http.StatusNoContent {
			bodyBytes, _ := io.ReadAll(deleteResp.Body)
			_ = deleteResp.Body.Close()
			return s.handleHTTPError(deleteResp.StatusCode, bodyBytes)
		}
		_ = deleteResp.Body.Close()
	}

	return nil
}

// List returns all Published resources of a given type
func (s *PorchStorage) List(ctx context.Context, resourceType ResourceType) ([]interface{}, error) {
	// List all PackageRevisions
	path := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisions", s.namespace)
	resp, err := s.makeRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, NewStorageError(ErrorCodeStorageFailure, "failed to list PackageRevisions", err)
	}

	// Parse response
	var prList map[string]interface{}
	if err := s.parseResponse(resp, http.StatusOK, &prList); err != nil {
		return nil, err
	}

	// Find latest Published PackageRevision for each unique resource
	latestRevisions := make(map[string]map[string]interface{})

	items, ok := prList["items"].([]interface{})
	if ok {
		for _, item := range items {
			pr, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			spec, ok := pr["spec"].(map[string]interface{})
			if !ok {
				continue
			}

			repo, _ := spec["repository"].(string)
			lifecycle, _ := spec["lifecycle"].(string)
			packageName, _ := spec["packageName"].(string)

			// Revision is now an integer in new Porch API
			var revNum int
			switch v := spec["revision"].(type) {
			case int:
				revNum = v
			case float64:
				revNum = int(v)
			case string:
				_, _ = fmt.Sscanf(v, "v%d", &revNum)
			}

			// Only consider Published revisions from our repository
			if repo != s.repository || lifecycle != PorchLifecyclePublished || revNum == 0 {
				continue
			}

			// Keep only the latest revision for each resource
			if existing, exists := latestRevisions[packageName]; exists {
				existingSpec, _ := existing["spec"].(map[string]interface{})
				var existingRevNum int
				switch v := existingSpec["revision"].(type) {
				case int:
					existingRevNum = v
				case float64:
					existingRevNum = int(v)
				case string:
					_, _ = fmt.Sscanf(v, "v%d", &existingRevNum)
				}
				if revNum <= existingRevNum {
					continue
				}
			}

			latestRevisions[packageName] = pr
		}
	}

	// Retrieve and parse each resource
	var resources []interface{}
	for _, pr := range latestRevisions {
		// Extract PackageRevision name
		metadata, ok := pr["metadata"].(map[string]interface{})
		if !ok {
			continue
		}
		prName, ok := metadata["name"].(string)
		if !ok {
			continue
		}

		// Retrieve PackageRevisionResources
		prrPath := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisionresources/%s",
			s.namespace, prName)
		prrResp, err := s.makeRequest(ctx, http.MethodGet, prrPath, nil)
		if err != nil {
			// Skip resources that can't be retrieved
			continue
		}

		var prr map[string]interface{}
		if err := s.parseResponse(prrResp, http.StatusOK, &prr); err != nil {
			continue
		}

		// Extract resources from spec
		prrSpec, ok := prr["spec"].(map[string]interface{})
		if !ok {
			continue
		}
		resourcesMap, ok := prrSpec["resources"].(map[string]interface{})
		if !ok {
			continue
		}

		// Determine resource filename based on type
		var resourceFilename string
		switch resourceType {
		case ResourceTypeOCloud:
			resourceFilename = "ocloud.yaml"
		case ResourceTypeTemplateInfo:
			resourceFilename = "templateinfo.yaml"
		case ResourceTypeFocomProvisioningRequest:
			resourceFilename = "focomprovisioningrequest.yaml"
		default:
			return nil, NewStorageError(ErrorCodeStorageFailure,
				fmt.Sprintf("unsupported resource type: %s", resourceType), nil)
		}

		// Get resource YAML content
		resourceYAML, ok := resourcesMap[resourceFilename].(string)
		if !ok {
			continue
		}

		// Parse YAML to Go struct
		resource, err := s.parseResourceYAML(resourceYAML, resourceType)
		if err != nil {
			continue
		}

		// Set state to APPROVED since List only returns Published revisions
		s.setResourceState(resource, PorchLifecyclePublished)

		// Convert to models type
		modelResource, err := convertToModelsType(resource)
		if err != nil {
			continue
		}

		resources = append(resources, modelResource)
	}

	return resources, nil
}

// GetRevisions returns all Published revisions of a resource ordered by revision number
func (s *PorchStorage) GetRevisions(ctx context.Context, resourceType ResourceType, id string) ([]interface{}, error) {
	// List all PackageRevisions
	path := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisions", s.namespace)
	resp, err := s.makeRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, NewStorageError(ErrorCodeStorageFailure, "failed to list PackageRevisions", err)
	}

	// Parse response
	var prList map[string]interface{}
	if err := s.parseResponse(resp, http.StatusOK, &prList); err != nil {
		return nil, err
	}

	// Find all Published PackageRevisions for this resource
	type revisionInfo struct {
		pr        map[string]interface{}
		revNum    int
		revString string
	}
	var revisions []revisionInfo

	items, ok := prList["items"].([]interface{})
	if ok {
		for _, item := range items {
			pr, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			spec, ok := pr["spec"].(map[string]interface{})
			if !ok {
				continue
			}

			pkgName, _ := spec["packageName"].(string)
			repo, _ := spec["repository"].(string)
			lifecycle, _ := spec["lifecycle"].(string)

			// Revision is now an integer in new Porch API
			var revNum int
			var revString string
			switch v := spec["revision"].(type) {
			case int:
				revNum = v
				revString = fmt.Sprintf("v%d", v)
			case float64:
				revNum = int(v)
				revString = fmt.Sprintf("v%d", int(v))
			case string:
				_, _ = fmt.Sscanf(v, "v%d", &revNum)
				revString = v
			}

			// Only consider Published revisions for this package
			if pkgName == id && repo == s.repository && lifecycle == PorchLifecyclePublished && revNum > 0 {
				revisions = append(revisions, revisionInfo{
					pr:        pr,
					revNum:    revNum,
					revString: revString,
				})
			}
		}
	}

	// Sort by revision number
	sort.Slice(revisions, func(i, j int) bool {
		return revisions[i].revNum < revisions[j].revNum
	})

	// Retrieve and parse each revision
	var results []interface{}
	for _, revInfo := range revisions {
		// Extract PackageRevision name
		metadata, ok := revInfo.pr["metadata"].(map[string]interface{})
		if !ok {
			continue
		}
		prName, ok := metadata["name"].(string)
		if !ok {
			continue
		}

		// Retrieve PackageRevisionResources
		prrPath := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisionresources/%s",
			s.namespace, prName)
		prrResp, err := s.makeRequest(ctx, http.MethodGet, prrPath, nil)
		if err != nil {
			continue
		}

		var prr map[string]interface{}
		if err := s.parseResponse(prrResp, http.StatusOK, &prr); err != nil {
			continue
		}

		// Extract resources from spec
		prrSpec, ok := prr["spec"].(map[string]interface{})
		if !ok {
			continue
		}
		resourcesMap, ok := prrSpec["resources"].(map[string]interface{})
		if !ok {
			continue
		}

		// Determine resource filename based on type
		var resourceFilename string
		switch resourceType {
		case ResourceTypeOCloud:
			resourceFilename = "ocloud.yaml"
		case ResourceTypeTemplateInfo:
			resourceFilename = "templateinfo.yaml"
		case ResourceTypeFocomProvisioningRequest:
			resourceFilename = "focomprovisioningrequest.yaml"
		default:
			return nil, NewStorageError(ErrorCodeStorageFailure,
				fmt.Sprintf("unsupported resource type: %s", resourceType), nil)
		}

		// Get resource YAML content
		resourceYAML, ok := resourcesMap[resourceFilename].(string)
		if !ok {
			continue
		}

		// Parse YAML to Go struct
		resource, err := s.parseResourceYAML(resourceYAML, resourceType)
		if err != nil {
			continue
		}

		// Set revision ID and state
		if resource != nil {
			s.setResourceRevisionID(resource, revInfo.revString)
			s.setResourceState(resource, PorchLifecyclePublished)
		}

		// Convert storage types to models types for the handler
		modelResource, err := convertToModelsType(resource)
		if err != nil {
			continue
		}

		results = append(results, modelResource)
	}

	return results, nil
}

// GetRevision retrieves a specific Published revision of a resource
func (s *PorchStorage) GetRevision(ctx context.Context, resourceType ResourceType, id string, revisionID string) (interface{}, error) {
	// List all PackageRevisions
	path := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisions", s.namespace)
	resp, err := s.makeRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, NewStorageError(ErrorCodeStorageFailure, "failed to list PackageRevisions", err)
	}

	// Parse response
	var prList map[string]interface{}
	if err := s.parseResponse(resp, http.StatusOK, &prList); err != nil {
		return nil, err
	}

	// Find the specific Published PackageRevision
	var targetPR map[string]interface{}
	items, ok := prList["items"].([]interface{})
	if ok {
		for _, item := range items {
			pr, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			spec, ok := pr["spec"].(map[string]interface{})
			if !ok {
				continue
			}

			pkgName, _ := spec["packageName"].(string)
			repo, _ := spec["repository"].(string)
			lifecycle, _ := spec["lifecycle"].(string)

			// Revision is now an integer in new Porch API, but revisionID parameter is still string "v1"
			var revString string
			switch v := spec["revision"].(type) {
			case int:
				revString = fmt.Sprintf("v%d", v)
			case float64:
				revString = fmt.Sprintf("v%d", int(v))
			case string:
				revString = v
			}

			// Find matching Published revision
			if pkgName == id && repo == s.repository && lifecycle == PorchLifecyclePublished && revString == revisionID {
				targetPR = pr
				break
			}
		}
	}

	if targetPR == nil {
		return nil, NewStorageError(ErrorCodeInvalidRevision,
			fmt.Sprintf("revision %s not found for resource %s", revisionID, id), nil)
	}

	// Extract PackageRevision name
	metadata, ok := targetPR["metadata"].(map[string]interface{})
	if !ok {
		return nil, NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevision: missing metadata", nil)
	}
	prName, ok := metadata["name"].(string)
	if !ok {
		return nil, NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevision: missing name", nil)
	}

	// Retrieve PackageRevisionResources
	prrPath := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisionresources/%s",
		s.namespace, prName)
	prrResp, err := s.makeRequest(ctx, http.MethodGet, prrPath, nil)
	if err != nil {
		return nil, NewStorageError(ErrorCodeStorageFailure, "failed to retrieve package contents", err)
	}

	var prr map[string]interface{}
	if err := s.parseResponse(prrResp, http.StatusOK, &prr); err != nil {
		return nil, err
	}

	// Extract resources from spec
	prrSpec, ok := prr["spec"].(map[string]interface{})
	if !ok {
		return nil, NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevisionResources: missing spec", nil)
	}
	resourcesMap, ok := prrSpec["resources"].(map[string]interface{})
	if !ok {
		return nil, NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevisionResources: missing resources", nil)
	}

	// Determine resource filename based on type
	var resourceFilename string
	switch resourceType {
	case ResourceTypeOCloud:
		resourceFilename = "ocloud.yaml"
	case ResourceTypeTemplateInfo:
		resourceFilename = "templateinfo.yaml"
	case ResourceTypeFocomProvisioningRequest:
		resourceFilename = "focomprovisioningrequest.yaml"
	default:
		return nil, NewStorageError(ErrorCodeStorageFailure,
			fmt.Sprintf("unsupported resource type: %s", resourceType), nil)
	}

	// Get resource YAML content
	resourceYAML, ok := resourcesMap[resourceFilename].(string)
	if !ok {
		return nil, NewStorageError(ErrorCodeStorageFailure,
			fmt.Sprintf("resource file %s not found in package", resourceFilename), nil)
	}

	// Parse YAML to Go struct
	resource, err := s.parseResourceYAML(resourceYAML, resourceType)
	if err != nil {
		return nil, NewStorageError(ErrorCodeStorageFailure, "failed to parse resource YAML", err)
	}

	// Set revision ID and state
	if resource != nil {
		s.setResourceRevisionID(resource, revisionID)
		s.setResourceState(resource, PorchLifecyclePublished)
	}

	// Convert storage types to models types for the handler
	modelResource, err := convertToModelsType(resource)
	if err != nil {
		return nil, NewStorageError(ErrorCodeStorageFailure, "failed to convert resource type", err)
	}

	return modelResource, nil
}

// CreateDraftFromRevision creates a new Draft PackageRevision from a specific Published revision.
// This uses Porch's edit task with sourceRef to derive a new draft from the source revision.
// Porch handles the resource copying automatically during the async creation process.
func (s *PorchStorage) CreateDraftFromRevision(ctx context.Context, resourceType ResourceType, id string, revisionID string) error {
	// Check if draft already exists
	existingDraft, err := s.findPackageRevision(ctx, id, PorchLifecycleDraft, PorchLifecycleProposed)
	if err != nil {
		return err
	}
	if existingDraft != nil {
		return NewStorageError(ErrorCodeAlreadyExists,
			fmt.Sprintf("draft for resource %s already exists", id), ErrResourceExists)
	}

	// Get the source revision's PackageRevision resource name for the edit task.
	// This verifies the revision exists and provides the exact name needed for sourceRef.
	sourceRevisionName, err := s.getPackageRevisionName(ctx, id, revisionID)
	if err != nil {
		return NewStorageError(ErrorCodeInvalidRevision,
			fmt.Sprintf("revision %s not found for resource %s", revisionID, id), err)
	}

	// Calculate the next revision ID (e.g., "v2" if v1 exists)
	nextRevision, err := s.generateNextRevisionID(ctx, id)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to generate next revision ID", err)
	}

	// Parse revision number from "v2" format to integer for Porch API
	var revNum int
	if _, err := fmt.Sscanf(nextRevision, "v%d", &revNum); err != nil {
		return NewStorageError(ErrorCodeStorageFailure, fmt.Sprintf("failed to parse revision number from %s", nextRevision), err)
	}

	// Generate workspace name for the new draft
	workspaceName := fmt.Sprintf("%s-draft", nextRevision)

	// Construct the full resource name: {repo}.{packageName}.{workspaceName}
	resourceName := fmt.Sprintf("%s.%s.%s", s.repository, id, workspaceName)

	// Create PackageRevision request with edit task.
	// The edit task creates a new draft revision by referencing an existing PackageRevision via sourceRef.
	// Porch requires metadata.name, spec.revision (as integer), and the edit task with sourceRef.
	prRequest := map[string]interface{}{
		"apiVersion": "porch.kpt.dev/v1alpha1",
		"kind":       "PackageRevision",
		"metadata": map[string]interface{}{
			"namespace": s.namespace,
			"name":      resourceName,
		},
		"spec": map[string]interface{}{
			"packageName":   id,
			"repository":    s.repository,
			"revision":      revNum,
			"lifecycle":     PorchLifecycleDraft,
			"workspaceName": workspaceName,
			"tasks": []map[string]interface{}{
				{
					"type": "edit",
					"edit": map[string]interface{}{
						"sourceRef": map[string]interface{}{
							"name": sourceRevisionName,
						},
					},
				},
			},
		},
	}

	// Make POST request to create PackageRevision
	path := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisions", s.namespace)
	resp, err := s.makeRequest(ctx, http.MethodPost, path, prRequest)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to create draft PackageRevision", err)
	}

	// Check for actual errors (not async processing)
	if resp.StatusCode == http.StatusInternalServerError {
		bodyBytes, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		// Check if this is an "already exists" error
		bodyStr := string(bodyBytes)
		if strings.Contains(bodyStr, "already exists") {
			return NewStorageError(ErrorCodeAlreadyExists,
				fmt.Sprintf("draft for resource %s already exists", id), ErrResourceExists)
		}

		// For other 500 errors, treat as async processing and continue
		// (Porch sometimes returns 500 during async operations)
	} else if resp.StatusCode != http.StatusCreated {
		// For non-201, non-500 responses, this is an error
		bodyBytes, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return s.handleHTTPError(resp.StatusCode, bodyBytes)
	} else {
		// 201 Created - close the body
		_ = resp.Body.Close()
	}

	// Wait for PackageRevision to exist (handles async creation)
	_, err = s.waitForPackageRevision(ctx, id, PorchLifecycleDraft, s.packageRevisionTimeout)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "draft PackageRevision not created", err)
	}

	// Porch automatically copies all resources from the source revision via the edit task.
	// No manual resource copying is needed.
	return nil
}

// getPackageRevisionName finds the PackageRevision name for a specific resource ID and revision ID
func (s *PorchStorage) getPackageRevisionName(ctx context.Context, resourceID string, revisionID string) (string, error) {
	// List all PackageRevisions for this package
	path := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisions", s.namespace)
	resp, err := s.makeRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := s.parseResponse(resp, http.StatusOK, &result); err != nil {
		return "", err
	}

	items, ok := result["items"].([]interface{})
	if !ok {
		return "", fmt.Errorf("invalid response format")
	}

	// Parse the expected revision number from revisionID (e.g., "v1" -> 1)
	var expectedRevNum int
	if _, err := fmt.Sscanf(revisionID, "v%d", &expectedRevNum); err != nil {
		return "", fmt.Errorf("invalid revision ID format: %s (expected format: v1, v2, etc.)", revisionID)
	}

	// Find the PackageRevision with matching packageName and revision
	for _, item := range items {
		pr, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		spec, ok := pr["spec"].(map[string]interface{})
		if !ok {
			continue
		}

		packageName, _ := spec["packageName"].(string)

		// Handle revision as either int or float64 (JSON numbers)
		var revNum int
		switch v := spec["revision"].(type) {
		case int:
			revNum = v
		case float64:
			revNum = int(v)
		case string:
			// Handle legacy string format if it exists
			_, _ = fmt.Sscanf(v, "v%d", &revNum)
		default:
			continue
		}

		if packageName == resourceID && revNum == expectedRevNum {
			metadata, ok := pr["metadata"].(map[string]interface{})
			if !ok {
				continue
			}
			name, ok := metadata["name"].(string)
			if ok {
				return name, nil
			}
		}
	}

	return "", fmt.Errorf("PackageRevision not found for resource %s revision %s", resourceID, revisionID)
}

// ValidateDependencies validates resource dependencies
func (s *PorchStorage) ValidateDependencies(ctx context.Context, resourceType ResourceType, resource interface{}) error {
	switch resourceType {
	case ResourceTypeFocomProvisioningRequest:
		// For FPR creation/update, verify referenced OCloud and TemplateInfo exist
		return s.validateFPRDependencies(ctx, resource)

	case ResourceTypeOCloud:
		// For OCloud deletion, verify no FPRs reference it
		return s.validateOCloudReferences(ctx, resource)

	case ResourceTypeTemplateInfo:
		// For TemplateInfo deletion, verify no FPRs reference it
		return s.validateTemplateInfoReferences(ctx, resource)

	default:
		return nil
	}
}

// validateFPRDependencies validates that an FPR's dependencies exist
func (s *PorchStorage) validateFPRDependencies(ctx context.Context, resource interface{}) error {
	// Extract FPR data from resource
	fpr, ok := resource.(*FocomProvisioningRequestData)
	if !ok {
		// Try value type
		if fprValue, ok := resource.(FocomProvisioningRequestData); ok {
			fpr = &fprValue
		} else {
			return NewStorageError(ErrorCodeDependencyFailed,
				"invalid resource type for FocomProvisioningRequest dependency validation",
				fmt.Errorf("expected FocomProvisioningRequestData, got %T", resource))
		}
	}

	// Verify OCloud exists
	if fpr.OCloudID != "" {
		_, err := s.Get(ctx, ResourceTypeOCloud, fpr.OCloudID)
		if err != nil {
			return NewStorageError(ErrorCodeDependencyFailed,
				fmt.Sprintf("referenced OCloud %s not found", fpr.OCloudID), err)
		}
	}

	// Verify TemplateInfo exists (using templateName as ID)
	if fpr.TemplateName != "" {
		_, err := s.Get(ctx, ResourceTypeTemplateInfo, fpr.TemplateName)
		if err != nil {
			return NewStorageError(ErrorCodeDependencyFailed,
				fmt.Sprintf("referenced TemplateInfo %s not found", fpr.TemplateName), err)
		}
	}

	return nil
}

// validateOCloudReferences validates that no FPRs reference this OCloud (for deletion prevention)
func (s *PorchStorage) validateOCloudReferences(ctx context.Context, resource interface{}) error {
	// Extract OCloud ID from resource
	var ocloudID string
	switch r := resource.(type) {
	case *OCloudData:
		ocloudID = r.ID
	case OCloudData:
		ocloudID = r.ID
	case *models.OCloudData:
		ocloudID = r.ID
	case models.OCloudData:
		ocloudID = r.ID
	default:
		return NewStorageError(ErrorCodeDependencyFailed,
			"invalid resource type for OCloud reference validation",
			fmt.Errorf("expected OCloudData, got %T", resource))
	}

	// List all FocomProvisioningRequests
	fprs, err := s.List(ctx, ResourceTypeFocomProvisioningRequest)
	if err != nil {
		return err
	}

	// Check if any FPR references this OCloud
	var referencingFPRs []string
	for _, item := range fprs {
		var fprOCloudID, fprID string
		switch fpr := item.(type) {
		case *FocomProvisioningRequestData:
			fprOCloudID = fpr.OCloudID
			fprID = fpr.ID
		case *models.FocomProvisioningRequestData:
			fprOCloudID = fpr.OCloudID
			fprID = fpr.ID
		default:
			continue
		}

		if fprOCloudID == ocloudID {
			referencingFPRs = append(referencingFPRs, fprID)
		}
	}

	if len(referencingFPRs) > 0 {
		return NewStorageError(ErrorCodeDependencyFailed,
			fmt.Sprintf("cannot delete OCloud %s: referenced by FocomProvisioningRequest(s): %v", ocloudID, referencingFPRs), nil)
	}

	return nil
}

// validateTemplateInfoReferences validates that no FPRs reference this TemplateInfo (for deletion prevention)
func (s *PorchStorage) validateTemplateInfoReferences(ctx context.Context, resource interface{}) error {
	// Extract TemplateInfo name from resource
	var templateName string
	switch r := resource.(type) {
	case *TemplateInfoData:
		templateName = r.TemplateName
	case TemplateInfoData:
		templateName = r.TemplateName
	case *models.TemplateInfoData:
		templateName = r.TemplateName
	case models.TemplateInfoData:
		templateName = r.TemplateName
	default:
		return NewStorageError(ErrorCodeDependencyFailed,
			"invalid resource type for TemplateInfo reference validation",
			fmt.Errorf("expected TemplateInfoData, got %T", resource))
	}

	// List all FocomProvisioningRequests
	fprs, err := s.List(ctx, ResourceTypeFocomProvisioningRequest)
	if err != nil {
		return err
	}

	// Check if any FPR references this TemplateInfo
	var referencingFPRs []string
	for _, item := range fprs {
		var fprTemplateName, fprID string
		switch fpr := item.(type) {
		case *FocomProvisioningRequestData:
			fprTemplateName = fpr.TemplateName
			fprID = fpr.ID
		case *models.FocomProvisioningRequestData:
			fprTemplateName = fpr.TemplateName
			fprID = fpr.ID
		default:
			continue
		}

		if fprTemplateName == templateName {
			referencingFPRs = append(referencingFPRs, fprID)
		}
	}

	if len(referencingFPRs) > 0 {
		return NewStorageError(ErrorCodeDependencyFailed,
			fmt.Sprintf("cannot delete TemplateInfo %s: referenced by FocomProvisioningRequest(s): %v", templateName, referencingFPRs), nil)
	}

	return nil
}

// findPackageRevision finds a PackageRevision by packageName and lifecycle states
func (s *PorchStorage) findPackageRevision(ctx context.Context, packageName string, lifecycles ...string) (map[string]interface{}, error) {
	// List all PackageRevisions
	path := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisions", s.namespace)
	resp, err := s.makeRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, NewStorageError(ErrorCodeStorageFailure, "failed to list PackageRevisions", err)
	}

	// Parse response
	var prList map[string]interface{}
	if err := s.parseResponse(resp, http.StatusOK, &prList); err != nil {
		return nil, err
	}

	// Extract items
	items, ok := prList["items"].([]interface{})
	if !ok {
		return nil, nil // No items found
	}

	// Find matching PackageRevision
	for _, item := range items {
		pr, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		spec, ok := pr["spec"].(map[string]interface{})
		if !ok {
			continue
		}

		pkgName, _ := spec["packageName"].(string)
		repo, _ := spec["repository"].(string)
		lifecycle, _ := spec["lifecycle"].(string)

		if pkgName == packageName && repo == s.repository {
			// Check if lifecycle matches any of the requested states
			for _, lc := range lifecycles {
				if lifecycle == lc {
					// For Published lifecycle, filter out invalid revisions (like -1 from .main)
					if lifecycle == PorchLifecyclePublished {
						var revNum int
						switch v := spec["revision"].(type) {
						case int:
							revNum = v
						case float64:
							revNum = int(v)
						case string:
							_, _ = fmt.Sscanf(v, "v%d", &revNum)
						}
						// Skip PackageRevisions with invalid revision numbers
						if revNum <= 0 {
							continue
						}
					}
					return pr, nil
				}
			}
		}
	}

	return nil, nil // Not found
}

// waitForPackageRevision polls for a PackageRevision to exist
// This handles Porch's asynchronous resource creation via admission webhooks
func (s *PorchStorage) waitForPackageRevision(ctx context.Context, packageName string, lifecycle string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(500 * time.Millisecond) // Poll every 500ms
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return "", fmt.Errorf("timeout waiting for PackageRevision %s (lifecycle=%s) after %v", packageName, lifecycle, timeout)
			}

			// Try to find the PackageRevision
			pr, err := s.findPackageRevision(ctx, packageName, lifecycle)
			if err != nil {
				// If it's not a "not found" error, return it
				if !isNotFoundError(err) {
					return "", err
				}
				// Otherwise continue polling
				continue
			}

			if pr != nil {
				// Extract the name
				metadata, ok := pr["metadata"].(map[string]interface{})
				if !ok {
					return "", fmt.Errorf("invalid PackageRevision: missing metadata")
				}
				name, ok := metadata["name"].(string)
				if !ok {
					return "", fmt.Errorf("invalid PackageRevision: missing name")
				}
				return name, nil
			}
		}
	}
}

// isNotFoundError checks if an error is a "not found" error
func isNotFoundError(err error) bool {
	if storageErr, ok := err.(*StorageError); ok {
		return storageErr.Code == ErrorCodeNotFound
	}
	return false
}

// CreateRevision creates a specific revision directly
// This is primarily used for test fixture setup to create approved resources with specific revision IDs
func (s *PorchStorage) CreateRevision(ctx context.Context, resourceType ResourceType, resourceID string, revisionID string, data interface{}) error {
	// Convert models types to storage types if needed
	data = s.convertToStorageType(data, resourceType)

	// Generate package name from resource ID
	packageName := resourceID

	// Create Kptfile content
	kptfileContent, err := s.createKptfile(resourceID, resourceType)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to create Kptfile", err)
	}

	// Create resource YAML content
	resourceYAML, err := s.createResourceYAML(data, resourceType)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to create resource YAML", err)
	}

	// Determine resource filename based on type
	var resourceFilename string
	switch resourceType {
	case ResourceTypeOCloud:
		resourceFilename = "ocloud.yaml"
	case ResourceTypeTemplateInfo:
		resourceFilename = "templateinfo.yaml"
	case ResourceTypeFocomProvisioningRequest:
		resourceFilename = "focomprovisioningrequest.yaml"
	default:
		return NewStorageError(ErrorCodeStorageFailure, fmt.Sprintf("unsupported resource type: %s", resourceType), nil)
	}

	// Porch requires Draft→Proposed→Published workflow, cannot create Published directly
	// Step 1: Create as Draft with workspace
	// Ensure workspace name stays within 63 character limit
	// Format: "fixture-{resourceID}-{timestamp}" where timestamp is 10 digits
	// So: 8 + resourceID + 1 + 10 = 19 + resourceID <= 63
	// Therefore resourceID must be <= 44 characters
	truncatedID := resourceID
	if len(resourceID) > 44 {
		truncatedID = resourceID[:44]
		truncatedID = strings.TrimRight(truncatedID, "-")
	}
	workspaceName := fmt.Sprintf("fixture-%s-%d", truncatedID, time.Now().Unix())
	packageRevision := map[string]interface{}{
		"apiVersion": "porch.kpt.dev/v1alpha1",
		"kind":       "PackageRevision",
		"spec": map[string]interface{}{
			"packageName":   packageName,
			"repository":    s.repository,
			"lifecycle":     PorchLifecycleDraft,
			"workspaceName": workspaceName,
		},
	}

	// Create the PackageRevision as Draft
	path := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisions", s.namespace)
	resp, err := s.makeRequest(ctx, "POST", path, packageRevision)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to create PackageRevision", err)
	}

	// Accept both 201 (immediate success) and 500 (async processing)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusInternalServerError {
		bodyBytes, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return s.handleHTTPError(resp.StatusCode, bodyBytes)
	}
	_ = resp.Body.Close()

	// Wait for PackageRevision to exist as Draft
	prName, err := s.waitForPackageRevision(ctx, packageName, PorchLifecycleDraft, s.packageRevisionTimeout)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "PackageRevision not created", err)
	}

	// Update PackageRevisionResources with package contents
	// Retry with backoff since PackageRevisionResources might not be immediately available
	resourcesPath := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisionresources/%s", s.namespace, prName)

	maxRetries := 10
	retryDelay := 500 * time.Millisecond
	var existingPRR map[string]interface{}

	for i := 0; i < maxRetries; i++ {
		// First, GET the existing PackageRevisionResources to merge with our changes
		getResp, getErr := s.makeRequest(ctx, http.MethodGet, resourcesPath, nil)
		if getErr == nil && getResp.StatusCode == http.StatusOK {
			// Parse existing resources
			if parseErr := s.parseResponse(getResp, http.StatusOK, &existingPRR); parseErr == nil {
				// Extract existing resources
				spec, ok := existingPRR["spec"].(map[string]interface{})
				if ok {
					resources, ok := spec["resources"].(map[string]interface{})
					if !ok {
						resources = make(map[string]interface{})
					}

					// Merge our files with existing resources
					resources["Kptfile"] = kptfileContent
					resources[resourceFilename] = resourceYAML

					// Create PUT request with merged resources
					packageResources := map[string]interface{}{
						"apiVersion": "porch.kpt.dev/v1alpha1",
						"kind":       "PackageRevisionResources",
						"metadata":   existingPRR["metadata"],
						"spec": map[string]interface{}{
							"resources": resources,
						},
					}

					// PUT the updated resources
					resp, err = s.makeRequest(ctx, "PUT", resourcesPath, packageResources)
					if err == nil && resp.StatusCode == http.StatusOK {
						break
					}
				}
			}
		}

		// If GET failed or PUT failed, the resource might not be ready yet, retry
		time.Sleep(retryDelay)
	}

	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to update PackageRevisionResources after retries", err)
	}
	if resp == nil || resp.StatusCode != http.StatusOK {
		return NewStorageError(ErrorCodeStorageFailure, "failed to update PackageRevisionResources: resource not ready", nil)
	}

	if err := s.parseResponse(resp, http.StatusOK, nil); err != nil {
		return err
	}

	// Step 2: Transition Draft→Proposed (validation)
	// First, update the PackageRevision to Proposed state
	prPath := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisions/%s", s.namespace, prName)

	// Get the current PackageRevision to preserve fields
	getResp, err := s.makeRequest(ctx, http.MethodGet, prPath, nil)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to get PackageRevision", err)
	}

	var currentPR map[string]interface{}
	if err := s.parseResponse(getResp, http.StatusOK, &currentPR); err != nil {
		return err
	}

	// Update lifecycle to Proposed
	spec, _ := currentPR["spec"].(map[string]interface{})
	spec["lifecycle"] = PorchLifecycleProposed

	proposeResp, err := s.makeRequest(ctx, http.MethodPut, prPath, currentPR)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to propose PackageRevision", err)
	}

	if proposeResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(proposeResp.Body)
		_ = proposeResp.Body.Close()
		return s.handleHTTPError(proposeResp.StatusCode, bodyBytes)
	}
	_ = proposeResp.Body.Close()

	// Wait for Proposed state
	prName, err = s.waitForPackageRevision(ctx, packageName, PorchLifecycleProposed, s.packageRevisionTimeout)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "PackageRevision not proposed", err)
	}

	// Step 3: Approve Proposed→Published with specific revision
	// Get the latest PackageRevision to get resourceVersion
	getResp2, err := s.makeRequest(ctx, http.MethodGet, prPath, nil)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to get PackageRevision for approval", err)
	}

	var proposedPR map[string]interface{}
	if err := s.parseResponse(getResp2, http.StatusOK, &proposedPR); err != nil {
		return err
	}

	approvalPath := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisions/%s/approval", s.namespace, prName)

	// Generate next Porch revision number for this package
	nextRevision, err := s.generateNextRevisionID(ctx, packageName)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to generate revision number", err)
	}

	// Parse revision number from "v1" format to integer for new Porch API
	var revNum int
	if _, err := fmt.Sscanf(nextRevision, "v%d", &revNum); err != nil {
		return NewStorageError(ErrorCodeStorageFailure, fmt.Sprintf("failed to parse revision number from %s", nextRevision), err)
	}

	// Use the full PackageRevision with resourceVersion
	proposedSpec, _ := proposedPR["spec"].(map[string]interface{})
	proposedSpec["lifecycle"] = PorchLifecyclePublished
	proposedSpec["revision"] = revNum // Use integer instead of string for new Porch API

	approvalRequest := proposedPR

	approvalResp, err := s.makeRequest(ctx, http.MethodPut, approvalPath, approvalRequest)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to approve PackageRevision", err)
	}

	if approvalResp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(approvalResp.Body)
		_ = approvalResp.Body.Close()
		return s.handleHTTPError(approvalResp.StatusCode, bodyBytes)
	}
	_ = approvalResp.Body.Close()

	// Wait for PackageRevision to be Published
	_, err = s.waitForPackageRevision(ctx, packageName, PorchLifecyclePublished, s.packageRevisionTimeout)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "PackageRevision not published", err)
	}

	return nil
}

// UpdateDraftState updates the state field in a draft resource
// This is primarily used for test fixture setup to set drafts in specific states
// Note: This updates the resource data's state field, not the Porch lifecycle
func (s *PorchStorage) UpdateDraftState(ctx context.Context, resourceType ResourceType, id string, state ResourceState) error {
	// Find the draft PackageRevision
	packageRevision, err := s.findPackageRevision(ctx, id, PorchLifecycleDraft, PorchLifecycleProposed)
	if err != nil {
		return NewStorageError(ErrorCodeNotFound, fmt.Sprintf("draft for resource %s not found", id), err)
	}

	// Get PackageRevision name
	metadata, ok := packageRevision["metadata"].(map[string]interface{})
	if !ok {
		return NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevision metadata", nil)
	}
	prName, ok := metadata["name"].(string)
	if !ok {
		return NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevision name", nil)
	}

	// Get current PackageRevisionResources
	resourcesPath := fmt.Sprintf("/apis/porch.kpt.dev/v1alpha1/namespaces/%s/packagerevisionresources/%s", s.namespace, prName)
	resp, err := s.makeRequest(ctx, "GET", resourcesPath, nil)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to get PackageRevisionResources", err)
	}

	var prResources map[string]interface{}
	if err := s.parseResponse(resp, http.StatusOK, &prResources); err != nil {
		return err
	}

	// Extract resources
	spec, ok := prResources["spec"].(map[string]interface{})
	if !ok {
		return NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevisionResources spec", nil)
	}
	resources, ok := spec["resources"].(map[string]interface{})
	if !ok {
		return NewStorageError(ErrorCodeStorageFailure, "invalid PackageRevisionResources resources", nil)
	}

	// Determine resource filename
	var resourceFilename string
	switch resourceType {
	case ResourceTypeOCloud:
		resourceFilename = "ocloud.yaml"
	case ResourceTypeTemplateInfo:
		resourceFilename = "templateinfo.yaml"
	case ResourceTypeFocomProvisioningRequest:
		resourceFilename = "focomprovisioningrequest.yaml"
	default:
		return NewStorageError(ErrorCodeStorageFailure, fmt.Sprintf("unsupported resource type: %s", resourceType), nil)
	}

	// Get current resource YAML
	resourceYAML, ok := resources[resourceFilename].(string)
	if !ok {
		return NewStorageError(ErrorCodeStorageFailure, "resource YAML not found", nil)
	}

	// Parse the resource
	resource, err := s.parseResourceYAML(resourceYAML, resourceType)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to parse resource YAML", err)
	}

	// Update the state field in the resource
	switch r := resource.(type) {
	case *OCloudData:
		r.State = state
	case *TemplateInfoData:
		r.State = state
	case *FocomProvisioningRequestData:
		r.State = state
	default:
		return NewStorageError(ErrorCodeStorageFailure, fmt.Sprintf("unsupported resource type: %T", resource), nil)
	}

	// Convert back to YAML
	updatedYAML, err := s.createResourceYAML(resource, resourceType)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to create updated resource YAML", err)
	}

	// Update the resources map
	resources[resourceFilename] = updatedYAML

	// Update PackageRevisionResources
	resp, err = s.makeRequest(ctx, "PUT", resourcesPath, prResources)
	if err != nil {
		return NewStorageError(ErrorCodeStorageFailure, "failed to update PackageRevisionResources", err)
	}

	if err := s.parseResponse(resp, http.StatusOK, nil); err != nil {
		return err
	}

	return nil
}
