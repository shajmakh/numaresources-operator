package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Input struct {
	MustGatherPath string `json:"mustGatherPath" jsonschema:"the path to the must-gather"`
}

type Output struct {
	Result string `json:"result" jsonschema:"the result of the operation"`
}

type ResourceDataOutput struct {
	Configuration string `json:"configuration" jsonschema:"the configuration of the resource"`
	Status        string `json:"status" jsonschema:"the status of the resource"`
}

func readPath(filePath string) ([]byte, error) {
	// TODO
	// Use os.OpenRoot to prevent directory traversal attacks
	// This ensures file access is scoped to the validated absolute path

	// assumption:
	// 1.the must-gather root is the current position (pwd) of the mcp server;
	// 2.if the must-gather exists in different directory, the user must provide the full correct path
	// 3.the mcp server depends on these assumptions and it does not process
	// TODO: ensure this is the safest validation of teh numaresources must-gather
	root, err := os.OpenRoot(filepath.Dir(filePath))
	if err != nil {
		return nil, fmt.Errorf("error opening root directory for %s (%s): %v", filePath, filepath.Dir(filePath), err)
	}
	defer func() {
		_ = root.Close()
	}()

	file, err := root.Open(filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("error opening file %s: %v", filePath, err)
	}
	defer func() {
		_ = file.Close()
	}()

	return io.ReadAll(file)
}

func checkMustGatherPath(ctx context.Context, req *mcp.CallToolRequest, input Input) (
	*mcp.CallToolResult, // TODO: do we need non-nil returned value
	Output,
	error,
) {
	_, err := readfile(input.MustGatherPath)
	if err != nil {
		return nil, Output{Result: fmt.Sprintf("error reading file %s: %v", input.MustGatherPath, err)}, err
	}
	return nil, Output{Result: fmt.Sprintf("must-gather %s is valid and for numaresources", input.MustGatherPath)}, nil
}

// func getResourceData(ctx context.Context, req *mcp.CallToolRequest, input Input) (
// 	*mcp.CallToolResult,
// 	Output,
// 	error,
// ) {
// 	// define the path of the
// 	return nil, Output{Result: fmt.Sprintf("resource data for %s", input.ResourceName)}, nil
// }

func getNumaresourcesOperatorConfig(ctx context.Context, req *mcp.CallToolRequest, input Input) (
	*mcp.CallToolResult,
	ResourceDataOutput,
	error,
) {
	// define the path of the nro file
	nroPathInMUSTGather := filepath.Join(input.MustGatherPath, "cluster-scoped-resources/nodetopology.openshift.io/numaresourcesoperators/numaresourcesoperators.yaml")
	resourceData, err := readfile(nroPathInMUSTGather)
	if err != nil {
		return nil, ResourceDataOutput{Configuration: fmt.Sprintf("error reading file %s: %v", nroPathInMUSTGather, err)}, err
	}

	//extend to status
	return nil, ResourceDataOutput{Configuration: string(resourceData)}, nil
}

func main() {
	// Create a server with a single tool.
	server := mcp.NewServer(&mcp.Implementation{Name: "greeter", Version: "v1.0.0"}, nil)

	mcp.AddTool(server, &mcp.Tool{Name: "check-path", Description: "verify if the must-gather is valid and for numaresources"}, checkMustGatherPath)
	// should all of the tools be called after ^ passes
	mcp.AddTool(server, &mcp.Tool{Name: "get-numaresources-operator-config", Description: "get the numaresources-operator yaml configuration"}, getNumaresourcesOperatorConfig)
	// Run the server over stdin/stdout, until the client disconnects.
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}

/* queries:
from the mustgather in /home/shajmakh/must-gather.local.7570010382959034801 show me the configuration of NRO CR?
invalid:
from the mustgather in /home/shajmakh/must-gather.local.5700103 show me the configuration of NRO CR?
*/
