package tfregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ProviderDetails creates a tool to get provider details from registry.
func ProviderDetails(registryClient *http.Client, logger *log.Logger) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("providerDetails",
			mcp.WithDescription("Get Terraform provider details by namespace, name and version from the Terraform registry."),
			mcp.WithString("name", mcp.Required(), mcp.Description("The name of the provider to retrieve")),
			mcp.WithString("namespace", mcp.Description("The namespace of the provider to retrieve"), mcp.DefaultString("hashicorp")),
			mcp.WithString("version", mcp.Description("The version of the provider to retrieve"), mcp.DefaultString("latest")),
			// TODO: Add pagination parameters here using the correct mcp-go mechanism
			// Example (conceptual):
			// mcp.WithInteger("page_number", mcp.Description("Page number"), mcp.Optional()),
			// mcp.WithInteger("page_size", mcp.Description("Page size"), mcp.Optional()),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// TODO: Parse pagination options
			// pageNumber, _ := OptionalParam[int](request, "page_number")
			// pageSize, _ := OptionalParam[int](request, "page_size")

			name := request.Params.Arguments["name"].(string)
			namespace := request.Params.Arguments["namespace"]
			version := request.Params.Arguments["version"]

			if ns, ok := namespace.(string); ok && ns != "" {
				namespace = ns
			} else {
				namespace = "hashicorp"
			}

			if v, ok := version.(string); ok && v != "" && v != "latest" {
				version = v
			} else {
				version = GetLatestProviderVersion(registryClient, namespace, name, logger)
			}

			providerUri := ConstructProviderVersionURI(namespace, name, version)
			logger.Debugf("Constructed provider URI: %s", providerUri)

			providerVersionID, providerVersionUri, err := GetProviderDetails(registryClient, providerUri, logger)
			if err != nil {
				return nil, logAndReturnError(logger, "getting provider details", err)
			}

			uri := fmt.Sprintf("provider-docs?filter[provider-version]=%s", providerVersionID)
			response, err := sendRegistryCall(registryClient, "GET", uri, logger, "v2")
			if err != nil {
				return nil, logAndReturnError(logger, "sending provider docs request", err)
			}

			var providerDocs ProviderDocs
			if err := json.Unmarshal(response, &providerDocs); err != nil {
				return nil, logAndReturnError(logger, "unmarshalling provider docs", err)
			}

			content := fmt.Sprintf("# %s provider docs\n\n", name)
			for _, doc := range providerDocs.Data {
				content += fmt.Sprintf("## %s \n\n**Id:** %s \n\n**Category:** %s\n\n**Subcategory:** %s\n\n**Path:** %s\n\n",
					doc.Attributes.Title, doc.ID, doc.Attributes.Category, doc.Attributes.Subcategory, doc.Attributes.Path)
			}

			resourceContent := mcp.TextResourceContents{
				MIMEType: "text/markdown",
				URI:      providerVersionUri,
				Text:     content,
			}
			return mcp.NewToolResultResource(providerVersionUri, resourceContent), nil
		}
}

func providerResourceDetails(registryClient *http.Client, logger *log.Logger) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("providerResourceDetails",
			mcp.WithDescription("Retrieve details about deploying resources using a specific Terraform provider."),
			mcp.WithString("name", mcp.Required(), mcp.Description("The name of the provider to retrieve")),
			mcp.WithString("namespace", mcp.Description("The namespace of the provider to retrieve"), mcp.DefaultString("hashicorp")),
			mcp.WithString("version", mcp.Description("The version of the provider to retrieve"), mcp.DefaultString("latest")),
			mcp.WithString("resource", mcp.Required(), mcp.Description("The resource of the Terraform provider to retrieve")),
			// TODO: Add pagination parameters here using the appropriate mcp-go mechanism
			// Example (conceptual):
			// mcp.WithInteger("page_number", mcp.Description("Page number"), mcp.Optional()),
			// mcp.WithInteger("page_size", mcp.Description("Page size"), mcp.Optional()),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// TODO: Parse pagination options
			// pageNumber, _ := OptionalParam[int](request, "page_number")
			// pageSize, _ := OptionalParam[int](request, "page_size")

			name := request.Params.Arguments["name"].(string)
			namespace := request.Params.Arguments["namespace"].(string)
			version := request.Params.Arguments["version"].(string)
			resource := request.Params.Arguments["resource"].(string)

			if version == "" || version == "latest" {
				version = GetLatestProviderVersion(registryClient, namespace, name, logger)
			}

			providerUri := ConstructProviderVersionURI(namespace, name, version)
			logger.Debugf("Constructed provider URI: %s", providerUri)

			providerVersionID, providerVersionUri, err := GetProviderDetails(registryClient, providerUri, logger)
			if err != nil {
				return nil, logAndReturnError(logger, "retrieving provider details", err)
			}

			content, err := GetProviderResourceDetails(registryClient, providerVersionID, resource, logger)
			if err != nil {
				return nil, err
			}

			if content == "" {
				content = fmt.Sprintf("Resource '%s' not found in the provider documentation", resource)
			}

			resourceContent := mcp.TextResourceContents{
				MIMEType: "text/markdown",
				URI:      providerVersionUri,
				Text:     content,
			}
			return mcp.NewToolResultResource(providerVersionUri, resourceContent), nil
		}
}

const MODULE_BASE_PATH = "registry://modules"

func ListModules(registryClient *http.Client, logger *log.Logger) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	listModulesTool := mcp.NewTool("listModules",
		mcp.WithDescription("List Terraform modules based on name and namespace from the Terraform registry."),
		mcp.WithString("name",
			mcp.DefaultString(""),
			mcp.Description("The name of the modules to retrieve"),
		),
		mcp.WithString("namespace",
			mcp.DefaultString(""),
			mcp.Description("The namespace of the modules to retrieve"),
		),
	)

	listModulesHandler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name := request.Params.Arguments["name"].(string)
		namespace := request.Params.Arguments["namespace"].(string)

		response, moduleUri, err := GetModuleDetails(registryClient, namespace, name, logger)
		if err != nil {
			logger.Errorf("Error getting modules: %v", err)
			return nil, err
		}

		var content *string
		content, err = UnmarshalTFModulePlural(response)
		if err != nil {
			logger.Errorf("Error unmarshalling modules: %v", err)
			return nil, err
		}
		if namespace == "" {

		} else {
			content, err = UnmarshalTFModuleSingular(response)
			if err != nil {
				logger.Errorf("Error unmarshalling module: %v", err)
				return nil, err
			}
		}

		resourceContent := mcp.TextResourceContents{
			MIMEType: "text/markdown",
			URI:      moduleUri,
			Text:     *content,
		}
		return mcp.NewToolResultResource(moduleUri, resourceContent), nil
	}

	return listModulesTool, listModulesHandler
}

func UnmarshalTFModulePlural(response []byte) (*string, error) {
	// Get the list of modules
	var terraformModules TerraformModules
	err := json.Unmarshal(response, &terraformModules)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling modules: %w", err)
	}

	content := fmt.Sprintf("# %s modules\n\n", MODULE_BASE_PATH)
	for _, module := range terraformModules.Data {
		content += fmt.Sprintf("## %s \n\n**Id:** %s \n\n**OwnerName:** %s\n\n**Namespace:** %s\n\n**Source:** %s\n\n",
			module.Name,
			module.ID,
			module.Owner,
			module.Namespace,
			module.Source,
		)
	}
	return &content, nil
}

func UnmarshalTFModuleSingular(response []byte) (*string, error) {
	// Handles one module
	var terraformModules TerraformModuleVersionDetails
	err := json.Unmarshal(response, &terraformModules)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling module: %w", err)
	}
	content := fmt.Sprintf("# %s modules\n\n", MODULE_BASE_PATH)
	content += fmt.Sprintf("## %s \n\n**Id:** %s \n\n**OwnerName:** %s\n\n**Namespace:** %s\n\n**Source:** %s\n\n",
		terraformModules.Name,
		terraformModules.ID,
		terraformModules.Owner,
		terraformModules.Namespace,
		terraformModules.Source,
		// TODO: Add more details
	)
	return &content, nil
}
