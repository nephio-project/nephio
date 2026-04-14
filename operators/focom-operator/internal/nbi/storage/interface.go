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
	"context"
	"encoding/json"
	"time"
)

// ResourceType represents the type of resource
type ResourceType string

const (
	ResourceTypeOCloud                   ResourceType = "ocloud"
	ResourceTypeTemplateInfo             ResourceType = "templateinfo"
	ResourceTypeFocomProvisioningRequest ResourceType = "focomprovisioningrequest"
)

// ResourceState represents the state of a resource
type ResourceState string

const (
	StateDraft     ResourceState = "DRAFT"
	StateValidated ResourceState = "VALIDATED"
	StateApproved  ResourceState = "APPROVED"
)

// StorageInterface defines the interface for all storage operations
type StorageInterface interface {
	// Generic CRUD operations for approved resources
	Create(ctx context.Context, resourceType ResourceType, resource interface{}) error
	Get(ctx context.Context, resourceType ResourceType, id string) (interface{}, error)
	Update(ctx context.Context, resourceType ResourceType, id string, resource interface{}) error
	Delete(ctx context.Context, resourceType ResourceType, id string) error
	List(ctx context.Context, resourceType ResourceType) ([]interface{}, error)

	// Draft management operations
	CreateDraft(ctx context.Context, resourceType ResourceType, draft interface{}) error
	GetDraft(ctx context.Context, resourceType ResourceType, id string) (interface{}, error)
	UpdateDraft(ctx context.Context, resourceType ResourceType, id string, draft interface{}) error
	DeleteDraft(ctx context.Context, resourceType ResourceType, id string) error
	ValidateDraft(ctx context.Context, resourceType ResourceType, id string) error
	ApproveDraft(ctx context.Context, resourceType ResourceType, id string) error
	RejectDraft(ctx context.Context, resourceType ResourceType, id string) error

	// Revision management operations
	GetRevisions(ctx context.Context, resourceType ResourceType, id string) ([]interface{}, error)
	GetRevision(ctx context.Context, resourceType ResourceType, id string, revisionId string) (interface{}, error)
	CreateRevision(ctx context.Context, resourceType ResourceType, resourceID string, revisionID string, data interface{}) error
	CreateDraftFromRevision(ctx context.Context, resourceType ResourceType, id string, revisionId string) error

	// Draft state management
	UpdateDraftState(ctx context.Context, resourceType ResourceType, id string, state ResourceState) error

	// Dependency validation
	ValidateDependencies(ctx context.Context, resourceType ResourceType, resource interface{}) error

	// Health and status
	HealthCheck(ctx context.Context) error
}

// BaseResource represents the common fields for all resources
type BaseResource struct {
	ID          string                 `json:"id"`
	RevisionID  string                 `json:"revisionId"`
	Namespace   string                 `json:"namespace"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	State       ResourceState          `json:"state"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// DraftResource represents a draft version of a resource
type DraftResource struct {
	ResourceID   string        `json:"resourceId"`
	ResourceType ResourceType  `json:"resourceType"`
	DraftData    interface{}   `json:"draftData"`
	State        ResourceState `json:"state"`
	CreatedAt    time.Time     `json:"createdAt"`
	UpdatedAt    time.Time     `json:"updatedAt"`
	ValidatedAt  *time.Time    `json:"validatedAt,omitempty"`
}

// RevisionResource represents an approved revision of a resource
type RevisionResource struct {
	ResourceID   string       `json:"resourceId"`
	RevisionID   string       `json:"revisionId"`
	ResourceType ResourceType `json:"resourceType"`
	RevisionData interface{}  `json:"revisionData"`
	ApprovedAt   time.Time    `json:"approvedAt"`
}

// OCloudData represents OCloud configuration data
type OCloudData struct {
	BaseResource
	O2IMSSecret O2IMSSecretRef `json:"o2imsSecret" yaml:"o2imsSecret"`
}

// MarshalJSON customizes JSON marshaling to use OpenAPI field names
func (o *OCloudData) MarshalJSON() ([]byte, error) {
	type Alias OCloudData
	return json.Marshal(&struct {
		OCloudID            string `json:"oCloudId"`
		OCloudRevisionState string `json:"oCloudRevisionState"`
		*Alias
	}{
		OCloudID:            o.ID,
		OCloudRevisionState: string(o.State),
		Alias:               (*Alias)(o),
	})
}

// O2IMSSecretRef represents a reference to an O2IMS secret
type O2IMSSecretRef struct {
	SecretRef SecretReference `json:"secretRef" yaml:"secretRef"`
}

// SecretReference represents a Kubernetes secret reference
type SecretReference struct {
	Name      string `json:"name" yaml:"name"`
	Namespace string `json:"namespace" yaml:"namespace"`
}

// TemplateInfoData represents TemplateInfo configuration data
type TemplateInfoData struct {
	BaseResource
	TemplateName            string `json:"templateName"`
	TemplateVersion         string `json:"templateVersion"`
	TemplateParameterSchema string `json:"templateParameterSchema"`
}

// FocomProvisioningRequestData represents FocomProvisioningRequest data
type FocomProvisioningRequestData struct {
	BaseResource
	OCloudID           string                 `json:"oCloudId"`
	OCloudNamespace    string                 `json:"oCloudNamespace"`
	TemplateName       string                 `json:"templateName"`
	TemplateVersion    string                 `json:"templateVersion"`
	TemplateParameters map[string]interface{} `json:"templateParameters"`
}
