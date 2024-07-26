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

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/apps/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/k8stopologyawareschedwg/deployer/pkg/deployer/platform"

	"github.com/openshift-kni/numaresources-operator/internal/api/features"
	"github.com/openshift-kni/numaresources-operator/internal/podlist"
	"github.com/openshift-kni/numaresources-operator/internal/remoteexec"
	"github.com/openshift-kni/numaresources-operator/test/utils/clients"
)

var _ = Describe("[tools] Auxiliary tools", Label("tools"), func() {
	Context("with the binary available", func() {
		It("[lsplatform] lsplatform should detect the cluster", Label("inspectfeatures"), func() {
			cmdline := []string{
				filepath.Join(BinariesPath, "lsplatform"),
			}

			expectExecutableExists(cmdline[0])

			fmt.Fprintf(GinkgoWriter, "running: %v\n", cmdline)

			cmd := exec.Command(cmdline[0], cmdline[1:]...)
			cmd.Stderr = GinkgoWriter
			out, err := cmd.Output()
			Expect(err).ToNot(HaveOccurred())

			text := strings.TrimSpace(string(out))
			_, ok := platform.ParsePlatform(text)
			Expect(ok).To(BeTrue(), "cannot recognize detected platform: %s", text)
		})

		It("[api][inspectfeatures] should expose correct active features and create filter", Label("api", "inspectfeatures"), func(ctx context.Context) {
			By("inspect active features from controller pod")
			var controllerDp v1.Deployment
			err := clients.Client.Get(context.TODO(), client.ObjectKey{Namespace: "numaresources-operator", Name: "test1"}, &controllerDp)
			Expect(err).ToNot(HaveOccurred())

			controllerPods, err := podlist.With(clients.Client).ByDeployment(ctx, controllerDp)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(controllerPods)).To(Equal(1))

			cmd := []string{"bin/numaresources-operator", "-inspect-features"}
			stdoutFeatures, stderr, err := remoteexec.CommandOnPod(context.Background(), clients.K8sClient, &controllerPods[0], cmd...)
			Expect(err).ToNot(HaveOccurred(), "err=%v stderr=%s", err, stderr)
			klog.Infof("active features from the deployed operator:\n%s", string(stdoutFeatures))

			var tp features.TopicInfo
			err = json.Unmarshal(stdoutFeatures, &tp)
			Expect(err).ToNot(HaveOccurred())

			cmd = []string{"bin/numaresources-operator", "-version"}
			stdoutVersion, stderr, err := remoteexec.CommandOnPod(context.Background(), clients.K8sClient, &controllerPods[0], cmd...)
			Expect(err).ToNot(HaveOccurred(), "err=%v", err)
			Expect(stdoutVersion).ToNot(BeEmpty())
			klog.Infof("deployed version: %s\n", string(stdoutVersion))

			re := regexp.MustCompile("numaresources-operator\\s+([0-9]+.[0-9]+.[0-9]+)\\s+")
			match := re.FindStringSubmatch("numaresources-operator 4.17.0 v0.4.16-rc2.dev36+g395eebc9 395eebc9 go1.22.5") //string(stdoutVersion))
			klog.Infof("\n\n%+v\n\n", match)
			Expect(len(match)).To(BeNumerically(">", 1), "different pattern of version was found:\n%s", string(stdoutVersion))
			tp.Metadata.Version = fmt.Sprintf("v%s", match[1])
			klog.Infof("version to validate: %s", tp.Metadata.Version)

			By("validate api output vs the expected")
			expected, err := tp.Validate()
			Expect(err).ToNot(HaveOccurred(), "api output failed validation with err %v\nexpected:\n%+v\nfound:\n%+v\n", expected, tp)
		})
	})
})
