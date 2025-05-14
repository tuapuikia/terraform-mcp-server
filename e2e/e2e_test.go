// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package e2e

import (
	"context"
	"fmt"
	"log"
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
	args := []string{
		"docker",
		"run",
		"-i",
		"--rm",
		"terraform-mcp-server:test-e2e",
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
		require.Equal(t, "terraform-mcp-server", result.ServerInfo.Name)
	})

	for _, testCase := range providerDetailsTestCases {
		t.Run("CallTool providerDetails", func(t *testing.T) {
			// t.Parallel()
			t.Logf("TOOL providerDetails %s", testCase.TestDescription)
			t.Logf("Test payload: %v", testCase.TestPayload)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			request := mcp.CallToolRequest{}
			request.Params.Name = "providerDetails"
			request.Params.Arguments = testCase.TestPayload

			response, err := client.CallTool(ctx, request)
			if testCase.TestShouldFail {
				require.Error(t, err, "expected to call 'providerDetails' tool with error")
				t.Logf("Error: %v", err)
				// require.True(t, response.IsError, "expected result to be an error")
			} else {
				require.NoError(t, err, "expected to call 'providerDetails' tool successfully")
				require.False(t, response.IsError, "expected result not to be an error")
				require.Len(t, response.Content, 1, "expected content to have one item")

				textContent, ok := response.Content[0].(mcp.TextContent)
				require.True(t, ok, "expected content to be of type TextContent")
				t.Logf("Content length: %d", len(textContent.Text))

				// TODO: Implement a better way to test this
				if testCase.TestContentType == CONST_TYPE_DATA_SOURCE {
					require.NotContains(t, textContent.Text, "**Category:** resources", "expected content not to contain resources")
				} else if testCase.TestContentType == CONST_TYPE_RESOURCE {
					require.NotContains(t, textContent.Text, "**Category:** data-sources", "expected content not to contain data-sources")
				} else if testCase.TestContentType == CONST_TYPE_BOTH {
					require.Contains(t, textContent.Text, "**Category:** resources", "expected content to contain resources")
					require.Contains(t, textContent.Text, "**Category:** data-sources", "expected content to contain data-sources")
				} else if testCase.TestContentType == CONST_TYPE_GUIDES {
					require.Contains(t, textContent.Text, "**Category:** guides", "expected content to contain guides")
				} else if testCase.TestContentType == CONST_TYPE_FUNCTIONS {
					require.Contains(t, textContent.Text, "**Category:** functions", "expected content to contain functions")
				}
			}
		})
	}

	for _, testCase := range providerDetailsTestCases {
		t.Run("CallTool providerResourceDetails", func(t *testing.T) {
			// t.Parallel()
			t.Logf("TOOL providerResourceDetails %s", testCase.TestDescription)
			t.Logf("Test payload: %v", testCase.TestPayload)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			request := mcp.CallToolRequest{}
			request.Params.Name = "providerResourceDetails"
			request.Params.Arguments = testCase.TestPayload

			response, err := client.CallTool(ctx, request)
			if testCase.TestShouldFail {
				require.Error(t, err, "expected to call 'providerResourceDetails' tool with error")
				t.Logf("Error: %v", err)
				// require.True(t, response.IsError, "expected result to be an error")
			} else {
				require.NoError(t, err, "expected to call 'providerResourceDetails' tool successfully")
				require.False(t, response.IsError, "expected result not to be an error")
				require.Len(t, response.Content, 1, "expected content to have one item")

				textContent, ok := response.Content[0].(mcp.TextContent)
				require.True(t, ok, "expected content to be of type TextContent")
				t.Logf("Content length: %d", len(textContent.Text))

				if testCase.TestContentType == CONST_TYPE_DATA_SOURCE {
					require.NotContains(t, textContent.Text, "**Category:** resources", "expected content not to contain resources")
				} else if testCase.TestContentType == CONST_TYPE_RESOURCE {
					require.NotContains(t, textContent.Text, "**Category:** data-sources", "expected content not to contain data-sources")
				} else if testCase.TestContentType == CONST_TYPE_BOTH {
					require.Contains(t, textContent.Text, "resource", "expected content to contain resources")
					require.Contains(t, textContent.Text, "data source", "expected content to contain data-sources")
				} else if testCase.TestContentType == CONST_TYPE_GUIDES {
					require.Contains(t, textContent.Text, "guide", "expected content to contain guide")
				} else if testCase.TestContentType == CONST_TYPE_FUNCTIONS {
					require.Contains(t, textContent.Text, "functions", "expected content to contain functions")
				}
			}
		})
	}

	for _, testCase := range searchModulesTestCases {
		t.Run("CallTool searchModules", func(t *testing.T) {
			// t.Parallel()
			t.Logf("TOOL searchModules %s", testCase.TestDescription)
			t.Logf("Test payload: %v", testCase.TestPayload)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			request := mcp.CallToolRequest{}
			request.Params.Name = "searchModules"
			request.Params.Arguments = testCase.TestPayload

			response, err := client.CallTool(ctx, request)
			if testCase.TestShouldFail {
				require.Error(t, err, "expected to call 'searchModules' tool with error")
				t.Logf("Error: %v", err)
			} else {
				require.NoError(t, err, "expected to call 'searchModules' tool successfully")
				require.False(t, response.IsError, "expected result not to be an error")
				// For searchModules, we expect one content item which is the text list of modules.
				// If no modules are found for a valid query, it might return an empty list or a message,
				// but the call itself should succeed.
				if len(response.Content) > 0 { // Check if content is present before trying to access it
					textContent, ok := response.Content[0].(mcp.TextContent)
					require.True(t, ok, "expected content to be of type TextContent")
					t.Logf("Content length: %d", len(textContent.Text))
					// Add more specific content assertions here if needed, e.g., check for "module" keyword
					// require.Contains(t, textContent.Text, "module", "expected content to contain 'module'")
				} else {
					// Handle cases where successful calls might return no content items (e.g. empty list of modules)
					// This depends on the expected behavior of the listModules tool for such cases.
					// For now, we just log it.
					t.Log("Response content is empty for successful call.")
				}
			}
		})
	}

	for _, testCase := range moduleDetailsTestCases {
		t.Run("CallTool moduleDetails", func(t *testing.T) {
			// t.Parallel()
			t.Logf("TOOL moduleDetails %s", testCase.TestDescription)
			t.Logf("Test payload: %v", testCase.TestPayload)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			request := mcp.CallToolRequest{}
			request.Params.Name = "moduleDetails"
			request.Params.Arguments = testCase.TestPayload

			response, err := client.CallTool(ctx, request)
			if testCase.TestShouldFail {
				require.Error(t, err, "expected to call 'moduleDetails' tool with error")
				t.Logf("Error: %v", err)
				// require.True(t, response.IsError, "expected result to be an error")
			} else {
				require.NoError(t, err, "expected to call 'moduleDetails' tool successfully")
				require.False(t, response.IsError, "expected result not to be an error")
				require.Len(t, response.Content, 1, "expected content to have one item")

				textContent, ok := response.Content[0].(mcp.TextContent)
				require.True(t, ok, "expected content to be of type TextContent")
				t.Logf("Content length: %d", len(textContent.Text))

				if testCase.TestContentType == CONST_TYPE_DATA_SOURCE {
					require.NotContains(t, textContent.Text, "**Category:** resources", "expected content not to contain resources")
				} else if testCase.TestContentType == CONST_TYPE_RESOURCE {
					require.NotContains(t, textContent.Text, "**Category:** data-sources", "expected content not to contain data-sources")
				} else if testCase.TestContentType == CONST_TYPE_BOTH {
					require.Contains(t, textContent.Text, "resource", "expected content to contain resources")
					require.Contains(t, textContent.Text, "data source", "expected content to contain data-sources")
				}
			}
		})
	}
}

func buildDockerImage(t *testing.T) {
	t.Log("Building Docker image for e2e tests...")

	cmd := exec.Command("make", "VERSION=test-e2e", "docker-build")
	cmd.Dir = ".." // Run this in the context of the root, where the Makefile is located.
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "expected to build Docker image successfully, output: %s", string(output))
}
