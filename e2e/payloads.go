package e2e

type ContentType string

const (
	CONST_TYPE_RESOURCE    ContentType = "resources"
	CONST_TYPE_DATA_SOURCE ContentType = "data-sources"
	CONST_TYPE_BOTH        ContentType = "both"
)

type ProviderTestCase struct {
	TestShouldFail  bool                   `json:"testShouldFail"`
	TestDescription string                 `json:"testDescription"`
	TestContentType ContentType            `json:"testResourceOnly,omitempty"`
	TestPayload     map[string]interface{} `json:"testPayload,omitempty"`
}

var providerDetailsTestCases = []ProviderTestCase{
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
}
