package rte

import "time"

const (
	NUMAResourcesOperatorNamespace = "numaresources" // local installation

	RTEContainerName = "resource-topology-exporter"
)

type GetRTENodeMetricsInput struct {
	NodeName string `json:"node_name"`
}

type GetRTEPodMetricsInput struct {
	PodName string `json:"pod_name"`
}

type GetFilteredMetricsInput struct {
	MetricNames []string `json:"metric_names"`
}

type RTEMetricsOutput struct {
	Output []PodMetricsInfo `json:"output"`
}

// RTEMetrics represents the 4 main RTE metrics
type RTEMetrics struct {
	PodResourceAPICallFailures []MetricValue `json:"podresource_api_call_failures"`
	NodeResourceTopologyWrites []MetricValue `json:"noderesourcetopology_writes"`
	OperationDelay             []MetricValue `json:"operation_delay"`
	WakeupDelay                []MetricValue `json:"wakeup_delay"`
}

// MetricValue represents a single metric value with labels
type MetricValue struct {
	Labels map[string]string `json:"labels"`
	Value  float64           `json:"value"`
	//Timestamp time.Time         `json:"timestamp"`
}

// PodMetricsInfo contains metrics for a specific pod;
// this is the building block of the output of every handler call
type PodMetricsInfo struct {
	PodName   string     `json:"pod_name"`
	NodeName  string     `json:"node_name"`
	Namespace string     `json:"namespace"`
	Metrics   RTEMetrics `json:"metrics"`
	Timestamp time.Time  `json:"timestamp"`
}

type metricsFilter func() []string

type FilterParams struct {
	MetricName []string
	NodeName   []string
	PodName    []string
}
