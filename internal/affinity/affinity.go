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
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

var ErrNoPodTemplateLabels = errors.New("no labels provided for PodAffinity")

func GetPodAntiAffinity(labels map[string]string) (*corev1.PodAntiAffinity, error) {
	if len(labels) == 0 {
		return nil, ErrNoPodTemplateLabels
	}

	return &corev1.PodAntiAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
			{
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
				TopologyKey: "kubernetes.io/hostname",
			},
		},
	}, nil
}

func deploymentStrategySafeForPodAntiAffinity() appsv1.DeploymentStrategy {
	return appsv1.DeploymentStrategy{
		Type: appsv1.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &appsv1.RollingUpdateDeployment{
			MaxUnavailable: ptr.To(intstr.FromInt(1)),
			MaxSurge:       ptr.To(intstr.FromInt(0)),
		},
	}
}

// maxSurgeAllowsExtraPods is true when the API would allow surging extra pods
// during rollout: nil MaxSurge uses the Deployment default (25%), non-zero int,
// or a percentage string other than "0%".
func maxSurgeAllowsExtraPods(maxSurge *intstr.IntOrString) bool {
	if maxSurge == nil {
		return true
	}
	switch maxSurge.Type {
	case intstr.Int:
		return maxSurge.IntVal != 0
	case intstr.String:
		s := strings.TrimSpace(maxSurge.StrVal)
		if s == "" {
			return true
		}
		return s != "0%"
	default:
		return true
	}
}

// maxUnavailableIsExplicitZero is true when MaxUnavailable is set to a value
// that means zero unavailable pods (invalid together with maxSurge 0 for
// RollingUpdate, but may appear in specs).
func maxUnavailableIsExplicitZero(maxUnavailable *intstr.IntOrString) bool {
	if maxUnavailable == nil {
		return false
	}
	switch maxUnavailable.Type {
	case intstr.Int:
		return maxUnavailable.IntVal == 0
	case intstr.String:
		s := strings.TrimSpace(maxUnavailable.StrVal)
		return s == "0" || s == "0%"
	default:
		return false
	}
}

// GetDeploymentStrategyForPodAntiAffinity checks if the provided
// strategy rollout settings cause deadlock when required podAntiAffinity is set.
// It returns a safe rolling strategy with mainly maxSurge 0 and any maxUnavailable value >0.
// If the provided strategy is safe, it is returned unchanged.
func GetDeploymentStrategyForPodAntiAffinity(strategy appsv1.DeploymentStrategy) appsv1.DeploymentStrategy {
	safe := deploymentStrategySafeForPodAntiAffinity()

	switch strategy.Type {
	case appsv1.RecreateDeploymentStrategyType:
		return strategy
	case appsv1.RollingUpdateDeploymentStrategyType:
		if strategy.RollingUpdate == nil {
			return safe
		}
		ru := strategy.RollingUpdate
		if maxUnavailableIsExplicitZero(ru.MaxUnavailable) {
			return safe
		}
		if !maxSurgeAllowsExtraPods(ru.MaxSurge) {
			return strategy
		}
		return safe
	default:
		return safe
	}
}
