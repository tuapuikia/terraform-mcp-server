package tfregistry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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
			mcp.WithString("sourceType", mcp.Description("The source type of the Terraform provider to retrieve, can be 'resources' or 'data-sources'")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// TODO: Parse pagination options
			// pageNumber, _ := OptionalParam[int](request, "page_number")
			// pageSize, _ := OptionalParam[int](request, "page_size")

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

			providerUri := ConstructProviderVersionURI(namespace, name, version)
			logger.Debugf("Constructed provider URI: %s", providerUri)

			providerVersionID, _, err := GetProviderDetails(registryClient, providerUri, logger)
			if err != nil {
				return nil, logAndReturnError(logger, "getting provider details", err)
			}
			var uri string
			content := fmt.Sprintf("# %s provider docs\n\n", name)
			var pageNumber float64 = 1
			for {
				if sourceType, ok := sourceType.(string); ok && sourceType != "" {
					uri = fmt.Sprintf("provider-docs?filter[provider-version]=%s&filter[category]=%s&page[number]=%v", providerVersionID, sourceType, pageNumber)
				} else {
					uri = fmt.Sprintf("provider-docs?filter[provider-version]=%s&page[number]=%v", providerVersionID, pageNumber)
				}

				response, err := sendRegistryCall(registryClient, "GET", uri, logger, "v2")
				if err != nil {
					return nil, logAndReturnError(logger, "sending provider docs request", err)
				}

				var providerDocs ProviderDocs
				if err := json.Unmarshal(response, &providerDocs); err != nil {
					return nil, logAndReturnError(logger, "unmarshalling provider docs", err)
				}

				if len(providerDocs.Data) == 0 {
					break
				} else {
					for _, doc := range providerDocs.Data {
						content += fmt.Sprintf("## %s \n\n**Id:** %s \n\n**Category:** %s\n\n**Subcategory:** %s\n\n**Path:** %s\n\n",
							doc.Attributes.Title, doc.ID, doc.Attributes.Category, doc.Attributes.Subcategory, doc.Attributes.Path)
					}
				}
				pageNumber++
			}
			return mcp.NewToolResultText(content), nil
		}
}

func providerResourceDetails(registryClient *http.Client, logger *log.Logger) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("providerResourceDetails",
			mcp.WithDescription("Retrieve details about deploying resources using a specific Terraform provider."),
			mcp.WithString("sourceType", mcp.Description("The source type of the Terraform provider to retrieve")),
			mcp.WithString("name", mcp.Required(), mcp.Description("The name of the provider to retrieve")),
			mcp.WithString("sourceName", mcp.Required(), mcp.Description("The resource of the Terraform provider to retrieve")),
			mcp.WithString("namespace", mcp.Description("The namespace of the provider to retrieve"), mcp.DefaultString("hashicorp")),
			mcp.WithString("version", mcp.Description("The version of the provider to retrieve"), mcp.DefaultString("latest")),
			mcp.WithNumber("pageNumber", mcp.Description("Page number"), mcp.DefaultNumber(1)),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// TODO: Parse pagination options
			// pageNumber, _ := OptionalParam[int](request, "page_number")
			// pageSize, _ := OptionalParam[int](request, "page_size")

			name := request.Params.Arguments["name"].(string)
			sourceName := request.Params.Arguments["sourceName"].(string)
			namespace := request.Params.Arguments["namespace"]
			version := request.Params.Arguments["version"]
			sourceType := request.Params.Arguments["sourceType"]
			pageNumber := request.Params.Arguments["pageNumber"]
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

			providerVersionID, _, err := GetProviderDetails(registryClient, providerUri, logger)
			if err != nil {
				return nil, logAndReturnError(logger, "retrieving provider details", err)
			}

			var sourceTypeSlice []string
			if s, ok := sourceType.(string); ok && s != "" {
				sourceTypeSlice = []string{s}
			} else {
				sourceTypeSlice = []string{"resources", "data-sources"}
			}
			content, err := GetProviderResourceDetails(registryClient, providerVersionID, sourceName, sourceTypeSlice, pageNumber, logger)
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
	listModulesTool := mcp.NewTool("list_modules",
		mcp.WithDescription("List modules."),
		mcp.WithString("namespace",
			mcp.DefaultString(""),
			mcp.Description("The namespace of the modules to retrieve"),
		),
		mcp.WithNumber("currentOffset",
			mcp.Description("Current offset for pagination"),
			mcp.Min(0),
			mcp.DefaultNumber(0),
		),
		mcp.WithString("name",
			mcp.DefaultString(""),
			mcp.Description("The name of the module to retrieve"),
		),
		// TODO: We shouldn't need to include provider as an input, we could potentially grab the provider value from first GET and then perform a second GET with the provider value
		mcp.WithString("provider",
			mcp.DefaultString(""),
			mcp.Description("The provider to retrieve"),
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
		content += fmt.Sprintf("## %s \n\n**Description:** %s \n\n**Module Version:** %s\n\n**Namespace:** %s\n\n**Source:** %s\n\n",
			module.Name,
			module.Description,
			module.Version,
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

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("# %s/%s/%s\n\n", MODULE_BASE_PATH, terraformModules.Namespace, terraformModules.Name))
	builder.WriteString(fmt.Sprintf("**Description:** %s\n\n", terraformModules.Description))
	builder.WriteString(fmt.Sprintf("**Module Version:** %s\n\n", terraformModules.Version))
	builder.WriteString(fmt.Sprintf("**Namespace:** %s\n\n", terraformModules.Namespace))
	builder.WriteString(fmt.Sprintf("**Source:** %s\n\n", terraformModules.Source))

	// Format Inputs
	if len(terraformModules.Root.Inputs) > 0 {
		builder.WriteString("### Inputs\n\n")
		builder.WriteString("| Name | Type | Description | Default | Required |\n")
		builder.WriteString("|---|---|---|---|---|\n")
		for _, input := range terraformModules.Root.Inputs {
			builder.WriteString(fmt.Sprintf("| %s | %s | %s | `%v` | %t |\n",
				input.Name,
				input.Type,
				input.Description, // Consider cleaning potential newlines/markdown
				input.Default,
				input.Required,
			))
		}
		builder.WriteString("\n")
	}

	// Format Outputs
	if len(terraformModules.Root.Outputs) > 0 {
		builder.WriteString("### Outputs\n\n")
		builder.WriteString("| Name | Description |\n")
		builder.WriteString("|---|---|\n")
		for _, output := range terraformModules.Root.Outputs {
			builder.WriteString(fmt.Sprintf("| %s | %s |\n",
				output.Name,
				output.Description, // Consider cleaning potential newlines/markdown
			))
		}
		builder.WriteString("\n")
	}

	// Format Provider Dependencies
	if len(terraformModules.Root.ProviderDependencies) > 0 {
		builder.WriteString("### Provider Dependencies\n\n")
		builder.WriteString("| Name | Namespace | Source | Version |\n")
		builder.WriteString("|---|---|---|---|\n")
		for _, dep := range terraformModules.Root.ProviderDependencies {
			builder.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
				dep.Name,
				dep.Namespace,
				dep.Source,
				dep.Version,
			))
		}
		builder.WriteString("\n")
	}

	// Format Examples
	if len(terraformModules.Examples) > 0 {
		builder.WriteString("### Examples\n\n")
		for _, example := range terraformModules.Examples {
			builder.WriteString(fmt.Sprintf("#### %s\n\n", example.Name))
			// Optionally, include more details from example if needed, like inputs/outputs
			// For now, just listing the name.
			if example.Readme != "" {
				builder.WriteString("**Readme:**\n\n")
				// Append readme content, potentially needs markdown escaping/sanitization depending on source
				builder.WriteString(example.Readme)
				builder.WriteString("\n\n")
			}
		}
		builder.WriteString("\n")
	}

	content := builder.String()
	return &content, nil
}
