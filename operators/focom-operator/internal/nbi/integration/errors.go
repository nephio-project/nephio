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

package integration

import "errors"

// Integration-specific errors
var (
	ErrKubernetesClientUnavailable = errors.New("kubernetes client unavailable")
	ErrO2IMSEndpointUnavailable    = errors.New("O2IMS endpoint unavailable")
	ErrCRCreationFailed            = errors.New("custom resource creation failed")
	ErrO2IMSRequestFailed          = errors.New("O2IMS request failed")
	ErrInvalidCRDMapping           = errors.New("invalid CRD mapping")
	ErrUnsupportedOperation        = errors.New("unsupported operation for current stage")
)

// IntegrationError represents an integration-specific error
type IntegrationError struct {
	Code    string
	Message string
	Cause   error
}

func (e *IntegrationError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *IntegrationError) Unwrap() error {
	return e.Cause
}

// NewIntegrationError creates a new integration error
func NewIntegrationError(code, message string, cause error) *IntegrationError {
	return &IntegrationError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// Error codes for integration operations
const (
	ErrorCodeKubernetesUnavailable = "KUBERNETES_UNAVAILABLE"
	ErrorCodeO2IMSUnavailable      = "O2IMS_UNAVAILABLE"
	ErrorCodeCRCreationFailed      = "CR_CREATION_FAILED"
	ErrorCodeO2IMSRequestFailed    = "O2IMS_REQUEST_FAILED"
	ErrorCodeInvalidMapping        = "INVALID_MAPPING"
	ErrorCodeUnsupportedOperation  = "UNSUPPORTED_OPERATION"
)
