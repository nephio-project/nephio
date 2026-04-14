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

package models

import (
	"time"
)

// ErrorResponse represents the standard error response structure
type ErrorResponse struct {
	Error     string    `json:"error" validate:"required"`
	Code      string    `json:"code" validate:"required"`
	Details   string    `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	RequestID string    `json:"requestId,omitempty"`
}

// NewErrorResponse creates a new ErrorResponse
func NewErrorResponse(error, code, details, requestID string) *ErrorResponse {
	return &ErrorResponse{
		Error:     error,
		Code:      code,
		Details:   details,
		Timestamp: time.Now(),
		RequestID: requestID,
	}
}

// Error codes for programmatic error handling
const (
	ErrorCodeValidation    = "VALIDATION_ERROR"
	ErrorCodeNotFound      = "NOT_FOUND"
	ErrorCodeConflict      = "CONFLICT"
	ErrorCodeDependency    = "DEPENDENCY_ERROR"
	ErrorCodeInvalidState  = "INVALID_STATE"
	ErrorCodeInternalError = "INTERNAL_ERROR"
	ErrorCodeBadRequest    = "BAD_REQUEST"
	ErrorCodeUnauthorized  = "UNAUTHORIZED"
	ErrorCodeForbidden     = "FORBIDDEN"
)

// DependencyError represents a dependency validation error
type DependencyError struct {
	ResourceType  ResourceType `json:"resourceType"`
	ResourceID    string       `json:"resourceId"`
	DependentType ResourceType `json:"dependentType"`
	DependentID   string       `json:"dependentId"`
	Message       string       `json:"message"`
}

// NewDependencyError creates a new DependencyError
func NewDependencyError(resourceType ResourceType, resourceID string, dependentType ResourceType, dependentID, message string) *DependencyError {
	return &DependencyError{
		ResourceType:  resourceType,
		ResourceID:    resourceID,
		DependentType: dependentType,
		DependentID:   dependentID,
		Message:       message,
	}
}

// Error implements the error interface
func (de *DependencyError) Error() string {
	return de.Message
}

// ValidationError represents a validation error with field-specific details
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value,omitempty"`
	Message string `json:"message"`
}

// NewValidationError creates a new ValidationError
func NewValidationError(field, value, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// Error implements the error interface
func (ve *ValidationError) Error() string {
	return ve.Message
}
