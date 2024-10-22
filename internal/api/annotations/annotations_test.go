/*
Copyright 2024.

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

package annotations

import "testing"

func TestIsCustomPolicyEnabled(t *testing.T) {
	testcases := []struct {
		description string
		annotations map[string]string
		expected    bool
	}{
		{
			description: "empty map",
			annotations: map[string]string{},
			expected:    false,
		},
		{
			description: "annotation set but not to identified value means the default",
			annotations: map[string]string{
				SELinuxPolicyConfigAnnotation: "true",
			},
			expected: false,
		},
		{
			description: "enabled custom policy",
			annotations: map[string]string{
				SELinuxPolicyConfigAnnotation: "custom",
			},
			expected: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			if got := IsCustomPolicyEnabled(tc.annotations); got != tc.expected {
				t.Errorf("expected %v got %v", tc.expected, got)
			}
		})
	}
}

func TestIsMultiplePoolsPerTree(t *testing.T) {
	testcases := []struct {
		description string
		annotations map[string]string
		expected    bool
	}{
		{
			description: "empty map",
			annotations: map[string]string{},
			expected:    false,
		},
		{
			description: "annotation set but not to allowed value means single pool",
			annotations: map[string]string{
				MultiplePoolsPerTreeAnnotation: "true",
			},
			expected: false,
		},
		{
			description: "enabled multiple pools per tree",
			annotations: map[string]string{
				MultiplePoolsPerTreeAnnotation: "enabled",
			},
			expected: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			if got := IsMultiplePoolsPerTree(tc.annotations); got != tc.expected {
				t.Errorf("expected %v got %v", tc.expected, got)
			}
		})
	}
}
