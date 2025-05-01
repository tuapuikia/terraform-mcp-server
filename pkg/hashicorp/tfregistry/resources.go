package tfregistry

import (
	"context"
	"fmt"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	log "github.com/sirupsen/logrus"
)

func RegisterResources(hcServer *server.MCPServer, registryClient *http.Client, logger *log.Logger) {
	// Add resources for official and partner providers
	hcServer.AddResource(ProviderResource(registryClient, "registry://providers/official", "Official Providers list", "official", logger))
	hcServer.AddResource(ProviderResource(registryClient, "registry://providers/partner", "Partner Providers list", "partner", logger))
}

func ProviderResource(registryClient *http.Client, resourceURI string, description string, providerType string, logger *log.Logger) (mcp.Resource, server.ResourceHandlerFunc) {
	return mcp.NewResource(
			resourceURI,
			description,
			mcp.WithMIMEType("text/markdown"),
			mcp.WithResourceDescription(description),
			// TODO: Add pagination parameters here using the correct mcp-go mechanism
			// Example (conceptual):
			// mcp.WithInteger("page_number", mcp.Description("Page number"), mcp.Optional()),
			// mcp.WithInteger("page_size", mcp.Description("Page size"), mcp.Optional()),
		),
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			listOfProviders := GetProviderList(registryClient, providerType, logger)
			resourceContents := make([]mcp.ResourceContents, len(listOfProviders))
			for i, provider := range listOfProviders {
				content := fmt.Sprintf("## %s Provider - [%s](https://registry.terraform.io/providers/hashicorp/%s/latest/docs)\n", provider, provider, provider)
				resourceContents[i] = mcp.TextResourceContents{
					MIMEType: "text/markdown",
					URI:      "registry://providers/" + provider,
					Text:     content,
				}
			}
			return resourceContents, nil
		}
}
