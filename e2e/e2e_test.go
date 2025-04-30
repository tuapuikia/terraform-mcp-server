package e2e_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"testing"
	"time"

	mcpClient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

func TestE2E(t *testing.T) {
	// Build the Docker image for the MCP server.
	buildDockerImage(t)

	e2eServerToken := os.Getenv("HCP_TFE_TOKEN")
	t.Setenv("HCP_TFE_TOKEN", e2eServerToken) // The MCP Client merges the existing environment.
	args := []string{
		"docker",
		"run",
		"-i",
		"--rm",
		"-e",
		"HCP_TFE_TOKEN",
		"cmd/hcp-terraform-mcp-server",
	}
	t.Log("Starting Stdio MCP client...")
	client, err := mcpClient.NewStdioMCPClient(args[0], []string{}, args[1:]...)
	require.NoError(t, err, "expected to create client successfully")
	defer client.Close()

	t.Run("Initialize", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		request := mcp.InitializeRequest{}
		request.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
		request.Params.ClientInfo = mcp.Implementation{
			Name:    "e2e-test-client",
			Version: "0.0.1",
		}

		result, err := client.Initialize(ctx, request)
		if err != nil {
			log.Fatalf("Failed to initialize: %v", err)
		}
		fmt.Printf(
			"Initialized with server: %s %s\n\n",
			result.ServerInfo.Name,
			result.ServerInfo.Version,
		)
		require.Equal(t, "hcp-terraform-mcp-server", result.ServerInfo.Name)
	})

	// TODO: split the tests into multiple files
	t.Run("CallTool providerDetails", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// When we call the "get_me" tool
		request := mcp.CallToolRequest{}
		request.Params.Name = "providerDetails"
		request.Params.Arguments = map[string]interface{}{
			"name":      "aws",
			"namespace": "hashicorp",
			"version":   "latest",
		}

		response, err := client.CallTool(ctx, request)
		require.NoError(t, err, "expected to call 'providerDetails' tool successfully")

		require.False(t, response.IsError, "expected result not to be an error")
		require.Len(t, response.Content, 2, "expected content to have one item")

		textContent, ok := response.Content[0].(mcp.TextContent)
		require.True(t, ok, "expected content to be of type TextContent")

		t.Logf("Raw response content: %s", textContent.Text)

		// TODO: Need to fix this: it is static and should be updated to test with the actual API response.
		require.Len(t, textContent.Text, 53, "expected content to have two items")
	})

	// TODO: split the tests into multiple files
	t.Run("CallTool listModules", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// When we call the "get_me" tool
		request := mcp.CallToolRequest{}
		request.Params.Name = "listModules"
		request.Params.Arguments = map[string]interface{}{
			"name":      "",
			"namespace": "",
		}
		response, err := client.CallTool(ctx, request)
		require.NoError(t, err, "expected to call 'listModules' tool successfully")

		require.False(t, response.IsError, "expected result not to be an error")
		require.Len(t, response.Content, 2, "expected content to have two items")

		textResourceContents, ok := response.Content[1].(mcp.EmbeddedResource).Resource.(mcp.TextResourceContents)
		require.True(t, ok, "expected content to be of type TextResourceContents")

		t.Logf("Raw response content: %s", textResourceContents.Text)

		// TODO: this is just copying the markdown/text output, though we should refine this to be more explicit of how we test the individual fields from the text response
		require.Equal(t, "# registry://modules modules\n\n## lb-http \n\n**Id:** GoogleCloudPlatform/lb-http/google/12.1.4 \n\n**OwnerName:** \n\n**Namespace:** GoogleCloudPlatform\n\n**Source:** https://github.com/terraform-google-modules/terraform-google-lb-http\n\n## managed-instance-group \n\n**Id:** GoogleCloudPlatform/managed-instance-group/google/1.1.15 \n\n**OwnerName:** \n\n**Namespace:** GoogleCloudPlatform\n\n**Source:** https://github.com/GoogleCloudPlatform/terraform-google-managed-instance-group\n\n## lb-internal \n\n**Id:** GoogleCloudPlatform/lb-internal/google/7.0.0 \n\n**OwnerName:** \n\n**Namespace:** GoogleCloudPlatform\n\n**Source:** https://github.com/terraform-google-modules/terraform-google-lb-internal\n\n## nat-gateway \n\n**Id:** GoogleCloudPlatform/nat-gateway/google/1.2.3 \n\n**OwnerName:** \n\n**Namespace:** GoogleCloudPlatform\n\n**Source:** https://github.com/GoogleCloudPlatform/terraform-google-nat-gateway\n\n## ecs-instance \n\n**Id:** alibaba/ecs-instance/alicloud/2.12.0 \n\n**OwnerName:** \n\n**Namespace:** alibaba\n\n**Source:** https://github.com/alibabacloud-automation/terraform-alicloud-ecs-instance\n\n## slb \n\n**Id:** alibaba/slb/alicloud/2.1.0 \n\n**OwnerName:** \n\n**Namespace:** alibaba\n\n**Source:** https://github.com/alibabacloud-automation/terraform-alicloud-slb\n\n## vpc \n\n**Id:** alibaba/vpc/alicloud/1.11.0 \n\n**OwnerName:** \n\n**Namespace:** alibaba\n\n**Source:** https://github.com/alibabacloud-automation/terraform-alicloud-vpc\n\n## compute-instance \n\n**Id:** oracle/compute-instance/opc/1.0.1 \n\n**OwnerName:** \n\n**Namespace:** oracle\n\n**Source:** https://github.com/oracle/terraform-opc-compute-instance\n\n## security-group \n\n**Id:** alibaba/security-group/alicloud/2.4.0 \n\n**OwnerName:** \n\n**Namespace:** alibaba\n\n**Source:** https://github.com/alibabacloud-automation/terraform-alicloud-security-group\n\n## sql-db \n\n**Id:** GoogleCloudPlatform/sql-db/google/25.2.2 \n\n**OwnerName:** \n\n**Namespace:** GoogleCloudPlatform\n\n**Source:** https://github.com/terraform-google-modules/terraform-google-sql-db\n\n## ip-networks \n\n**Id:** oracle/ip-networks/opc/1.0.0 \n\n**OwnerName:** \n\n**Namespace:** oracle\n\n**Source:** https://github.com/oracle/terraform-opc-ip-networks\n\n## lb \n\n**Id:** GoogleCloudPlatform/lb/google/5.0.0 \n\n**OwnerName:** \n\n**Namespace:** GoogleCloudPlatform\n\n**Source:** https://github.com/terraform-google-modules/terraform-google-lb\n\n## slb-listener \n\n**Id:** terraform-alicloud-modules/slb-listener/alicloud/1.5.0 \n\n**OwnerName:** \n\n**Namespace:** terraform-alicloud-modules\n\n**Source:** https://github.com/alibabacloud-automation/terraform-alicloud-slb-listener\n\n## kubernetes-wordpress \n\n**Id:** terraform-alicloud-modules/kubernetes-wordpress/alicloud/1.1.0 \n\n**OwnerName:** \n\n**Namespace:** terraform-alicloud-modules\n\n**Source:** https://github.com/alibabacloud-automation/terraform-alicloud-kubernetes-wordpress\n\n## disk \n\n**Id:** terraform-alicloud-modules/disk/alicloud/2.0.0 \n\n**OwnerName:** \n\n**Namespace:** terraform-alicloud-modules\n\n**Source:** https://github.com/alibabacloud-automation/terraform-alicloud-disk\n\n", textResourceContents.Text, "expected modules to match")
	})
}

func buildDockerImage(t *testing.T) {
	t.Log("Building Docker image for e2e tests...")

	cmd := exec.Command("docker", "build", "-t", "cmd/hcp-terraform-mcp-server", ".")
	cmd.Dir = ".." // Run this in the context of the root, where the Dockerfile is located.
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "expected to build Docker image successfully, output: %s", string(output))
}
