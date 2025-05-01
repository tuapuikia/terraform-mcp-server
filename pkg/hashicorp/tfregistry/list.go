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
			mcp.WithDescription("Get information about a terraform provider such as guides, examples, resources, data sources, etc."),
			mcp.WithString("name", mcp.Required(), mcp.Description("The name of the provider to retrieve")),
			mcp.WithString("namespace", mcp.Description("The namespace of the provider to retrieve")),
			mcp.WithString("version", mcp.Description("The version of the provider to retrieve")),
			mcp.WithString("sourceType", mcp.Description("The source type of the Terraform provider to retrieve, can be 'resources' or 'data-sources'")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			name := request.Params.Arguments["name"].(string)
			namespace := request.Params.Arguments["namespace"]
			version := request.Params.Arguments["version"]
			sourceType := request.Params.Arguments["sourceType"]

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

			uri := fmt.Sprintf("providers/%s/%s/%s", namespace, name, version)
			response, err := sendRegistryCall(registryClient, "GET", uri, logger)
			if err != nil {
				return nil, logAndReturnError(logger, "getting provider details", err)
			}

			var providerDocs ProviderDocs
			if err := json.Unmarshal(response, &providerDocs); err != nil {
				return nil, logAndReturnError(logger, "unmarshalling provider docs", err)
			}

			content := fmt.Sprintf("# %s provider docs\n\n", name)
			s, sourceTypeProvided := sourceType.(string) // Get the sourceType and check if it was provided

			for _, doc := range providerDocs.Docs {
				// Include the doc if sourceType was not provided/empty OR if the doc category matches the provided sourceType
				if !sourceTypeProvided || s == "" || doc.Category == s {
					if doc.Language == "hcl" {
						content += fmt.Sprintf("## %s \n\n**Id:** %s \n\n**Category:** %s\n\n**Subcategory:** %s\n\n**Path:** %s\n\n",
							doc.Title, doc.ID, doc.Category, doc.Subcategory, doc.Path)
					}
				}
			}
			return mcp.NewToolResultText(content), nil
		}
}

func providerResourceDetails(registryClient *http.Client, logger *log.Logger) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("providerResourceDetails",
			mcp.WithDescription("Retrieve details about deploying resources using a specific Terraform provider."),
			mcp.WithString("sourceType", mcp.Description("The source type of the Terraform provider to retrieve, resource or data-source")),
			mcp.WithString("name", mcp.Required(), mcp.Description("The name of the provider to retrieve")),
			mcp.WithString("sourceName", mcp.Required(), mcp.Description("The resource of the Terraform provider to retrieve")),
			mcp.WithString("namespace", mcp.Description("The namespace of the provider to retrieve")),
			mcp.WithString("version", mcp.Description("The version of the provider to retrieve")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			name := request.Params.Arguments["name"].(string)
			sourceName := request.Params.Arguments["sourceName"].(string)
			namespace := request.Params.Arguments["namespace"]
			version := request.Params.Arguments["version"]
			sourceType := request.Params.Arguments["sourceType"]
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

			content, err := GetProviderResourceDetails(registryClient, version, name, namespace, sourceName, sourceType, logger)
			if err != nil {
				return nil, err
			}

			if content == "" {
				content = fmt.Sprintf("Resource '%s' not found in the provider documentation", sourceName)
			}

			return mcp.NewToolResultText(content), nil
		}
}

const MODULE_BASE_PATH = "registry://modules"

func ListModules(registryClient *http.Client, logger *log.Logger) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	listModulesTool := mcp.NewTool("listModules",
		mcp.WithDescription("List Terraform modules based on name and namespace from the Terraform registry."),
		mcp.WithString("name",
			mcp.Description("The name of the modules to retrieve"),
		),
		mcp.WithString("namespace",
			mcp.Description("The namespace of the modules to retrieve"),
		),
		mcp.WithNumber("currentOffset",
			mcp.Description("Current offset for pagination"),
			mcp.Min(0),
			mcp.DefaultNumber(0),
		),
	)

	listModulesHandler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name := request.Params.Arguments["name"]
		namespace := request.Params.Arguments["namespace"]
		currentOffset := request.Params.Arguments["currentOffset"]

		response, err := getModuleDetails(registryClient, namespace, name, currentOffset, logger)
		if err != nil {
			logger.Errorf("Error getting modules: %v", err)
			return nil, err
		}

		var content *string
		if ns, ok := namespace.(string); !ok || ns == "" {
			content, err = UnmarshalTFModulePlural(response)
			if err != nil {
				logger.Errorf("Error unmarshalling modules: %v", err)
				return nil, err
			}
		} else {
			content, err = UnmarshalTFModuleSingular(response)
			if err != nil {
				logger.Errorf("Error unmarshalling module: %v", err)
				return nil, err
			}
		}

		return mcp.NewToolResultText(*content), nil
	}

	return listModulesTool, listModulesHandler
}

func getModuleDetails(providerClient *http.Client, namespace interface{}, name interface{}, currentOffset interface{}, logger *log.Logger) ([]byte, error) {
	// Clean up the URI
	uri := "modules"
	if ns, ok := namespace.(string); ok && ns != "" {
		if n, ok := name.(string); ok && n != "" {
			uri = fmt.Sprintf("%s/%s/%s", uri, ns, n)
		} else {
			uri = fmt.Sprintf("%s/%s", uri, ns)
		}
	}

	if cO, ok := currentOffset.(float64); ok {
		uri = fmt.Sprintf("%s?offset=%v", uri, cO)
	} else {
		uri = fmt.Sprintf("%s?offset=%v", uri, 0)
	}
	response, err := sendRegistryCall(providerClient, "GET", uri, logger)
	if err != nil {
		logger.Errorf("Error sending request: %v", err)
		return nil, fmt.Errorf("error sending request: %w", err)
	}

	// Return the filtered JSON as a string
	return response, nil
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
