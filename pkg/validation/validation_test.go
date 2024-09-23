/*
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2021 Red Hat, Inc.
 */

package validation

import (
	"strings"
	"testing"

	machineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nropv1 "github.com/openshift-kni/numaresources-operator/api/numaresourcesoperator/v1"
	nodegroupv1 "github.com/openshift-kni/numaresources-operator/api/numaresourcesoperator/v1/helper/nodegroup"

	testobjs "github.com/openshift-kni/numaresources-operator/internal/objects"
)

func TestMachineConfigPoolDuplicates(t *testing.T) {
	type testCase struct {
		name                 string
		trees                []nodegroupv1.Tree
		expectedError        bool
		expectedErrorMessage string
	}

	testCases := []testCase{
		{
			name: "duplicate MCP name",
			trees: []nodegroupv1.Tree{
				{
					MachineConfigPools: []*machineconfigv1.MachineConfigPool{
						testobjs.NewMachineConfigPool("test", nil, nil, nil),
						testobjs.NewMachineConfigPool("test", nil, nil, nil),
					},
				},
			},
			expectedError:        true,
			expectedErrorMessage: "selected by at least two node groups",
		},
		{
			name: "no duplicates",
			trees: []nodegroupv1.Tree{
				{
					MachineConfigPools: []*machineconfigv1.MachineConfigPool{
						testobjs.NewMachineConfigPool("test", nil, nil, nil),
						testobjs.NewMachineConfigPool("test1", nil, nil, nil),
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := MachineConfigPoolDuplicates(tc.trees)
			if err == nil && tc.expectedError {
				t.Errorf("expected error, succeeded")
			}
			if err != nil && !tc.expectedError {
				t.Errorf("expected success, failed")
			}
			if tc.expectedErrorMessage != "" {
				if !strings.Contains(err.Error(), tc.expectedErrorMessage) {
					t.Errorf("unexpected error: %v (expected %q)", err, tc.expectedErrorMessage)
				}
			}
		})
	}
}

func TestNodeGroupsSanity(t *testing.T) {
	var (
		valid            = "mcp-test"
		invalidWithSpace = "mcp test"
		invalidEmpty1    = "  \t  "
		invalidEmpty2    = ""
	)

	type testCase struct {
		name                 string
		nodeGroups           []nropv1.NodeGroup
		expectedError        bool
		expectedErrorMessage string
	}

	testCases := []testCase{
		{
			name: "both nil selectors",
			nodeGroups: []nropv1.NodeGroup{
				{
					MachineConfigPoolSelector: nil,
					NodeSelector:              nil,
				},
				{
					MachineConfigPoolSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test": "test",
						},
					},
				},
			},
			expectedError:        true,
			expectedErrorMessage: "one of the node groups does not specify a selector",
		},
		{
			name: "both selectors are set",
			nodeGroups: []nropv1.NodeGroup{
				{
					MachineConfigPoolSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test": "test",
						},
					},
					NodeSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test2": "test2",
						},
					},
				},
				{
					MachineConfigPoolSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test": "test",
						},
					},
				},
			},
			expectedError:        true,
			expectedErrorMessage: "only one selector is allowed to be specified under a node group",
		},
		{
			name: "with duplicates - mcp selector",
			nodeGroups: []nropv1.NodeGroup{
				{
					MachineConfigPoolSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test": "test",
						},
					},
				},
				{
					MachineConfigPoolSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test": "test",
						},
					},
				},
			},
			expectedError:        true,
			expectedErrorMessage: "has duplicates",
		},
		{
			name: "with duplicates - node selector label",
			nodeGroups: []nropv1.NodeGroup{
				{
					NodeSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test": "test",
						},
					},
				},
				{
					NodeSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test": "test",
						},
					},
				},
			},
			expectedError:        true,
			expectedErrorMessage: "has duplicates",
		},
		{
			name: "overlapping but different labels - node selector",
			nodeGroups: []nropv1.NodeGroup{
				{
					NodeSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test": "test",
						},
					},
				},
				{
					NodeSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test":  "test",
							"test1": "test1",
						},
					},
				},
			},
			expectedError: false,
		},
		{
			name: "bad MCP selector",
			nodeGroups: []nropv1.NodeGroup{
				{
					MachineConfigPoolSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "test",
								Operator: "bad-operator",
								Values:   []string{"test"},
							},
						},
					},
				},
			},

			expectedError:        true,
			expectedErrorMessage: "not a valid label selector operator",
		},
		{
			name: "correct values - multiple node group different selector each",
			nodeGroups: []nropv1.NodeGroup{
				{
					MachineConfigPoolSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test": "test",
						},
					},
				},
				{
					MachineConfigPoolSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test1": "test1",
						},
					},
				},
				{
					NodeSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test": "test", // intentionally duplicate MCPselector; this should work fine because MCP selector is a label on MCP while NodeSelector is a label on the compute node
						},
					},
				},
			},
		},
		{
			name: "invalid node group name",
			nodeGroups: []nropv1.NodeGroup{
				{
					Name: &invalidWithSpace,
					MachineConfigPoolSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test": "test",
						},
					},
				},
			},
			expectedError:        true,
			expectedErrorMessage: "node group name should not contain spaces or be empty",
		},
		{
			name: "node group name all spaces",
			nodeGroups: []nropv1.NodeGroup{
				{
					Name: &invalidEmpty1,
					MachineConfigPoolSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test": "test",
						},
					},
				},
			},
			expectedError:        true,
			expectedErrorMessage: "node group name should not contain spaces",
		},
		{
			name: "node group name as empty string",
			nodeGroups: []nropv1.NodeGroup{
				{
					Name: &invalidEmpty2,
					MachineConfigPoolSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test": "test",
						},
					},
				},
			},
			expectedError:        true,
			expectedErrorMessage: "node group name should not contain spaces or be empty",
		},
		{
			name: "valid node group name",
			nodeGroups: []nropv1.NodeGroup{
				{
					Name: &valid,
					MachineConfigPoolSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"test": "test",
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := NodeGroups(tc.nodeGroups)
			if err == nil && tc.expectedError {
				t.Errorf("expected error, succeeded")
			}
			if err != nil && !tc.expectedError {
				t.Errorf("expected success, failed")
			}
			if tc.expectedErrorMessage != "" {
				if !strings.Contains(err.Error(), tc.expectedErrorMessage) {
					t.Errorf("unexpected error: %v (expected %q)", err, tc.expectedErrorMessage)
				}
			}
		})
	}
}

func TestEqualNamespacedDSSlicesByName(t *testing.T) {
	type testCase struct {
		name          string
		s1            []nropv1.NamespacedName
		s2            []nropv1.NamespacedName
		expectedError bool
	}

	testCases := []testCase{
		{
			name:          "nil slices",
			s1:            []nropv1.NamespacedName{},
			s2:            []nropv1.NamespacedName{},
			expectedError: false,
		},
		{
			name: "equal slices by name",
			s1: []nropv1.NamespacedName{
				{
					Name: "foo",
				},
				{
					Namespace: "ns1",
					Name:      "bar",
				},
			},
			s2: []nropv1.NamespacedName{
				{
					Name: "bar",
				},
				{
					Namespace: "ns2",
					Name:      "foo",
				},
			},
			expectedError: false,
		},
		{
			name: "different slices by length",
			s1: []nropv1.NamespacedName{
				{
					Namespace: "ns1",
					Name:      "foo",
				},
				{
					Namespace: "ns1",
					Name:      "bar",
				},
			},
			s2: []nropv1.NamespacedName{
				{
					Namespace: "ns2",
					Name:      "bar",
				},
				{
					Namespace: "ns2",
					Name:      "foo",
				},
				{
					Namespace: "ns2",
					Name:      "foo",
				},
			},
			expectedError: true,
		},
		{
			name: "different slices by name",
			s1: []nropv1.NamespacedName{
				{
					Namespace: "ns1",
					Name:      "foo",
				},
				{
					Namespace: "ns1",
					Name:      "bar",
				},
			},
			s2: []nropv1.NamespacedName{
				{
					Namespace: "ns1",
					Name:      "foo",
				},
				{
					Namespace: "ns1",
					Name:      "foo",
				},
			},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := EqualNamespacedDSSlicesByName(tc.s1, tc.s2)
			if err == nil && tc.expectedError {
				t.Errorf("expected error, succeeded")
			}
			if err != nil && !tc.expectedError {
				t.Errorf("expected success, failed")
			}
		})
	}
}
