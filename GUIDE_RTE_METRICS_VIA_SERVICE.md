# Guide: Listing RTE Pods and Metrics via Kubernetes Service CR

This guide shows you how to manually discover RTE pods and fetch their metrics using the Kubernetes Service Custom Resource (`numaresources-rte-metrics-service`) via terminal commands.

## Prerequisites

- Access to a Kubernetes cluster with NUMAResources Operator installed
- `kubectl` or `oc` (OpenShift CLI) configured and authenticated
- RTE pods should be running in the cluster

**Note**: This guide uses `kubectl` commands, but they can be replaced with `oc` for OpenShift clusters.

## Step 1: Discover the RTE Metrics Service

First, let's check if the Service exists and examine its configuration:

```bash
# Get the Service resource
kubectl get service numaresources-rte-metrics-service -n numaresources -o yaml

# Or get a simpler view
kubectl get service numaresources-rte-metrics-service -n numaresources
```

Expected output should show:
- **Name**: `numaresources-rte-metrics-service`
- **Namespace**: `numaresources`
- **Selector**: `name=resource-topology` (this is what we'll use to find pods)
- **Port**: `2112` (metrics port)

## Step 2: List RTE Pods Using Service Selector

The Service has a selector that matches RTE pods. Use this selector to list pods:

```bash
# Method 1: Use the Service selector directly
kubectl get pods -n numaresources -l name=resource-topology

# Method 2: Extract selector from Service and use it
SERVICE_SELECTOR=$(kubectl get service numaresources-rte-metrics-service -n numaresources -o jsonpath='{.spec.selector}')
echo "Service selector: $SERVICE_SELECTOR"

# Use the extracted selector (format: key1=value1,key2=value2)
kubectl get pods -n numaresources -l "$(kubectl get service numaresources-rte-metrics-service -n numaresources -o jsonpath='{.spec.selector}' | tr ',' ' ' | sed 's/:/=/g')"
```

**Simpler approach** - Since we know the selector is `name=resource-topology`:

```bash
# List all RTE pods matching the Service selector
kubectl get pods -n numaresources -l name=resource-topology

# Get detailed information
kubectl get pods -n numaresources -l name=resource-topology -o wide

# Get pod names only
kubectl get pods -n numaresources -l name=resource-topology -o jsonpath='{.items[*].metadata.name}'
```

## Step 3: Discover Pods via Service Endpoints

Another way to find RTE pods is through the Service's Endpoints resource:

```bash
# Get the Endpoints resource (same name as Service)
kubectl get endpoints numaresources-rte-metrics-service -n numaresources -o yaml

# Extract pod names from endpoints
kubectl get endpoints numaresources-rte-metrics-service -n numaresources -o jsonpath='{.subsets[*].addresses[*].targetRef.name}'

# Get pod details from endpoints
kubectl get endpoints numaresources-rte-metrics-service -n numaresources -o jsonpath='{range .subsets[*]}{range .addresses[*]}{.targetRef.name}{"\n"}{end}{end}'
```

## Step 4: Access Metrics from RTE Pods

**Important**: RTE metrics endpoints use HTTPS (TLS). The `kubectl proxy` command's HTTP REST API doesn't support specifying HTTPS scheme in the URL path. However, there are ways to use the proxy API with HTTPS backends.

### Method 1: Using API Server Proxy Directly (Works with HTTPS!)

You can use the Kubernetes API server's proxy endpoint directly with curl, bypassing `kubectl proxy`. This method supports HTTPS backends:

```bash
# Get your API server URL and credentials
APISERVER=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')
TOKEN=$(kubectl get secret $(kubectl get sa default -o jsonpath='{.secrets[0].name}') -o jsonpath='{.data.token}' | base64 -d)

# Or use your current kubeconfig context
APISERVER=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')
TOKEN=$(kubectl -n default create token default 2>/dev/null || kubectl get secret -n default -o jsonpath='{.items[?(@.metadata.annotations.kubernetes\.io/service-account\.name=="default")].data.token}' | head -1 | base64 -d)

# Get a pod name
POD_NAME=$(kubectl get pods -n numaresources -l name=resource-topology -o jsonpath='{.items[0].metadata.name}')

# Access metrics via API server proxy (supports HTTPS backend via Go client API)
curl -k -H "Authorization: Bearer $TOKEN" \
  "$APISERVER/api/v1/namespaces/numaresources/pods/$POD_NAME/proxy/https:2112/metrics"
```

**Note**: The API server's proxy endpoint internally uses the Go client API which supports HTTPS backends. The URL format `/proxy/https:2112/metrics` tells the API server to use HTTPS when connecting to the pod.

### Method 2: Using kubectl proxy with API Server Direct Call

If you want to use `kubectl proxy` but still access HTTPS backends, you need to call the API server's proxy endpoint directly (not through kubectl proxy's HTTP endpoint):

```bash
# Start kubectl proxy (still useful for authentication)
kubectl proxy --port=8001 &
PROXY_PID=$!

# Get API server URL (through proxy)
APISERVER="http://localhost:8001"

# Get pod name
POD_NAME=$(kubectl get pods -n numaresources -l name=resource-topology -o jsonpath='{.items[0].metadata.name}')

# IMPORTANT: This still won't work because kubectl proxy's HTTP endpoint doesn't support HTTPS backends
# The API server proxy endpoint does, but kubectl proxy wraps it in HTTP

# Clean up
kill $PROXY_PID
```

**Reality Check**: Even through kubectl proxy, the REST API endpoint doesn't support HTTPS backends. You need to use the API server directly (Method 1) or use port-forward (Method 3).

### Method 3: Port-Forward (Simplest for HTTPS endpoints)

```bash
# Get a pod name
POD_NAME=$(kubectl get pods -n numaresources -l name=resource-topology -o jsonpath='{.items[0].metadata.name}')

# Port-forward to the pod's metrics port
kubectl port-forward -n numaresources $POD_NAME 2112:2112 &
PORT_FORWARD_PID=$!

# Wait a moment for port-forward to establish
sleep 2

# Fetch metrics (use -k flag to skip certificate verification for localhost)
curl -k https://localhost:2112/metrics

# Clean up port-forward
kill $PORT_FORWARD_PID
```

### Method 2: Access Metrics from All Pods

```bash
# Get all RTE pod names
PODS=$(kubectl get pods -n numaresources -l name=resource-topology -o jsonpath='{.items[*].metadata.name}')

# Iterate through each pod
for pod in $PODS; do
    echo "=== Metrics for pod: $pod ==="
    
    # Start port-forward in background
    kubectl port-forward -n numaresources $pod 2112:2112 > /dev/null 2>&1 &
    PF_PID=$!
    sleep 2
    
    # Fetch metrics
    curl -k -s https://localhost:2112/metrics | grep -E "^rte_"
    
    # Clean up
    kill $PF_PID 2>/dev/null
    echo ""
done
```

### Method 4: Execute curl Inside Pod (Alternative)

If port-forward doesn't work, you can execute curl from inside the pod:

```bash
# Get a pod name
POD_NAME=$(kubectl get pods -n numaresources -l name=resource-topology -o jsonpath='{.items[0].metadata.name}')

# Execute curl inside the pod
kubectl exec -n numaresources $POD_NAME -c resource-topology-exporter -- curl -k https://localhost:2112/metrics
```

### Stopping Port-Forward

**Method 1: If running in foreground**
```bash
# Press Ctrl+C to stop
```

**Method 2: If running in background**
```bash
# Find the process
ps aux | grep "port-forward"

# Kill by PID
kill <PID>

# Or kill by name pattern
pkill -f "port-forward.*<pod-name>"
```

**Method 3: Find and kill by port**
```bash
# Find what's using port 2112
lsof -i :2112
# or
netstat -tulpn | grep 2112

# Kill the process using the port
kill $(lsof -t -i:2112)
```

## Step 5: Complete Testing Script

Here's a complete script to test everything:

```bash
#!/bin/bash

NAMESPACE="numaresources"
SERVICE_NAME="numaresources-rte-metrics-service"

echo "=== Step 1: Check Service ==="
kubectl get service $SERVICE_NAME -n $NAMESPACE

echo -e "\n=== Step 2: List RTE Pods via Service Selector ==="
kubectl get pods -n $NAMESPACE -l name=resource-topology -o wide

echo -e "\n=== Step 3: Check Service Endpoints ==="
kubectl get endpoints $SERVICE_NAME -n $NAMESPACE -o yaml

echo -e "\n=== Step 4: Extract Pod Names from Endpoints ==="
POD_NAMES=$(kubectl get endpoints $SERVICE_NAME -n $NAMESPACE -o jsonpath='{.subsets[*].addresses[*].targetRef.name}')
echo "RTE Pods found: $POD_NAMES"

echo -e "\n=== Step 5: Fetch Metrics from Each Pod ==="
for pod in $POD_NAMES; do
    echo "--- Metrics from pod: $pod ---"
    
    # Start port-forward
    kubectl port-forward -n $NAMESPACE $pod 2112:2112 > /dev/null 2>&1 &
    PF_PID=$!
    sleep 2
    
    # Fetch metrics
    curl -k -s https://localhost:2112/metrics | grep "^rte_" | head -10
    
    # Clean up
    kill $PF_PID 2>/dev/null
    echo ""
done
```

Save this as `test-rte-metrics.sh`, make it executable (`chmod +x test-rte-metrics.sh`), and run it.

## Step 6: Verify Specific RTE Metrics

Focus on the main RTE metrics:

```bash
# Get all RTE pods
PODS=$(kubectl get pods -n numaresources -l name=resource-topology -o jsonpath='{.items[*].metadata.name}')

# Check for the 4 main RTE metrics from each pod
for pod in $PODS; do
    echo "=== Pod: $pod ==="
    
    # Start port-forward
    kubectl port-forward -n numaresources $pod 2112:2112 > /dev/null 2>&1 &
    PF_PID=$!
    sleep 2
    
    # Fetch and filter metrics
    curl -k -s https://localhost:2112/metrics | \
        grep -E "(rte_podresource_api_call_failures_total|rte_noderesourcetopology_writes_total|rte_operation_delay_milliseconds|rte_wakeup_delay_milliseconds)"
    
    # Clean up
    kill $PF_PID 2>/dev/null
    echo ""
done
```

## Quick Reference Commands

```bash
# 1. Get Service details
kubectl get service numaresources-rte-metrics-service -n numaresources -o yaml

# 2. List pods using Service selector
kubectl get pods -n numaresources -l name=resource-topology

# 3. Get pods from Service endpoints
kubectl get endpoints numaresources-rte-metrics-service -n numaresources -o jsonpath='{.subsets[*].addresses[*].targetRef.name}'

# 4. Access metrics via port-forward (recommended for HTTPS)
POD_NAME=$(kubectl get pods -n numaresources -l name=resource-topology -o jsonpath='{.items[0].metadata.name}')
kubectl port-forward -n numaresources $POD_NAME 2112:2112 &
curl -k https://localhost:2112/metrics

# 5. Access metrics by executing curl inside pod
POD_NAME=$(kubectl get pods -n numaresources -l name=resource-topology -o jsonpath='{.items[0].metadata.name}')
kubectl exec -n numaresources $POD_NAME -c resource-topology-exporter -- curl -k https://localhost:2112/metrics
```

## Troubleshooting

### Service Not Found
```bash
# Check if Service exists
kubectl get service -n numaresources | grep rte-metrics
```

### No Pods Found
```bash
# Check if RTE pods are running
kubectl get pods -n numaresources -l name=resource-topology

# Check pod status
kubectl get pods -n numaresources -l name=resource-topology -o jsonpath='{.items[*].status.phase}'
```

### Endpoints Empty
```bash
# Check if Service endpoints are populated
kubectl get endpoints numaresources-rte-metrics-service -n numaresources

# If empty, check if pods match the selector
kubectl get pods -n numaresources --show-labels | grep resource-topology
```

### Metrics Endpoint Not Accessible
```bash
# Check if pod has metrics port exposed
kubectl get pod <pod-name> -n numaresources -o jsonpath='{.spec.containers[*].ports[*].name}'

# Verify pod is running
kubectl get pod <pod-name> -n numaresources -o jsonpath='{.status.phase}'

# Check container readiness
kubectl get pod <pod-name> -n numaresources -o jsonpath='{.status.containerStatuses[*].ready}'
```

### "Client sent an HTTP request to an HTTPS server" Error

This error occurs because:
1. RTE metrics endpoints use HTTPS (TLS)
2. `kubectl proxy` command's HTTP REST API doesn't support specifying HTTPS scheme in the URL path
3. The proxy endpoint defaults to HTTP

**Solutions**:

**Option 1: Use API Server Proxy Directly (Supports HTTPS!)**
```bash
# Get API server and token
APISERVER=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')
TOKEN=$(kubectl -n default create token default 2>/dev/null || kubectl get secret -n default -o jsonpath='{.items[?(@.metadata.annotations.kubernetes\.io/service-account\.name=="default")].data.token}' | head -1 | base64 -d)

# Use API server proxy endpoint directly (supports HTTPS backend)
curl -k -H "Authorization: Bearer $TOKEN" \
  "$APISERVER/api/v1/namespaces/numaresources/pods/$pod/proxy/https:2112/metrics"
```

**Option 2: Use port-forward (Simpler)**
```bash
# Port-forward preserves HTTPS protocol
kubectl port-forward -n numaresources $pod 2112:2112 &
curl -k https://localhost:2112/metrics
```

**Why this happens**: 
- The Kubernetes API server's proxy endpoint (used by Go client) **does support HTTPS backends** via `/proxy/https:2112/metrics`
- But `kubectl proxy` wraps this in an HTTP-only endpoint, so the HTTPS scheme gets lost
- Using the API server directly or port-forward bypasses this limitation

### Port Already in Use

If port 2112 is already in use:

```bash
# Find what's using the port
lsof -i :2112

# Kill the process or use a different local port
kubectl port-forward -n numaresources $pod 2113:2112  # Use 2113 locally instead
curl -k https://localhost:2113/metrics
```

