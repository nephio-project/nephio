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

// SecretReference represents a reference to a Kubernetes secret
type SecretReference struct {
	Name      string `json:"name" validate:"required"`
	Namespace string `json:"namespace" validate:"required"`
}

// O2IMSSecretRef represents the O2IMS secret reference structure
type O2IMSSecretRef struct {
	SecretRef SecretReference `json:"secretRef" validate:"required"`
}

// OCloudData represents the internal data model for OCloud configurations
type OCloudData struct {
	BaseResource
	O2IMSSecret O2IMSSecretRef `json:"o2imsSecret" validate:"required"`
}

// NewOCloudData creates a new OCloudData instance
func NewOCloudData(namespace, name, description string, o2imsSecret O2IMSSecretRef) *OCloudData {
	baseResource := NewBaseResource(namespace, name, description)
	// For OClouds, the ID should be the same as the name to match Kubernetes resource naming
	baseResource.ID = name
	return &OCloudData{
		BaseResource: baseResource,
		O2IMSSecret:  o2imsSecret,
	}
}

// Clone creates a deep copy of the OCloudData
func (o *OCloudData) Clone() *OCloudData {
	clone := &OCloudData{
		BaseResource: o.BaseResource,
		O2IMSSecret:  o.O2IMSSecret,
	}

	// Deep copy metadata if it exists
	if o.Metadata != nil {
		clone.Metadata = make(map[string]interface{})
		for k, v := range o.Metadata {
			clone.Metadata[k] = v
		}
	}

	return clone
}

// Update applies updates to the OCloudData
func (o *OCloudData) Update(name, description *string, o2imsSecret *O2IMSSecretRef) {
	if name != nil {
		o.Name = *name
	}
	if description != nil {
		o.Description = *description
	}
	if o2imsSecret != nil {
		o.O2IMSSecret = *o2imsSecret
	}
	o.UpdateTimestamp()
}

// OCloudDataUpdate represents the update structure for OCloud configurations
type OCloudDataUpdate struct {
	Name        *string         `json:"name,omitempty"`
	Description *string         `json:"description,omitempty"`
	O2IMSSecret *O2IMSSecretRef `json:"o2imsSecret,omitempty"`
}

// OCloudInfo represents complete information about an OCloud configuration including status
type OCloudInfo struct {
	OCloudData   *OCloudData   `json:"oCloudData" validate:"required"`
	OCloudStatus *OCloudStatus `json:"oCloudStatus,omitempty"`
}

// OCloudStatus represents status information for OCloud configurations
type OCloudStatus struct {
	Message string `json:"message,omitempty"`
}

// MarshalJSON customizes JSON marshaling to use OpenAPI field names
func (o *OCloudData) MarshalJSON() ([]byte, error) {
	type Alias OCloudData
	return json.Marshal(&struct {
		ResourceID          string `json:"resourceId"`
		OCloudID            string `json:"oCloudId"`
		OCloudRevisionState string `json:"oCloudRevisionState"`
		*Alias
	}{
		ResourceID:          o.ID,
		OCloudID:            o.ID,
		OCloudRevisionState: string(o.State),
		Alias:               (*Alias)(o),
	})
}
