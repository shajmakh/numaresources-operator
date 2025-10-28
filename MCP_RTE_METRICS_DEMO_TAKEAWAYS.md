# MCP RTE Metrics Server - Key Takeaways for Demo

## 🎯 What We Built

**MCP Server for RTE Metrics** - A Model Context Protocol (MCP) server that exposes RTE (Resource Topology Exporter) metrics from Kubernetes clusters, enabling AI assistants and tools to query cluster metrics programmatically.

## 🏗️ Architecture & Design

### Core Components
- **MCP Server** (`model-context-protocol/metrics/main.go`) - Runs over stdin/stdout, implements MCP protocol
- **5 MCP Tools** exposing different query patterns:
  1. `list_rte_pods` - Discover RTE pods in cluster
  2. `get_all_rte_metrics` - Fetch metrics from all RTE pods
  3. `get_rte_pod_metrics` - Query specific pod by name
  4. `get_rte_node_metrics` - Query by node name (1:1 pod-to-node mapping)
  5. `get_specific_rte_metrics` - Filter by metric names

### Key Technical Implementation
- **Kubernetes API Proxy** - Uses `/api/v1/namespaces/{ns}/pods/{pod}/proxy/https:2112/metrics` endpoint
- **HTTPS/TLS Support** - Critical: RTE metrics endpoints use HTTPS, not HTTP
- **Prometheus Format Parsing** - Parses Prometheus text format into structured JSON
- **Automatic Pod Discovery** - Uses Service selector (`name=resource-topology`) to find RTE pods

## 📊 The 4 Main RTE Metrics Tracked

1. **`rte_podresource_api_call_failures_total`** - Tracks failures calling PodResource API
2. **`rte_noderesourcetopology_writes_total`** - Counts NRT object updates (periodic/reactive)
3. **`rte_operation_delay_milliseconds`** - Measures operation latency (podresources_scan, node_resource_object_update)
4. **`rte_wakeup_delay_milliseconds`** - Tracks wakeup delays for periodic/reactive triggers

## 🔑 Key Technical Challenges Solved

### Challenge 1: HTTPS/TLS Metrics Endpoints
- **Problem**: RTE pods expose metrics over HTTPS (port 2112), not HTTP
- **Solution**: Use Kubernetes API server proxy with `https:2112` scheme specification
- **Why Important**: Standard `kubectl proxy` only supports HTTP; direct API proxy bypasses this limitation

### Challenge 2: Prometheus Text Format → Structured JSON
- **Problem**: Metrics come as Prometheus text format (lines like `metric{labels} value`)
- **Solution**: Custom parser extracts metric names, labels, and values into structured `MetricValue` objects
- **Benefit**: Enables filtering, aggregation, and programmatic access

### Challenge 3: Pod Discovery & Filtering
- **Problem**: Need to find RTE pods and filter by node/metric type
- **Solution**: Use Kubernetes label selector (`name=resource-topology`) and service discovery
- **Pattern**: List pods → Filter by node → Scrape metrics → Parse → Filter metrics

## 💡 Use Cases Demonstrated

1. **Query by Node**: "Get operation_delay metrics for pods on node cnfdd3"
   - Use case: Debugging specific node performance issues
   
2. **Filter by Metric Type**: "List only podresource_api_call_failures metrics"
   - Use case: Monitoring API health and error rates

3. **Cluster-wide Metrics**: "Get all RTE metrics from all pods"
   - Use case: Overall cluster health monitoring

## 🎬 Demo Flow (4 Minutes)

### Slide 1: Problem Statement (30s)
- RTE metrics are hard to query programmatically
- Need structured access for AI tools and automation
- Traditional kubectl/curl approaches are cumbersome

### Slide 2: MCP Solution (1min)
- What is MCP? Protocol for AI tools to access context
- Built MCP server exposing RTE metrics as tools
- 5 query patterns covering common use cases

### Slide 3: Technical Deep Dive (1.5min)
- Architecture: MCP server → K8s API proxy → HTTPS endpoints → Parser → JSON
- Key challenge: HTTPS/TLS support via API proxy
- Metrics: 4 main RTE metrics tracked with labels

### Slide 4: Live Demo (1min)
- Show query: "Get operation_delay metrics for node cnfdd3"
- Show structured JSON output
- Highlight filtering capabilities

## 🚀 Key Value Propositions

1. **Programmatic Access**: AI assistants can query metrics without manual kubectl commands
2. **Structured Data**: JSON output vs raw Prometheus text - easier to process
3. **Filtering**: Query by node, pod, or metric type - flexible queries
4. **HTTPS Support**: Properly handles TLS-secured metrics endpoints
5. **Extensibility**: Easy to add more metrics or query patterns

## 📝 Notes for Demo

- **Emphasize**: MCP enables AI tools to understand cluster state
- **Highlight**: HTTPS/TLS handling was critical technical challenge
- **Show**: Real queries from your testing (operation_delay, podresource failures)
- **Connect**: How this enables better observability and debugging


