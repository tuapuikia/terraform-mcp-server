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
			mcp.WithDescription(`This tool helps users deploy services on cloud, on-premise and SaaS application environments by retrieving a specific Terraform provider. 
			It helps users understand everything that can be provisioned and managed using the Terraform provider by listing out its resources (write operations), data sources (read operations), and functions (utility operations). 
			For each item, note the existence and path of its documentation.
			`),
			mcp.WithString("providerName", mcp.Required(), mcp.Description("The name of the Terraform provider to perform the read or deployment operation.")),
			mcp.WithString("providerNamespace", mcp.Required(), mcp.Description("The publisher of the Terraform provider, typically the name of the company, or their GitHub organization name that created the provider.")),
			mcp.WithString("providerVersion", mcp.Description("The version of the Terraform provider to retrieve in the format 'x.y.z', or 'latest' to get the latest version.")),
			mcp.WithString("providerDataType", mcp.Description("The source type of the Terraform provider to retrieve, can be 'resources' or 'data-sources'."),
				mcp.Enum("resources", "data-sources")), // TODO: Limitation due to the v1 API, we need to implement v2
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

			// For typical provider and namespace hallucinations
			defaultErrorGuide := "please check the provider name or the namespace, perhaps the provider is published under a different namespace or company name"
			providerName, providerNamespace, providerVersion, providerDataType, err := resolveProviderDetails(request, registryClient, defaultErrorGuide, logger)
			if err != nil {
				return nil, err
			}

			uri := fmt.Sprintf("providers/%s/%s/%s", providerNamespace, providerName, providerVersion)
			response, err := sendRegistryCall(registryClient, "GET", uri, logger)
			if err != nil {
				return nil, logAndReturnError(logger, "getting provider details", err)
			}

			var providerDocs ProviderDocs
			if err := json.Unmarshal(response, &providerDocs); err != nil {
				return nil, logAndReturnError(logger, "unmarshalling provider docs", err)
			}

			content := fmt.Sprintf("# %s provider docs\n\n", providerName)
			contentAvailable := false
			for _, doc := range providerDocs.Docs {
				// restrictData determines whether the data should be restricted based on the provider data type.
				// It evaluates to true if providerDataType is not empty and does not match the doc's category.
				restrictData := providerDataType != "" && providerDataType != doc.Category
				if !restrictData {
					if doc.Language == "hcl" {
						contentAvailable = true
						content += fmt.Sprintf("## %s \n\n**Id:** %s \n\n**Category:** %s\n\n**Subcategory:** %s\n\n**Path:** %s\n\n",
							doc.Title, doc.ID, doc.Category, doc.Subcategory, doc.Path)
					}
				}
			}

			// Check if the content data is not fulfilled
			if !contentAvailable {
				errMessage := fmt.Sprintf(`No documentation found for provider '%s' in the '%s' namespace, %s`, providerName, providerNamespace, defaultErrorGuide)
				return nil, logAndReturnError(logger, errMessage, err)
			}
			return mcp.NewToolResultText(content), nil
		}
}

func providerResourceDetails(registryClient *http.Client, logger *log.Logger) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("providerResourceDetails",
			mcp.WithDescription(`This tool is used to obtain the documentation, schema, and code examples from a given Terraform provider version, which will guide you in deploying a specific service on cloud, on-premise, and SaaS application environments. 
			Please specify the provider name, namespace, and the service name you wish to provision to utilize this tool.`),
			mcp.WithString("providerName", mcp.Required(), mcp.Description("The name of the Terraform provider to perform the read or deployment operation.")),
			mcp.WithString("providerNamespace", mcp.Required(), mcp.Description("The publisher of the Terraform provider, typically the name of the company or their GitHub organization name that created the provider.")),
			mcp.WithString("providerVersion", mcp.Description("The version of the Terraform provider to retrieve in the format 'x.y.z', or 'latest' to get the latest version.")),
			mcp.WithString("providerDataType", mcp.Description("The source type of the Terraform provider to retrieve, can be 'resources' or 'data-sources'."),
				mcp.Enum("resources", "data-sources")),
			mcp.WithString("serviceName", mcp.Required(), mcp.Description("The name of the service or resource for read or deployment operations.")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

			serviceName, ok := request.Params.Arguments["serviceName"].(string)
			if !ok || serviceName == "" {
				return nil, fmt.Errorf("serviceName is required and must be a string")
			}

			// For typical provider and namespace hallucinations
			defaultErrorGuide := "please check the provider name or the namespace, perhaps the provider is published under a different namespace or company name"
			providerName, providerNamespace, providerVersion, providerDataType, err := resolveProviderDetails(request, registryClient, defaultErrorGuide, logger)
			if err != nil {
				return nil, err
			}

			content, err := GetProviderResourceDetails(registryClient, providerVersion, providerName, providerNamespace, serviceName, providerDataType, logger)
			if err != nil {
				return nil, err
			}

			if content == "" {
				content = fmt.Sprintf("Resource '%s' not found in the provider documentation", serviceName)
			}

			return mcp.NewToolResultText(content), nil
		}
}

const MODULE_BASE_PATH = "registry://modules"

var providerToNamespaceModule = map[string]interface{}{
	"google":  []interface{}{"GoogleCloudPlatform", "terraform-google-modules"},
	"aws":     []interface{}{"aws-ia", "terraform-aws-modules"},
	"azurerm": []interface{}{"Azure", "aztfmod"},
	"oracle":  []interface{}{"oracle", "oracle-terraform-modules"},
	"alibaba": []interface{}{"alibaba", "terraform-alicloud-modules"},
}

func ListModules(registryClient *http.Client, logger *log.Logger) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("listModules",
			mcp.WithDescription(`This tool helps users deploy complex services on cloud and on-premise environments by retrieving a list of Terraform modules.
			Please specify the provider name to utilize this tool. You can also use this tool without specifying a provider to get a list of all available modules.`),
			mcp.WithString("moduleProvider",
				mcp.Description("The name of the provider for the Terraform module to use."),
			),
			mcp.WithNumber("currentOffset",
				mcp.Description("Current offset for pagination"),
				mcp.Min(0),
				mcp.DefaultNumber(0),
			),
		), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			moduleProvider := request.Params.Arguments["moduleProvider"]
			currentOffset := request.Params.Arguments["currentOffset"]
			var modulesData string
			if moduleProvider == nil {
				response, err := getModuleDetails(registryClient, nil, nil, nil, currentOffset, logger)
				if err != nil {
					logger.Errorf("Error getting modules: %v", err)
					return nil, err
				}
				content, err := UnmarshalTFModulePlural(response)
				if err != nil {
					logger.Errorf("Error unmarshalling modules: %v", err)
					return nil, err
				}
				modulesData += *content
				return mcp.NewToolResultText(modulesData), nil
			}

			for _, namespace := range providerToNamespaceModule[moduleProvider.(string)].([]interface{}) {
				response, err := getModuleDetails(registryClient, namespace, nil, nil, currentOffset, logger)
				if err != nil {
					logger.Errorf("Error listing modules: %v", err)
					return nil, err
				}

				content, err := UnmarshalTFModulePlural(response)
				if err != nil {
					logger.Errorf("Error unmarshalling modules list: %v", err)
					return nil, err
				}
				modulesData += *content
			}
			return mcp.NewToolResultText(modulesData), nil
		}
}

func ModuleDetails(registryClient *http.Client, logger *log.Logger) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("moduleDetails",
			mcp.WithDescription(`This tool provides comprehensive details about a Terraform module, including inputs, outputs, and examples, enabling users to understand its effective usage. 
		To use it, please specify the module name and its associated provider.`),
			mcp.WithString("moduleName",
				mcp.Required(),
				mcp.Description("The name of the module to to access its detailed information."),
			),
			// TODO: We shouldn't need to include provider as an input, we could potentially grab the provider value from first GET and then perform a second GET with the provider value
			mcp.WithString("moduleProvider",
				mcp.Required(),
				mcp.Description("The provider associated with the module, used to determine the correct namespace or the publisher of the module."),
			),
		), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			moduleName := request.Params.Arguments["moduleName"]
			moduleProvider := request.Params.Arguments["moduleProvider"]
			if _, ok := moduleProvider.(string); !ok {
				moduleProvider = ""
			}
			var detailData string
			for _, moduleNamespace := range providerToNamespaceModule[moduleProvider.(string)].([]interface{}) {
				response, err := getModuleDetails(registryClient, moduleNamespace, moduleName, moduleProvider, nil, logger)
				if err != nil {
					logger.Errorf("Error getting module details: %v", err)
					return nil, err
				}

				content, err := UnmarshalTFModuleSingular(response)
				if err != nil {
					logger.Errorf("Error unmarshalling module details: %v", err)
					return nil, err
				}
				detailData += *content
			}

			return mcp.NewToolResultText(detailData), nil
		}
}

func getModuleDetails(providerClient *http.Client, namespace interface{}, name interface{}, provider interface{}, currentOffset interface{}, logger *log.Logger) ([]byte, error) {
	// Clean up the URI
	uri := "modules"
	if ns, ok := namespace.(string); ok && ns != "" {
		if n, ok := name.(string); ok && n != "" {
			uri = fmt.Sprintf("%s/%s/%s/%s", uri, ns, n, provider) // single module
		} else {
			uri = fmt.Sprintf("%s/%s", uri, ns) // plural module
		}
	}

	if cO, ok := currentOffset.(float64); ok {
		uri = fmt.Sprintf("%s?offset=%v", uri, cO)
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
