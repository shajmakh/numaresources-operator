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

const (
	SELinuxPolicyConfigAnnotation = "config.node.openshift-kni.io/selinux-policy"
	SELinuxPolicyCustom           = "custom"
	// MultiplePoolsPerTreeAnnotation an annotation used to re-enable the support of multiple node pools per tree; starting 4.18 it is disabled by default
	MultiplePoolsPerTreeAnnotation = "experimental.multiple-pools-per-tree"
	MultiplePoolsPerTreeEnabled    = "enabled"
)

func IsCustomPolicyEnabled(annot map[string]string) bool {
	if v, ok := annot[SELinuxPolicyConfigAnnotation]; ok && v == SELinuxPolicyCustom {
		return true
	}
	return false
}

func IsMultiplePoolsPerTree(annot map[string]string) bool {
	if v, ok := annot[MultiplePoolsPerTreeAnnotation]; ok && v == MultiplePoolsPerTreeEnabled {
		return true
	}
	return false
}
