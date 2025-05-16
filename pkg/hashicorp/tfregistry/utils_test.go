// +build !integration

package tfregistry

import (
	"net/http"
	"testing"

	log "github.com/sirupsen/logrus"
)

// --- sendRegistryCall ---

var client = &http.Client{
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	},
}

var logger = log.New()

func TestSendRegistryCall_Success_v1(t *testing.T) {
	_, err := sendRegistryCall(client, "GET", "providers/hashicorp/aws", logger)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestSendRegistryCall_Success_v2(t *testing.T) {
	_, err := sendRegistryCall(client, "GET", "provider-docs?filter[provider-version]=6221", logger, "v2")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestSendRegistryCall_404NotFound_v2(t *testing.T) {
	_, err := sendRegistryCall(client, "GET", "test-uri", logger, "v2")
	if err == nil || err.Error() != "error: 404 Not Found" {
		t.Errorf("expected 404 error, got %v", err)
	}
}

func TestSendRegistryCall_404NotFound_v1(t *testing.T) {
	_, err := sendRegistryCall(client, "GET", "test-uri", logger)
	if err == nil || err.Error() != "error: 404 Not Found" {
		t.Errorf("expected 404 error, got %v", err)
	}
}
