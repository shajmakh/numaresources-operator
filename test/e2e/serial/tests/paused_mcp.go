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

package tests

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metahelper "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
	"k8s.io/kubelet/config/v1beta1"
	"k8s.io/utils/ptr"

	"sigs.k8s.io/controller-runtime/pkg/client"

	machineconfigv1 "github.com/openshift/api/machineconfiguration/v1"

	nropv1 "github.com/openshift-kni/numaresources-operator/api/v1"
	"github.com/openshift-kni/numaresources-operator/api/v1/helper/nodegroup"
	nropmcp "github.com/openshift-kni/numaresources-operator/internal/machineconfigpools"
	"github.com/openshift-kni/numaresources-operator/internal/wait"
	"github.com/openshift-kni/numaresources-operator/pkg/kubeletconfig"
	"github.com/openshift-kni/numaresources-operator/pkg/objectnames"
	"github.com/openshift-kni/numaresources-operator/pkg/status"
	rteconfig "github.com/openshift-kni/numaresources-operator/rte/pkg/config"
	"github.com/openshift-kni/numaresources-operator/test/e2e/label"
	serialconfig "github.com/openshift-kni/numaresources-operator/test/e2e/serial/config"
	e2efixture "github.com/openshift-kni/numaresources-operator/test/internal/fixture"
	"github.com/openshift-kni/numaresources-operator/test/internal/objects"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	masterPoolName            = "master"
	masterMCPPoolLabelKey     = "pools.operator.machineconfiguration.openshift.io/master"
	pausedMCPE2EAnnotationKey = "numaresources.openshift-kni.io/paused-mcp-e2e"
)

var _ = FDescribe("[serial][disruptive] numaresources paused MCP support", Serial, Label("disruptive"), Label("feature:pausedmcp"), func() {
	var fxt *e2efixture.Fixture

	BeforeEach(func() {
		Expect(serialconfig.Config).ToNot(BeNil())
		Expect(serialconfig.Config.Ready()).To(BeTrue(), "NUMA fixture initialization failed")

		var err error
		fxt, err = e2efixture.Setup("e2e-test-paused-mcp", serialconfig.Config.NRTList)
		Expect(err).ToNot(HaveOccurred(), "unable to setup test fixture")
	})

	AfterEach(func() {
		err := e2efixture.Teardown(fxt)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("when MachineConfigPools are paused", func() {
		When("a new paused master pool is added and config map doesn't exist", func() {
			FIt("should create RTE DaemonSet and ConfigMap for a newly added paused master pool from the active MachineConfig", Label(label.Tier3, label.OpenShift), func(ctx context.Context) {
				nroKey := objects.NROObjectKey()
				nroObj := &nropv1.NUMAResourcesOperator{}
				Expect(fxt.Client.Get(ctx, nroKey, nroObj)).To(Succeed(), "cannot get %q in the cluster", nroKey.String())
				Expect(nroObj.Status.DaemonSets).ToNot(BeEmpty(), "NUMAResourcesOperator has no DaemonSets in status")

				initialNodeGroups := nodegroup.CloneList(nroObj.Spec.NodeGroups)
				for _, ng := range initialNodeGroups {
					if ng.PoolName != nil && *ng.PoolName == masterPoolName {
						e2efixture.Skip(fxt, "master node group already exists")
					}
				}

				// ensure there are no kubeletconfigs that targets the master MachineConfigPool for test simplification
				// we want this to be loud if there are any so we make it Fail
				kcList := &machineconfigv1.KubeletConfigList{}
				Expect(fxt.Client.List(ctx, kcList)).To(Succeed())
				for _, kc := range kcList.Items {
					found := kc.Spec.MachineConfigPoolSelector != nil && kc.Spec.MachineConfigPoolSelector.MatchLabels[masterMCPPoolLabelKey] != ""
					Expect(found).To(BeFalse(), "foundkubeletconfig %q targets the master MachineConfigPool, the test is not designed for that and it need to be adjusted", kc.Name)
				}

				masterMCP := &machineconfigv1.MachineConfigPool{}
				Expect(fxt.Client.Get(ctx, client.ObjectKey{Name: masterPoolName}, masterMCP)).To(Succeed(), "cannot get MCP %q", masterPoolName)
				Expect(masterMCP.Status.Configuration.Name).ToNot(BeEmpty(), "master MCP has empty status.configuration")

				e2efixture.By("pausing the master MachineConfigPool")
				setMCPPaused(ctx, fxt.Client, masterPoolName, true)
				defer func() {
					By("CLEANUP: unpausing the master MachineConfigPool")
					setMCPPaused(context.TODO(), fxt.Client, masterPoolName, false)
				}()

				tmScopePhase1 := "pod"
				e2efixture.By("creating a standalone KubeletConfig targeting master with TopologyManager policy set to %s", tmScopePhase1)
				kc, err := objects.TestKC(map[string]string{masterMCPPoolLabelKey: ""}, objects.WithTMScope("pod"))
				Expect(err).ToNot(HaveOccurred())
				kc.Name = "nrop-paused-master-kc"
				Expect(fxt.Client.Create(ctx, kc)).To(Succeed(), "cannot create KubeletConfig %q", kc.Name)

				defer func() {
					By("CLEANUP: deleting the standalone KubeletConfig")
					ctx := context.TODO()
					err := fxt.Client.Delete(ctx, kc)
					Expect(client.IgnoreNotFound(err)).ToNot(HaveOccurred())
					Expect(wait.With(fxt.Client).ForMCOKubeletConfigDeleted(ctx, kc.Name)).To(Succeed(), " failed while waiting for KubeletConfig %q to be deleted", kc.Name)
				}()
				e2efixture.By("adding master to NUMAResourcesOperator nodeGroups")
				newNodeGroups := nodegroup.CloneList(initialNodeGroups)
				newNodeGroups = append(newNodeGroups, nropv1.NodeGroup{PoolName: ptr.To(masterPoolName)})

				var updatedNRO nropv1.NUMAResourcesOperator
				Eventually(func(g Gomega) {
					g.Expect(fxt.Client.Get(ctx, nroKey, &updatedNRO)).To(Succeed())
					updatedNRO.Spec.NodeGroups = newNodeGroups
					g.Expect(fxt.Client.Update(ctx, &updatedNRO)).To(Succeed())
				}).WithTimeout(10*time.Minute).WithPolling(30*time.Second).Should(Succeed(), "failed to add master node group")

				defer func() {
					e2efixture.By("CLEANUP: reverting NUMAResourcesOperator nodeGroups")
					ctx := context.TODO()
					var revertNRO nropv1.NUMAResourcesOperator
					Eventually(func(g Gomega) {
						g.Expect(fxt.Client.Get(ctx, nroKey, &revertNRO)).To(Succeed())
						revertNRO.Spec.NodeGroups = initialNodeGroups
						g.Expect(fxt.Client.Update(ctx, &revertNRO)).To(Succeed())
					}).WithTimeout(10 * time.Minute).WithPolling(30 * time.Second).Should(Succeed())
				}()

				masterKey := client.ObjectKey{
					Namespace: nroObj.Status.DaemonSets[0].Namespace,
					Name:      objectnames.GetComponentName(nroObj.Name, masterPoolName),
				}
				By(fmt.Sprintf("waiting for RTE ConfigMap %q to reflect the active MachineConfig", masterKey.Name))
				prePauseExpected := getNewTopologyManagerScopeValue(tmScopePhase1)
				Eventually(func(g Gomega) {
					cm := &corev1.ConfigMap{}
					g.Expect(fxt.Client.Get(ctx, masterKey, cm)).To(Succeed())
					data, ok := cm.Data[rteconfig.Key]
					klog.Infof("found configMap %q payload: %s", masterKey.String(), data)
					g.Expect(ok).To(BeTrue(), "ConfigMap %q missing %q key", masterKey.String(), rteconfig.Key)
					g.Expect(data).To(ContainSubstring("\"topologyManagerScope\": \""+prePauseExpected+"\""),
						"ConfigMap should match active MachineConfig while MCP is paused; pending KC payload must not be applied")
				}).WithTimeout(5 * time.Minute).WithPolling(10 * time.Second).Should(Succeed())

				By(fmt.Sprintf("waiting for RTE DaemonSet %q to be created", masterKey.Name))
				// because master nodes are unschedualable we don't check for daemonSet status to be complete, it is enough to verify that is was created
				Eventually(func(g Gomega) {
					var ds appsv1.DaemonSet
					g.Expect(fxt.Client.Get(ctx, masterKey, &ds)).To(Succeed())
				}).WithTimeout(5*time.Minute).WithPolling(10*time.Second).Should(Succeed(), "DaemonSet for paused master pool was not created")

				By("verifying operator MachineConfigPoolPaused condition is True")
				// Available condition won't be true because master nodes are unschedualable which okay for this test
				Eventually(func(g Gomega) {
					g.Expect(fxt.Client.Get(ctx, nroKey, &updatedNRO)).To(Succeed())
					paused := metahelper.FindStatusCondition(updatedNRO.Status.Conditions, status.ConditionMachineConfigPoolPaused)
					g.Expect(paused).ToNot(BeNil())
					g.Expect(paused.Status).To(Equal(metav1.ConditionTrue), "expected MachineConfigPoolPaused=True, got: %+v", updatedNRO.Status.Conditions)
				}).WithTimeout(5 * time.Minute).WithPolling(10 * time.Second).Should(Succeed())
			})
		})

		When("config map exists and the matching MCP is paused", func() {
			It("should keep the ConfigMap matching the active config on paused and unpaused MCPs", Label(label.Tier3, label.OpenShift), func(ctx context.Context) {
				By("initializing dynamic client for performance profile operations")
				dynamicClient := newDynamicClient()

				nroKey := objects.NROObjectKey()
				nroObj := &nropv1.NUMAResourcesOperator{}
				Expect(fxt.Client.Get(ctx, nroKey, nroObj)).To(Succeed(), "cannot get %q in the cluster", nroKey.String())
				Expect(nroObj.Status.DaemonSets).ToNot(BeEmpty(), "NUMAResourcesOperator has no DaemonSets in status")
				Expect(nroObj.Spec.NodeGroups).ToNot(BeEmpty(), "NUMAResourcesOperator has no nodeGroups")

				mcps, err := nropmcp.GetListByNodeGroupsV1(ctx, fxt.Client, nroObj.Spec.NodeGroups)
				Expect(err).ToNot(HaveOccurred(), "cannot get MCPs associated with NUMAResourcesOperator %q", nroObj.Name)
				Expect(mcps).ToNot(BeEmpty())

				var targetedMCP *machineconfigv1.MachineConfigPool
				for _, mcp := range mcps {
					if mcp.Name == masterPoolName {
						continue
					}
					targetedMCP = mcp
					break
				}
				if targetedMCP == nil {
					targetedMCP = mcps[0]
				}
				e2efixture.By("using MachineConfigPool %q for the paused dry-update scenario", targetedMCP.Name)

				kcList := &machineconfigv1.KubeletConfigList{}
				Expect(fxt.Client.List(ctx, kcList)).To(Succeed())

				var targetedKC *machineconfigv1.KubeletConfig
				for i := range kcList.Items {
					kc := &kcList.Items[i]
					kcMcpSel, err := metav1.LabelSelectorAsSelector(kc.Spec.MachineConfigPoolSelector)
					Expect(err).ToNot(HaveOccurred())
					if kcMcpSel.Matches(labels.Set(targetedMCP.Labels)) {
						targetedKC = kc
						break
					}
				}
				Expect(targetedKC).ToNot(BeNil(), "there should be at least one kubeletconfig.machineconfiguration object matching MCP %q", targetedMCP.Name)
				if len(targetedKC.OwnerReferences) != 1 || targetedKC.OwnerReferences[0].Kind != "PerformanceProfile" {
					Skip(fmt.Sprintf("KubeletConfig %q is not owned by a PerformanceProfile (owners=%v)", targetedKC.Name, targetedKC.OwnerReferences))
				}
				ref := targetedKC.OwnerReferences[0]

				kcObj, err := kubeletconfig.MCOKubeletConfToKubeletConf(targetedKC)
				Expect(err).ToNot(HaveOccurred())
				initialTopologyManagerPolicy := kcObj.TopologyManagerPolicy
				Expect(initialTopologyManagerPolicy).ToNot(BeEmpty(), "TopologyManagerPolicy is empty on KubeletConfig %q", targetedKC.Name)

				newTopologyManagerPolicy := v1beta1.NoneTopologyManagerPolicy
				if initialTopologyManagerPolicy == v1beta1.NoneTopologyManagerPolicy {
					newTopologyManagerPolicy = v1beta1.SingleNumaNodeTopologyManagerPolicy
				}

				nroNamespace := nroObj.Status.DaemonSets[0].Namespace
				cmKey := client.ObjectKey{
					Namespace: nroNamespace,
					Name:      objectnames.GetComponentName(nroObj.Name, targetedMCP.Name),
				}
				cm := &corev1.ConfigMap{}
				Expect(fxt.Client.Get(ctx, cmKey, cm)).To(Succeed(), "cannot get RTE ConfigMap %q", cmKey.String())
				initialCMData, ok := cm.Data[rteconfig.Key]
				Expect(ok).To(BeTrue(), "ConfigMap %q missing %q key", cmKey.String(), rteconfig.Key)
				initialConf, err := rteConfigFrom(initialCMData)
				Expect(err).ToNot(HaveOccurred())
				Expect(initialConf.Kubelet.TopologyManagerPolicy).To(Equal(initialTopologyManagerPolicy),
					"RTE ConfigMap policy does not match KubeletConfig before the test")

				By(fmt.Sprintf("pausing MachineConfigPool %q", targetedMCP.Name))
				setMCPPaused(ctx, fxt.Client, targetedMCP.Name, true)
				defer func() {
					By(fmt.Sprintf("CLEANUP: unpausing MachineConfigPool %q", targetedMCP.Name))
					setMCPPaused(context.Background(), fxt.Client, targetedMCP.Name, false)
				}()

				By(fmt.Sprintf("dry-updating PerformanceProfile %q topologyPolicy to %q while MCP is paused", ref.Name, newTopologyManagerPolicy))
				updatePerformanceProfileFieldUnstructured(dynamicClient, ctx, ref.Name, newTopologyManagerPolicy, "spec", "numa", "topologyPolicy")
				defer func() {
					By(fmt.Sprintf("CLEANUP: restoring PerformanceProfile %q topologyPolicy to %q while MCP is still paused", ref.Name, initialTopologyManagerPolicy))
					updatePerformanceProfileFieldUnstructured(dynamicClient, context.Background(), ref.Name, initialTopologyManagerPolicy, "spec", "numa", "topologyPolicy")
				}()

				By("waiting for the KubeletConfig CR to reflect the PerformanceProfile change")
				Eventually(func(g Gomega) {
					g.Expect(fxt.Client.Get(ctx, client.ObjectKeyFromObject(targetedKC), targetedKC)).To(Succeed())
					updatedKcObj, err := kubeletconfig.MCOKubeletConfToKubeletConf(targetedKC)
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(updatedKcObj.TopologyManagerPolicy).To(Equal(newTopologyManagerPolicy),
						"KubeletConfig CR was not updated from PerformanceProfile")
				}).WithTimeout(5 * time.Minute).WithPolling(10 * time.Second).Should(Succeed())

				By("verifying the RTE ConfigMap keeps the pre-pause (active) TopologyManagerPolicy for about 2 minutes")
				Consistently(func(g Gomega) {
					updatedCM := &corev1.ConfigMap{}
					g.Expect(fxt.Client.Get(ctx, cmKey, updatedCM)).To(Succeed())
					data, ok := updatedCM.Data[rteconfig.Key]
					g.Expect(ok).To(BeTrue(), "ConfigMap %q missing %q key", cmKey.String(), rteconfig.Key)
					conf, err := rteConfigFrom(data)
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(conf.Kubelet.TopologyManagerPolicy).To(Equal(initialTopologyManagerPolicy),
						"ConfigMap TopologyManagerPolicy changed while MCP is paused; got %q want %q",
						conf.Kubelet.TopologyManagerPolicy, initialTopologyManagerPolicy)
					g.Expect(data).To(Equal(initialCMData), "ConfigMap payload changed while MCP is paused")
				}).WithTimeout(2 * time.Minute).WithPolling(15 * time.Second).Should(Succeed())

				By("verifying operator reports Available with MachineConfigPoolPaused")
				Eventually(func(g Gomega) {
					updatedNRO := &nropv1.NUMAResourcesOperator{}
					g.Expect(fxt.Client.Get(ctx, nroKey, updatedNRO)).To(Succeed())
					avail := metahelper.FindStatusCondition(updatedNRO.Status.Conditions, status.ConditionAvailable)
					g.Expect(avail).ToNot(BeNil())
					g.Expect(avail.Status).To(Equal(metav1.ConditionTrue), "expected Available=True, got: %+v", updatedNRO.Status.Conditions)
					paused := metahelper.FindStatusCondition(updatedNRO.Status.Conditions, status.ConditionMachineConfigPoolPaused)
					g.Expect(paused).ToNot(BeNil())
					g.Expect(paused.Status).To(Equal(metav1.ConditionTrue), "expected MachineConfigPoolPaused=True, got: %+v", updatedNRO.Status.Conditions)
				}).WithTimeout(5 * time.Minute).WithPolling(10 * time.Second).Should(Succeed())
			})
		})
	})
})

func setMCPPaused(ctx context.Context, cli client.Client, mcpName string, paused bool) {
	Eventually(func(g Gomega) {
		err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			mcp := &machineconfigv1.MachineConfigPool{}
			if err := cli.Get(ctx, client.ObjectKey{Name: mcpName}, mcp); err != nil {
				return err
			}
			if mcp.Spec.Paused == paused {
				return nil
			}
			mcp.Spec.Paused = paused
			return cli.Update(ctx, mcp)
		})
		g.Expect(err).ToNot(HaveOccurred())

		mcp := &machineconfigv1.MachineConfigPool{}
		g.Expect(cli.Get(ctx, client.ObjectKey{Name: mcpName}, mcp)).To(Succeed())
		g.Expect(mcp.Spec.Paused).To(Equal(paused))
	}).WithTimeout(2 * time.Minute).WithPolling(5 * time.Second).Should(Succeed())
}
