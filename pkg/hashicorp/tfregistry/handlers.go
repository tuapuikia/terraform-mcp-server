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

// ResolveProviderDocID creates a tool to get provider details from registry.
func ResolveProviderDocID(registryClient *http.Client, logger *log.Logger) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("resolveProviderDocID",
			mcp.WithDescription(`This tool retrieves a specific Terraform provider version. You MUST call this function before 'getProviderDocs' to obtain a valid tfprovider-compatible providerDocID. 
			When selecting the best match, consider: - Name similarity to the query - Description relevance Return the selected providerDocID and explain your choice. 
			If there are multiple good matches, mention this but proceed with the most relevant one.`),
			mcp.WithString("providerName", mcp.Required(), mcp.Description("The name of the Terraform provider to perform the read or deployment operation")),
			mcp.WithString("providerNamespace", mcp.Required(), mcp.Description("The publisher of the Terraform provider, typically the name of the company, or their GitHub organization name that created the provider")),
			mcp.WithString("serviceName", mcp.Required(), mcp.Description("The name of the service you want to deploy or read using the Terraform provider")),
			mcp.WithString("providerVersion", mcp.Description("The version of the Terraform provider to retrieve in the format 'x.y.z', or 'latest' to get the latest version")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

			// For typical provider and namespace hallucinations
			defaultErrorGuide := "please check the provider name, namespace or the service you're looking for, perhaps the provider is published under a different namespace or company name or service name is incorrect"
			providerDetail, err := resolveProviderDetails(request, registryClient, defaultErrorGuide, logger)
			if err != nil {
				return nil, err
			}

			serviceName, ok := request.Params.Arguments["serviceName"].(string)
			if !ok || serviceName == "" {
				return nil, logAndReturnError(logger, "serviceName is required and must be a string", nil)
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

			content := fmt.Sprintf("# %s provider docs\n\n Each result includes: \n\n- providerDocID: tfprovider-compatible identifier (format: Integer)\n- Title: Service or resource name\n- Category: Type of document (e.g., 'resources', 'data-sources', 'guides')\nFor best results, select libraries based on the name match. \n\n ---", providerDetail.ProviderName)
			contentAvailable := false
			for _, doc := range providerDocs.Docs {
				cs, err := containsSlug(doc.Slug, serviceName)
				cs_pn, err_pn := containsSlug(fmt.Sprintf("%s_%s", providerDetail.ProviderName, doc.Slug), serviceName)
				if doc.Language == "hcl" && (cs || cs_pn) && err == nil && err_pn == nil {
					contentAvailable = true
					content += fmt.Sprintf("\n- providerDocID: %s\n- Title: %s\n- Category: %s \n ---",
						doc.ID, doc.Title, doc.Category)
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

// GetProviderDocs creates a tool to get provider docs for a specific service from registry.
func GetProviderDocs(registryClient *http.Client, logger *log.Logger) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("getProviderDocs",
			mcp.WithDescription(`Fetches up-to-date documentation fora specific service from a Terraform provider. You must call 'resolveProviderDocID' first to obtain the exact tfprovider-compatible providerDocID required to use this tool.`),
			mcp.WithString("providerDocID", mcp.Required(), mcp.Description("Exact tfprovider-compatible providerDocID, (e.g., '8894603', '8906901') retrieved from 'resolveProviderDocID'")), // TODO: fix this
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			providerDocID, ok := request.Params.Arguments["providerDocID"].(string)
			if !ok || providerDocID == "" {
				return nil, fmt.Errorf("providerDocID is required and must be a string")
			}

			detailResp, err := sendRegistryCall(registryClient, "GET", fmt.Sprintf("provider-docs/%s", providerDocID), logger, "v2")
			if err != nil {
				return nil, logAndReturnError(logger, fmt.Sprintf("Error fetching provider-docs/%s, please make sure providerDocID is valid and the resolveProviderDocID tool has run prior", providerDocID), err)
			}

			var details ProviderResourceDetails
			if err := json.Unmarshal(detailResp, &details); err != nil {
				return nil, logAndReturnError(logger, fmt.Sprintf("error unmarshalling provider-docs/%s", providerDocID), err)
			}
			return mcp.NewToolResultText(details.Data.Attributes.Content), nil
		}
}

func SearchModules(registryClient *http.Client, logger *log.Logger) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	return mcp.NewTool("searchModules",
			mcp.WithDescription(`Resolves a Terraform module name to obtain a compatible moduleID for the moduleDetails tool and returns a list of matching Terraform modules. You MUST call this function before 'moduleDetails' to obtain a valid and compatible moduleID. When selecting the best match, consider: - Name similarity to the query - Description relevance - Verification status (verified) - Download counts (popularity) Return the selected moduleID and explain your choice. If there are multiple good matches, mention this but proceed with the most relevant one. If no modules were found, reattempt the search with a new moduleName query.`),
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
			if val, ok := currentOffset.(float64); ok {
				currentOffsetValue = int(val)
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
