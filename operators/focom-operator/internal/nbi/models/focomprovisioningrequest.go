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
	"time"
)

// FocomProvisioningRequestData represents the internal data model for FOCOM provisioning requests
type FocomProvisioningRequestData struct {
	BaseResource
	OCloudID           string                 `json:"oCloudId" validate:"required"`
	OCloudNamespace    string                 `json:"oCloudNamespace" validate:"required"`
	TemplateName       string                 `json:"templateName" validate:"required"`
	TemplateVersion    string                 `json:"templateVersion" validate:"required"`
	TemplateParameters map[string]interface{} `json:"templateParameters" validate:"required"`
}

// NewFocomProvisioningRequestData creates a new FocomProvisioningRequestData instance
func NewFocomProvisioningRequestData(namespace, name, description, oCloudID, oCloudNamespace, templateName, templateVersion string, templateParameters map[string]interface{}) *FocomProvisioningRequestData {
	baseResource := NewBaseResource(namespace, name, description)
	// FPR ID uses the name field directly, matching the OCloud pattern.
	// This gives users control over the resource identifier.
	baseResource.ID = name
	return &FocomProvisioningRequestData{
		BaseResource:       baseResource,
		OCloudID:           oCloudID,
		OCloudNamespace:    oCloudNamespace,
		TemplateName:       templateName,
		TemplateVersion:    templateVersion,
		TemplateParameters: templateParameters,
	}
}

// Clone creates a deep copy of the FocomProvisioningRequestData
func (f *FocomProvisioningRequestData) Clone() *FocomProvisioningRequestData {
	clone := &FocomProvisioningRequestData{
		BaseResource:    f.BaseResource,
		OCloudID:        f.OCloudID,
		OCloudNamespace: f.OCloudNamespace,
		TemplateName:    f.TemplateName,
		TemplateVersion: f.TemplateVersion,
	}

	// Deep copy template parameters
	if f.TemplateParameters != nil {
		clone.TemplateParameters = make(map[string]interface{})
		for k, v := range f.TemplateParameters {
			clone.TemplateParameters[k] = v
		}
	}

	// Deep copy metadata if it exists
	if f.Metadata != nil {
		clone.Metadata = make(map[string]interface{})
		for k, v := range f.Metadata {
			clone.Metadata[k] = v
		}
	}

	return clone
}

// Update applies updates to the FocomProvisioningRequestData
func (f *FocomProvisioningRequestData) Update(name, description, templateName, templateVersion *string, templateParameters map[string]interface{}) {
	if name != nil {
		f.Name = *name
	}
	if description != nil {
		f.Description = *description
	}
	if templateName != nil {
		f.TemplateName = *templateName
	}
	if templateVersion != nil {
		f.TemplateVersion = *templateVersion
	}
	if templateParameters != nil {
		f.TemplateParameters = templateParameters
	}
	f.UpdateTimestamp()
}

// FocomProvisioningRequestDataUpdate represents the update structure for FOCOM provisioning requests
type FocomProvisioningRequestDataUpdate struct {
	Name               *string                `json:"name,omitempty"`
	Description        *string                `json:"description,omitempty"`
	TemplateName       *string                `json:"templateName,omitempty"`
	TemplateVersion    *string                `json:"templateVersion,omitempty"`
	TemplateParameters map[string]interface{} `json:"templateParameters,omitempty"`
}

// FocomProvisioningRequestInfo represents complete information about a FOCOM provisioning request including status
type FocomProvisioningRequestInfo struct {
	FocomProvisioningRequestData       *FocomProvisioningRequestData `json:"focomProvisioningRequestData" validate:"required"`
	OCloudProvisioningRequestReference string                        `json:"oCloudProvisioningRequestReference"`
	FocomProvisioningStatus            *FocomProvisioningStatus      `json:"focomProvisioningStatus,omitempty"`
	FocomProvisionedResourceSet        *FocomProvisionedResourceSet  `json:"focomProvisionedResourceSet,omitempty"`
}

// FocomProvisioningStatus represents status information for FOCOM provisioning requests
type FocomProvisioningStatus struct {
	Phase       string     `json:"phase,omitempty"`
	Message     string     `json:"message,omitempty"`
	LastUpdated *time.Time `json:"lastUpdated,omitempty"`
	RemoteName  string     `json:"remoteName,omitempty"`
}

// FocomProvisionedResourceSet represents resources that have been successfully provisioned
type FocomProvisionedResourceSet struct {
	ClusterID       string                `json:"clusterId,omitempty"`
	ClusterEndpoint string                `json:"clusterEndpoint,omitempty"`
	Resources       []ProvisionedResource `json:"resources,omitempty"`
}

// ProvisionedResource represents individual provisioned resource information
type ProvisionedResource struct {
	ResourceID   string `json:"resourceId" validate:"required"`
	ResourceType string `json:"resourceType" validate:"required"`
	ResourceName string `json:"resourceName" validate:"required"`
	Status       string `json:"status" validate:"required,oneof=ACTIVE INACTIVE ERROR"`
}

// MarshalJSON customizes JSON marshaling to use OpenAPI field names
func (f *FocomProvisioningRequestData) MarshalJSON() ([]byte, error) {
	type Alias FocomProvisioningRequestData
	return json.Marshal(&struct {
		ResourceID                            string `json:"resourceId"`
		ProvisioningRequestID                 string `json:"provisioningRequestId"`
		FocomProvisioningRequestRevisionState string `json:"focomProvisioningRequestRevisionState"`
		*Alias
	}{
		ResourceID:                            f.ID,
		ProvisioningRequestID:                 f.ID,
		FocomProvisioningRequestRevisionState: string(f.State),
		Alias:                                 (*Alias)(f),
	})
}
