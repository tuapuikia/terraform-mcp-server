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
	t.Run("CallTool list_providers", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// When we call the "get_me" tool
		request := mcp.CallToolRequest{}
		request.Params.Name = "list_providers"

		response, err := client.CallTool(ctx, request)
		require.NoError(t, err, "expected to call 'list_providers' tool successfully")

		require.False(t, response.IsError, "expected result not to be an error")
		require.Len(t, response.Content, 1, "expected content to have one item")

		textContent, ok := response.Content[0].(mcp.TextContent)
		require.True(t, ok, "expected content to be of type TextContent")

		t.Logf("Raw response content: %s", textContent.Text)

		// TODO: this is static and should be updated to test with the actual API response.
		require.Equal(t, "aws, google, azurerm, kubernetes, github, docker, null, random", textContent.Text, "expected providers to match")
	})
}

func buildDockerImage(t *testing.T) {
	t.Log("Building Docker image for e2e tests...")

	cmd := exec.Command("docker", "build", "-t", "cmd/hcp-terraform-mcp-server", ".")
	cmd.Dir = ".." // Run this in the context of the root, where the Dockerfile is located.
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "expected to build Docker image successfully, output: %s", string(output))
}
