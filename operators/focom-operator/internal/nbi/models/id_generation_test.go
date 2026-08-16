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
	"testing"
)

func TestSanitizeID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple lowercase",
			input:    "my-template-v1.0.0",
			expected: "my-template-v1-0-0",
		},
		{
			name:     "uppercase to lowercase",
			input:    "MyTemplate-V1.0.0",
			expected: "mytemplate-v1-0-0",
		},
		{
			name:     "special characters",
			input:    "my_template@v1.0.0",
			expected: "my-template-v1-0-0",
		},
		{
			name:     "consecutive hyphens",
			input:    "my---template--v1",
			expected: "my-template-v1",
		},
		{
			name:     "leading and trailing hyphens",
			input:    "-my-template-",
			expected: "my-template",
		},
		{
			name:     "spaces",
			input:    "my template v1",
			expected: "my-template-v1",
		},
		{
			name:     "mixed special chars",
			input:    "ocloud_01-template@v1.2.3",
			expected: "ocloud-01-template-v1-2-3",
		},
		{
			name:     "exceeds 63 character limit",
			input:    "very-long-ocloud-name-with-very-long-template-name-and-very-long-version-v1.0.0",
			expected: "very-long-ocloud-name-with-very-long-template-name-and-very-lon",
		},
		{
			name:     "exactly 63 characters",
			input:    "ocloud-template-v1-0-0-with-exactly-sixty-three-characters",
			expected: "ocloud-template-v1-0-0-with-exactly-sixty-three-characters",
		},
		{
			name:     "truncation removes trailing hyphen",
			input:    "very-long-name-that-will-be-truncated-at-exactly-a-hyphen-position-v1",
			expected: "very-long-name-that-will-be-truncated-at-exactly-a-hyphen-posit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeID(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeID(%q) = %q, want %q", tt.input, result, tt.expected)
			}
			// Verify it's within the 63 character limit
			if len(result) > 63 {
				t.Errorf("SanitizeID(%q) resulted in %d characters, exceeds 63 character limit", tt.input, len(result))
			}
		})
	}
}

func TestTemplateInfoIDGeneration(t *testing.T) {
	templateInfo := NewTemplateInfoData(
		"default",
		"My Template",
		"Test template",
		"my-template",
		"v1.0.0",
		`{"type": "object"}`,
	)

	expected := "my-template-v1-0-0"
	if templateInfo.ID != expected {
		t.Errorf("TemplateInfo ID = %q, want %q", templateInfo.ID, expected)
	}
}

func TestFocomProvisioningRequestIDGeneration(t *testing.T) {
	fpr := NewFocomProvisioningRequestData(
		"default",
		"My FPR",
		"Test FPR",
		"ocloud-01",
		"default",
		"my-template",
		"v1.0.0",
		map[string]interface{}{"key": "value"},
	)

	// FPR ID uses the name field directly, matching the OCloud pattern
	expected := "My FPR"
	if fpr.ID != expected {
		t.Errorf("FocomProvisioningRequest ID = %q, want %q", fpr.ID, expected)
	}
}

func TestOCloudIDGeneration(t *testing.T) {
	ocloud := NewOCloudData(
		"default",
		"my-ocloud",
		"Test OCloud",
		O2IMSSecretRef{
			SecretRef: SecretReference{
				Name:      "secret",
				Namespace: "default",
			},
		},
	)

	expected := "my-ocloud"
	if ocloud.ID != expected {
		t.Errorf("OCloud ID = %q, want %q", ocloud.ID, expected)
	}
}
