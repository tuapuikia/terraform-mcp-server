package tfregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	log "github.com/sirupsen/logrus"
)

func RegisterResourceTemplates(hcServer *server.MCPServer, registryClient *http.Client, logger *log.Logger) {
	hcServer.AddResourceTemplate(ProviderResourceTemplate(registryClient, "registry://provider/{namespace}/name/{name}/version/{version}", "Provider details", logger))
}

func ProviderResourceTemplate(registryClient *http.Client, resourceURI string, description string, logger *log.Logger) (mcp.ResourceTemplate, server.ResourceTemplateHandlerFunc) {
	return mcp.NewResourceTemplate(
			resourceURI,
			description,
			mcp.WithTemplateDescription("Describes details for a Terraform provider"),
			mcp.WithTemplateMIMEType("application/json"),
			// TODO: Add pagination parameters here using the correct mcp-go mechanism
			// Example (conceptual):
			// mcp.WithInteger("page_number", mcp.Description("Page number"), mcp.Optional()),
			// mcp.WithInteger("page_size", mcp.Description("Page size"), mcp.Optional()),
		),
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			providerVersionID, providerVersionUri, err := GetProviderDetails(registryClient, request.Params.URI, logger)
			logger.Debugf("Provider resource template - providerVersionID: %s, providerVersionUri: %s", providerVersionID, providerVersionUri)
			if err != nil {
				return nil, logAndReturnError(logger, "getting provider details", err)
			}

			// Filter docs by provider version
			uri := fmt.Sprintf("provider-docs?filter[provider-version]=%s", providerVersionID)
			response, err := sendRegistryCall(registryClient, "GET", uri, logger, "v2")
			if err != nil {
				return nil, logAndReturnError(logger, "sending provider docs request", err)
			}

			return buildResourceContents(response, providerVersionUri, logger)
		}
}

func buildResourceContents(response []byte, baseUri string, logger *log.Logger) ([]mcp.ResourceContents, error) {
	var providerDocs ProviderDocs
	if err := json.Unmarshal(response, &providerDocs); err != nil {
		return nil, logAndReturnError(logger, "unmarshalling provider docs", err)
	}

	resourceContents := make([]mcp.ResourceContents, len(providerDocs.Docs))
	for i, doc := range providerDocs.Docs {
		content := fmt.Sprintf("## %s \n\n**Id:** %s \n\n**Category:** %s\n\n**Subcategory:** %s\n\n**Path:** %s\n\n",
			doc.Title, doc.ID, doc.Category, doc.Subcategory, doc.Path)
		resourceContents[i] = mcp.TextResourceContents{
			MIMEType: "text/markdown",
			URI:      fmt.Sprintf("%s/%s", baseUri, doc.ID),
			Text:     content,
		}
	}
	return resourceContents, nil
}
