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
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// TestConfig holds configuration for integration tests
type TestConfig struct {
	Storage StorageConfig `yaml:"storage"`
	Porch   PorchConfig   `yaml:"porch"`
	Test    TestOptions   `yaml:"test"`
}

// StorageConfig specifies which storage backend to use
type StorageConfig struct {
	Backend string `yaml:"backend"` // "memory" or "porch"
}

// PorchConfig holds Porch-specific configuration
type PorchConfig struct {
	UseKubeconfig bool   `yaml:"useKubeconfig,omitempty"` // Use kubeconfig for auth (handles exec, certs, etc.)
	Kubeconfig    string `yaml:"kubeconfig,omitempty"`    // Path to kubeconfig (optional, auto-detected)
	KubernetesURL string `yaml:"kubernetesURL,omitempty"` // Optional, auto-detected from KUBECONFIG
	Token         string `yaml:"token,omitempty"`         // Optional, auto-detected
	Namespace     string `yaml:"namespace"`
	Repository    string `yaml:"repository"`
	HTTPSVerify   bool   `yaml:"httpsVerify"`
}

// TestOptions holds test execution options
type TestOptions struct {
	Cleanup        bool          `yaml:"cleanup"`        // Clean up resources after tests
	UseUniqueIDs   bool          `yaml:"useUniqueIDs"`   // Generate unique IDs to avoid conflicts
	ResourcePrefix string        `yaml:"resourcePrefix"` // Prefix for test resource IDs
	Timeout        time.Duration `yaml:"timeout"`        // Timeout for test operations
}

// DefaultTestConfig returns the default test configuration (InMemoryStorage)
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		Storage: StorageConfig{
			Backend: "memory",
		},
		Porch: PorchConfig{
			Namespace:   "default",
			Repository:  "focom-resources",
			HTTPSVerify: false,
		},
		Test: TestOptions{
			Cleanup:        true,
			UseUniqueIDs:   true,
			ResourcePrefix: "test-",
			Timeout:        30 * time.Second,
		},
	}
}

// LoadTestConfig loads test configuration from file and environment variables
// Environment variables take precedence over config file values
func LoadTestConfig(configPath string) (*TestConfig, error) {
	config := DefaultTestConfig()

	// Try to load from file if it exists
	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath) // #nosec G304 -- config path is a known local file
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Environment variables override config file
	if backend := os.Getenv("NBI_STORAGE_BACKEND"); backend != "" {
		config.Storage.Backend = backend
	}

	if kubeURL := os.Getenv("KUBERNETES_BASE_URL"); kubeURL != "" {
		config.Porch.KubernetesURL = kubeURL
	}

	if token := os.Getenv("TOKEN"); token != "" {
		config.Porch.Token = token
	}

	if namespace := os.Getenv("PORCH_NAMESPACE"); namespace != "" {
		config.Porch.Namespace = namespace
	}

	if repository := os.Getenv("PORCH_REPOSITORY"); repository != "" {
		config.Porch.Repository = repository
	}

	return config, nil
}

// LoadTestConfigOrDefault loads test configuration, falling back to defaults on error
func LoadTestConfigOrDefault() *TestConfig {
	// Look for config file in current directory or internal/nbi directory
	configPaths := []string{
		"testconfig.yaml",
		"internal/nbi/testconfig.yaml",
		filepath.Join("internal", "nbi", "testconfig.yaml"),
	}

	for _, path := range configPaths {
		if config, err := LoadTestConfig(path); err == nil {
			return config
		}
	}

	// Fall back to defaults (InMemoryStorage)
	return DefaultTestConfig()
}
