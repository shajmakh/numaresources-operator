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

package objectnames

import (
	"fmt"
	"strings"
)

const (
	DefaultNUMAResourcesOperatorCrName  = "numaresourcesoperator"
	DefaultNUMAResourcesSchedulerCrName = "numaresourcesscheduler"
)

func GetMachineConfigName(instanceName, mcpName string) string {
	return fmt.Sprintf("51-%s-%s", instanceName, mcpName)
}

func GetComponentName(instanceName, mcpName string) string {
	return fmt.Sprintf("%s-%s", instanceName, mcpName)
}

func ExtractAssociatedNameFromRTEDaemonset(dsName, instanceName string) string {
	return strings.Replace(dsName, fmt.Sprintf("%s-", instanceName), "", 1)
}
