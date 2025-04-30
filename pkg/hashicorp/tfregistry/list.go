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

const MODULE_BASE_PATH = "registry://modules"

func ListModules(registryClient *http.Client, logger *log.Logger) (tool mcp.Tool, handler server.ToolHandlerFunc) {
	listModulesTool := mcp.NewTool("list_modules",
		mcp.WithDescription("List modules."),
		mcp.WithString("name",
			mcp.DefaultString(""),
			mcp.Description("The name of the provider to retrieve"),
		),
		mcp.WithString("namespace",
			mcp.DefaultString(""),
			mcp.Description("The namespace of the provider to retrieve"),
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
	response, err := SendRegistryCall(providerClient, "GET", uri, logger)
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
