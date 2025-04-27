package tfregistry

import (
	"fmt"
	"io"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func SendRegistryCall(client *http.Client, method string, uri string, logger *log.Logger, apiVersion ...string) ([]byte, error) {
	version := "v1"
	if len(apiVersion) > 0 {
		version = apiVersion[0]
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
