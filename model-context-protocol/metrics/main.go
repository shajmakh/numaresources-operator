package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/openshift-kni/numaresources-operator/model-context-protocol/metrics/rte"
)

func main() {
	// NOTE: the server assumes existence of a k8s cluster and that NRO is installed
	server := mcp.NewServer(&mcp.Implementation{Name: "metrics", Version: "v1.0.0"}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_rte_pods",
		Description: "List all RTE pods and their status",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
		pods, err := rte.ListRTEPods(ctx)
		if err != nil {
			return nil, nil, err
		}
		return nil, pods, nil
	})

	// mcp.AddTool(server, &mcp.Tool{
	// 	Name:        "get_all_rte_metrics",
	// 	Description: "Get metrics from all RTE pods",
	// }, rte.GetAllRTEMetricsHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_rte_pod_metrics",
		Description: "Get metrics from a specific RTE pod",
	}, rte.GetRTEPodMetricsHandler)

	// mcp.AddTool(server, &mcp.Tool{
	// 	Name:        "get_rte_node_metrics",
	// 	Description: "Get metrics from all RTE pods on a specific node", // it is 1:1 mapping between a node and a RTE pod but is valid (and easier) to have a single handler for this
	// }, rte.GetRTENodeMetricsHandler)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_specific_rte_metrics",
		Description: "Get specific metrics from a chosen RTE pods or nodes; the input is a list of pod names and node names, and a list of metrics to fetch",
	}, rte.GetFilteredMetricsHandler)

	log.Println("Starting MCP server")
	// Run the server over stdin/stdout, until the client disconnects.
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
