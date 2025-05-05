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
	hcServer.AddResource(ProviderResource(registryClient, fmt.Sprintf("%sproviders/official", PROVIDER_BASE_PATH), "Official Providers list", "official", logger))
	hcServer.AddResource(ProviderResource(registryClient, fmt.Sprintf("%sproviders/partner", PROVIDER_BASE_PATH), "Partner Providers list", "partner", logger))
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
			listOfProviders, err := GetProviderList(registryClient, providerType, logger)
			if err != nil {
				return nil, logAndReturnError(logger, fmt.Sprintf("Provider Resource: error getting %s provider list", providerType), err)
			}
			resourceContents := make([]mcp.ResourceContents, len(listOfProviders))
			for i, provider := range listOfProviders {
				namespace, name, version := ExtractProviderNameAndVersion(fmt.Sprintf("%s/%s/name/%s/version/latest", PROVIDER_BASE_PATH, provider["namespace"], provider["name"]))
				logger.Debugf("Extracted namespace: %s, name: %s, version: %s", namespace, name, version)

				versionNumber, err := GetLatestProviderVersion(registryClient, namespace, name, logger)
				if err != nil {
					return nil, logAndReturnError(logger, fmt.Sprintf("Provider Resource: error getting %s/%s provider version %s", namespace, name, versionNumber), err)
				}

				providerVersionUri := fmt.Sprintf("%s/%s/name/%s/version/%s", PROVIDER_BASE_PATH, namespace, name, versionNumber)
				logger.Debugf("Provider resource - providerVersionUri: %s", providerVersionUri)

				providerDocs, err := ProviderResourceTemplateHandler(registryClient, providerVersionUri, logger)
				if err != nil {
					return nil, logAndReturnError(logger, fmt.Sprintf("Provider Resource: error with provider template handler %s/%s provider version %s details", namespace, name, versionNumber), err)
				}
				resourceContents[i] = mcp.TextResourceContents{
					MIMEType: "text/markdown",
					URI:      providerVersionUri,
					Text:     fmt.Sprintf("# %s Provider \n\n %s", provider["name"], providerDocs),
				}
			}
			return resourceContents, nil
		}
}
