// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfregistry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	log "github.com/sirupsen/logrus"
)

const PROVIDER_BASE_PATH = "registry://providers"

func GetProviderList(providerClient *http.Client, providerType string, logger *log.Logger) ([]map[string]string, error) {
	uri := fmt.Sprintf("providers?filter[tier]=%s", providerType)
	jsonData, err := sendRegistryCall(providerClient, "GET", uri, logger, "v2")
	if err != nil {
		return nil, logAndReturnError(logger, fmt.Sprintf("%s provider API request", providerType), err)
	}

	var providerListJson ProviderList
	if err := json.Unmarshal(jsonData, &providerListJson); err != nil {
		return nil, logAndReturnError(logger, fmt.Sprintf("%s providers request unmarshalling", providerType), err)
	}

	providerDetails := make([]map[string]string, len(providerListJson.Data))

	for i, provider := range providerListJson.Data {
		providerDetails[i] = map[string]string{
			"name":        provider.Attributes.Name,
			"namespace":   provider.Attributes.Namespace,
			"description": provider.Attributes.Namespace,
			"downloads":   provider.Attributes.Namespace,
		}
	}
	return providerDetails, nil
}

// Every provider version has a unique ID, which is used to identify the provider version in the registry and its specific documentation
// https://registry.terraform.io/v2/providers/hashicorp/aws?include=provider-versions
func GetProviderVersionID(registryClient *http.Client, namespace string, name string, version string, logger *log.Logger) (string, error) {
	uri := fmt.Sprintf("providers/%s/%s?include=provider-versions", namespace, name)
	response, err := sendRegistryCall(registryClient, "GET", uri, logger, "v2")
	if err != nil {
		return "", logAndReturnError(logger, "provider version ID request", err)
	}
	var providerVersionList ProviderVersionList
	if err := json.Unmarshal(response, &providerVersionList); err != nil {
		return "", logAndReturnError(logger, "provider version ID request unmarshalling", err)
	}
	for _, providerVersion := range providerVersionList.Included {
		if providerVersion.Attributes.Version == version {
			return providerVersion.ID, nil
		}
	}
	return "", fmt.Errorf("provider version %s not found", version)
}

func GetProviderOverviewDocs(registryClient *http.Client, providerVersionID string, logger *log.Logger) (string, error) {
	// https://registry.terraform.io/v2/provider-docs?filter[provider-version]=21818&filter[category]=overview&filter[slug]=index
	uri := fmt.Sprintf("provider-docs?filter[provider-version]=%s&filter[category]=overview&filter[slug]=index", providerVersionID)
	response, err := sendRegistryCall(registryClient, "GET", uri, logger, "v2")
	if err != nil {
		return "", logAndReturnError(logger, "getting provider docs overview", err)
	}
	var providerOverview ProviderOverview
	if err := json.Unmarshal(response, &providerOverview); err != nil {
		return "", logAndReturnError(logger, "getting provider docs request unmarshalling", err)
	}

	resourceContent := ""
	for _, providerOverviewPage := range providerOverview.Data {
		resourceContentNew, err := GetProviderResouceDocs(registryClient, providerOverviewPage.ID, logger)
		resourceContent += resourceContentNew
		if err != nil {
			return "", logAndReturnError(logger, "getting provider resource docs looping", err)
		}
	}

	return resourceContent, nil
}

func GetProviderDocs(registryClient *http.Client, providerVersionID string, dataCategory string, logger *log.Logger) (string, error) {
	// https://registry.terraform.io/v2/provider-versions/70800?include=provider-docs&filter[language]=hcl
	uri := fmt.Sprintf("provider-versions/%s?include=provider-docs&filter[language]=hcl", providerVersionID)
	if dataCategory != "" {
		uri += fmt.Sprintf("&filter[category]=%s", dataCategory)
	}
	response, err := sendRegistryCall(registryClient, "GET", uri, logger, "v2")
	if err != nil {
		return "", logAndReturnError(logger, "Error getting provider docs", err)
	}
	var providerVersionResponse ProviderVersionResponse
	if err := json.Unmarshal(response, &providerVersionResponse); err != nil {
		return "", logAndReturnError(logger, "Error getting provider docs request unmarshalling", err)
	}
	content := fmt.Sprintf("# Provider: %s\n", providerVersionResponse.Data.Attributes.Description)
	content += fmt.Sprintf("## Total downloads for provider version %s: %d\n\n", providerVersionResponse.Data.Attributes.Version, providerVersionResponse.Data.Attributes.Downloads)

	for _, providerDetails := range providerVersionResponse.Included {
		resourceContent, err := GetProviderResouceDocs(registryClient, providerDetails.ID, logger)
		if err != nil {
			return "", logAndReturnError(logger, "Error getting provider resource docs", err)
		}
		content += fmt.Sprintf("%s \n\n", resourceContent)
	}
	return content, fmt.Errorf("provider version %s not found", providerVersionID)
}

func GetProviderResouceDocs(registryClient *http.Client, providerDocsID string, logger *log.Logger) (string, error) {
	// https://registry.terraform.io/v2/provider-docs/8862001
	uri := fmt.Sprintf("provider-docs/%s", providerDocsID)
	response, err := sendRegistryCall(registryClient, "GET", uri, logger, "v2")
	if err != nil {
		return "", logAndReturnError(logger, "Error getting provider resource docs ", err)
	}
	var providerServiceDetails ProviderResourceDetails
	if err := json.Unmarshal(response, &providerServiceDetails); err != nil {
		return "", logAndReturnError(logger, "Error unmarshalling provider resource docs", err)
	}
	return providerServiceDetails.Data.Attributes.Content, nil
}

func ExtractProviderNameAndVersion(uri string) (string, string, string) {
	uri = strings.TrimPrefix(uri, fmt.Sprintf("%s/", PROVIDER_BASE_PATH))
	parts := strings.Split(uri, "/")
	return parts[0], parts[2], parts[4]
}

func ConstructProviderVersionURI(providerNamespace interface{}, providerName string, providerVersion interface{}) string {
	return fmt.Sprintf("%s/%s/providers/%s/versions/%s", PROVIDER_BASE_PATH, providerNamespace, providerName, providerVersion)
}

func GetLatestProviderVersion(providerClient *http.Client, providerNamespace, providerName interface{}, logger *log.Logger) (string, error) {
	uri := fmt.Sprintf("providers/%s/%s", providerNamespace, providerName)
	jsonData, err := sendRegistryCall(providerClient, "GET", uri, logger, "v1")
	if err != nil {
		return "", logAndReturnError(logger, "latest provider version API request", err)
	}

	var providerVersionLatest ProviderVersionLatest
	if err := json.Unmarshal(jsonData, &providerVersionLatest); err != nil {
		return "", logAndReturnError(logger, "provider versions request unmarshalling", err)
	}

	logger.Debugf("Fetched latest provider version: %s", providerVersionLatest.Version)

	return providerVersionLatest.Version, nil
}

func GetProviderResourceDetails(client *http.Client, providerDetail ProviderDetail, serviceName string, logger *log.Logger) (string, error) {
	var content string

	uri := fmt.Sprintf("providers/%s/%s/%s", providerDetail.ProviderNamespace, providerDetail.ProviderName, providerDetail.ProviderVersion)
	response, err := sendRegistryCall(client, "GET", uri, logger)
	if err != nil {
		return "", logAndReturnError(logger, "getting provider details", err)
	}

	var providerDocs ProviderDocs
	if err := json.Unmarshal(response, &providerDocs); err != nil {
		return "", logAndReturnError(logger, "unmarshalling provider docs", err)
	}

	content = fmt.Sprintf("# %s provider docs\n\n", providerDetail.ProviderName)
	for _, doc := range providerDocs.Docs {
		// restrictData determines whether the data should be restricted based on the provider data type.
		// It evaluates to true if providerDataType is not empty and does not match the doc's category.
		restrictData := providerDetail.ProviderDataType != "" && providerDetail.ProviderDataType != doc.Category
		if !restrictData {
			if match, err := containsSlug(serviceName, doc.Slug); err == nil && match && doc.Language == "hcl" {
				response, err := sendRegistryCall(client, "GET", fmt.Sprintf("provider-docs/%s", doc.ID), logger, "v2")
				if err != nil {
					logger.Errorf("Error sending request for provider-docs/%s: %v", doc.ID, err)
					continue
				}
				var details ProviderResourceDetails
				if err := json.Unmarshal(response, &details); err == nil {
					content += details.Data.Attributes.Content
				} else {
					logger.Errorf("Error unmarshalling provider resource details: %v", err)
				}
			} else if err != nil {
				logger.Errorf("Error checking slug match: %v", err)
			}
		}
	}
	return content, nil
}

// containsSlug checks if the sourceName string contains the slug string anywhere within it.
// It safely handles potential regex metacharacters in the slug.
// TODO: include a unit test for this
func containsSlug(sourceName, slug string) (bool, error) {
	// Use regexp.QuoteMeta to escape any special regex characters in the slug.
	// This ensures the slug is treated as a literal string in the pattern.
	escapedSlug := regexp.QuoteMeta(slug)

	// Construct the regex pattern dynamically: ".*" + escapedSlug + ".*"
	// This pattern means "match any characters, then the escaped slug, then any characters".
	pattern := ".*" + escapedSlug + ".*"

	// regexp.MatchString compiles and runs the regex against the sourceName.
	// It returns true if a match is found, false otherwise.
	// It also returns an error if the pattern is invalid (unlikely here due to QuoteMeta).
	matched, err := regexp.MatchString(pattern, sourceName)
	if err != nil {
		fmt.Printf("Error compiling or matching regex pattern '%s': %v\n", pattern, err)
		return false, err // Propagate the error
	}

	return matched, nil
}

// isValidProviderVersionFormat checks if the provider version format is valid.
func isValidProviderVersionFormat(version string) bool {
	// Example regex for semantic versioning (e.g., "1.0.0", "1.0.0-beta").
	semverRegex := `^v?(\d+\.\d+\.\d+(-[a-zA-Z0-9]+)?)$`
	matched, _ := regexp.MatchString(semverRegex, version)
	return matched
}

func isValidProviderDataType(providerDataType string) bool {
	return providerDataType == "resources" || providerDataType == "data-sources" || providerDataType == "provider-guides"
}

func resolveProviderDetails(request mcp.CallToolRequest, registryClient *http.Client, defaultErrorGuide string, logger *log.Logger) (ProviderDetail, error) {
	providerDetail := ProviderDetail{}
	providerName, ok := request.Params.Arguments["providerName"].(string)
	if !ok || providerName == "" {
		return providerDetail, fmt.Errorf("providerName is required and must be a string")
	}

	providerNamespace, ok := request.Params.Arguments["providerNamespace"].(string)
	if !ok || providerNamespace == "" {
		logger.Debugf(`Error getting latest provider version in "%s" namespace, trying the hashicorp namespace`, providerNamespace)
		providerNamespace = "hashicorp"
	}

	providerVersion := request.Params.Arguments["providerVersion"]
	providerDataType := request.Params.Arguments["providerDataType"]

	var err error
	providerVersionValue := ""
	if v, ok := providerVersion.(string); ok && isValidProviderVersionFormat(v) {
		providerVersionValue = v
	} else {
		providerVersionValue, err = GetLatestProviderVersion(registryClient, providerNamespace, providerName, logger)
		if err != nil {
			providerVersionValue = ""
			logger.Debugf("Error getting latest provider version in %s namespace: %v", providerNamespace, err)
		}
	}

	// If the provider version doesn't exist, try the hashicorp namespace
	if providerVersionValue == "" {
		tryProviderNamespace := "hashicorp"
		providerVersionValue, err = GetLatestProviderVersion(registryClient, tryProviderNamespace, providerName, logger)
		if err != nil {
			// Just so we don't print the same namespace twice if they are the same
			if providerNamespace != tryProviderNamespace {
				tryProviderNamespace = fmt.Sprintf(`"%s" or the "%s"`, providerNamespace, tryProviderNamespace)
			}
			return providerDetail, logAndReturnError(logger, fmt.Sprintf(`Error getting the "%s" provider, 
			with version "%s" in the %s namespace, %s`, providerName, providerVersion, tryProviderNamespace, defaultErrorGuide), nil)
		}
		providerNamespace = tryProviderNamespace // Update the namespace to hashicorp, if successful
	}

	providerDataTypeValue := ""
	if pdt, ok := providerDataType.(string); ok && isValidProviderDataType(pdt) {
		providerDataTypeValue = pdt
	}

	providerDetail.ProviderName = providerName
	providerDetail.ProviderNamespace = providerNamespace
	providerDetail.ProviderVersion = providerVersionValue
	providerDetail.ProviderDataType = providerDataTypeValue
	return providerDetail, nil
}

const MODULE_BASE_PATH = "registry://modules"

var providerToNamespaceModule = map[string]interface{}{
	"aws":      []interface{}{"aws-ia", "terraform-aws-modules"},
	"azurerm":  []interface{}{"Azure", "aztfmod"},
	"google":   []interface{}{"GoogleCloudPlatform", "terraform-google-modules"},
	"ibm":      []interface{}{"ibm", "terraform-ibm-modules"},
	"oracle":   []interface{}{"oracle", "oracle-terraform-modules"},
	"alibaba":  []interface{}{"alibaba", "terraform-alicloud-modules"},
	"aviatrix": []interface{}{"aviatrix", "terraform-aviatrix-modules"},
}

func GetModuleDetails(providerClient *http.Client, namespace string, name string, logger *log.Logger) ([]byte, string, error) {
	// Clean up the URI
	moduleUri := "registry://modules"
	uri := "modules"
	if namespace != "" {
		moduleUri = fmt.Sprintf("%s/%s/%s", moduleUri, namespace, name)
		uri = fmt.Sprintf("%s/%s/%s", uri, namespace, name)
	}
	// Get the provider versions
	response, err := sendRegistryCall(providerClient, "GET", uri, logger)
	if err != nil {
		logger.Errorf("Error sending request: %v", err)
		return nil, moduleUri, fmt.Errorf("error sending request: %w", err)
	}

	// Return the filtered JSON as a string
	return response, moduleUri, nil
}

func getModuleDetails(providerClient *http.Client, moduleDetail ModuleDetail, currentOffset int, logger *log.Logger) ([]byte, error) {
	// Clean up the URI
	uri := "modules"
	if ns := moduleDetail.ModuleNamespace; ns != "" {
		if n := moduleDetail.ModuleName; n != "" {
			uri = fmt.Sprintf("%s/%s/%s/%s", uri, ns, n, moduleDetail.ModuleProvider) // single module
		} else {
			uri = fmt.Sprintf("%s/%s", uri, ns) // plural module
		}
	}

	uri = fmt.Sprintf("%s?offset=%v", uri, currentOffset)
	response, err := sendRegistryCall(providerClient, "GET", uri, logger)
	if err != nil {
		return nil, logAndReturnError(logger, fmt.Sprintf("getting module(s) for provider %s, not found please provider a better provider name like aws, azurerm or google etc.", moduleDetail.ModuleProvider), err)
	}

	// Return the filtered JSON as a string
	return response, nil
}

func UnmarshalTFModulePlural(response []byte) (*string, error) {
	// Get the list of modules
	var terraformModules TerraformModules
	err := json.Unmarshal(response, &terraformModules)
	if err != nil {
		return nil, logAndReturnError(nil, "unmarshalling modules", err)
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

func UnmarshalModuleSingular(response []byte) (*string, error) {
	// Handles one module
	var terraformModules TerraformModuleVersionDetails
	err := json.Unmarshal(response, &terraformModules)
	if err != nil {
		return nil, logAndReturnError(nil, "unmarshalling module details", err)
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

func sendRegistryCall(client *http.Client, method string, uri string, logger *log.Logger, callOptions ...string) ([]byte, error) {
	version := "v1"
	if len(callOptions) > 0 {
		version = callOptions[0] // API version will be the first optional arg to this function
	}

	url := fmt.Sprintf("https://registry.terraform.io/%s/%s", version, uri)
	logger.Debugf("Requested URL: %s", url)

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "MCP-Client")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error: %s", "404 Not Found")
	}

	defer resp.Body.Close()
	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	logger.Debugf("Response status: %s", resp.Status)
	logger.Tracef("Response body: %s", string(body))
	return body, nil
}

func logAndReturnError(logger *log.Logger, context string, err error) error {
	if err == nil {
		err = fmt.Errorf("%s", context)
	}
	logger.Errorf("Error in %s: %v", context, err)
	return err
}
