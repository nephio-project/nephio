/*
Copyright 2023 Nephio.

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

package v1

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GetConditionType returns a string based on the KRM object Reference
// It acts on APIVersion, Kind and Name. if these parameters dont exist
// the string does not contain these parameters
func GetConditionType(o *corev1.ObjectReference) string {
	var sb strings.Builder
	sb.Reset()
	if o.APIVersion != "" {
		gv, err := schema.ParseGroupVersion(o.APIVersion)
		if err == nil {
			sb.WriteString(gv.String())
		}
	}
	if o.Kind != "" {
		if sb.String() != "" {
			sb.WriteString(".")
		}
		sb.WriteString(o.Kind)
	}
	if o.Name != "" {
		if sb.String() != "" {
			sb.WriteString(".")
		}
		sb.WriteString(o.Name)
	}
	return sb.String()
}

// GetGVKNFromConditionType return a KRM ObjectReference from a string
// It expects an APIVersion with a / as a.b/c and a kind + name
// if not it retruns an empty ObjectReference
func GetGVKNFromConditionType(ct string) (o *corev1.ObjectReference) {
	split := strings.Split(ct, "/")
	group := ""
	vkn := ct
	if len(split) > 1 {
		group = split[0]
		vkn = split[1]
	}
	newsplit := strings.Split(vkn, ".")
	if len(newsplit) == 3 {
		return &corev1.ObjectReference{
			APIVersion: fmt.Sprintf("%s/%s", group, newsplit[0]),
			Kind:       newsplit[1],
			Name:       newsplit[2],
		}
	}
	return &corev1.ObjectReference{}
}
