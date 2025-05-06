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

	log "github.com/sirupsen/logrus"
)

const PROVIDER_BASE_PATH = "registry://providers"

func GetProviderList(providerClient *http.Client, providerType string, logger *log.Logger) ([]map[string]string, error) {
	uri := fmt.Sprintf("providers?filter[tier]=%s", providerType)
	jsonData, err := sendRegistryCall(providerClient, "GET", uri, logger, "v2")
	if err != nil {
		logError(logger, fmt.Sprintf("%s provider API request", providerType), err)
		return nil, err
	}

	var providerListJson ProviderList
	if err := json.Unmarshal(jsonData, &providerListJson); err != nil {
		logError(logger, fmt.Sprintf("%s providers request unmarshalling", providerType), err)
		return nil, err
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
		logError(logger, "provider version ID request", err)
		return "", err
	}
	var providerVersionList ProviderVersionList
	if err := json.Unmarshal(response, &providerVersionList); err != nil {
		logError(logger, "provider version ID request unmarshalling", err)
		return "", err
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
		logError(logger, "getting provider docs request unmarshalling", err)
		return "", err
	}

	resourceContent := ""
	for _, providerOverviewPage := range providerOverview.Data {
		resourceContentNew, err := GetProviderResouceDocs(registryClient, providerOverviewPage.ID, logger)
		resourceContent += resourceContentNew
		if err != nil {
			logError(logger, "getting provider resource docs looping", err)
			return "", err
		}
	}

	return resourceContent, nil
}

func GetProviderDocs(registryClient *http.Client, providerVersionID string, logger *log.Logger) (string, error) {
	// https://registry.terraform.io/v2/provider-versions/70800?include=provider-docs&filter[language]=hcl
	// TODO: Implement the logic to filter ?filter[category]=resources
	uri := fmt.Sprintf("provider-versions/%s?include=provider-docs&filter[language]=hcl", providerVersionID)
	response, err := sendRegistryCall(registryClient, "GET", uri, logger, "v2")
	if err != nil {
		return "", logAndReturnError(logger, "Error getting provider docs", err)
	}
	var providerVersionResponse ProviderVersionResponse
	if err := json.Unmarshal(response, &providerVersionResponse); err != nil {
		logError(logger, "Error getting provider docs request unmarshalling", err)
		return "", err
	}
	content := fmt.Sprintf("# Provider: %s\n", providerVersionResponse.Data.Attributes.Description)
	content += fmt.Sprintf("## Total downloads for provider version %s: %d\n\n", providerVersionResponse.Data.Attributes.Version, providerVersionResponse.Data.Attributes.Downloads)

	for _, providerDetails := range providerVersionResponse.Included {
		resourceContent, err := GetProviderResouceDocs(registryClient, providerDetails.ID, logger)
		if err != nil {
			logError(logger, "Error getting provider resource docs", err)
			return "", err
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
		logError(logger, "Error unmarshalling provider service details", err)
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
		logError(logger, "latest provider version API request", err)
		return "", err
	}

	var providerVersionLatest ProviderVersionLatest
	if err := json.Unmarshal(jsonData, &providerVersionLatest); err != nil {
		logError(logger, "provider versions request unmarshalling", err)
		return "", err
	}

	logger.Debugf("Fetched latest provider version: %s", providerVersionLatest.Version)

	return providerVersionLatest.Version, nil
}

func logError(logger *log.Logger, context string, err error) {
	logger.Errorf("Error in %s: %v", context, err)
}

func GetProviderResourceDetails(client *http.Client, version, providerName, providerNamespace, serviceName, providerDataType interface{}, logger *log.Logger) (string, error) {
	var content string

	uri := fmt.Sprintf("providers/%s/%s/%s", providerNamespace, providerName, version)
	response, err := sendRegistryCall(client, "GET", uri, logger)
	if err != nil {
		return "", logAndReturnError(logger, "getting provider details", err)
	}

	var providerDocs ProviderDocs
	if err := json.Unmarshal(response, &providerDocs); err != nil {
		return "", logAndReturnError(logger, "unmarshalling provider docs", err)
	}

	content = fmt.Sprintf("# %s provider docs\n\n", providerName)

	// Get the sourceType and check if it was provided, force it to be "resources" if any other value fails
	sourceTypeValue, sourceTypeProvided := providerDataType.(string)
	if !sourceTypeProvided || sourceTypeValue == "" || (sourceTypeValue != "resources" && sourceTypeValue != "data-sources" && sourceTypeValue != "provider-guides") {
		sourceTypeValue = "resources"
	}

	for _, doc := range providerDocs.Docs {
		// Include the doc to only provide info from the sourceType (enum - "resource", "data-source", "provider-guides")
		if doc.Category == sourceTypeValue {
			if match, err := containsSlug(serviceName.(string), doc.Slug); err == nil && match && doc.Language == "hcl" {
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

// isValidProviderVersionFormat checks if the provider version format is valid.
func isValidProviderVersionFormat(version string) bool {
	// Example regex for semantic versioning (e.g., "1.0.0", "1.0.0-beta").
	semverRegex := `^v?(\d+\.\d+\.\d+(-[a-zA-Z0-9]+)?)$`
	matched, _ := regexp.MatchString(semverRegex, version)
	return matched
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
	logger.Errorf("Error in %s: %v", context, err)
	return err
}
