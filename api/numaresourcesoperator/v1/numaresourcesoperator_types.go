/*
 * Copyright 2023 Red Hat, Inc.
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

package v1

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	mcov1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
)

// NUMAResourcesOperatorSpec defines the desired state of NUMAResourcesOperator
type NUMAResourcesOperatorSpec struct {
	// Group of Nodes to enable RTE on
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Group of nodes to enable RTE on"
	NodeGroups []NodeGroup `json:"nodeGroups,omitempty"`
	// Optional Resource Topology Exporter image URL
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Optional RTE image URL",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	ExporterImage string `json:"imageSpec,omitempty"`
	// Valid values are: "Normal", "Debug", "Trace", "TraceAll".
	// Defaults to "Normal".
	// +optional
	// +kubebuilder:default=Normal
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="RTE log verbosity"
	LogLevel operatorv1.LogLevel `json:"logLevel,omitempty"`
	// Optional Namespace/Name glob patterns of pod to ignore at node level
	// +optional
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Optional ignore pod namespace/name glob patterns"
	PodExcludes []NamespacedName `json:"podExcludes,omitempty"`
}

// +kubebuilder:validation:Enum=Disabled;Enabled;EnabledExclusiveResources
type PodsFingerprintingMode string

const (
	// PodsFingerprintingDisabled disables the pod fingerprinting reporting.
	PodsFingerprintingDisabled PodsFingerprintingMode = "Disabled"

	// PodsFingerprintingEnabled enables the pod fingerprint considering all the pods running on nodes. It is the default.
	PodsFingerprintingEnabled PodsFingerprintingMode = "Enabled"

	// PodsFingerprintingEnabledExclusiveResources enables the pod fingerprint considering only pods which have exclusive resources assigned.
	PodsFingerprintingEnabledExclusiveResources PodsFingerprintingMode = "EnabledExclusiveResources"
)

// +kubebuilder:validation:Enum=Disabled;Enabled
type InfoRefreshPauseMode string

const (
	// InfoRefreshPauseDisabled enables RTE and NRT sync
	InfoRefreshPauseDisabled InfoRefreshPauseMode = "Disabled"

	// InfoRefreshPauseEnabled pauses RTE and disables the NRT sync
	InfoRefreshPauseEnabled InfoRefreshPauseMode = "Enabled"
)

// +kubebuilder:validation:Enum=Periodic;Events;PeriodicAndEvents
type InfoRefreshMode string

const (
	// InfoRefreshPeriodic is the default. Periodically polls the state and reports it.
	InfoRefreshPeriodic InfoRefreshMode = "Periodic"

	// InfoRefreshEvents reports a new state each time a pod lifecycle event is received.
	InfoRefreshEvents InfoRefreshMode = "Events"

	// InfoRefreshPeriodicAndEvents enables both periodic and event-based reporting.
	InfoRefreshPeriodicAndEvents InfoRefreshMode = "PeriodicAndEvents"
)

// NodeGroupConfig exposes topology info reporting setting per node group
type NodeGroupConfig struct {
	// PodsFingerprinting defines if pod fingerprint should be reported for the machines belonging to this group
	// +optional
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable or disable the pods fingerprinting setting",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	PodsFingerprinting *PodsFingerprintingMode `json:"podsFingerprinting,omitempty"`
	// InfoRefreshMode sets the mechanism which will be used to refresh the topology info.
	// +optional
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Topology info mechanism setting",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	InfoRefreshMode *InfoRefreshMode `json:"infoRefreshMode,omitempty"`
	// InfoRefreshPeriod sets the topology info refresh period. Use explicit 0 to disable.
	// +optional
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Topology info refresh period setting",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	InfoRefreshPeriod *metav1.Duration `json:"infoRefreshPeriod,omitempty"`
	// InfoRefreshPause defines if updates to NRTs are paused for the machines belonging to this group
	// +optional
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable or disable the RTE pause setting",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	InfoRefreshPause *InfoRefreshPauseMode `json:"infoRefreshPause,omitempty"`
	// Tolerations overrides tolerations to be set into RTE daemonsets for this NodeGroup. If not empty, the tolerations will be the one set here.
	// Leave empty to make the system use the default tolerations.
	// +optional
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Extra tolerations for the topology updater daemonset",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

// NodeGroup defines group of nodes that will run resource topology exporter daemon set
// You can choose the group of node by MachineConfigPoolSelector or by NodeSelector
type NodeGroup struct {
	// Name is the name used to identify this node group. Has to be unique among node groups.
	// If not provided, is determined by the system.
	// +optional
	Name *string `json:"name,omitempty"`
	// NodeSelector allows to set directly the labels which nodes belonging to this nodegroup must match.
	// This offer greater flexibility than using a MachineConfigPoolSelector, You can use either NodeSelector
	// or machineConfigPoolSelector, not both at the same time.
	// +optional
	NodeSelector *metav1.LabelSelector `json:"nodeSelector,omitempty"`
	// MachineConfigPoolSelector defines label selector for the Machine Config Pool. If used, a NodeGroup will
	// match the same nodes this Machine Config Pool is matching.
	// You can use either NodeSelector or machineConfigPoolSelector, not both at the same time.
	// +optional
	MachineConfigPoolSelector *metav1.LabelSelector `json:"machineConfigPoolSelector,omitempty"`
	// Config defines the RTE behavior for this NodeGroup
	// +optional
	Config *NodeGroupConfig `json:"config,omitempty"`
}

// NodeGroupStatus reports the status of a NodeGroup once matches an actual set of nodes and it is correctly processed
// by the system. In other words, is not possible to have a NodeGroupStatus which does not represent a valid NodeGroup
// which in turn correctly references unambiguously a set of nodes in the cluster.
// Hence, if a NodeGroupStatus is published, its `Name` must be present, because it refers back to a NodeGroup whose
// config was correctly processed in the Spec. And its DaemonSet will be nonempty, because matches correctly a set
// of nodes in the cluster. The Config is best-effort always represented, possibly reflecting the system defaults.
// If the system cannot process a NodeGroup correctly from the Spec, it will report Degraded state in the top-level
// condition, and will provide details using the aforementioned conditions.
type NodeGroupStatus struct {
	// Name matches the name of a configured NodeGroup
	Name string `json:"name"`
	// DaemonSet of the configured RTEs, for this node group
	//+operator-sdk:csv:customresourcedefinitions:type=status,displayName="RTE DaemonSets"
	DaemonSets []NamespacedName `json:"daemonsets,omitempty"`
	// NodeGroupConfig represents the latest available configuration applied to this NodeGroup
	// +optional
	//+operator-sdk:csv:customresourcedefinitions:type=status,displayName="Optional configuration enforced on this NodeGroup"
	Config *NodeGroupConfig `json:"config,omitempty"`
	// Selector represents label selector for this node group that was set by either MachineConfigPoolSelector or NodeSelector
	//+operator-sdk:csv:customresourcedefinitions:type=status,displayName="Label selector of node group status"
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
}

// NUMAResourcesOperatorStatus defines the observed state of NUMAResourcesOperator
type NUMAResourcesOperatorStatus struct {
	// DaemonSets of the configured RTEs, one per node group
	// This field is not populated on HyperShift. Use NodeGroups instead.
	//+operator-sdk:csv:customresourcedefinitions:type=status,displayName="RTE DaemonSets"
	DaemonSets []NamespacedName `json:"daemonsets,omitempty"`
	// MachineConfigPools resolved from configured node groups
	// This field is not populated on HyperShift. Use NodeGroups instead.
	//+operator-sdk:csv:customresourcedefinitions:type=status,displayName="RTE MCPs from node groups"
	MachineConfigPools []MachineConfigPool `json:"machineconfigpools,omitempty"`
	// NodeGroups report the observed status of the configured NodeGroups, matching by their name
	// +optional
	//+operator-sdk:csv:customresourcedefinitions:type=status,displayName="Node groups observed status"
	NodeGroups []NodeGroupStatus `json:"nodeGroups,omitempty"`
	// Conditions show the current state of the NUMAResourcesOperator Operator
	//+operator-sdk:csv:customresourcedefinitions:type=status,displayName="Condition reported"
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// RelatedObjects list of objects of interest for this operator
	//+operator-sdk:csv:customresourcedefinitions:type=status,displayName="Related Objects"
	RelatedObjects []configv1.ObjectReference `json:"relatedObjects,omitempty"`
}

// MachineConfigPool defines the observed state of each MachineConfigPool selected by node groups
type MachineConfigPool struct {
	// Name the name of the machine config pool
	Name string `json:"name"`
	// Conditions represents the latest available observations of MachineConfigPool current state.
	// +optional
	//+operator-sdk:csv:customresourcedefinitions:type=status,displayName="Optional conditions reported for this NodeGroup"
	Conditions []mcov1.MachineConfigPoolCondition `json:"conditions,omitempty"`
	// NodeGroupConfig represents the latest available configuration applied to this MachineConfigPool
	// +optional
	//+operator-sdk:csv:customresourcedefinitions:type=status,displayName="Optional configuration enforced on this NodeGroup"
	Config *NodeGroupConfig `json:"config,omitempty"`
	// NodeGroupName the name of the node group this MCP belongs to
	// +optional
	NodeGroupName string `json:"nodeGroupName,omitempty"`
}

//+genclient
//+genclient:nonNamespaced
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=numaresop,path=numaresourcesoperators,scope=Cluster
//+kubebuilder:storageversion

// NUMAResourcesOperator is the Schema for the numaresourcesoperators API
// +operator-sdk:csv:customresourcedefinitions:displayName="NUMA Resources Operator",resources={{DaemonSet,v1,rte-daemonset,ConfigMap,v1,rte-configmap}}
type NUMAResourcesOperator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NUMAResourcesOperatorSpec   `json:"spec,omitempty"`
	Status NUMAResourcesOperatorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NUMAResourcesOperatorList contains a list of NUMAResourcesOperator
type NUMAResourcesOperatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NUMAResourcesOperator `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NUMAResourcesOperator{}, &NUMAResourcesOperatorList{})
}

func (ngc *NodeGroupConfig) ToString() string {
	if ngc != nil {
		ngc.SetDefaults()
		return fmt.Sprintf("PodsFingerprinting mode: %s InfoRefreshMode: %s InfoRefreshPeriod: %s InfoRefreshPause: %s", *ngc.PodsFingerprinting, *ngc.InfoRefreshMode, *ngc.InfoRefreshPeriod, *ngc.InfoRefreshPause)
	}
	return ""
}
