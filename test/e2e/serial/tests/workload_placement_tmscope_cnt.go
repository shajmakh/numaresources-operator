package tests

import (
	"context"
	"fmt"
	"regexp"
	"time"

	nrtv1alpha2 "github.com/k8stopologyawareschedwg/noderesourcetopology-api/pkg/apis/topology/v1alpha2"
	nropv1 "github.com/openshift-kni/numaresources-operator/api/v1"
	nodegroupv1 "github.com/openshift-kni/numaresources-operator/api/v1/helper/nodegroup"
	intnrt "github.com/openshift-kni/numaresources-operator/internal/noderesourcetopology"
	e2ereslist "github.com/openshift-kni/numaresources-operator/internal/resourcelist"
	"github.com/openshift-kni/numaresources-operator/internal/wait"
	"github.com/openshift-kni/numaresources-operator/pkg/objectnames"
	"github.com/openshift-kni/numaresources-operator/test/e2e/label"
	serialconfig "github.com/openshift-kni/numaresources-operator/test/e2e/serial/config"
	"github.com/openshift-kni/numaresources-operator/test/internal/configuration"
	e2efixture "github.com/openshift-kni/numaresources-operator/test/internal/fixture"
	e2enrt "github.com/openshift-kni/numaresources-operator/test/internal/noderesourcetopologies"
	"github.com/openshift-kni/numaresources-operator/test/internal/nrosched"
	"github.com/openshift-kni/numaresources-operator/test/internal/objects"
	machineconfigv1 "github.com/openshift/api/machineconfiguration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("[serial] numaresources workload placement considering TM scope count", Serial, Label("disruptive", "scheduler"), Label("feature:wlplacement", "feature:tmscope_cnt"), func() {
	var fxt *e2efixture.Fixture
	var nrtList nrtv1alpha2.NodeResourceTopologyList
	var nrts []nrtv1alpha2.NodeResourceTopology

	BeforeEach(func() {
		Expect(serialconfig.Config).ToNot(BeNil())
		Expect(serialconfig.Config.Ready()).To(BeTrue(), "NUMA fixture initialization failed")

		var err error
		fxt, err = e2efixture.Setup("e2e-test-workload-placement-tmpol", serialconfig.Config.NRTList)
		Expect(err).ToNot(HaveOccurred(), "unable to setup test fixture")

		Expect(fxt.Client.List(context.TODO(), &nrtList)).To(Succeed())

		// Note that this test, being part of "serial", expects NO OTHER POD being scheduled
		// in between, so we consider this information current and valid when the It()s run.

		tmScope, err := getTopologyManagerScope(context.TODO(), fxt.Client)
		Expect(err).ToNot(HaveOccurred(), "failed to get topology manager scope")
		Expect(tmScope).ToNot(BeEmpty(), "topology manager scope not found")

		nrts = e2enrt.FilterByTopologyManagerPolicyAndScope(nrtList.Items, intnrt.SingleNUMANode, tmScope)
		Expect(nrts).ToNot(BeEmpty(), "no nodes with topology manager policy %q and scope %q found", intnrt.SingleNUMANode, tmScope)
	})

	AfterEach(func() {
		Expect(e2efixture.Teardown(fxt)).To(Succeed())
	})

	DescribeTable(
		"[placement][unsched] cluster with one worker nodes suitable", Label("placement", "unsched"), Label("feature:unsched"),
		func(policyFuncs tmScopeFuncs, errMsg string, podRes podResourcesRequest, unsuitableFreeRes, targetFreeResPerNUMA []corev1.ResourceList) {
			hostsRequired := 2
			if len(nrts) < hostsRequired {
				e2efixture.Skipf(fxt, "not enough nodes with policy %q - found %d", policyFuncs.policyName(), len(nrts))
			}

			Expect(unsuitableFreeRes).To(HaveLen(hostsRequired), "mismatch unsuitable resource declarations expected %d items, but found %d", hostsRequired, len(unsuitableFreeRes))

			for _, nrt := range nrts {
				for _, zone := range nrt.Zones {
					avail := e2enrt.AvailableFromZone(zone)
					if !isHugePageInAvailable(avail) && isHugepageNeeded(podRes) {
						e2efixture.Skipf(fxt, "hugepages requested but not found under node: %q", nrt.Name)
					}
				}
			}

			pod := objects.NewTestPodPause(fxt.Namespace.Name, "testpod")
			pod.Spec.SchedulerName = serialconfig.Config.SchedulerName
			pod.Spec.NodeSelector = map[string]string{
				serialconfig.MultiNUMALabel: "2",
			}
			pod.Spec.Containers[0].Name = "testcnt-0"
			pod.Spec.Containers[0].Resources.Limits = podRes.appCnt[0]
			for i := 1; i < len(podRes.appCnt); i++ {
				pod.Spec.Containers = append(pod.Spec.Containers, pod.Spec.Containers[0])
				pod.Spec.Containers[i].Name = fmt.Sprintf("testcnt-%d", i)
				pod.Spec.Containers[i].Resources.Limits = podRes.appCnt[i]
			}

			// TODO remove completely
			// // we expect init containers to be required less often than app containers, so we delegate that
			// makeInitTestContainers(pod, podRes.initCnt)

			requiredRes := e2ereslist.FromGuaranteedPod(*pod)

			numaZonesRequired := 2

			By(fmt.Sprintf("filtering available nodes with at least %d NUMA zones", numaZonesRequired))
			nrtCandidates := e2enrt.FilterZoneCountEqual(nrts, numaZonesRequired)
			if len(nrtCandidates) < hostsRequired {
				e2efixture.Skipf(fxt, "not enough nodes with %d NUMA Zones: found %d", numaZonesRequired, len(nrtCandidates))
			}
			By("filtering available nodes with allocatable resources on each NUMA zone that can match request")
			nrtCandidates = e2enrt.FilterAnyZoneMatchingResources(nrtCandidates, requiredRes)
			if len(nrtCandidates) < hostsRequired {
				e2efixture.Skipf(fxt, "not enough nodes with NUMA zones each of them can match requests: found %d", len(nrtCandidates))
			}

			candidateNodeNames := e2enrt.AccumulateNames(nrtCandidates)
			// nodes we have now are all equal for our purposes. Pick one at random
			targetNodeName, ok := e2efixture.PopNodeName(candidateNodeNames)
			Expect(ok).To(BeTrue(), "cannot select a target node among %#v", e2efixture.ListNodeNames(candidateNodeNames))
			unsuitableNodeNames := e2efixture.ListNodeNames(candidateNodeNames)

			By(fmt.Sprintf("selecting target node %q and unsuitable nodes %#v (random pick)", targetNodeName, unsuitableNodeNames))

			// make targetFreeResPerNUMA the complement of the test pod's resources
			// IOW targetFreeResPerNUMA + baseload + podResourcesRequest equals to all node's allocatable resources
			if len(targetFreeResPerNUMA) == 0 {
				for i := 0; i < len(podRes.appCnt); i++ {
					// appending a copy so mutating one object won't implicitly change the other
					targetFreeResPerNUMA = append(targetFreeResPerNUMA, podRes.appCnt[i].DeepCopy())
				}
			}
			padInfo := paddingInfo{
				pod:                  pod,
				targetNodeName:       targetNodeName,
				targetFreeResPerNUMA: targetFreeResPerNUMA,
				unsuitableNodeNames:  unsuitableNodeNames,
				unsuitableFreeRes:    unsuitableFreeRes,
			}

			By("Padding nodes to create the test workload scenario")
			paddingPods := setupPadding(fxt, nrtList, padInfo)

			By("Waiting for padding pods to be ready")
			failedPodIds := e2efixture.WaitForPaddingPodsRunning(context.Background(), fxt, paddingPods)
			Expect(failedPodIds).To(BeEmpty(), "some padding pods have failed to run")

			By("waiting for the NRT data to settle")
			e2efixture.MustSettleNRT(fxt)

			for _, unsuitableNodeName := range unsuitableNodeNames {
				dumpNRTForNode(fxt.Client, unsuitableNodeName, "unsuitable")
			}
			dumpNRTForNode(fxt.Client, targetNodeName, "target")

			By("running the test pod")
			klog.Info(objects.DumpPODResourceRequirements(pod))
			err := fxt.Client.Create(context.TODO(), pod)
			Expect(err).ToNot(HaveOccurred())

			By("verify the pod keep on pending")
			_, err = wait.With(fxt.Client).Interval(10*time.Second).Steps(5).WhileInPodPhase(context.TODO(), pod.Namespace, pod.Name, corev1.PodPending)
			if err != nil {
				_ = objects.LogEventsForPod(fxt.K8sClient, pod.Namespace, pod.Name)
				dumpNRTForNode(fxt.Client, targetNodeName, "target")
			}
			Expect(err).ToNot(HaveOccurred())

			By("checking the scheduler report the expected error in the pod events`")
			loggedEvents := false
			Eventually(func() bool {
				events, err := objects.GetEventsForPod(fxt.K8sClient, pod.Namespace, pod.Name)
				if err != nil {
					klog.ErrorS(err, "failed to get events for pod", "namespace", pod.Namespace, "name", pod.Name)
				}
				for _, e := range events {
					ok, err := regexp.MatchString(errMsg, e.Message)
					if err != nil {
						klog.ErrorS(err, "bad message regex", "pattern", errMsg, "eventMessage", e.Message)
					}
					if e.Reason == "FailedScheduling" && ok {
						return true
					}
				}
				klog.InfoS("failed to find the expected event with Reason=\"FailedScheduling\" and Message contains", "expected", errMsg)
				if !loggedEvents {
					_ = objects.LogEventsForPod(fxt.K8sClient, pod.Namespace, pod.Name)
					loggedEvents = true
				}
				return false
			}).WithTimeout(2*time.Minute).WithPolling(10*time.Second).Should(BeTrue(), "pod %s/%s doesn't contains the expected event error", pod.Namespace, pod.Name)

			By("deleting the test pod")
			err = fxt.Client.Delete(context.TODO(), pod)
			Expect(err).ToNot(HaveOccurred())

			By("checking the test pod is removed")
			err = wait.With(fxt.Client).Timeout(3*time.Minute).ForPodDeleted(context.TODO(), pod.Namespace, pod.Name)
			Expect(err).ToNot(HaveOccurred())

			// we don't need to wait for NRT update since we already checked it hasn't changed in prior step
		},

		// below tests try to schedule a multi-container pod, when having only one worker node with available resources (target node) for the total pod's containers,
		// but only one container can be aligned to a single numa node while the second container cannot. Because of that, the pod should keep on pending and we expect
		// to see the reason for not scheduling the pod on that target node as "cannot align container: testcnt-1", because the other worker nodes have insufficient
		// free resources to accommodate the pod thus they will be rejected as candidates at earlier stage
		Entry("[test_id:85007] pod with two gu cnt keep on pending because cannot align the second container to a single numa node",
			Label(label.Tier0, "unsched", "tmscope:cnt", "cpu"),
			newContainerScopeSingleNUMANodeFuncs(),
			nrosched.ErrorCannotAlignContainer,
			podResourcesRequest{
				appCnt: []corev1.ResourceList{
					{
						corev1.ResourceCPU:    resource.MustParse("5"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
						"hugepages-2Mi":       resource.MustParse("32Mi"),
						"hugepages-1Gi":       resource.MustParse("1Gi"),
					},
					{
						corev1.ResourceCPU:    resource.MustParse("5"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
						"hugepages-2Mi":       resource.MustParse("32Mi"),
						"hugepages-1Gi":       resource.MustParse("1Gi"),
					},
				},
			},
			// we need keep the gap between Node level fit and NUMA level fit wide enough.
			// for example if only 2 cpus are separating unsuitable node from becoming suitable,
			// it's not good because the baseload should be added as well (which is around 2 cpus)
			// and then the pod might land on the unsuitable node.
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
					"hugepages-2Mi":       resource.MustParse("32Mi"),
					"hugepages-1Gi":       resource.MustParse("1Gi"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
					"hugepages-2Mi":       resource.MustParse("32Mi"),
					"hugepages-1Gi":       resource.MustParse("1Gi"),
				},
			},
			// the free resources that should be left on the target node should not depend that there will be some baseload added upon padding the node,
			// those free resources should match the pod requests in total. The reason behind that is that Noderesourcesfit plugin (the plugin that is
			// responsible for accepting/rejecting compute nodes as candidates for placing the pod) actually accounts for the baseload, it compares the
			// actual available resources on node with the pod requested resources, if the available resources can accommodate the pod resources then it
			// will mark the node as a possible candidate, if not it will reject it.
			[]corev1.ResourceList{
				{
					// the baseload will be added to the first numa zone upon padding, this need to consider
					// that baseCpus + targetNodeFreeCpus does not make the first numa a candidate for any of the containers. Take into account that the baseCpus can be at least 2 cpus
					//so for example if cpus(cont1) = 5 and cpus(cont2) = 5 then cpus(numa0)<5 and since the basecpus usually is 2 then we should make pass at most 2 free cpus as the free cpus in numa0
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
					"hugepages-2Mi":       resource.MustParse("32Mi"),
					"hugepages-1Gi":       resource.MustParse("1Gi"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("9"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
					"hugepages-2Mi":       resource.MustParse("32Mi"),
					"hugepages-1Gi":       resource.MustParse("1Gi"),
				},
			},
		),
		Entry("[test_id:74256] guaranteed pod with multi cnt with fractional cpus keep on pending because cannot align the second container to a single numa node",
			Label(label.Tier3, "unsched", "tmscope:pod", "cpu"),
			newPodScopeSingleNUMANodeFuncs(),
			nrosched.ErrorCannotAlignPod,
			podResourcesRequest{
				appCnt: []corev1.ResourceList{
					{
						corev1.ResourceCPU:    resource.MustParse("4300m"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
						"hugepages-2Mi":       resource.MustParse("32Mi"),
					},
					{
						corev1.ResourceCPU:    resource.MustParse("7500m"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
						"hugepages-2Mi":       resource.MustParse("32Mi"),
					},
					{
						corev1.ResourceCPU:    resource.MustParse("2500m"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
					},
				},
			},
			// available resources on non-target nodes
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU: resource.MustParse("1"),
				},
				{
					corev1.ResourceCPU: resource.MustParse("1"),
				},
			},
			// available resources on target node
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU:    resource.MustParse("11"),
					corev1.ResourceMemory: resource.MustParse("12Gi"),
					"hugepages-2Mi":       resource.MustParse("128Mi"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("6"),
					corev1.ResourceMemory: resource.MustParse("12Gi"),
					"hugepages-2Mi":       resource.MustParse("128Mi"),
				},
			},
		),
		Entry("[test_id:74257] burstable pod with multi cnt with fractional cpus keep on pending because of not enough free cpus",
			Label(label.Tier3, "unsched", "tmscope:pod", "cpu"),
			Label("feature:nonreg"),
			newPodScopeSingleNUMANodeFuncs(),
			"0.* nodes are available: [0-9]* Insufficient cpu",
			podResourcesRequest{
				appCnt: []corev1.ResourceList{
					{
						corev1.ResourceCPU: resource.MustParse("4300m"),
						"hugepages-2Mi":    resource.MustParse("32Mi"),
					},
					{
						corev1.ResourceCPU: resource.MustParse("10"),
					},
					{
						corev1.ResourceCPU:    resource.MustParse("3500m"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
					},
				},
			},
			// available resources on non-target nodes
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU: resource.MustParse("1"),
				},
				{
					corev1.ResourceCPU: resource.MustParse("1"),
				},
			},
			// available resources on target node
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU:    resource.MustParse("8"),
					corev1.ResourceMemory: resource.MustParse("12Gi"),
					"hugepages-2Mi":       resource.MustParse("128Mi"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("7"),
					corev1.ResourceMemory: resource.MustParse("12Gi"),
					"hugepages-2Mi":       resource.MustParse("128Mi"),
				},
			},
		),
		Entry("[test_id:85008] pod with two gu cnt keep on pending because cannot align the second container to a single numa node",
			Label(label.Tier0, "unsched", "tmscope:cnt", "memory"),
			newContainerScopeSingleNUMANodeFuncs(),
			nrosched.ErrorCannotAlignContainer,
			podResourcesRequest{
				appCnt: []corev1.ResourceList{
					{
						corev1.ResourceCPU:    resource.MustParse("4"),
						corev1.ResourceMemory: resource.MustParse("7Gi"),
						"hugepages-2Mi":       resource.MustParse("32Mi"),
						"hugepages-1Gi":       resource.MustParse("1Gi"),
					},
					{
						corev1.ResourceCPU:    resource.MustParse("4"),
						corev1.ResourceMemory: resource.MustParse("7Gi"),
						"hugepages-2Mi":       resource.MustParse("32Mi"),
						"hugepages-1Gi":       resource.MustParse("1Gi"),
					},
				},
			},
			// we need keep the gap between Node level fit and NUMA level fit wide enough.
			// for example if only 2 cpus are separating unsuitable node from becoming suitable,
			// it's not good because the baseload should be added as well (which is around 2 cpus)
			// and then the pod might land on the unsuitable node.
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
					"hugepages-2Mi":       resource.MustParse("32Mi"),
					"hugepages-1Gi":       resource.MustParse("1Gi"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("7Gi"),
					"hugepages-2Mi":       resource.MustParse("32Mi"),
					"hugepages-1Gi":       resource.MustParse("1Gi"),
				},
			},
			// the free resources that should be left on the target node should not depend that there will be some baseload added upon padding the node,
			// those free resources should match the pod requests in total. The reason behind that is that Noderesourcesfit plugin (the plugin that is
			// responsible for accepting/rejecting compute nodes as candidates for placing the pod) actually accounts for the baseload, it compares the
			// actual available resources on node with the pod requested resources, if the available resources can accommodate the pod resources then it
			// will mark the node as a possible candidate, if not it will reject it.
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU: resource.MustParse("4"),
					//the base memory on the node could be 4.5Gi, so we need to consider that 4.5Gi + 1Gi is not enough for any of the pod containers
					corev1.ResourceMemory: resource.MustParse("1Gi"),
					"hugepages-2Mi":       resource.MustParse("32Mi"),
					"hugepages-1Gi":       resource.MustParse("1Gi"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("13Gi"),
					"hugepages-2Mi":       resource.MustParse("32Mi"),
					"hugepages-1Gi":       resource.MustParse("1Gi"),
				},
			},
		),
		Entry("[test_id:85009] pod with two gu cnt keep on pending because cannot align the second container to a single numa node",
			Label(label.Tier0, "unsched", "tmscope:cnt", "hugepages2Mi"),
			newContainerScopeSingleNUMANodeFuncs(),
			nrosched.ErrorCannotAlignContainer,
			podResourcesRequest{
				appCnt: []corev1.ResourceList{
					{
						corev1.ResourceCPU:    resource.MustParse("4"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
						"hugepages-2Mi":       resource.MustParse("16Mi"),
						"hugepages-1Gi":       resource.MustParse("1Gi"),
					},
					{
						corev1.ResourceCPU:    resource.MustParse("4"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
						"hugepages-2Mi":       resource.MustParse("48Mi"),
						"hugepages-1Gi":       resource.MustParse("1Gi"),
					},
				},
			},
			// we need keep the gap between Node level fit and NUMA level fit wide enough.
			// for example if only 2 cpus are separating unsuitable node from becoming suitable,
			// it's not good because the baseload should be added as well (which is around 2 cpus)
			// and then the pod might land on the unsuitable node.
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
					"hugepages-2Mi":       resource.MustParse("32Mi"),
					"hugepages-1Gi":       resource.MustParse("1Gi"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
					"hugepages-2Mi":       resource.MustParse("32Mi"),
					"hugepages-1Gi":       resource.MustParse("1Gi"),
				},
			},
			// the free resources that should be left on the target node should not depend that there will be some baseload added upon padding the node,
			// those free resources should match the pod requests in total. The reason behind that is that Noderesourcesfit plugin (the plugin that is
			// responsible for accepting/rejecting compute nodes as candidates for placing the pod) actually accounts for the baseload, it compares the
			// actual available resources on node with the pod requested resources, if the available resources can accommodate the pod resources then it
			// will mark the node as a possible candidate, if not it will reject it.
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
					"hugepages-2Mi":       resource.MustParse("32Mi"),
					"hugepages-1Gi":       resource.MustParse("1Gi"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
					"hugepages-2Mi":       resource.MustParse("32Mi"),
					"hugepages-1Gi":       resource.MustParse("1Gi"),
				},
			},
		),
		Entry("[test_id:85011] pod with two gu cnt keep on pending because cannot align the second container to a single numa node",
			Label(label.Tier0, "unsched", "tmscope:cnt", "hugepages1Gi"),
			newContainerScopeSingleNUMANodeFuncs(),
			nrosched.ErrorCannotAlignContainer,
			podResourcesRequest{
				appCnt: []corev1.ResourceList{
					{
						corev1.ResourceCPU:    resource.MustParse("4"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
						"hugepages-2Mi":       resource.MustParse("32Mi"),
					},
					{
						corev1.ResourceCPU:    resource.MustParse("4"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
						"hugepages-2Mi":       resource.MustParse("32Mi"),
						"hugepages-1Gi":       resource.MustParse("2Gi"),
					},
				},
			},
			// we need keep the gap between Node level fit and NUMA level fit wide enough.
			// for example if only 2 cpus are separating unsuitable node from becoming suitable,
			// it's not good because the baseload should be added as well (which is around 2 cpus)
			// and then the pod might land on the unsuitable node.
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
					"hugepages-2Mi":       resource.MustParse("32Mi"),
					"hugepages-1Gi":       resource.MustParse("1Gi"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
					"hugepages-2Mi":       resource.MustParse("32Mi"),
					"hugepages-1Gi":       resource.MustParse("1Gi"),
				},
			},
			// the free resources that should be left on the target node should not depend that there will be some baseload added upon padding the node,
			// those free resources should match the pod requests in total. The reason behind that is that Noderesourcesfit plugin (the plugin that is
			// responsible for accepting/rejecting compute nodes as candidates for placing the pod) actually accounts for the baseload, it compares the
			// actual available resources on node with the pod requested resources, if the available resources can accommodate the pod resources then it
			// will mark the node as a possible candidate, if not it will reject it.
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
					"hugepages-2Mi":       resource.MustParse("32Mi"),
					"hugepages-1Gi":       resource.MustParse("1Gi"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
					"hugepages-2Mi":       resource.MustParse("32Mi"),
					"hugepages-1Gi":       resource.MustParse("1Gi"),
				},
			},
		),
		Entry("[test_id:54020] pod with two gu cnt requesting multiple device types keep on pending because cannot align the second container to a single numa node",
			Label(label.Tier2, "unsched", "tmscope:cnt", "devices"),
			newContainerScopeSingleNUMANodeFuncs(),
			nrosched.ErrorCannotAlignContainer,
			podResourcesRequest{
				appCnt: []corev1.ResourceList{
					{
						corev1.ResourceCPU:    resource.MustParse("4"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
						corev1.ResourceName(e2efixture.GetDeviceType2Name()): resource.MustParse("2"),
					},
					{
						corev1.ResourceCPU:    resource.MustParse("4"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
						corev1.ResourceName(e2efixture.GetDeviceType1Name()): resource.MustParse("2"),
						corev1.ResourceName(e2efixture.GetDeviceType3Name()): resource.MustParse("2"),
					},
				},
			},
			// we need keep the gap between Node level fit and NUMA level fit wide enough.
			// for example if only 2 cpus are separating unsuitable node from becoming suitable,
			// it's not good because the baseload should be added as well (which is around 2 cpus)
			// and then the pod might land on the unsuitable node.
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
				},
			},
			// the free resources that should be left on the target node should not depend that there will be some baseload added upon padding the node,
			// those free resources should match the pod requests in total. The reason behind that is that Noderesourcesfit plugin (the plugin that is
			// responsible for accepting/rejecting compute nodes as candidates for placing the pod) actually accounts for the baseload, it compares the
			// actual available resources on node with the pod requested resources, if the available resources can accommodate the pod resources then it
			// will mark the node as a possible candidate, if not it will reject it.
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
					corev1.ResourceName(e2efixture.GetDeviceType1Name()): resource.MustParse("1"),
					corev1.ResourceName(e2efixture.GetDeviceType3Name()): resource.MustParse("2"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
					corev1.ResourceName(e2efixture.GetDeviceType1Name()): resource.MustParse("1"),
					corev1.ResourceName(e2efixture.GetDeviceType2Name()): resource.MustParse("2"),
				},
			},
		),
		Entry("[test_id:54019] pod with two gu cnt keep on pending because cannot align the second container to a single numa node",
			Label(label.Tier1, "unsched", "tmscope:container", "devices"),
			newContainerScopeSingleNUMANodeFuncs(),
			nrosched.ErrorCannotAlignContainer,
			podResourcesRequest{
				appCnt: []corev1.ResourceList{
					{
						corev1.ResourceCPU:    resource.MustParse("5"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
						corev1.ResourceName(e2efixture.GetDeviceType1Name()): resource.MustParse("3"),
					},
					{
						corev1.ResourceCPU:    resource.MustParse("5"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
						corev1.ResourceName(e2efixture.GetDeviceType1Name()): resource.MustParse("2"),
					},
				},
			},
			// we need keep the gap between Node level fit and NUMA level fit wide enough.
			// for example if only 2 cpus are separating unsuitable node from becoming suitable,
			// it's not good because the baseload should be added as well (which is around 2 cpus)
			// and then the pod might land on the unsuitable node.
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("2Gi"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("2"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
				},
			},
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU:    resource.MustParse("5"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
					corev1.ResourceName(e2efixture.GetDeviceType1Name()): resource.MustParse("1"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("5"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
					corev1.ResourceName(e2efixture.GetDeviceType1Name()): resource.MustParse("4"),
				},
			},
		),
		Entry("[test_id:54017] pod with two gu cnt keep on pending because cannot align the both containers on single numa",
			Label(label.Tier1, "unsched", "tmscope:pod", "devices"),
			newPodScopeSingleNUMANodeFuncs(),
			nrosched.ErrorCannotAlignPod,
			podResourcesRequest{
				appCnt: []corev1.ResourceList{
					{
						corev1.ResourceCPU:    resource.MustParse("8"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
						corev1.ResourceName(e2efixture.GetDeviceType1Name()): resource.MustParse("3"),
					},
					{
						corev1.ResourceCPU:    resource.MustParse("8"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
						corev1.ResourceName(e2efixture.GetDeviceType2Name()): resource.MustParse("2"),
					},
				},
			},
			// we need keep the gap between Node level fit and NUMA level fit wide enough.
			// for example if only 2 cpus are separating unsuitable node from becoming suitable,
			// it's not good because the baseload should be added as well (which is around 2 cpus)
			// and then the pod might land on the unsuitable node.
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("2Gi"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("2"),
					corev1.ResourceMemory: resource.MustParse("3Gi"),
				},
			},
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU:    resource.MustParse("8"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
					corev1.ResourceName(e2efixture.GetDeviceType1Name()): resource.MustParse("3"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("8"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
					corev1.ResourceName(e2efixture.GetDeviceType2Name()): resource.MustParse("2"),
				},
			},
		),
		Entry("[test_id:55430] besteffort pod requesting multiple device types keep on pending because cannot align the container to a single numa node",
			Label(label.Tier2, "unsched", "tmscope:pod", "devices"),
			newPodScopeSingleNUMANodeFuncs(),
			nrosched.ErrorCannotAlignPod,
			podResourcesRequest{
				appCnt: []corev1.ResourceList{
					{
						corev1.ResourceName(e2efixture.GetDeviceType1Name()): resource.MustParse("3"),
						corev1.ResourceName(e2efixture.GetDeviceType2Name()): resource.MustParse("4"),
					},
				},
			},
			// we need keep the gap between Node level fit and NUMA level fit wide enough.
			// for example if only 2 cpus are separating unsuitable node from becoming suitable,
			// it's not good because the baseload should be added as well (which is around 2 cpus)
			// and then the pod might land on the unsuitable node.
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
					corev1.ResourceName(e2efixture.GetDeviceType1Name()): resource.MustParse("1"),
					corev1.ResourceName(e2efixture.GetDeviceType2Name()): resource.MustParse("1"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
					corev1.ResourceName(e2efixture.GetDeviceType1Name()): resource.MustParse("1"),
					corev1.ResourceName(e2efixture.GetDeviceType2Name()): resource.MustParse("1"),
				},
			},
			// the free resources that should be left on the target node should not depend that there will be some baseload added upon padding the node,
			// those free resources should match the pod requests in total. The reason behind that is that Noderesourcesfit plugin (the plugin that is
			// responsible for accepting/rejecting compute nodes as candidates for placing the pod) actually accounts for the baseload, it compares the
			// actual available resources on node with the pod requested resources, if the available resources can accommodate the pod resources then it
			// will mark the node as a possible candidate, if not it will reject it.
			[]corev1.ResourceList{
				{
					corev1.ResourceName(e2efixture.GetDeviceType1Name()): resource.MustParse("1"),
					corev1.ResourceName(e2efixture.GetDeviceType2Name()): resource.MustParse("2"),
				},
				{
					corev1.ResourceName(e2efixture.GetDeviceType1Name()): resource.MustParse("2"),
					corev1.ResourceName(e2efixture.GetDeviceType2Name()): resource.MustParse("2"),
				},
			},
		),
		Entry("[test_id:55429] burstable pod requesting multiple device types keep on pending because cannot align the container to a single numa node",
			Label(label.Tier2, "unsched", "tmscope:pod", "devices"),
			newPodScopeSingleNUMANodeFuncs(),
			nrosched.ErrorCannotAlignPod,
			podResourcesRequest{
				appCnt: []corev1.ResourceList{
					{
						corev1.ResourceCPU: resource.MustParse("1"),
						corev1.ResourceName(e2efixture.GetDeviceType1Name()): resource.MustParse("3"),
						corev1.ResourceName(e2efixture.GetDeviceType2Name()): resource.MustParse("4"),
					},
				},
			},
			// we need keep the gap between Node level fit and NUMA level fit wide enough.
			// for example if only 2 cpus are separating unsuitable node from becoming suitable,
			// it's not good because the baseload should be added as well (which is around 2 cpus)
			// and then the pod might land on the unsuitable node.
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
					corev1.ResourceName(e2efixture.GetDeviceType1Name()): resource.MustParse("1"),
					corev1.ResourceName(e2efixture.GetDeviceType2Name()): resource.MustParse("1"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("1Gi"),
					corev1.ResourceName(e2efixture.GetDeviceType1Name()): resource.MustParse("1"),
					corev1.ResourceName(e2efixture.GetDeviceType2Name()): resource.MustParse("1"),
				},
			},
			// the free resources that should be left on the target node should not depend that there will be some baseload added upon padding the node,
			// those free resources should match the pod requests in total. The reason behind that is that Noderesourcesfit plugin (the plugin that is
			// responsible for accepting/rejecting compute nodes as candidates for placing the pod) actually accounts for the baseload, it compares the
			// actual available resources on node with the pod requested resources, if the available resources can accommodate the pod resources then it
			// will mark the node as a possible candidate, if not it will reject it.
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
					corev1.ResourceName(e2efixture.GetDeviceType1Name()): resource.MustParse("1"),
					corev1.ResourceName(e2efixture.GetDeviceType2Name()): resource.MustParse("2"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
					corev1.ResourceName(e2efixture.GetDeviceType1Name()): resource.MustParse("2"),
					corev1.ResourceName(e2efixture.GetDeviceType2Name()): resource.MustParse("2"),
				},
			},
		),
		Entry("[test_id:54023] besteffort pod requesting multiple device types keep on pending because cannot align the container to a single numa node",
			Label(label.Tier2, "unsched", "tmscope:cnt", "devices"),
			newContainerScopeSingleNUMANodeFuncs(),
			nrosched.ErrorCannotAlignContainer,
			podResourcesRequest{
				appCnt: []corev1.ResourceList{
					{
						corev1.ResourceName(e2efixture.GetDeviceType2Name()): resource.MustParse("2"),
					},
					{
						corev1.ResourceName(e2efixture.GetDeviceType1Name()): resource.MustParse("2"),
						corev1.ResourceName(e2efixture.GetDeviceType3Name()): resource.MustParse("2"),
					},
				},
			},
			// we need keep the gap between Node level fit and NUMA level fit wide enough.
			// for example if only 2 cpus are separating unsuitable node from becoming suitable,
			// it's not good because the baseload should be added as well (which is around 2 cpus)
			// and then the pod might land on the unsuitable node.
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
					corev1.ResourceName(e2efixture.GetDeviceType1Name()): resource.MustParse("1"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
					corev1.ResourceName(e2efixture.GetDeviceType1Name()): resource.MustParse("1"),
				},
			},
			// the free resources that should be left on the target node should not depend that there will be some baseload added upon padding the node,
			// those free resources should match the pod requests in total. The reason behind that is that Noderesourcesfit plugin (the plugin that is
			// responsible for accepting/rejecting compute nodes as candidates for placing the pod) actually accounts for the baseload, it compares the
			// actual available resources on node with the pod requested resources, if the available resources can accommodate the pod resources then it
			// will mark the node as a possible candidate, if not it will reject it.
			[]corev1.ResourceList{
				{
					corev1.ResourceName(e2efixture.GetDeviceType1Name()): resource.MustParse("1"),
					corev1.ResourceName(e2efixture.GetDeviceType3Name()): resource.MustParse("2"),
				},
				{
					corev1.ResourceName(e2efixture.GetDeviceType1Name()): resource.MustParse("1"),
					corev1.ResourceName(e2efixture.GetDeviceType2Name()): resource.MustParse("2"),
				},
			},
		),
		Entry("[test_id:54022] burstable pod requesting multiple device types keep on pending because cannot align the container to a single numa node",
			Label(label.Tier2, "unsched", "tmscope:cnt", "devices"),
			newContainerScopeSingleNUMANodeFuncs(),
			nrosched.ErrorCannotAlignContainer,
			podResourcesRequest{
				appCnt: []corev1.ResourceList{
					{
						corev1.ResourceCPU: resource.MustParse("4"),
						corev1.ResourceName(e2efixture.GetDeviceType2Name()): resource.MustParse("2"),
					},
					{
						corev1.ResourceName(e2efixture.GetDeviceType1Name()): resource.MustParse("2"),
						corev1.ResourceName(e2efixture.GetDeviceType3Name()): resource.MustParse("2"),
					},
				},
			},
			// we need keep the gap between Node level fit and NUMA level fit wide enough.
			// for example if only 2 cpus are separating unsuitable node from becoming suitable,
			// it's not good because the baseload should be added as well (which is around 2 cpus)
			// and then the pod might land on the unsuitable node.
			[]corev1.ResourceList{
				{
					corev1.ResourceCPU:    resource.MustParse("1"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
					corev1.ResourceName(e2efixture.GetDeviceType2Name()): resource.MustParse("1"),
				},
				{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("4Gi"),
					corev1.ResourceName(e2efixture.GetDeviceType2Name()): resource.MustParse("1"),
				},
			},
			// the free resources that should be left on the target node should not depend that there will be some baseload added upon padding the node,
			// those free resources should match the pod requests in total. The reason behind that is that Noderesourcesfit plugin (the plugin that is
			// responsible for accepting/rejecting compute nodes as candidates for placing the pod) actually accounts for the baseload, it compares the
			// actual available resources on node with the pod requested resources, if the available resources can accommodate the pod resources then it
			// will mark the node as a possible candidate, if not it will reject it.
			[]corev1.ResourceList{
				{
					corev1.ResourceName(e2efixture.GetDeviceType1Name()): resource.MustParse("1"),
					corev1.ResourceName(e2efixture.GetDeviceType3Name()): resource.MustParse("2"),
				},
				{
					corev1.ResourceName(e2efixture.GetDeviceType1Name()): resource.MustParse("1"),
					corev1.ResourceName(e2efixture.GetDeviceType2Name()): resource.MustParse("2"),
				},
			},
		),
	)
})

func getTopologyManagerScope(ctx context.Context, cli client.Client) (string, error) {
	var nro nropv1.NUMAResourcesOperator
	if err := cli.Get(ctx, client.ObjectKey{Name: objectnames.DefaultNUMAResourcesOperatorCrName}, &nro); err != nil {
		return "", err
	}

	// assume the CR only has one node group
	nodeGroup := nro.Spec.NodeGroups[0]
	poolName := nodeGroup.PoolName
	if poolName == nil || *poolName == "" {
		mcps := &machineconfigv1.MachineConfigPoolList{}
		if err := cli.List(ctx, mcps); err != nil {
			return "", err
		}
		pools, err := nodegroupv1.FindTreesOpenshift(mcps, []nropv1.NodeGroup{nodeGroup})
		if err != nil {
			return "", err
		}
		poolName = &pools[0].MachineConfigPools[0].Name
	}

	cmName := objectnames.GetComponentName(nro.Name, *poolName)
	var cm corev1.ConfigMap
	if err := cli.Get(ctx, client.ObjectKey{Name: cmName}, &cm); err != nil {
		return "", err
	}

	cfg, err := configuration.ValidateAndExtractRTEConfigData(&cm)
	if err != nil {
		return "", err
	}
	return cfg.Kubelet.TopologyManagerScope, nil
}
