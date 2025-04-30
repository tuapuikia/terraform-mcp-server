package tfregistry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

func GetProviderDetails(providerClient *http.Client, uri string, logger *log.Logger) (string, string, error) {
	// Clean up the URI
	namespace, name, version := ExtractProviderNameAndVersion(uri)
	logger.Debugf("Extracted namespace: %s, name: %s, version: %s", namespace, name, version)

	if version == "" || version == "latest" {
		version = GetLatestProviderVersion(providerClient, namespace, name, logger)
	}
	providerVersionUri := fmt.Sprintf("registry://provider/%s/name/%s/version/%s", namespace, name, version)

	// Get the provider versions
	uri = fmt.Sprintf("providers/%s/%s?include=provider-versions", namespace, name)
	response, err := sendRegistryCall(providerClient, "GET", uri, logger, "v2")
	if err != nil {
		logger.Errorf("Error sending request: %v", err)
		return "", providerVersionUri, fmt.Errorf("error sending request: %w", err)
	}
	// Get the provider version-id
	providerVersionID, err := GetProviderVersionID(response, version)
	if err != nil {
		logger.Errorf("Error getting provider version ID: %v", err)
		return "", providerVersionUri, fmt.Errorf("error getting provider version ID: %w", err)
	}

	// Return the filtered JSON as a string
	return providerVersionID, providerVersionUri, nil
}

func GetProviderList(providerClient *http.Client, providerType string, logger *log.Logger) []string {
	uri := fmt.Sprintf("providers?filter[tier]=%s", providerType)
	jsonData, err := sendRegistryCall(providerClient, "GET", uri, logger, "v2")
	if err != nil {
		logError(logger, fmt.Sprintf("%s provider API request", providerType), err)
		return []string{}
	}

	var providerListJson ProviderList
	if err := json.Unmarshal(jsonData, &providerListJson); err != nil {
		logError(logger, fmt.Sprintf("%s providers request unmarshalling", providerType), err)
		return []string{}
	}

	providerList := make([]string, len(providerListJson.Data))
	for i, provider := range providerListJson.Data {
		providerList[i] = provider.Attributes.Name
	}
	return providerList
}

func ExtractProviderNameAndVersion(uri string) (string, string, string) {
	uri = strings.TrimPrefix(uri, "registry://provider/")
	parts := strings.Split(uri, "/")
	return parts[0], parts[2], parts[4]
}

func ConstructProviderVersionURI(providerNamespace interface{}, providerName string, providerVersion interface{}) string {
	return fmt.Sprintf("registry://provider/%s/providers/%s/versions/%s", providerNamespace, providerName, providerVersion)
}

func GetLatestProviderVersion(providerClient *http.Client, namespace, name interface{}, logger *log.Logger) string {
	uri := fmt.Sprintf("providers/%s/%s", namespace, name)
	jsonData, err := sendRegistryCall(providerClient, "GET", uri, logger, "v1")
	if err != nil {
		logError(logger, "latest provider version API request", err)
		return ""
	}

	var providerVersionLatest ProviderVersionLatest
	if err := json.Unmarshal(jsonData, &providerVersionLatest); err != nil {
		logError(logger, "provider versions request unmarshalling", err)
		return ""
	}

	logger.Debugf("Fetched latest provider version: %s", providerVersionLatest.Version)

	return providerVersionLatest.Version
}

func logError(logger *log.Logger, context string, err error) {
	logger.Errorf("Error in %s: %v", context, err)
}

// Function to extract the provider version ID.
func GetProviderVersionID(jsonData []byte, targetVersion string) (string, error) {
	// Unmarshal the JSON data into the struct.
	var response ProviderVersionResponse
	err := json.Unmarshal(jsonData, &response)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling JSON: %w", err)
	}

	// Iterate through the "included" array to find the matching version.
	for _, included := range response.Included {
		if included.Type == "provider-versions" && included.Attributes.Version == targetVersion {
			return included.ID, nil // Return the ID if version matches.
		}
	}

	return "", fmt.Errorf("provider version '%s' not found", targetVersion) // Return an error if not found.
}

// Function to get the provider version documentation.
func GetProviderDocs(jsonData []byte, logger *log.Logger) (ProviderDocs, error) {
	var response ProviderDocs
	err := json.Unmarshal(jsonData, &response)
	if err != nil {
		return ProviderDocs{}, fmt.Errorf("error unmarshalling JSON: %w", err)
	}

	markdown := ""
	// Create a markdown string using the fields from ProviderDocs.
	for _, doc := range response.Data {
		markdown += fmt.Sprintf("# %s \n\n**Id:** %s \n\n**Category:** %s\n\n**Subcategory:** %s\n\n**Path:** %s\n\n",
			doc.Attributes.Title,
			doc.ID,
			doc.Attributes.Category,
			doc.Attributes.Subcategory,
			doc.Attributes.Path,
		)
	}

	return response, nil
}

func GetProviderResourceDetails(client *http.Client, providerVersionID, resource string, sourceTypes []string, logger *log.Logger) (string, error) {
	var content string
	for _, category := range sourceTypes {
		uri := fmt.Sprintf("provider-docs?filter[provider-version]=%s&filter[category]=%s", providerVersionID, category)
		response, err := sendRegistryCall(client, "GET", uri, logger, "v2")
		if err != nil {
			return "", logAndReturnError(logger, "sending provider docs request", err)
		}

		var providerDocs ProviderDocs
		if err := json.Unmarshal(response, &providerDocs); err != nil {
			return "", logAndReturnError(logger, "unmarshalling provider docs", err)
		}

		for _, doc := range providerDocs.Data {
			if resource == doc.Attributes.Slug {
				uri := fmt.Sprintf("provider-docs/%s", doc.ID)
				response, err := sendRegistryCall(client, "GET", uri, logger, "v2")
				if err != nil {
					return "", logAndReturnError(logger, "sending provider details docs request", err)
				}

				var providerResourceDetails ProviderResourceDetails
				if err := json.Unmarshal(response, &providerResourceDetails); err != nil {
					return "", logAndReturnError(logger, "unmarshalling provider resource docs", err)
				}
				content += providerResourceDetails.Data.Attributes.Content
			}
		}
	}
	return content, nil
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

	return body, nil
}

func logAndReturnError(logger *log.Logger, context string, err error) error {
	logger.Errorf("Error in %s: %v", context, err)
	return err
}
