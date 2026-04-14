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

package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/nephio-project/nephio/operators/focom-operator/internal/nbi/models"
)

// Common service errors
var (
	ErrInvalidContext     = errors.New("invalid context")
	ErrResourceNotFound   = errors.New("resource not found")
	ErrResourceExists     = errors.New("resource already exists")
	ErrInvalidState       = errors.New("invalid resource state")
	ErrDependencyNotFound = errors.New("dependency not found")
	ErrValidationFailed   = errors.New("validation failed")
)

// EarlyValidationError is returned when early schema validation fails during CreateDraft or UpdateDraft.
type EarlyValidationError struct {
	Errors       []string
	SchemaErrors []models.SchemaValidationError
}

func (e *EarlyValidationError) Error() string {
	return fmt.Sprintf("schema validation failed: %s", strings.Join(e.Errors, "; "))
}
