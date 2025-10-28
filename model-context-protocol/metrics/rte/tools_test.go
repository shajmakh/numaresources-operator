package rte

import (
	"context"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_listRTEPodsFromClient(t *testing.T) {
	pods := &corev1.PodList{
		Items: []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rte-pod-1",
					Namespace: NUMAResourcesOperatorNamespace,
					Labels: map[string]string{
						"name": "resource-topology",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rte-pod-2",
					Namespace: NUMAResourcesOperatorNamespace,
					Labels: map[string]string{
						"name": "resource-topology",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rte-pod-3",
					Namespace: NUMAResourcesOperatorNamespace,
					Labels: map[string]string{
						"name": "resource-topology-not-included",
					},
				},
			},
		},
	}

	// implement fake client that returns the pods
	runtimeObjects := make([]runtime.Object, len(pods.Items))
	for i := range pods.Items {
		runtimeObjects[i] = &pods.Items[i]
	}
	cli := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithRuntimeObjects(runtimeObjects...).Build()
	got, err := listRTEPodsFromClient(context.Background(), cli)
	if err != nil {
		t.Errorf("listRTEPodsFromClient() failed: %v", err)
	}

	expectedPods := &corev1.PodList{
		Items: []corev1.Pod{pods.Items[0], pods.Items[1]},
	}
	if !reflect.DeepEqual(got, expectedPods) {
		t.Errorf("listRTEPodsFromClient() = %v, want %v", got, pods)
	}
}

func Test_extractRTEMetrics(t *testing.T) {
	tests := []struct {
		name          string
		parsedMetrics map[string][]MetricValue
		filter        metricsFilter
		expected      RTEMetrics
	}{
		{
			name: "no filter - all metrics present",
			parsedMetrics: map[string][]MetricValue{
				"rte_podresource_api_call_failures_total": {
					{Labels: map[string]string{"node": "node1"}, Value: 5.0},
				},
				"rte_noderesourcetopology_writes_total": {
					{Labels: map[string]string{"node": "node1"}, Value: 10.0},
				},
				"rte_operation_delay_milliseconds": {
					{Labels: map[string]string{"trigger": "periodic"}, Value: 15.5},
				},
				"rte_wakeup_delay_milliseconds": {
					{Labels: map[string]string{"node": "node1"}, Value: 20.0},
				},
			},
			filter: func() []string {
				return nil
			},
			expected: RTEMetrics{
				PodResourceAPICallFailures: []MetricValue{
					{Labels: map[string]string{"node": "node1"}, Value: 5.0},
				},
				NodeResourceTopologyWrites: []MetricValue{
					{Labels: map[string]string{"node": "node1"}, Value: 10.0},
				},
				OperationDelay: []MetricValue{
					{Labels: map[string]string{"trigger": "periodic"}, Value: 15.5},
				},
				WakeupDelay: []MetricValue{
					{Labels: map[string]string{"node": "node1"}, Value: 20.0},
				},
			},
		},
		{
			name: "with filter - specific metrics only",
			parsedMetrics: map[string][]MetricValue{
				"rte_podresource_api_call_failures_total": {
					{Labels: map[string]string{"node": "node1"}, Value: 5.0},
				},
				"rte_noderesourcetopology_writes_total": {
					{Labels: map[string]string{"node": "node1"}, Value: 10.0},
				},
				"rte_operation_delay_milliseconds": {
					{Labels: map[string]string{"trigger": "periodic"}, Value: 15.5},
				},
				"rte_wakeup_delay_milliseconds": {
					{Labels: map[string]string{"node": "node1"}, Value: 20.0},
				},
			},
			filter: func() []string {
				return []string{"rte_wakeup_delay_milliseconds", "rte_operation_delay_milliseconds"}
			},
			expected: RTEMetrics{
				PodResourceAPICallFailures: []MetricValue{},
				NodeResourceTopologyWrites: []MetricValue{},
				OperationDelay: []MetricValue{
					{Labels: map[string]string{"trigger": "periodic"}, Value: 15.5},
				},
				WakeupDelay: []MetricValue{
					{Labels: map[string]string{"node": "node1"}, Value: 20.0},
				},
			},
		},
		{
			name: "no filter - missing metrics",
			parsedMetrics: map[string][]MetricValue{
				"rte_wakeup_delay_milliseconds": {
					{Labels: map[string]string{"node": "node1"}, Value: 20.0},
				},
			},
			filter: func() []string {
				return nil
			},
			expected: RTEMetrics{
				PodResourceAPICallFailures: []MetricValue{},
				NodeResourceTopologyWrites: []MetricValue{},
				OperationDelay:             []MetricValue{},
				WakeupDelay: []MetricValue{
					{Labels: map[string]string{"node": "node1"}, Value: 20.0},
				},
			},
		},
		{
			name:          "empty parsed metrics",
			parsedMetrics: map[string][]MetricValue{},
			filter: func() []string {
				return nil
			},
			expected: RTEMetrics{
				PodResourceAPICallFailures: []MetricValue{},
				NodeResourceTopologyWrites: []MetricValue{},
				OperationDelay:             []MetricValue{},
				WakeupDelay:                []MetricValue{},
			},
		},
		{
			name: "with filter - missing requested metric",
			parsedMetrics: map[string][]MetricValue{
				"rte_wakeup_delay_milliseconds": {
					{Labels: map[string]string{"node": "node1"}, Value: 20.0},
				},
			},
			filter: func() []string {
				return []string{"rte_wakeup_delay_milliseconds", "rte_noderesourcetopology_writes_total"}
			},
			expected: RTEMetrics{
				PodResourceAPICallFailures: []MetricValue{},
				NodeResourceTopologyWrites: []MetricValue{},
				OperationDelay:             []MetricValue{},
				WakeupDelay: []MetricValue{
					{Labels: map[string]string{"node": "node1"}, Value: 20.0},
				},
			},
		},
		{
			name: "no filter - multiple values per metric",
			parsedMetrics: map[string][]MetricValue{
				"rte_wakeup_delay_milliseconds": {
					{Labels: map[string]string{"node": "node1"}, Value: 20.0},
					{Labels: map[string]string{"node": "node2"}, Value: 25.0},
				},
				"rte_operation_delay_milliseconds": {
					{Labels: map[string]string{"trigger": "periodic"}, Value: 15.5},
					{Labels: map[string]string{"trigger": "event"}, Value: 12.3},
				},
			},
			filter: func() []string {
				return nil
			},
			expected: RTEMetrics{
				PodResourceAPICallFailures: []MetricValue{},
				NodeResourceTopologyWrites: []MetricValue{},
				OperationDelay: []MetricValue{
					{Labels: map[string]string{"trigger": "periodic"}, Value: 15.5},
					{Labels: map[string]string{"trigger": "event"}, Value: 12.3},
				},
				WakeupDelay: []MetricValue{
					{Labels: map[string]string{"node": "node1"}, Value: 20.0},
					{Labels: map[string]string{"node": "node2"}, Value: 25.0},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRTEMetrics(tt.parsedMetrics, tt.filter)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("extractRTEMetrics() failed: got\n%+v\nexpected\n%+v", got, tt.expected)
			}
		})
	}
}
