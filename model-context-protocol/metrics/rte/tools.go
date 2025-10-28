package rte

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	corev1 "k8s.io/api/core/v1"
)

func GetRTEPodMetricsHandler(ctx context.Context, req *mcp.CallToolRequest, input GetRTEPodMetricsInput) (*mcp.CallToolResult, RTEMetricsOutput, error) {
	metricsInfo, err := getRTEPodMetrics(ctx, input.PodName, nil)
	if err != nil {
		return nil, RTEMetricsOutput{}, err
	}
	return nil, RTEMetricsOutput{Output: []PodMetricsInfo{metricsInfo}}, nil
}

func GetAllRTEMetricsHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, RTEMetricsOutput, error) {
	pods, err := ListRTEPods(ctx)
	if err != nil {
		return nil, RTEMetricsOutput{}, err
	}

	if len(pods.Items) == 0 {
		return nil, RTEMetricsOutput{}, errors.New("no RTE pods found")
	}

	podsMetrics := make([]PodMetricsInfo, 0)
	for _, pod := range pods.Items {
		metricsInfo, err := getRTEPodMetrics(ctx, pod.Name, nil)
		if err != nil {
			return nil, RTEMetricsOutput{}, err
		}
		podsMetrics = append(podsMetrics, metricsInfo)
	}
	return nil, RTEMetricsOutput{Output: podsMetrics}, nil
}

func GetRTENodeMetricsHandler(ctx context.Context, req *mcp.CallToolRequest, input GetRTENodeMetricsInput) (*mcp.CallToolResult, RTEMetricsOutput, error) {
	pods, err := ListRTEPods(ctx)
	if err != nil {
		return nil, RTEMetricsOutput{}, err
	}

	if len(pods.Items) == 0 {
		return nil, RTEMetricsOutput{}, errors.New("no RTE pods found")
	}

	targetNode := input.NodeName
	if targetNode == "" {
		return nil, RTEMetricsOutput{}, errors.New("node name is required")
	}

	podsMetrics := make([]PodMetricsInfo, 0)
	for _, pod := range pods.Items {
		if pod.Spec.NodeName == targetNode {
			metricsInfo, err := getRTEPodMetrics(ctx, pod.Name, nil)
			if err != nil {
				return nil, RTEMetricsOutput{}, err
			}
			podsMetrics = append(podsMetrics, metricsInfo)
		}
	}
	return nil, RTEMetricsOutput{Output: podsMetrics}, nil
}

func GetFilteredMetricsHandler(ctx context.Context, req *mcp.CallToolRequest, input GetFilteredMetricsInput) (*mcp.CallToolResult, RTEMetricsOutput, error) {
	pods, err := ListRTEPods(ctx)
	if err != nil {
		return nil, RTEMetricsOutput{}, err
	}

	if len(pods.Items) == 0 {
		return nil, RTEMetricsOutput{}, errors.New("no RTE pods found")
	}
	withFilters := metricsFilter(func() []string {
		return nil
	})

	if len(input.MetricNames) != 0 {
		withFilters = func() []string {
			return input.MetricNames
		}
	}

	podsMetrics := make([]PodMetricsInfo, 0)
	for _, pod := range pods.Items {
		metricsInfo, err := getRTEPodMetrics(ctx, pod.Name, withFilters)
		if err != nil {
			return nil, RTEMetricsOutput{}, err
		}
		podsMetrics = append(podsMetrics, metricsInfo)
	}
	return nil, RTEMetricsOutput{Output: podsMetrics}, nil

}

func getRTEPodMetrics(ctx context.Context, podName string, withFilters metricsFilter) (PodMetricsInfo, error) {
	pods, err := ListRTEPods(ctx)
	if err != nil {
		return PodMetricsInfo{}, err
	}

	targetPod := corev1.Pod{}
	for _, pod := range pods.Items {
		if pod.Name == podName {
			targetPod = pod
			break
		}
	}
	if targetPod.Status.Phase != corev1.PodRunning {
		return PodMetricsInfo{}, fmt.Errorf("RTE pod %s is not running", targetPod.Name)
	}
	for _, cnt := range targetPod.Status.ContainerStatuses {
		if cnt.Name == RTEContainerName && !cnt.Ready {
			return PodMetricsInfo{}, fmt.Errorf("RTE container in pod %s is not ready: %+v", targetPod.Name, cnt)
		}
	}

	metricsText, err := scrapePodMetrics(targetPod)
	if err != nil {
		return PodMetricsInfo{}, err
	}

	parsedMetrics := parseMetricsText(metricsText)
	rteMetrics := extractRTEMetrics(parsedMetrics, withFilters)

	return PodMetricsInfo{
		PodName:   targetPod.Name,
		NodeName:  targetPod.Spec.NodeName,
		Namespace: targetPod.Namespace,
		Metrics:   rteMetrics,
		Timestamp: time.Now(),
	}, nil
}

// parseMetricsText parses Prometheus format metrics text
func parseMetricsText(metricsText string) map[string][]MetricValue {
	// one line example of metrics output that we want to consider:
	// rte_wakeup_delay_milliseconds{node="cnfdd3.t5g-dev.eng.rdu2.dc.redhat.com",trigger="periodic"} 10000

	metrics := make(map[string][]MetricValue)
	lines := strings.Split(metricsText, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse metric line: name{labels} value
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		metricName := parts[0]
		valueStr := parts[1]

		// Extract labels if present
		labels := make(map[string]string)
		if strings.Contains(metricName, "{") {
			start := strings.Index(metricName, "{")
			end := strings.LastIndex(metricName, "}")
			if end > start {
				labelStr := metricName[start+1 : end]
				metricName = metricName[:start]

				// Parse labels
				labelPairs := strings.Split(labelStr, ",")
				for _, pair := range labelPairs {
					if strings.Contains(pair, "=") {
						kv := strings.SplitN(pair, "=", 2)
						if len(kv) == 2 {
							key := strings.TrimSpace(kv[0])
							val := strings.TrimSpace(kv[1])
							val = strings.Trim(val, "\"")
							labels[key] = val
						}
					}
				}
			}
		}

		// Parse value
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			continue
		}

		if metrics[metricName] == nil {
			metrics[metricName] = []MetricValue{}
		}

		metrics[metricName] = append(metrics[metricName], MetricValue{
			Labels: labels,
			Value:  value,
		})
	}

	return metrics
}

// extractRTEMetrics extracts the 4 main RTE metrics or filters the metrics based on the input filters
func extractRTEMetrics(parsedMetrics map[string][]MetricValue, withFilters metricsFilter) RTEMetrics {

	rteMetrics := RTEMetrics{
		PodResourceAPICallFailures: []MetricValue{},
		NodeResourceTopologyWrites: []MetricValue{},
		OperationDelay:             []MetricValue{},
		WakeupDelay:                []MetricValue{},
	}
	mainMetrics := map[string]*[]MetricValue{
		"rte_podresource_api_call_failures_total": &rteMetrics.PodResourceAPICallFailures,
		"rte_noderesourcetopology_writes_total":   &rteMetrics.NodeResourceTopologyWrites,
		"rte_operation_delay_milliseconds":        &rteMetrics.OperationDelay,
		"rte_wakeup_delay_milliseconds":           &rteMetrics.WakeupDelay,
	}

	metricNamesToCollect := make(map[string]*[]MetricValue)
	var filter []string
	if withFilters != nil {
		filter = withFilters()
	}
	if len(filter) == 0 {
		metricNamesToCollect = mainMetrics
	} else {
		// Map filtered metrics to RTEMetrics fields if they match main metrics
		for _, metric := range filter {
			if target, exists := mainMetrics[metric]; exists {
				metricNamesToCollect[metric] = target
			} else {
				// For unknown metrics, create a new slice (won't be stored in RTEMetrics)
				metricNamesToCollect[metric] = &[]MetricValue{}
			}
		}
	}

	for metricName, target := range metricNamesToCollect {
		if values, exists := parsedMetrics[metricName]; exists {
			*target = values
		}
	}

	return rteMetrics
}

// scrapePodMetrics scrapes metrics from a single RTE pod using Kubernetes Pod proxy API
// This uses the cluster's API server proxy (similar to kubectl proxy) to access pod metrics
// Format: /api/v1/namespaces/{namespace}/pods/{pod}/proxy/{path}
// This works from outside the cluster and handles TLS/authentication automatically via REST client
func scrapePodMetrics(pod corev1.Pod) (string, error) {
	k8sClient, err := getKubernetesClient()
	if err != nil {
		return "", fmt.Errorf("failed to get kubernetes client: %w", err)
	}

	// Find the metrics port from the pod spec
	metricsPort := "2112"
	for _, container := range pod.Spec.Containers {
		for _, port := range container.Ports {
			if port.Name == "metrics-port" {
				metricsPort = strconv.Itoa(int(port.ContainerPort))
				break
			}
		}
	}

	// Use Kubernetes Pod proxy API via PodExpansion ProxyGet method
	// This properly formats the proxy URL with scheme, name, port, and path
	body, err := k8sClient.CoreV1().Pods(pod.Namespace).ProxyGet("https", pod.Name, metricsPort, "metrics", nil).
		DoRaw(context.Background())

	if err != nil {
		return "", fmt.Errorf("failed to fetch metrics from %s via pod proxy: %w", pod.Name, err)
	}

	return string(body), nil
}

func ListRTEPods(ctx context.Context) (*corev1.PodList, error) {
	cli, err := getClient()
	if err != nil {
		return nil, err
	}

	return listRTEPodsFromClient(ctx, cli)
}

func listRTEPodsFromClient(ctx context.Context, cli client.Client) (*corev1.PodList, error) {
	pods := &corev1.PodList{}
	err := cli.List(ctx, pods, client.InNamespace(NUMAResourcesOperatorNamespace), client.MatchingLabels{
		"name": "resource-topology",
	})
	if err != nil {
		return nil, err
	}
	return pods, nil
}

func getClient() (client.Client, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	c, err := client.New(cfg, client.Options{})
	return c, err
}

func getKubernetesClient() (kubernetes.Interface, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}
