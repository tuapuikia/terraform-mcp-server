package tfregistry

import (
	"context"
	"net/http"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ListProviders creates a tool to list Terraform providers.
func ListProviders(registryClient *http.Client) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("list_providers",
			mcp.WithDescription("List providers accessible by the credential."),
			// TODO: Add pagination parameters here using the correct mcp-go mechanism
			// Example (conceptual):
			// mcp.WithInteger("page_number", mcp.Description("Page number"), mcp.Optional()),
			// mcp.WithInteger("page_size", mcp.Description("Page size"), mcp.Optional()),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// TODO: Parse pagination options
			// pageNumber, _ := OptionalParam[int](request, "page_number")
			// pageSize, _ := OptionalParam[int](request, "page_size")

			commonProviders := []string{
				"aws", "google", "azurerm", "kubernetes",
				"github", "docker", "null", "random",
			}

			return mcp.NewToolResultText(strings.Join(commonProviders, ", ")), nil
		}
}
