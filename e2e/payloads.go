package e2e

type ContentType string

const (
	CONST_TYPE_RESOURCE    ContentType = "resources"
	CONST_TYPE_DATA_SOURCE ContentType = "data-sources"
	CONST_TYPE_BOTH        ContentType = "both"
	CONST_TYPE_GUIDES      ContentType = "guides"
	CONST_TYPE_FUNCTIONS   ContentType = "functions"
	CONST_TYPE_OVERVIEW    ContentType = "overview"
)

type RegistryTestCase struct {
	TestShouldFail  bool                   `json:"testShouldFail"`
	TestDescription string                 `json:"testDescription"`
	TestContentType ContentType            `json:"testContentType,omitempty"`
	TestPayload     map[string]interface{} `json:"testPayload,omitempty"`
}

var providerDetailsTestCases = []RegistryTestCase{
	{
		TestShouldFail:  true,
		TestDescription: "Testing with empty payload",
		TestPayload:     map[string]interface{}{},
	},
	{
		TestShouldFail:  true,
		TestDescription: "Testing without providerNamespace and providerVersion",
		TestPayload:     map[string]interface{}{"ProviderName": "google"},
	},
	{
		TestShouldFail:  false,
		TestDescription: "Testing without providerVersion",
		TestPayload: map[string]interface{}{
			"providerName":      "azurerm",
			"providerNamespace": "hashicorp",
			"serviceName":       "azurerm_iot_security_solution",
		},
	},
	{
		TestShouldFail:  false,
		TestDescription: "Testing without providerNamespace, but owned by hashicorp",
		TestPayload: map[string]interface{}{
			"providerName":    "aws",
			"providerVersion": "latest",
			"serviceName":     "aws_s3_bucket",
		},
	},
	{
		TestShouldFail:  true,
		TestDescription: "Testing without providerNamespace, but not-owned by hashicorp",
		TestPayload: map[string]interface{}{
			"providerName":    "snowflake",
			"providerVersion": "latest",
		},
	},
	{
		TestShouldFail:  false,
		TestDescription: "Testing only with required values",
		TestContentType: CONST_TYPE_BOTH,
		TestPayload: map[string]interface{}{
			"providerName":      "dns",
			"providerNamespace": "hashicorp",
			"serviceName":       "dns_ns_record_set",
		},
	},
	{
		TestShouldFail:  false,
		TestDescription: "Testing resources with all values for hashicorp providerNamespace",
		TestContentType: CONST_TYPE_RESOURCE,
		TestPayload: map[string]interface{}{
			"providerName":      "pinecone",
			"providerNamespace": "pinecone-io",
			"providerVersion":   "latest",
			"providerDataType":  "resources",
			"serviceName":       "pinecone_index",
		},
	},
	{
		TestShouldFail:  false,
		TestDescription: "Testing data-sources for non-hashicorp providerNamespace",
		TestContentType: CONST_TYPE_DATA_SOURCE,
		TestPayload: map[string]interface{}{
			"providerName":      "terracurl",
			"providerNamespace": "devops-rob",
			"providerDataType":  "data-sources",
			"serviceName":       "terracurl",
		},
	},
	{
		TestShouldFail:  false,
		TestDescription: "Testing payload with malformed providerNamespace",
		TestPayload: map[string]interface{}{
			"providerName":      "vault",
			"providerNamespace": "hashicorp-malformed",
			"providerVersion":   "latest",
			"serviceName":       "vault_aws_auth_backend_role",
		},
	},
	{
		TestShouldFail:  true,
		TestDescription: "Testing payload with malformed providerName",
		TestPayload: map[string]interface{}{
			"providerName":      "vaults",
			"providerNamespace": "hashicorp",
			"providerVersion":   "latest",
		},
	},
	{
		TestShouldFail:  false,
		TestDescription: "Testing guides documentation with v2 API",
		TestContentType: CONST_TYPE_GUIDES,
		TestPayload: map[string]interface{}{
			"providerName":      "aws",
			"providerNamespace": "hashicorp",
			"providerVersion":   "latest",
			"providerDataType":  "guides",
			"serviceName":       "custom-service-endpoints",
		},
	},
	{
		TestShouldFail:  false,
		TestDescription: "Testing functions documentation with v2 API",
		TestContentType: CONST_TYPE_FUNCTIONS,
		TestPayload: map[string]interface{}{
			"providerName":      "google",
			"providerNamespace": "hashicorp",
			"providerVersion":   "latest",
			"providerDataType":  "functions",
			"serviceName":       "name_from_id",
		},
	},
	{
		TestShouldFail:  false,
		TestDescription: "Testing overview documentation with v2 API",
		TestContentType: CONST_TYPE_OVERVIEW,
		TestPayload: map[string]interface{}{
			"providerName":      "google",
			"providerNamespace": "hashicorp",
			"providerVersion":   "latest",
			"providerDataType":  "overview",
			"serviceName":       "index",
		},
	},
}

var listModulesTestCases = []RegistryTestCase{
	{
		TestShouldFail:  false,
		TestDescription: "Testing listModules with no parameters",
		TestPayload:     map[string]interface{}{},
	},
	{
		TestShouldFail:  false,
		TestDescription: "Testing listModules with moduleProvider 'aws' - no offset",
		TestPayload: map[string]interface{}{
			"moduleProvider": "aws",
		},
	},
	{
		TestShouldFail:  false,
		TestDescription: "Testing listModules with moduleProvider 'google' and currentOffset 10",
		TestPayload: map[string]interface{}{
			"moduleProvider": "google",
			"currentOffset":  10,
		},
	},
	{
		TestShouldFail:  false,
		TestDescription: "Testing listModules with currentOffset 5 only",
		TestPayload: map[string]interface{}{
			"currentOffset": 5,
		},
	},
	{
		TestShouldFail:  false,
		TestDescription: "Testing listModules with invalid currentOffset (negative)",
		TestPayload: map[string]interface{}{
			"currentOffset": -1,
		},
	},
	{
		TestShouldFail:  true,
		TestDescription: "Testing listModules with a moduleProvider not in the map (e.g., 'unknownprovider')",
		TestPayload: map[string]interface{}{
			"moduleProvider": "unknownprovider",
		},
	},
}

var moduleDetailsTestCases = []RegistryTestCase{
	{
		TestShouldFail:  false,
		TestDescription: "Testing moduleDetails with valid 'vpc' module for 'aws' provider",
		TestPayload: map[string]interface{}{
			"moduleName":     "vpc",
			"moduleProvider": "aws",
		},
	},
	{
		TestShouldFail:  true,
		TestDescription: "Testing moduleDetails missing moduleName",
		TestPayload: map[string]interface{}{
			"moduleProvider": "aws",
		},
	},
	{
		TestShouldFail:  true,
		TestDescription: "Testing moduleDetails missing moduleProvider",
		TestPayload: map[string]interface{}{
			"moduleName": "vpc",
		},
	},
	{
		TestShouldFail:  true,
		TestDescription: "Testing moduleDetails with empty moduleName",
		TestPayload: map[string]interface{}{
			"moduleName":     "",
			"moduleProvider": "aws",
		},
	},
	{
		TestShouldFail:  true,
		TestDescription: "Testing moduleDetails with empty moduleProvider",
		TestPayload: map[string]interface{}{
			"moduleName":     "vpc",
			"moduleProvider": "",
		},
	},
	{
		TestShouldFail:  true,
		TestDescription: "Testing moduleDetails with non-existent module 'nonexistentmodule' for 'aws' provider",
		TestPayload: map[string]interface{}{
			"moduleName":     "nonexistentmodule",
			"moduleProvider": "aws",
		},
	},
	{
		TestShouldFail:  true, // Expecting empty or error, tool call might succeed but return no useful data
		TestDescription: "Testing moduleDetails with moduleProvider 'unknownprovider'",
		TestPayload: map[string]interface{}{
			"moduleName":     "vpc",
			"moduleProvider": "unknownprovider",
		},
	},
}
