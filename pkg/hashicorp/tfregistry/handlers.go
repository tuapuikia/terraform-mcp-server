// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

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
			mcp.WithDescription(`This tool helps users deploy services on cloud, on-premise and SaaS application environments by retrieving a specific Terraform provider. 
			It helps users understand everything that can be provisioned and managed using the Terraform provider by listing out its resources (write operations), data sources (read operations), and functions (utility operations). 
			For each item, note the existence and path of its documentation.
			`),
			mcp.WithString("providerName", mcp.Required(), mcp.Description("The name of the Terraform provider to perform the read or deployment operation.")),
			mcp.WithString("providerNamespace", mcp.Required(), mcp.Description("The publisher of the Terraform provider, typically the name of the company, or their GitHub organization name that created the provider.")),
			mcp.WithString("providerVersion", mcp.Description("The version of the Terraform provider to retrieve in the format 'x.y.z', or 'latest' to get the latest version.")),
			mcp.WithString("providerDataType", mcp.Description("The source type of the Terraform provider to retrieve."),
				mcp.Enum("resources", "data-sources", "functions", "guides", "overview")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

			// For typical provider and namespace hallucinations
			defaultErrorGuide := "please check the provider name or the namespace, perhaps the provider is published under a different namespace or company name"
			providerDetail, err := resolveProviderDetails(request, registryClient, defaultErrorGuide, logger)
			if err != nil {
				return nil, err
			}

			// Check if we need to use v2 API for guides, functions, or overview
			if isV2ProviderDataType(providerDetail.ProviderDataType) {
				content, err := GetProviderDocsV2(registryClient, providerDetail, logger)
				if err != nil {
					errMessage := fmt.Sprintf(`No %s documentation found for provider '%s' in the '%s' namespace, %s`,
						providerDetail.ProviderDataType, providerDetail.ProviderName, providerDetail.ProviderNamespace, defaultErrorGuide)
					return nil, logAndReturnError(logger, errMessage, err)
				}

				fullContent := fmt.Sprintf("# %s provider docs\n\n%s",
					providerDetail.ProviderName, content)

				return mcp.NewToolResultText(fullContent), nil
			}

			// For resources/data-sources, use the v1 API for better performance (single response)
			uri := fmt.Sprintf("providers/%s/%s/%s", providerDetail.ProviderNamespace, providerDetail.ProviderName, providerDetail.ProviderVersion)
			response, err := sendRegistryCall(registryClient, "GET", uri, logger)
			if err != nil {
				return nil, logAndReturnError(logger, "getting provider details", err)
			}

			var providerDocs ProviderDocs
			if err := json.Unmarshal(response, &providerDocs); err != nil {
				return nil, logAndReturnError(logger, "unmarshalling provider docs", err)
			}

			content := fmt.Sprintf("# %s provider docs\n\n", providerDetail.ProviderName)
			contentAvailable := false
			for _, doc := range providerDocs.Docs {
				// restrictData determines whether the data should be restricted based on the provider data type.
				// It evaluates to true if providerDataType is not empty and does not match the doc's category.
				restrictData := providerDetail.ProviderDataType != "" && providerDetail.ProviderDataType != doc.Category
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
				errMessage := fmt.Sprintf(`No documentation found for provider '%s' in the '%s' namespace, %s`, providerDetail.ProviderName, providerDetail.ProviderNamespace, defaultErrorGuide)
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
				mcp.Enum("resources", "data-sources", "functions", "guides")),
			mcp.WithString("serviceName", mcp.Required(), mcp.Description("The name of the service or resource for read or deployment operations.")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

			serviceName, ok := request.Params.Arguments["serviceName"].(string)
			if !ok || serviceName == "" {
				return nil, fmt.Errorf("serviceName is required and must be a string")
			}

			// For typical provider and namespace hallucinations
			defaultErrorGuide := "please check the provider name or the namespace, perhaps the provider is published under a different namespace or company name"
			providerDetail, err := resolveProviderDetails(request, registryClient, defaultErrorGuide, logger)
			if err != nil {
				return nil, err
			}

			var content string
			if isV2ProviderDataType(providerDetail.ProviderDataType) {
				content, err = GetProviderResourceDetailsV2(registryClient, providerDetail, serviceName, logger)
			} else {
				content, err = GetProviderResourceDetails(registryClient, providerDetail, serviceName, logger)
			}
			if err != nil {
				return nil, err
			}

			if content == "" {
				content = fmt.Sprintf("Resource '%s' not found in the provider documentation", serviceName)
			}

			return mcp.NewToolResultText(content), nil
		}
}

func SearchModules(registryClient *http.Client, logger *log.Logger) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("searchModules",
			mcp.WithDescription(`This tool helps users deploy complex services on cloud and on-premise environments by searching for a list of Terraform modules. It resolves a module name to obtain a compatible moduleID for the moduleDetails tool and returns a list of matching libraries. You MUST call this function before 'moduleDetails' to obtain a valid and compatible moduleID. When selecting the best match, consider: - Name similarity to the query - Description relevance - Code Snippet count (documentation coverage) - Download counts (popularity) Return the selected moduleID and explain your choice. If there are multiple good matches, mention this but proceed with the most relevant one. If no modules were found, reattempt the search with a new moduleName query.`),
			mcp.WithString("moduleQuery",
				mcp.Required(),
				mcp.Description("The query to search for Terraform modules."),
			),
			mcp.WithNumber("currentOffset",
				mcp.Description("Current offset for pagination"),
				mcp.Min(0),
				mcp.DefaultNumber(0),
			),
		), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			moduleQuery := request.Params.Arguments["moduleQuery"]
			currentOffset := request.Params.Arguments["currentOffset"]
			currentOffsetValue := 0
			if _, ok := currentOffset.(int); ok {
				currentOffsetValue = currentOffset.(int)
			}

			if mq, ok := moduleQuery.(string); !ok {
				return nil, logAndReturnError(logger, "error finding the module name;", nil)
			} else {
				var modulesData, errMsg string
				response, err := searchModules(registryClient, mq, currentOffsetValue, logger)
				if err != nil {
					return nil, logAndReturnError(logger, fmt.Sprintf("no module(s) found for moduleName: %s", mq), err)
				} else {
					modulesData, err = UnmarshalTFModulePlural(response, mq)
					if err != nil {
						return nil, logAndReturnError(logger, fmt.Sprintf("unmarshalling modules for moduleName: %s", mq), err)
					}
				}

				if modulesData == "" {
					errMsg = fmt.Sprintf("getting module(s), none found! query used: %s; error: %s", mq, errMsg)
					return nil, logAndReturnError(logger, errMsg, nil)
				}
				return mcp.NewToolResultText(modulesData), nil
			}
		}
}

func ModuleDetails(registryClient *http.Client, logger *log.Logger) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("moduleDetails",
			mcp.WithDescription(`Fetches up-to-date documentation on how to use a Terraform module. You must call 'searchModules' first to obtain the exact valid and compatible moduleID required to use this tool.`),
			mcp.WithString("moduleID",
				mcp.Required(),
				mcp.Description("Exact valid and compatible moduleID retrieved from searchModules (e.g., 'squareops/terraform-kubernetes-mongodb/mongodb/2.1.1', 'GoogleCloudPlatform/vertex-ai/google/0.2.0')"),
			),
		), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			moduleID := request.Params.Arguments["moduleID"]
			
			if mn, ok := moduleID.(string); !ok || mn == "" {
				return nil, logAndReturnError(logger, "moduleID is required and must be a valid string. It represents the ID of the module to retrieve detailed information about", nil)
			} else {
				var errMsg string
				response, err := GetModuleDetails(registryClient, mn, 0, logger)
				if err != nil {
					errMsg = fmt.Sprintf("no module(s) found for %v,", mn)
					return nil, logAndReturnError(logger, errMsg, nil)
				}
				moduleData, err := UnmarshalModuleSingular(response)
				if err != nil {
					return nil, logAndReturnError(logger, "unmarshalling module details", err)
				}
				if moduleData == "" {
					errMsg = fmt.Sprintf("getting module(s), none found! %s please provider a different moduleProvider", errMsg)
					return nil, logAndReturnError(logger, errMsg, nil)
				}
				return mcp.NewToolResultText(moduleData), nil
				}
		}
}
