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

package config

import (
	"fmt"
	"os"
	"strconv"
)

// NBIConfig holds configuration for the NBI system
type NBIConfig struct {
	// HTTP server configuration
	Port int `json:"port"`

	// Storage configuration
	StorageBackend StorageBackend `json:"storageBackend"`

	// Implementation stage configuration
	Stage ImplementationStage `json:"stage"`

	// O2IMS configuration (for stage 3)
	O2IMSConfig O2IMSConfig `json:"o2imsConfig"`

	// Default namespace for FOCOM resources
	// If set, this namespace will be used for all resources unless overridden in the request body
	Namespace string `json:"namespace"`

	// EarlySchemaValidation enables schema validation of FPR templateParameters
	// during CreateDraft and UpdateDraft operations (not just ValidateDraft).
	// Controlled by the FOCOM_EARLY_SCHEMA_VALIDATION environment variable.
	EarlySchemaValidation bool `json:"earlySchemaValidation"`
}

// StorageBackend represents the storage backend type
type StorageBackend string

const (
	StorageBackendMemory StorageBackend = "memory"
	StorageBackendPorch  StorageBackend = "porch"
)

// ImplementationStage represents the implementation stage
type ImplementationStage int

const (
	Stage1 ImplementationStage = 1 // In-memory storage + Kubernetes API
	Stage2 ImplementationStage = 2 // Nephio Porch storage + Kubernetes API
	Stage3 ImplementationStage = 3 // Nephio Porch storage + O2IMS REST endpoints
)

// O2IMSConfig holds O2IMS-specific configuration
type O2IMSConfig struct {
	Endpoints []O2IMSEndpoint `json:"endpoints"`
}

// O2IMSEndpoint represents an O2IMS server endpoint
type O2IMSEndpoint struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// DefaultNBIConfig returns the default NBI configuration
func DefaultNBIConfig() *NBIConfig {
	return &NBIConfig{
		Port:           8080,
		StorageBackend: StorageBackendMemory,
		Stage:          Stage1,
		O2IMSConfig:    O2IMSConfig{},
		Namespace:      "focom-system", // Default namespace
	}
}

// LoadFromEnvironment loads configuration from environment variables
func (c *NBIConfig) LoadFromEnvironment() error {
	// Load port
	if portStr := os.Getenv("NBI_PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("invalid NBI_PORT: %w", err)
		}
		c.Port = port
	}

	// Load storage backend
	if backend := os.Getenv("NBI_STORAGE_BACKEND"); backend != "" {
		c.StorageBackend = StorageBackend(backend)
	}

	// Load implementation stage
	if stageStr := os.Getenv("NBI_STAGE"); stageStr != "" {
		stage, err := strconv.Atoi(stageStr)
		if err != nil {
			return fmt.Errorf("invalid NBI_STAGE: %w", err)
		}
		c.Stage = ImplementationStage(stage)
	}

	// Load default namespace
	if namespace := os.Getenv("FOCOM_NAMESPACE"); namespace != "" {
		c.Namespace = namespace
	}

	// Load early schema validation flag
	if earlyValidation := os.Getenv("FOCOM_EARLY_SCHEMA_VALIDATION"); earlyValidation != "" {
		c.EarlySchemaValidation = earlyValidation == "true"
	}

	return nil
}

// Validate validates the configuration
func (c *NBIConfig) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}

	if c.StorageBackend != StorageBackendMemory && c.StorageBackend != StorageBackendPorch {
		return fmt.Errorf("invalid storage backend: %s", c.StorageBackend)
	}

	if c.Stage < Stage1 || c.Stage > Stage3 {
		return fmt.Errorf("invalid stage: %d", c.Stage)
	}

	// Validate namespace
	if c.Namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}

	// Validate stage-specific requirements
	switch c.Stage {
	case Stage1:
		if c.StorageBackend != StorageBackendMemory {
			return fmt.Errorf("stage 1 requires memory storage backend")
		}
	case Stage2:
		if c.StorageBackend != StorageBackendPorch {
			return fmt.Errorf("stage 2 requires porch storage backend")
		}
	case Stage3:
		if c.StorageBackend != StorageBackendPorch {
			return fmt.Errorf("stage 3 requires porch storage backend")
		}
		if len(c.O2IMSConfig.Endpoints) == 0 {
			return fmt.Errorf("stage 3 requires O2IMS endpoints configuration")
		}
	}

	return nil
}
