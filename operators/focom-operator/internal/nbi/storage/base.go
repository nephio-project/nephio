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

import "errors"

// Common storage errors
var (
	ErrResourceNotFound   = errors.New("resource not found")
	ErrResourceExists     = errors.New("resource already exists")
	ErrInvalidResourceID  = errors.New("invalid resource ID")
	ErrInvalidRevisionID  = errors.New("invalid revision ID")
	ErrStorageUnavailable = errors.New("storage unavailable")
	ErrConcurrentAccess   = errors.New("concurrent access conflict")
)

// StorageError represents a storage-specific error
type StorageError struct {
	Code    string
	Message string
	Cause   error
}

func (e *StorageError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *StorageError) Unwrap() error {
	return e.Cause
}

// NewStorageError creates a new storage error
func NewStorageError(code, message string, cause error) *StorageError {
	return &StorageError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// Error codes for storage operations
const (
	ErrorCodeNotFound         = "RESOURCE_NOT_FOUND"
	ErrorCodeAlreadyExists    = "RESOURCE_ALREADY_EXISTS"
	ErrorCodeInvalidID        = "INVALID_RESOURCE_ID"
	ErrorCodeInvalidRevision  = "INVALID_REVISION_ID"
	ErrorCodeInvalidState     = "INVALID_STATE"
	ErrorCodeDependencyFailed = "DEPENDENCY_VALIDATION_FAILED"
	ErrorCodeStorageFailure   = "STORAGE_FAILURE"
	ErrorCodeConcurrentAccess = "CONCURRENT_ACCESS_CONFLICT"
)
