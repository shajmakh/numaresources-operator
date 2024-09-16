//go:build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1

import (
	configv1 "github.com/openshift/api/config/v1"
	machineconfiguration_openshift_iov1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineConfigPool) DeepCopyInto(out *MachineConfigPool) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]machineconfiguration_openshift_iov1.MachineConfigPoolCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = new(NodeGroupConfig)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineConfigPool.
func (in *MachineConfigPool) DeepCopy() *MachineConfigPool {
	if in == nil {
		return nil
	}
	out := new(MachineConfigPool)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NUMAResourcesOperator) DeepCopyInto(out *NUMAResourcesOperator) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NUMAResourcesOperator.
func (in *NUMAResourcesOperator) DeepCopy() *NUMAResourcesOperator {
	if in == nil {
		return nil
	}
	out := new(NUMAResourcesOperator)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NUMAResourcesOperator) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NUMAResourcesOperatorList) DeepCopyInto(out *NUMAResourcesOperatorList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]NUMAResourcesOperator, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NUMAResourcesOperatorList.
func (in *NUMAResourcesOperatorList) DeepCopy() *NUMAResourcesOperatorList {
	if in == nil {
		return nil
	}
	out := new(NUMAResourcesOperatorList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NUMAResourcesOperatorList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NUMAResourcesOperatorSpec) DeepCopyInto(out *NUMAResourcesOperatorSpec) {
	*out = *in
	if in.NodeGroups != nil {
		in, out := &in.NodeGroups, &out.NodeGroups
		*out = make([]NodeGroup, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.PodExcludes != nil {
		in, out := &in.PodExcludes, &out.PodExcludes
		*out = make([]NamespacedName, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NUMAResourcesOperatorSpec.
func (in *NUMAResourcesOperatorSpec) DeepCopy() *NUMAResourcesOperatorSpec {
	if in == nil {
		return nil
	}
	out := new(NUMAResourcesOperatorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NUMAResourcesOperatorStatus) DeepCopyInto(out *NUMAResourcesOperatorStatus) {
	*out = *in
	if in.DaemonSets != nil {
		in, out := &in.DaemonSets, &out.DaemonSets
		*out = make([]NamespacedName, len(*in))
		copy(*out, *in)
	}
	if in.MachineConfigPools != nil {
		in, out := &in.MachineConfigPools, &out.MachineConfigPools
		*out = make([]MachineConfigPool, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.RelatedObjects != nil {
		in, out := &in.RelatedObjects, &out.RelatedObjects
		*out = make([]configv1.ObjectReference, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NUMAResourcesOperatorStatus.
func (in *NUMAResourcesOperatorStatus) DeepCopy() *NUMAResourcesOperatorStatus {
	if in == nil {
		return nil
	}
	out := new(NUMAResourcesOperatorStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NUMAResourcesScheduler) DeepCopyInto(out *NUMAResourcesScheduler) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NUMAResourcesScheduler.
func (in *NUMAResourcesScheduler) DeepCopy() *NUMAResourcesScheduler {
	if in == nil {
		return nil
	}
	out := new(NUMAResourcesScheduler)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NUMAResourcesScheduler) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NUMAResourcesSchedulerList) DeepCopyInto(out *NUMAResourcesSchedulerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]NUMAResourcesScheduler, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NUMAResourcesSchedulerList.
func (in *NUMAResourcesSchedulerList) DeepCopy() *NUMAResourcesSchedulerList {
	if in == nil {
		return nil
	}
	out := new(NUMAResourcesSchedulerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NUMAResourcesSchedulerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NUMAResourcesSchedulerSpec) DeepCopyInto(out *NUMAResourcesSchedulerSpec) {
	*out = *in
	if in.CacheResyncPeriod != nil {
		in, out := &in.CacheResyncPeriod, &out.CacheResyncPeriod
		*out = new(metav1.Duration)
		**out = **in
	}
	if in.CacheResyncDebug != nil {
		in, out := &in.CacheResyncDebug, &out.CacheResyncDebug
		*out = new(CacheResyncDebugMode)
		**out = **in
	}
	if in.SchedulerInformer != nil {
		in, out := &in.SchedulerInformer, &out.SchedulerInformer
		*out = new(SchedulerInformerMode)
		**out = **in
	}
	if in.CacheResyncDetection != nil {
		in, out := &in.CacheResyncDetection, &out.CacheResyncDetection
		*out = new(CacheResyncDetectionMode)
		**out = **in
	}
	if in.ScoringStrategy != nil {
		in, out := &in.ScoringStrategy, &out.ScoringStrategy
		*out = new(ScoringStrategyParams)
		(*in).DeepCopyInto(*out)
	}
	if in.Replicas != nil {
		in, out := &in.Replicas, &out.Replicas
		*out = new(int32)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NUMAResourcesSchedulerSpec.
func (in *NUMAResourcesSchedulerSpec) DeepCopy() *NUMAResourcesSchedulerSpec {
	if in == nil {
		return nil
	}
	out := new(NUMAResourcesSchedulerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NUMAResourcesSchedulerStatus) DeepCopyInto(out *NUMAResourcesSchedulerStatus) {
	*out = *in
	out.Deployment = in.Deployment
	if in.CacheResyncPeriod != nil {
		in, out := &in.CacheResyncPeriod, &out.CacheResyncPeriod
		*out = new(metav1.Duration)
		**out = **in
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]metav1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.RelatedObjects != nil {
		in, out := &in.RelatedObjects, &out.RelatedObjects
		*out = make([]configv1.ObjectReference, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NUMAResourcesSchedulerStatus.
func (in *NUMAResourcesSchedulerStatus) DeepCopy() *NUMAResourcesSchedulerStatus {
	if in == nil {
		return nil
	}
	out := new(NUMAResourcesSchedulerStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NamespacedName) DeepCopyInto(out *NamespacedName) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NamespacedName.
func (in *NamespacedName) DeepCopy() *NamespacedName {
	if in == nil {
		return nil
	}
	out := new(NamespacedName)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeGroup) DeepCopyInto(out *NodeGroup) {
	*out = *in
	if in.MachineConfigPoolSelector != nil {
		in, out := &in.MachineConfigPoolSelector, &out.MachineConfigPoolSelector
		*out = new(metav1.LabelSelector)
		(*in).DeepCopyInto(*out)
	}
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = new(NodeSelector)
		(*in).DeepCopyInto(*out)
	}
	if in.Config != nil {
		in, out := &in.Config, &out.Config
		*out = new(NodeGroupConfig)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeGroup.
func (in *NodeGroup) DeepCopy() *NodeGroup {
	if in == nil {
		return nil
	}
	out := new(NodeGroup)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeGroupConfig) DeepCopyInto(out *NodeGroupConfig) {
	*out = *in
	if in.PodsFingerprinting != nil {
		in, out := &in.PodsFingerprinting, &out.PodsFingerprinting
		*out = new(PodsFingerprintingMode)
		**out = **in
	}
	if in.InfoRefreshMode != nil {
		in, out := &in.InfoRefreshMode, &out.InfoRefreshMode
		*out = new(InfoRefreshMode)
		**out = **in
	}
	if in.InfoRefreshPeriod != nil {
		in, out := &in.InfoRefreshPeriod, &out.InfoRefreshPeriod
		*out = new(metav1.Duration)
		**out = **in
	}
	if in.InfoRefreshPause != nil {
		in, out := &in.InfoRefreshPause, &out.InfoRefreshPause
		*out = new(InfoRefreshPauseMode)
		**out = **in
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]corev1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeGroupConfig.
func (in *NodeGroupConfig) DeepCopy() *NodeGroupConfig {
	if in == nil {
		return nil
	}
	out := new(NodeGroupConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeSelector) DeepCopyInto(out *NodeSelector) {
	*out = *in
	if in.LabelSelector != nil {
		in, out := &in.LabelSelector, &out.LabelSelector
		*out = new(metav1.LabelSelector)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeSelector.
func (in *NodeSelector) DeepCopy() *NodeSelector {
	if in == nil {
		return nil
	}
	out := new(NodeSelector)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceSpecParams) DeepCopyInto(out *ResourceSpecParams) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceSpecParams.
func (in *ResourceSpecParams) DeepCopy() *ResourceSpecParams {
	if in == nil {
		return nil
	}
	out := new(ResourceSpecParams)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ScoringStrategyParams) DeepCopyInto(out *ScoringStrategyParams) {
	*out = *in
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = make([]ResourceSpecParams, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ScoringStrategyParams.
func (in *ScoringStrategyParams) DeepCopy() *ScoringStrategyParams {
	if in == nil {
		return nil
	}
	out := new(ScoringStrategyParams)
	in.DeepCopyInto(out)
	return out
}
