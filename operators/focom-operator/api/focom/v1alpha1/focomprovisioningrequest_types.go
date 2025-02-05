/*
Copyright 2025.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FocomProvisioningRequestSpec defines the desired state of FocomProvisioningRequest
type FocomProvisioningRequestSpec struct {
	// +kubebuilder:validation:Required
	OCloudId string `json:"oCloudId"`

	// +kubebuilder:validation:Required
	OCloudNamespace string `json:"oCloudNamespace"`

	// +kubebuilder:validation:Optional
	Name string `json:"name,omitempty"`

	// +kubebuilder:validation:Optional
	Description string `json:"description,omitempty"`

	// +kubebuilder:validation:Required
	TemplateName string `json:"templateName"`

	// +kubebuilder:validation:Required
	TemplateVersion string `json:"templateVersion"`

	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Required
	TemplateParameters runtime.RawExtension `json:"templateParameters"`
}

// FocomProvisioningRequestStatus defines the observed state of FocomProvisioningRequest
type FocomProvisioningRequestStatus struct {
	Phase       string       `json:"phase,omitempty"`
	Message     string       `json:"message,omitempty"`
	LastUpdated *metav1.Time `json:"lastUpdated,omitempty"`
	// The name of the remote resource in the target cluster
	RemoteName string `json:"remoteName,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// FocomProvisioningRequest is the Schema for the focomprovisioningrequests API
type FocomProvisioningRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FocomProvisioningRequestSpec   `json:"spec,omitempty"`
	Status FocomProvisioningRequestStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FocomProvisioningRequestList contains a list of FocomProvisioningRequest
type FocomProvisioningRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FocomProvisioningRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FocomProvisioningRequest{}, &FocomProvisioningRequestList{})
}
