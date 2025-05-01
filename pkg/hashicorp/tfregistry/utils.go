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

func GetProviderDetails(providerClient *http.Client, uri string, version interface{}, logger *log.Logger) (string, string, error) {
	// Clean up the URI
	namespace, name, version := ExtractProviderNameAndVersion(uri)
	logger.Debugf("Extracted namespace: %s, name: %s, version: %s", namespace, name, version)

	if version == "" || version == "latest" {
		version = GetLatestProviderVersion(providerClient, namespace, name, logger)
	}
	providerVersionUri := fmt.Sprintf("registry://provider/%s/name/%s/version/%s", namespace, name, version)

	// Get the provider versions
	uri = fmt.Sprintf("providers/%s/%s/%s", namespace, name, version)
	response, err := sendRegistryCall(providerClient, "GET", uri, logger)
	if err != nil {
		logger.Errorf("Error sending request: %v", err)
		return "", providerVersionUri, fmt.Errorf("error sending request: %w", err)
	}
	// Get the provider version-id
	providerVersionID, err := GetProviderVersionID(response, version.(string))
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

func GetProviderResourceDetails(client *http.Client, version, name, namespace, sourceName, sourceType interface{}, logger *log.Logger) (string, error) {
	var content string

	uri := fmt.Sprintf("providers/%s/%s/%s", namespace, name, version)
	response, err := sendRegistryCall(client, "GET", uri, logger)
	if err != nil {
		return "", logAndReturnError(logger, "getting provider details", err)
	}

	var providerDocs ProviderDocs
	if err := json.Unmarshal(response, &providerDocs); err != nil {
		return "", logAndReturnError(logger, "unmarshalling provider docs", err)
	}

	content = fmt.Sprintf("# %s provider docs\n\n", name)
	s, sourceTypeProvided := sourceType.(string) // Get the sourceType and check if it was provided

	for _, doc := range providerDocs.Docs {
		// Include the doc if sourceType was not provided/empty OR if the doc category matches the provided sourceType
		if !sourceTypeProvided || s == "" || doc.Category == s {
			content += fmt.Sprintf("## %s \n\n**Id:** %s \n\n**Category:** %s\n\n**Subcategory:** %s\n\n**Path:** %s\n\n",
				doc.Title, doc.ID, doc.Category, doc.Subcategory, doc.Path)
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
	logger.Debugf("Response body: %s", string(body))
	return body, nil
}

func logAndReturnError(logger *log.Logger, context string, err error) error {
	logger.Errorf("Error in %s: %v", context, err)
	return err
}
