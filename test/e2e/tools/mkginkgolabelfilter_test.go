/*
 * Copyright 2024 Red Hat, Inc.
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
	"io"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/klog/v2"

	"github.com/openshift-kni/numaresources-operator/internal/api/features"
)

const FeatureKey = "feature"

var _ = Describe("[tools][mkginkgolabelfilter] Auxiliary tools", Label("tools", "mkginkgolabelfilter"), Ordered, func() {
	Context("with the binary available", func() {
		var cmdline []string

		BeforeAll(func(ctx context.Context) {
			cmdline = []string{
				filepath.Join(BinariesPath, "mkginkgolabelfilter"),
			}
			//expectExecutableExists(cmdline[0])
		})

		It("should create a filter that matches the a valid input", func(ctx context.Context) {
			input := "{\"active\":[" +
				"\"foo\"," +
				"\"bar\"," +
				"\"foobar\"" +
				"]}"

			var inputTp features.TopicInfo
			_ = json.Unmarshal([]byte(input), &inputTp)
			expectedToolOutput, _ := features.BuildFilterConsistAny(FeatureKey, inputTp.Active)

			klog.Infof("running: %v\n", cmdline)
			toolCmd := exec.Command(cmdline[0])
			r, w := io.Pipe()
			toolCmd.Stdin = r

			go func() {
				defer w.Close()
				w.Write(append([]byte(input), "\n"...))
			}()
			out, err := toolCmd.Output()
			Expect(err).ToNot(HaveOccurred())

			Expect(string(out)).To(Equal(expectedToolOutput), "different output found:\n%v\nexpected:\n%v", string(out), expectedToolOutput)

			topics, err := getListFromGinkgoQuery(string(out))
			Expect(err).ToNot(HaveOccurred())
			sort.Strings(topics)
			sort.Strings(inputTp.Active)
			Expect(reflect.DeepEqual(topics, inputTp.Active)).To(BeTrue(), "different active topics are displayed in the query:\n%v\nexpected:\n%v", topics, inputTp.Active)
		})

		It("should fail on an invalid json format", func(ctx context.Context) {
			input := "(\"active\"=[" +
				"\"foo\"," +
				"\"bar\"," +
				"\"foobar\"" +
				"])"

			klog.Infof("running: %v\n", cmdline)
			toolCmd := exec.Command(cmdline[0])
			r, w := io.Pipe()
			toolCmd.Stdin = r

			go func() {
				defer w.Close()
				w.Write(append([]byte(input), "\n"...))
			}()
			out, err := toolCmd.Output()
			Expect(err).To(HaveOccurred(), "expected to fail on invalid json formatted input but passed instead, binary output:%s", string(out))
		})

		It("should return empty features on a valid json format and non-matching topic info - no active", func(ctx context.Context) {
			input := "{\"\":[" +
				"\"foo\"," +
				"\"bar\"," +
				"\"foobar\"" +
				"]}"
			var inputTp features.TopicInfo
			_ = json.Unmarshal([]byte(input), &inputTp)
			expectedToolOutput, _ := features.BuildFilterConsistAny(FeatureKey, inputTp.Active)

			klog.Infof("running: %v\n", cmdline)
			toolCmd := exec.Command(cmdline[0])
			r, w := io.Pipe()
			toolCmd.Stdin = r

			go func() {
				defer w.Close()
				w.Write(append([]byte(input), "\n"...))
			}()
			out, err := toolCmd.Output()
			Expect(err).ToNot(HaveOccurred())

			Expect(string(out)).To(Equal(expectedToolOutput), "different output found:\n%v\nexpected:\n%v", string(out), expectedToolOutput)

			topics, err := getListFromGinkgoQuery(string(out))
			Expect(len(topics)).To(Equal(0), "features found: %+v", topics)
		})

		It("should fail on a valid json format and partially matching topic info", func(ctx context.Context) {
			input := "{\"supported\":[" +
				"\"foo\"," +
				"\"bar\"," +
				"]," +
				"\"active\": [\"foobar\"]}"

			klog.Infof("running: %v\n", cmdline)
			toolCmd := exec.Command(cmdline[0])
			r, w := io.Pipe()
			toolCmd.Stdin = r

			go func() {
				defer w.Close()
				w.Write(append([]byte(input), "\n"...))
			}()
			_, err := toolCmd.Output()
			Expect(err).To(HaveOccurred())

		})

	})
})

func getListFromGinkgoQuery(q string) ([]string, error) {
	re := regexp.MustCompile("consistAny\\s+{(.*)*}")
	match := re.FindStringSubmatch(q)
	if len(match) < 2 {
		return []string{}, fmt.Errorf("a list of active features is expected, found:\n %s", q)
	}
	topicsStr := strings.TrimSpace(match[1])
	if topicsStr == "" {
		return []string{}, nil
	}
	return strings.Split(topicsStr, ","), nil
}
