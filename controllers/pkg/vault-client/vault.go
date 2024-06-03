/*
Copyright 2023 The Nephio Authors.

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

package vaultClient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	vault "github.com/hashicorp/vault/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	corev1 "k8s.io/api/core/v1"
)

type LoginPayload struct {
	Role string `json:"role"`
	JWT  string `json:"jwt"`
}

type AuthResponse struct {
	Auth struct {
		ClientToken string `json:"client_token"`
	} `json:"auth"`
}

func AuthenticateToVault(vaultAddr, jwt, role string) (string, error) {
	// Create a Vault client
	config := vault.DefaultConfig()
	config.Address = vaultAddr
	client, err := vault.NewClient(config)
	if err != nil {
		return "", fmt.Errorf("unable to create Vault client: %w", err)
	}

	payload := LoginPayload{
		Role: role,
		JWT:  jwt,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("unable to marshal payload: %w", err)
	}

	// Perform the login request
	req := client.NewRequest("POST", "/v1/auth/jwt/login")
	req.Body = bytes.NewBuffer(payloadBytes)

	resp, err := client.RawRequest(req)
	if err != nil {
		return "", fmt.Errorf("unable to perform login request: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("unable to read response body: %w", err)
	}

	var authResp AuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return "", fmt.Errorf("unable to decode response: %w", err)
	}

	return authResp.Auth.ClientToken, err
}

func StoreKubeconfig(ctx context.Context, kubeconfigData corev1.Secret, client *vault.Client, secretPath, clusterName string) error {
	// Read the Kubeconfig file

	// Prepare the data to store
	data := map[string]interface{}{
		"data": map[string]interface{}{
			clusterName: kubeconfigData.Data,
		},
	}

	// Store the data in Vault
	_, err := client.KVv2("secret").Put(ctx, "kubeconfigs"+clusterName, data)
	if err != nil {
		return fmt.Errorf("unable to write secret to Vault: %w", err)
	}

	fmt.Println("VAULT STORE TESTTTTT")

	return nil
}

func FetchKubeconfig(client *vault.Client, secretPath, clusterName string) (string, error) {
	// Read the secret
	secret, err := client.Logical().Read(secretPath)
	if err != nil {
		return "", fmt.Errorf("unable to read secret: %w", err)
	}

	if secret == nil {
		return "", fmt.Errorf("secret not found at path: %s", secretPath)
	}

	// Extract the Kubeconfig data
	kubeconfig, ok := secret.Data[clusterName].(string)
	if !ok {
		return "", fmt.Errorf("kubeconfig for cluster %s not found", clusterName)
	}

	return kubeconfig, nil
}

// VaultJWTRoleSpec defines the desired state of VaultJWTRole
type VaultJWTRoleSpec struct {
	RoleType       string   `json:"roleType"`
	UserClaim      string   `json:"userClaim"`
	BoundAudiences []string `json:"boundAudiences"`
	BoundSubject   string   `json:"boundSubject"`
	TokenTtl       string   `json:"tokenTtl"`
	TokenPolicies  []string `json:"tokenPolicies"`
}

// VaultJWTRoleStatus defines the observed state of VaultJWTRole
type VaultJWTRoleStatus struct {
	Conditions []VaultJWTRoleCondition `json:"conditions,omitempty"`
}

// VaultJWTRoleCondition defines the condition of VaultJWTRole
type VaultJWTRoleCondition struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// VaultJWTRole is the Schema for the vaultjwtroles API
type VaultJWTRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VaultJWTRoleSpec   `json:"spec,omitempty"`
	Status VaultJWTRoleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VaultJWTRoleList contains a list of VaultJWTRole
type VaultJWTRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VaultJWTRole `json:"items"`
}

// DeepCopyInto copies all properties of this object into another object of the same type
func (in *VaultJWTRole) DeepCopyInto(out *VaultJWTRole) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	// If you have a Status field, uncomment the following line
	// in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy creates a deep copy of a VaultJWTRole
func (in *VaultJWTRole) DeepCopy() *VaultJWTRole {
	if in == nil {
		return nil
	}
	out := new(VaultJWTRole)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a generically typed copy of an object
func (in *VaultJWTRole) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto copies all properties of this object into another object of the same type
func (in *VaultJWTRoleSpec) DeepCopyInto(out *VaultJWTRoleSpec) {
	*out = *in
	if in.BoundAudiences != nil {
		in, out := &in.BoundAudiences, &out.BoundAudiences
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.TokenPolicies != nil {
		in, out := &in.TokenPolicies, &out.TokenPolicies
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy creates a deep copy of a VaultJWTRoleSpec
func (in *VaultJWTRoleSpec) DeepCopy() *VaultJWTRoleSpec {
	if in == nil {
		return nil
	}
	out := new(VaultJWTRoleSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto copies all properties of this object into another object of the same type
func (in *VaultJWTRoleList) DeepCopyInto(out *VaultJWTRoleList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]VaultJWTRole, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy creates a deep copy of a VaultJWTRoleList
func (in *VaultJWTRoleList) DeepCopy() *VaultJWTRoleList {
	if in == nil {
		return nil
	}
	out := new(VaultJWTRoleList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a generically typed copy of an object
func (in *VaultJWTRoleList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
