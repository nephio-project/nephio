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
	"encoding/json"
)

// TemplateInfoData represents the internal data model for TemplateInfo configurations
type TemplateInfoData struct {
	BaseResource
	TemplateName            string `json:"templateName" validate:"required"`
	TemplateVersion         string `json:"templateVersion" validate:"required"`
	TemplateParameterSchema string `json:"templateParameterSchema" validate:"required"`
}

// NewTemplateInfoData creates a new TemplateInfoData instance
func NewTemplateInfoData(namespace, name, description, templateName, templateVersion, templateParameterSchema string) *TemplateInfoData {
	baseResource := NewBaseResource(namespace, name, description)
	// Generate ID from templateName and templateVersion
	baseResource.ID = SanitizeID(templateName + "-" + templateVersion)
	return &TemplateInfoData{
		BaseResource:            baseResource,
		TemplateName:            templateName,
		TemplateVersion:         templateVersion,
		TemplateParameterSchema: templateParameterSchema,
	}
}

// Clone creates a deep copy of the TemplateInfoData
func (t *TemplateInfoData) Clone() *TemplateInfoData {
	clone := &TemplateInfoData{
		BaseResource:            t.BaseResource,
		TemplateName:            t.TemplateName,
		TemplateVersion:         t.TemplateVersion,
		TemplateParameterSchema: t.TemplateParameterSchema,
	}

	// Deep copy metadata if it exists
	if t.Metadata != nil {
		clone.Metadata = make(map[string]interface{})
		for k, v := range t.Metadata {
			clone.Metadata[k] = v
		}
	}

	return clone
}

// Update applies updates to the TemplateInfoData
func (t *TemplateInfoData) Update(name, description, templateName, templateVersion, templateParameterSchema *string) {
	if name != nil {
		t.Name = *name
	}
	if description != nil {
		t.Description = *description
	}
	if templateName != nil {
		t.TemplateName = *templateName
	}
	if templateVersion != nil {
		t.TemplateVersion = *templateVersion
	}
	if templateParameterSchema != nil {
		t.TemplateParameterSchema = *templateParameterSchema
	}
	t.UpdateTimestamp()
}

// TemplateInfoDataUpdate represents the update structure for TemplateInfo configurations
type TemplateInfoDataUpdate struct {
	Name                    *string `json:"name,omitempty"`
	Description             *string `json:"description,omitempty"`
	TemplateName            *string `json:"templateName,omitempty"`
	TemplateVersion         *string `json:"templateVersion,omitempty"`
	TemplateParameterSchema *string `json:"templateParameterSchema,omitempty"`
}

// TemplateInfoInfo represents complete information about a TemplateInfo configuration including status
type TemplateInfoInfo struct {
	TemplateInfoData   *TemplateInfoData   `json:"templateInfoData" validate:"required"`
	TemplateInfoStatus *TemplateInfoStatus `json:"templateInfoStatus,omitempty"`
}

// TemplateInfoStatus represents status information for TemplateInfo configurations
type TemplateInfoStatus struct {
	Message string `json:"message,omitempty"`
}

// MarshalJSON customizes JSON marshaling to use OpenAPI field names
func (t *TemplateInfoData) MarshalJSON() ([]byte, error) {
	type Alias TemplateInfoData
	return json.Marshal(&struct {
		ResourceID                string `json:"resourceId"`
		TemplateInfoID            string `json:"templateInfoId"`
		TemplateInfoRevisionState string `json:"templateInfoRevisionState"`
		*Alias
	}{
		ResourceID:                t.ID,
		TemplateInfoID:            t.ID,
		TemplateInfoRevisionState: string(t.State),
		Alias:                     (*Alias)(t),
	})
}
