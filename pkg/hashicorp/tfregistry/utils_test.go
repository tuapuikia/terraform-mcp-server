// +build !integration

package tfregistry

import (
	"net/http"
	"testing"
	"strings"

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

// --- UnmarshalTFModulePlural ---

func TestUnmarshalTFModulePlural_Valid(t *testing.T) {
	resp := []byte(`{
		"meta": {},
		"modules": [
			{
				"id": "namespace/name/provider/1.0.0",
				"owner": "owner",
				"namespace": "namespace",
				"name": "name",
				"version": "1.0.0",
				"provider": "provider",
				"description": "A test module",
				"source": "source",
				"tag": "",
				"published_at": "2023-01-01T00:00:00Z",
				"downloads": 1,
				"verified": true
			}
		]
	}`)
	out, err := UnmarshalTFModulePlural(resp, "test-query")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(out) == 0 || !strings.Contains(out, "A test module") {
		t.Errorf("expected output to contain module description, got %q", out)
	}
}

func TestUnmarshalTFModulePlural_NoModules(t *testing.T) {
	resp := []byte(`{"meta": {}, "modules": []}`)
	_, err := UnmarshalTFModulePlural(resp, "test-query")
	if err == nil || !strings.Contains(err.Error(), "no modules found") {
		t.Errorf("expected error about no modules found, got %v", err)
	}
}

func TestUnmarshalTFModulePlural_InvalidJSON(t *testing.T) {
	resp := []byte(`not a json`)
	_, err := UnmarshalTFModulePlural(resp, "test-query")
	if err == nil || !strings.Contains(err.Error(), "unmarshalling modules") {
		t.Errorf("expected unmarshalling error, got %v", err)
	}
}

// --- UnmarshalModuleSingular ---

func TestUnmarshalModuleSingular_ValidAllFields(t *testing.T) {
	resp := []byte(`{
		"id": "namespace/name/provider/1.0.0",
		"owner": "owner",
		"namespace": "namespace",
		"name": "name",
		"version": "1.0.0",
		"provider": "provider",
		"provider_logo_url": "",
		"description": "A test module",
		"source": "source",
		"tag": "",
		"published_at": "2023-01-01T00:00:00Z",
		"downloads": 1,
		"verified": true,
		"root": {
			"path": "",
			"name": "root",
			"readme": "",
			"empty": false,
			"inputs": [
				{"name": "input1", "type": "string", "description": "desc", "default": "val", "required": true}
			],
			"outputs": [
				{"name": "output1", "description": "desc"}
			],
			"dependencies": [],
			"provider_dependencies": [
				{"name": "prov1", "namespace": "ns", "source": "src", "version": "1.0.0"}
			],
			"resources": []
		},
		"submodules": [],
		"examples": [
			{"path": "", "name": "example1", "readme": "example readme", "empty": false, "inputs": [], "outputs": [], "dependencies": [], "provider_dependencies": [], "resources": []}
		],
		"providers": ["provider"],
		"versions": ["1.0.0"],
		"deprecation": null
	}`)
	out, err := UnmarshalModuleSingular(resp)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(out, "A test module") {
		t.Errorf("expected output to contain module description, got %q", out)
	}
	if !strings.Contains(out, "input1") {
		t.Errorf("expected output to contain input variable, got %q", out)
	}
	if !strings.Contains(out, "example1") {
		t.Errorf("expected output to contain example name, got %q", out)
	}
}

func TestUnmarshalModuleSingular_EmptySections(t *testing.T) {
	resp := []byte(`{
		"id": "namespace/name/provider/1.0.0",
		"owner": "owner",
		"namespace": "namespace",
		"name": "name",
		"version": "1.0.0",
		"provider": "provider",
		"provider_logo_url": "",
		"description": "A test module",
		"source": "source",
		"tag": "",
		"published_at": "2023-01-01T00:00:00Z",
		"downloads": 1,
		"verified": true,
		"root": {
			"path": "",
			"name": "root",
			"readme": "",
			"empty": false,
			"inputs": [],
			"outputs": [],
			"dependencies": [],
			"provider_dependencies": [],
			"resources": []
		},
		"submodules": [],
		"examples": [],
		"providers": ["provider"],
		"versions": ["1.0.0"],
		"deprecation": null
	}`)
	out, err := UnmarshalModuleSingular(resp)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(out, "A test module") {
		t.Errorf("expected output to contain description, got %q", out)
	}
}

func TestUnmarshalModuleSingular_InvalidJSON(t *testing.T) {
	resp := []byte(`not a json`)
	_, err := UnmarshalModuleSingular(resp)
	if err == nil || !strings.Contains(err.Error(), "unmarshalling module details") {
		t.Errorf("expected unmarshalling error, got %v", err)
	}
}
