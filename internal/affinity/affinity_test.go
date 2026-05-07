/*
 * Copyright 2026 Red Hat, Inc.
 *
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
 */

package affinity

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func TestGetPodAntiAffinity(t *testing.T) {
	tests := []struct {
		name    string
		labels  map[string]string
		want    *corev1.PodAntiAffinity
		wantErr error
	}{
		{
			name:    "nil labels",
			labels:  nil,
			wantErr: ErrNoPodTemplateLabels,
		},
		{
			name:    "empty labels",
			labels:  map[string]string{},
			wantErr: ErrNoPodTemplateLabels,
		},
		{
			name: "multiple labels",
			labels: map[string]string{
				"app":                          "secondary-scheduler",
				"pod-template-hash":            "abc123",
				"openshift.io/deployment.name": "secondary-scheduler",
			},
			want: &corev1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
					{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app":                          "secondary-scheduler",
								"pod-template-hash":            "abc123",
								"openshift.io/deployment.name": "secondary-scheduler",
							},
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetPodAntiAffinity(tt.labels)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error: want %v, got %v", tt.wantErr, err)
				}
				if got != nil {
					t.Fatalf("expected nil PodAntiAffinity on error, got %#v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got == nil {
				t.Fatal("GetPodAntiAffinity returned nil PodAntiAffinity")
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("GetPodAntiAffinity mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMaxSurgeAllowsExtraPods(t *testing.T) {
	if !maxSurgeAllowsExtraPods(nil) {
		t.Fatal("nil maxSurge should allow extra pods (API default surge)")
	}
	if !maxSurgeAllowsExtraPods(ptr.To(intstr.FromInt(1))) {
		t.Fatal("non-zero int surge should allow extra pods")
	}
	if maxSurgeAllowsExtraPods(ptr.To(intstr.FromInt(0))) {
		t.Fatal("zero int surge should not allow extra pods")
	}
	if !maxSurgeAllowsExtraPods(ptr.To(intstr.FromString("25%"))) {
		t.Fatal("25% surge should allow extra pods (not treated as zero surge)")
	}
	if !maxSurgeAllowsExtraPods(ptr.To(intstr.FromString(""))) {
		t.Fatal("empty string surge should be treated like unset (API default surge)")
	}
	if maxSurgeAllowsExtraPods(ptr.To(intstr.FromString("0%"))) {
		t.Fatal("0% surge should not allow extra pods")
	}
}

func TestMaxUnavailableIsExplicitZero(t *testing.T) {
	if maxUnavailableIsExplicitZero(nil) {
		t.Fatal("nil maxUnavailable is not explicit zero")
	}
	if !maxUnavailableIsExplicitZero(ptr.To(intstr.FromInt(0))) {
		t.Fatal("int 0 should be explicit zero")
	}
	if maxUnavailableIsExplicitZero(ptr.To(intstr.FromInt(1))) {
		t.Fatal("int 1 should not be explicit zero")
	}
	if !maxUnavailableIsExplicitZero(ptr.To(intstr.FromString("0"))) {
		t.Fatal("string 0 should be explicit zero")
	}
	if !maxUnavailableIsExplicitZero(ptr.To(intstr.FromString("0%"))) {
		t.Fatal("string 0% should be explicit zero")
	}
	if !maxUnavailableIsExplicitZero(ptr.To(intstr.FromString("  0  "))) {
		t.Fatal("whitespace-padded string zero should trim to explicit zero")
	}
}

func TestMaxSurgeAllowsExtraPodsUnknownType(t *testing.T) {
	ios := intstr.IntOrString{Type: intstr.Type(2), IntVal: 0}
	if !maxSurgeAllowsExtraPods(&ios) {
		t.Fatal("unknown intstr type should be conservative (treat as surge allowed)")
	}
}

func TestMaxUnavailableIsExplicitZeroUnknownType(t *testing.T) {
	ios := intstr.IntOrString{Type: intstr.Type(2), IntVal: 0}
	if maxUnavailableIsExplicitZero(&ios) {
		t.Fatal("unknown intstr type should not be explicit zero")
	}
}

func TestGetDeploymentStrategyForPodAntiAffinity(t *testing.T) {
	safe := deploymentStrategySafeForPodAntiAffinity()

	tests := []struct {
		name string
		in   appsv1.DeploymentStrategy
		want appsv1.DeploymentStrategy
	}{
		{
			name: "empty strategy type defaults to safe rolling",
			in:   appsv1.DeploymentStrategy{},
			want: safe,
		},
		{
			name: "recreate unchanged",
			in: appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType,
			},
			want: appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType},
		},
		{
			name: "rolling nil rollingUpdate returns safe",
			in: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
			},
			want: safe,
		},
		{
			name: "rolling with no maxSurge returns safe",
			in: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: ptr.To(intstr.FromInt(2)),
				},
			},
			want: safe,
		},
		{
			name: "rolling non-zero maxSurge returns safe",
			in: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge:       ptr.To(intstr.FromInt(2)),
					MaxUnavailable: ptr.To(intstr.FromInt(3)),
				},
			},
			want: safe,
		},
		{
			name: "rolling percentage surge returns safe",
			in: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge:       ptr.To(intstr.FromString("25%")),
					MaxUnavailable: ptr.To(intstr.FromString("25%")),
				},
			},
			want: safe,
		},
		{
			name: "rolling maxUnavailable string zero returns safe",
			in: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge:       ptr.To(intstr.FromInt(1)),
					MaxUnavailable: ptr.To(intstr.FromString("0%")),
				},
			},
			want: safe,
		},
		{
			name: "rolling maxSurge zero int preserves strategy",
			in: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge:       ptr.To(intstr.FromInt(0)),
					MaxUnavailable: ptr.To(intstr.FromInt(5)),
				},
			},
			want: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge:       ptr.To(intstr.FromInt(0)),
					MaxUnavailable: ptr.To(intstr.FromInt(5)),
				},
			},
		},
		{
			name: "rolling maxSurge zero percent preserves strategy",
			in: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge:       ptr.To(intstr.FromString("0%")),
					MaxUnavailable: ptr.To(intstr.FromInt(1)),
				},
			},
			want: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge:       ptr.To(intstr.FromString("0%")),
					MaxUnavailable: ptr.To(intstr.FromInt(1)),
				},
			},
		},
		{
			name: "rolling maxSurge zero nil maxUnavailable preserves strategy",
			in: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge: ptr.To(intstr.FromInt(0)),
				},
			},
			want: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge: ptr.To(intstr.FromInt(0)),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetDeploymentStrategyForPodAntiAffinity(tt.in)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("GetDeploymentStrategyForPodAntiAffinity mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
