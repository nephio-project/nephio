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

package condkptsdk

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// validateGVKRef returns an error if the ApiVersion or Kind 
// contain an empty string 
func validateGVKRef(ref corev1.ObjectReference) error {
	if ref.APIVersion == "" || ref.Kind == ""  {
		return fmt.Errorf("gvk not initialized, got: %v", ref)
	}
	return nil
}

// validateGVKNRef returns an error if the ApiVersion or Kind or Name
// contain an empty string 
func validateGVKNRef(ref corev1.ObjectReference) error {
	if ref.APIVersion == "" || ref.Kind == "" || ref.Name == "" {
		return fmt.Errorf("gvk or name not initialized, got: %v", ref)
	}
	return nil
}

// getGVKRefFromGVKNref return a new objectReference with only APIVersion and Kind
func getGVKRefFromGVKNref(ref *corev1.ObjectReference) *corev1.ObjectReference {
	return &corev1.ObjectReference{APIVersion: ref.APIVersion, Kind: ref.Kind}
}