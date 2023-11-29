/*
 * Copyright 2022 Red Hat, Inc.
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
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	manifests "github.com/k8stopologyawareschedwg/deployer/pkg/manifests"
	k8swgobjupdate "github.com/k8stopologyawareschedwg/deployer/pkg/objectupdate"
	nrtv1alpha2 "github.com/k8stopologyawareschedwg/noderesourcetopology-api/pkg/apis/topology/v1alpha2"

	nropv1 "github.com/openshift-kni/numaresources-operator/api/numaresourcesoperator/v1"
	"github.com/openshift-kni/numaresources-operator/internal/wait"
	serialconfig "github.com/openshift-kni/numaresources-operator/test/e2e/serial/config"
	e2eclient "github.com/openshift-kni/numaresources-operator/test/utils/clients"
	e2efixture "github.com/openshift-kni/numaresources-operator/test/utils/fixture"
	e2enrt "github.com/openshift-kni/numaresources-operator/test/utils/noderesourcetopologies"
	"github.com/openshift-kni/numaresources-operator/test/utils/nrosched"
	"github.com/openshift-kni/numaresources-operator/test/utils/objects"
)

type setupPodFunc func(pod *corev1.Pod)

var _ = Describe("[serial][fundamentals][scheduler][nonreg] numaresources fundamentals non-regression", Serial, func() {
	var fxt *e2efixture.Fixture
	var nrtList nrtv1alpha2.NodeResourceTopologyList

	BeforeEach(func() {
		Expect(serialconfig.Config).ToNot(BeNil())
		Expect(serialconfig.Config.Ready()).To(BeTrue(), "NUMA fixture initialization failed")

		var err error
		fxt, err = e2efixture.Setup("e2e-test-non-regression-fundamentals", serialconfig.Config.NRTList)
		Expect(err).ToNot(HaveOccurred(), "unable to setup test fixture")

		err = fxt.Client.List(context.TODO(), &nrtList)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		err := e2efixture.Teardown(fxt)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("using the NUMA-aware scheduler without updated NRT data", func() {
		// TODO The set of tests under this context are expecting that all worker nodes of the cluster
		// be NROP-configured. this case is covered. However, if the cluster has other worker nodes (>2)
		// and those nodes are not NROP-configured then the expected behavior of these tests will change
		// and currently is missing.
		var testPod *corev1.Pod
		nropObjInitial := &nropv1.NUMAResourcesOperator{}
		nroKey := objects.NROObjectKey()

		BeforeEach(func() {
			By("Disable RTE functionality hence NRT updated data publishing")
			err := fxt.Client.Get(context.TODO(), nroKey, nropObjInitial)
			Expect(err).ToNot(HaveOccurred(), "cannot get %q from the cluster", nroKey.String())

			updatedNROPObj := nropObjInitial.DeepCopy()
			infoRefreshPauseMode := nropv1.InfoRefreshPauseEnabled
			updatedNROPObj.Spec.NodeGroups[0].Config = &nropv1.NodeGroupConfig{
				InfoRefreshPause: &infoRefreshPauseMode,
			}

			By("wait long enough to verify NROP object is updated")
			err = fxt.Client.Update(context.TODO(), updatedNROPObj)
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() nropv1.InfoRefreshPauseMode {
				nropObjCurrent := &nropv1.NUMAResourcesOperator{}
				err = fxt.Client.Get(context.TODO(), nroKey, nropObjCurrent)
				Expect(err).ToNot(HaveOccurred(), "post NROP updated: cannot get %q from the cluster", nroKey.String())
				return *nropObjCurrent.Status.MachineConfigPools[0].Config.InfoRefreshPause
			}).WithTimeout(time.Minute).WithPolling(9*time.Second).Should(Equal(infoRefreshPauseMode), "failed to update the NROP object")

			By("wait for the ds to get its args updated")
			dsKey := wait.ObjectKey{
				Namespace: nropObjInitial.Status.DaemonSets[0].Namespace,
				Name:      nropObjInitial.Status.DaemonSets[0].Name,
			}

			Eventually(func() []string {
				dsObj := appsv1.DaemonSet{}
				err = fxt.Client.Get(context.TODO(), client.ObjectKey(dsKey), &dsObj)
				Expect(err).ToNot(HaveOccurred())
				cnt := k8swgobjupdate.FindContainerByName(dsObj.Spec.Template.Spec.Containers, manifests.ContainerNameRTE)
				Expect(cnt).NotTo(BeNil(), "cannot find container data for %q",  manifests.ContainerNameRTE)
				return cnt.Args

			}).WithTimeout(time.Minute).WithPolling(9*time.Second).Should(ContainElement(ContainSubstring("--no-publish")), "ds was not updated as expected: \"--no-pubish\" arg is missing.")
			By("waiting for DaemonSet to be ready")
			_, err = wait.With(e2eclient.Client).Interval(10*time.Second).Timeout(3*time.Minute).ForDaemonSetReadyByKey(context.TODO(), dsKey)
			Expect(err).ToNot(HaveOccurred(), "failed to get the daemonset %s: %v", dsKey.String(), err)

		})

		AfterEach(func() {
			NROToRestore := &nropv1.NUMAResourcesOperator{}
			err := fxt.Client.Get(context.TODO(), nroKey, NROToRestore)
			NROToRestore.Spec = nropObjInitial.Spec
			By("wait long enough to verify the NROP object is restored")
			err = fxt.Client.Update(context.TODO(), NROToRestore)
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() nropv1.NUMAResourcesOperatorStatus {
				nropObjCurrent := &nropv1.NUMAResourcesOperator{}
				err = fxt.Client.Get(context.TODO(), nroKey, nropObjCurrent)
				Expect(err).ToNot(HaveOccurred(), "post NROP updated: cannot get %q from the cluster", nroKey.String())

				return nropObjCurrent.Status
			}).WithTimeout(time.Minute).WithPolling(9*time.Second).Should(Equal(nropObjInitial.Status), "failed to restore the NROP object")

			By("wait for the ds to get its args updated")
			dsKey := wait.ObjectKey{
				Namespace: nropObjInitial.Status.DaemonSets[0].Namespace,
				Name:      nropObjInitial.Status.DaemonSets[0].Name,
			}

			Eventually(func() []string {
				dsObj := appsv1.DaemonSet{}
				err := fxt.Client.Get(context.TODO(), client.ObjectKey(dsKey), &dsObj)
				Expect(err).ToNot(HaveOccurred())
				cnt := k8swgobjupdate.FindContainerByName(dsObj.Spec.Template.Spec.Containers,  manifests.ContainerNameRTE)
				Expect(cnt).NotTo(BeNil(), "cannot find container data for %q",  manifests.ContainerNameRTE)
				return cnt.Args
			}).WithTimeout(time.Minute).WithPolling(9*time.Second).ShouldNot(ContainElement(ContainSubstring("--no-publish")), "ds was not updated as expected: \"--no-pubish\" arg is missing.")
			By("waiting for DaemonSet to be ready")

			_, err = wait.With(e2eclient.Client).Interval(10*time.Second).Timeout(3*time.Minute).ForDaemonSetReadyByKey(context.TODO(), dsKey)
			Expect(err).ToNot(HaveOccurred(), "failed to get the daemonset %s: %v", dsKey.String(), err)

		})

		It("[tier1] should make a best-effort pod pending", func() {
			testPod = objects.NewTestPodPause(fxt.Namespace.Name, "testpod")
			testPod.Spec.SchedulerName = serialconfig.Config.SchedulerName

			By(fmt.Sprintf("creating pod %s/%s", testPod.Namespace, testPod.Name))
			err := fxt.Client.Create(context.TODO(), testPod)
			Expect(err).ToNot(HaveOccurred())

			err = wait.With(fxt.Client).Interval(10*time.Second).Steps(3).WhileInPodPhase(context.TODO(), testPod.Namespace, testPod.Name, corev1.PodPending)
			if err != nil {
				_ = objects.LogEventsForPod(fxt.K8sClient, testPod.Namespace, testPod.Name)
			}
			Expect(err).ToNot(HaveOccurred())
		})

		It("[tier1] should make a burstable pod pending", func() {
			testPod = objects.NewTestPodPause(fxt.Namespace.Name, "testpod")
			testPod.Spec.SchedulerName = serialconfig.Config.SchedulerName
			testPod.Spec.Containers[0].Resources.Requests = corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("8"),
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			}

			By(fmt.Sprintf("creating pod %s/%s", testPod.Namespace, testPod.Name))
			err := fxt.Client.Create(context.TODO(), testPod)
			Expect(err).ToNot(HaveOccurred())

			err = wait.With(fxt.Client).Interval(10*time.Second).Steps(3).WhileInPodPhase(context.TODO(), testPod.Namespace, testPod.Name, corev1.PodPending)
			if err != nil {
				_ = objects.LogEventsForPod(fxt.K8sClient, testPod.Namespace, testPod.Name)
			}
			Expect(err).ToNot(HaveOccurred())
		})

		FIt("[tier1][test_id:47611] should make a guaranteed pod pending", func() {
			//TODO check pod events has "invalid node topology data" in case all workers are

			testPod = objects.NewTestPodPause(fxt.Namespace.Name, "testpod")
			testPod.Spec.SchedulerName = serialconfig.Config.SchedulerName
			testPod.Spec.Containers[0].Resources.Limits = corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("8"),
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			}

			By(fmt.Sprintf("creating pod %s/%s", testPod.Namespace, testPod.Name))
			err := fxt.Client.Create(context.TODO(), testPod)
			Expect(err).ToNot(HaveOccurred())

			err = wait.With(fxt.Client).Interval(10*time.Second).Steps(3).WhileInPodPhase(context.TODO(), testPod.Namespace, testPod.Name, corev1.PodPending)
			if err != nil {
				_ = objects.LogEventsForPod(fxt.K8sClient, testPod.Namespace, testPod.Name)
			}
			Expect(err).ToNot(HaveOccurred())

			By("verify pod events contain the correct message \"invalid node topology data\"")
			ok, err := verifyPodEvents(fxt, testPod, "FailedScheduling", "invalid node topology data")
			Expect(err).ToNot(HaveOccurred())
			if !ok {
				_ = objects.LogEventsForPod(fxt.K8sClient, testPod.Namespace, testPod.Name)
			}
			Expect(ok).To(BeTrue(), "failed to find the expected event with Reason=\"FailedScheduling\" and Message contains: \"invalid node topology data\"")

		})
	})

	Context("using the NUMA-aware scheduler with NRT data", func() {
		var cpusPerPod int64 = 2 // must be even. Must be >= 2

		DescribeTable("[node1] against a single node",
			// the ourpose of this test is to send a burst of pods towards a node. Each pod must require resources in such a way
			// that overreservation will allow only a chunk of pod to go running, while the other will be kept pending.
			// when scheduler cache resync happens, the scheduler will send the remaining pods, and all of them must eventually
			// go running for the test to succeed.
			// calibrating the pod number and requirements was done using trial and error, there are not hard numbers yet,
			// TODO: autocalibrate the numbers considering the NUMA zone count and their capacity (assuming all NUMA zones equal)

			func(setupPod setupPodFunc) {
				nroSchedObj := &nropv1.NUMAResourcesScheduler{}
				nroSchedKey := objects.NROSchedObjectKey()
				err := fxt.Client.Get(context.TODO(), nroSchedKey, nroSchedObj)
				Expect(err).ToNot(HaveOccurred())

				if nroSchedObj.Status.CacheResyncPeriod == nil {
					e2efixture.Skip(fxt, "Scheduler cache not enabled")
				}
				timeout := nroSchedObj.Status.CacheResyncPeriod.Round(time.Second) * 10
				klog.Infof("pod running timeout: %v", timeout)

				nrts := e2enrt.FilterZoneCountEqual(nrtList.Items, 2)
				if len(nrts) < 1 {
					e2efixture.Skip(fxt, "Not enough nodes found with at least 2 NUMA zones")
				}

				nodesNames := e2enrt.AccumulateNames(nrts)
				targetNodeName, ok := e2efixture.PopNodeName(nodesNames)
				Expect(ok).To(BeTrue())

				klog.Infof("selected target node name: %q", targetNodeName)

				nrtInfo, err := e2enrt.FindFromList(nrts, targetNodeName)
				Expect(err).ToNot(HaveOccurred())

				// we still are in the serial suite, so we assume;
				// - even number of CPUs per NUMA zone
				// - unloaded node - so available == allocatable
				// - identical NUMA zones
				// - at most 1/4 of the node resources took by baseload (!!!)
				// we use cpus as unit because it's the easiest thing to consider
				maxAllocPerNUMA := e2enrt.GetMaxAllocatableResourceNumaLevel(*nrtInfo, corev1.ResourceCPU)
				maxAllocPerNUMAVal, ok := maxAllocPerNUMA.AsInt64()
				Expect(ok).To(BeTrue(), "cannot convert allocatable CPU resource as int")

				cpusVal := (3 * maxAllocPerNUMAVal) / 2
				// 150% of detected allocatable capacity per NUMA zone. Any value > allocatable per NUMA is fine.
				// CAUTION: still assuming all NUMA zones are equal across all nodes
				numPods := int(cpusVal / cpusPerPod) // unlikely we will need more than a billion pods (!!)

				klog.Infof("creating %d pods consuming %d cpus each (found %d per NUMA zone)", numPods, cpusVal, maxAllocPerNUMAVal)

				var testPods []*corev1.Pod
				for idx := 0; idx < numPods; idx++ {
					testPod := objects.NewTestPodPause(fxt.Namespace.Name, fmt.Sprintf("testpod-%d", idx))
					testPod.Spec.SchedulerName = serialconfig.Config.SchedulerName

					setupPod(testPod)

					_, err := pinPodToNode(testPod, targetNodeName)
					Expect(err).ToNot(HaveOccurred())

					By(fmt.Sprintf("creating pod %s/%s", testPod.Namespace, testPod.Name))
					err = fxt.Client.Create(context.TODO(), testPod)
					Expect(err).ToNot(HaveOccurred())

					testPods = append(testPods, testPod)
				}

				failedPods, updatedPods := wait.With(fxt.Client).Timeout(timeout).ForPodListAllRunning(context.TODO(), testPods)

				for _, failedPod := range failedPods {
					_ = objects.LogEventsForPod(fxt.K8sClient, failedPod.Namespace, failedPod.Name)
				}
				Expect(failedPods).To(BeEmpty(), "pods failed to go running: %s", accumulatePodNamespacedNames(failedPods))

				for _, updatedPod := range updatedPods {
					schedOK, err := nrosched.CheckPODWasScheduledWith(fxt.K8sClient, updatedPod.Namespace, updatedPod.Name, serialconfig.Config.SchedulerName)
					Expect(err).ToNot(HaveOccurred())
					Expect(schedOK).To(BeTrue(), "pod %s/%s not scheduled with expected scheduler %s", updatedPod.Namespace, updatedPod.Name, serialconfig.Config.SchedulerName)
				}
			},
			Entry("should handle a burst of qos=guaranteed pods [tier1]", func(pod *corev1.Pod) {
				pod.Spec.Containers[0].Resources.Limits = corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewQuantity(cpusPerPod, resource.DecimalSI),
					corev1.ResourceMemory: resource.MustParse("64Mi"),
				}
			}),
			Entry("should handle a burst of qos=burstable pods [tier2]", func(pod *corev1.Pod) {
				pod.Spec.Containers[0].Resources.Requests = corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewQuantity(cpusPerPod, resource.DecimalSI),
					corev1.ResourceMemory: resource.MustParse("64Mi"),
				}
			}),
			// this is REALLY REALLY to prevent the most catastrophic regressions
			Entry("should handle a burst of qos=best-effort pods [tier3]", func(pod *corev1.Pod) {}),
		)

		DescribeTable("[nodeAll] against all the available worker nodes",
			// like [node1] tests, but flooding all the available worker nodes - not just one.
			// note this require different constants for calibration. Again values determined by trial and error,
			// no hard rules yet.
			// TODO: autocalibrate the numbers considering the NUMA zone count and their capacity (assuming all NUMA zones equal)

			func(setupPod setupPodFunc) {
				nroSchedObj := &nropv1.NUMAResourcesScheduler{}
				nroSchedKey := objects.NROSchedObjectKey()
				err := fxt.Client.Get(context.TODO(), nroSchedKey, nroSchedObj)
				Expect(err).ToNot(HaveOccurred())

				if nroSchedObj.Status.CacheResyncPeriod == nil {
					e2efixture.Skip(fxt, "Scheduler cache not enabled")
				}
				timeout := nroSchedObj.Status.CacheResyncPeriod.Round(time.Second) * 10
				klog.Infof("pod running timeout: %v", timeout)

				nrts := e2enrt.FilterZoneCountEqual(nrtList.Items, 2)
				if len(nrts) < 1 {
					e2efixture.Skip(fxt, "Not enough nodes found with at least 2 NUMA zones")
				}

				// CAUTION here: we assume all worker node identicals, so to estimate the available
				// resources we pick one at random and we use it as reference
				nodesNames := e2enrt.AccumulateNames(nrts)
				referenceNodeName, ok := e2efixture.PopNodeName(nodesNames)
				Expect(ok).To(BeTrue())

				klog.Infof("selected reference node name: %q", referenceNodeName)

				nrtInfo, err := e2enrt.FindFromList(nrts, referenceNodeName)
				Expect(err).ToNot(HaveOccurred())

				// we still are in the serial suite, so we assume;
				// - even number of CPUs per NUMA zone
				// - unloaded node - so available == allocatable
				// - identical NUMA zones
				// - at most 1/4 of the node resources took by baseload (!!!)
				// we use cpus as unit because it's the easiest thing to consider
				resQty := e2enrt.GetMaxAllocatableResourceNumaLevel(*nrtInfo, corev1.ResourceCPU)
				resVal, ok := resQty.AsInt64()
				Expect(ok).To(BeTrue(), "cannot convert allocatable CPU resource as int")

				cpusVal := (10 * resVal) / 8
				numPods := int(int64(len(nrts)) * cpusVal / cpusPerPod) // unlikely we will need more than a billion pods (!!)

				klog.Infof("creating %d pods consuming %d cpus each (found %d per NUMA zone)", numPods, cpusVal, resVal)

				var testPods []*corev1.Pod
				for idx := 0; idx < numPods; idx++ {
					testPod := objects.NewTestPodPause(fxt.Namespace.Name, fmt.Sprintf("testpod-%d", idx))
					testPod.Spec.SchedulerName = serialconfig.Config.SchedulerName

					setupPod(testPod)

					testPod.Spec.NodeSelector = map[string]string{
						serialconfig.MultiNUMALabel: "2",
					}

					By(fmt.Sprintf("creating pod %s/%s", testPod.Namespace, testPod.Name))
					err = fxt.Client.Create(context.TODO(), testPod)
					Expect(err).ToNot(HaveOccurred())

					testPods = append(testPods, testPod)
				}

				failedPods, updatedPods := wait.With(fxt.Client).Timeout(timeout).ForPodListAllRunning(context.TODO(), testPods)

				for _, failedPod := range failedPods {
					_ = objects.LogEventsForPod(fxt.K8sClient, failedPod.Namespace, failedPod.Name)
				}
				Expect(failedPods).To(BeEmpty(), "pods failed to go running: %s", accumulatePodNamespacedNames(failedPods))

				for _, updatedPod := range updatedPods {
					schedOK, err := nrosched.CheckPODWasScheduledWith(fxt.K8sClient, updatedPod.Namespace, updatedPod.Name, serialconfig.Config.SchedulerName)
					Expect(err).ToNot(HaveOccurred())
					Expect(schedOK).To(BeTrue(), "pod %s/%s not scheduled with expected scheduler %s", updatedPod.Namespace, updatedPod.Name, serialconfig.Config.SchedulerName)
				}
			},
			Entry("should handle a burst of qos=guaranteed pods [tier1]", func(pod *corev1.Pod) {
				pod.Spec.Containers[0].Resources.Limits = corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewQuantity(cpusPerPod, resource.DecimalSI),
					corev1.ResourceMemory: resource.MustParse("64Mi"),
				}
			}),
			Entry("should handle a burst of qos=burstable pods [tier2]", func(pod *corev1.Pod) {
				pod.Spec.Containers[0].Resources.Requests = corev1.ResourceList{
					corev1.ResourceCPU:    *resource.NewQuantity(cpusPerPod, resource.DecimalSI),
					corev1.ResourceMemory: resource.MustParse("64Mi"),
				}
			}),
			// this is REALLY REALLY to prevent the most catastrophic regressions
			Entry("should handle a burst of qos=best-effort pods [tier3]", func(pod *corev1.Pod) {}),
		)

		// TODO: mixed
	})
})

func accumulatePodNamespacedNames(pods []*corev1.Pod) string {
	podNames := []string{}
	for _, pod := range pods {
		podNames = append(podNames, pod.Namespace+"/"+pod.Name)
	}
	return strings.Join(podNames, ",")
}

func verifyPodEvents(fxt *e2efixture.Fixture, pod *corev1.Pod, ereason, emsg string) (bool, error) {
	events, err := objects.GetEventsForPod(fxt.K8sClient, pod.Namespace, pod.Name)
	if err != nil {
		return false, fmt.Errorf("failed to get events for pod %s/%s; error: %v", pod.Namespace, pod.Name, err)
	}
	for _, e := range events {
		if e.Reason == "FailedScheduling" && strings.Contains(e.Message, "invalid node topology data") {
			return true, nil
		}
	}
	return false, nil
}
