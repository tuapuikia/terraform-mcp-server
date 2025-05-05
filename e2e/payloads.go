package e2e

type ProviderTestCase struct {
	TestShouldFail  bool                   `json:"testShouldFail"`
	TestDescription string                 `json:"testDescription"`
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
		TestDescription: "Testing without provider version and namespace",
		TestPayload:     map[string]interface{}{"ProviderName": "aws"},
	},
	{
		TestShouldFail:  false,
		TestDescription: "Testing without provider version",
		TestPayload: map[string]interface{}{
			"providerName":      "azurerm",
			"providerNamespace": "hashicorp",
		},
	},
	{
		TestShouldFail:  false,
		TestDescription: "Testing without provider namespace, but owned by hashicorp",
		TestPayload: map[string]interface{}{
			"providerName":    "google",
			"providerVersion": "latest",
		},
	},
	{
		TestShouldFail:  false,
		TestDescription: "Testing without provider namespace, but not-owned by hashicorp",
		TestPayload: map[string]interface{}{
			"providerName":    "snowflake",
			"providerVersion": "latest",
		},
	},
	{
		TestShouldFail:  false,
		TestDescription: "Testing only with required values",
		TestPayload: map[string]interface{}{
			"providerName":      "aws",
			"providerNamespace": "hashicorp",
			"providerVersion":   "latest",
		},
	},
	{
		TestShouldFail:  false,
		TestDescription: "Testing data-sources with all values for hashicorp namespace",
		TestPayload: map[string]interface{}{
			"providerName":      "vault",
			"providerNamespace": "hashicorp",
			"providerVersion":   "latest",
			"providerDataType":  "resources",
		},
	},
	{
		TestShouldFail:  false,
		TestDescription: "Testing data-sources with all values for non-hashicorp namespace",
		TestPayload: map[string]interface{}{
			"providerName":      "terracurl",
			"providerNamespace": "devops-rob",
			"providerVersion":   "latest",
			"providerDataType":  "data-sources",
		},
	},
	{
		TestShouldFail:  false,
		TestDescription: "Testing payload with malformed namespace",
		TestPayload: map[string]interface{}{
			"providerName":      "vault",
			"providerNamespace": "hashicorp-malformed",
			"providerVersion":   "latest",
			"providerDataType":  "resources",
		},
	},
	{
		TestShouldFail:  false,
		TestDescription: "Testing payload with malformed name",
		TestPayload: map[string]interface{}{
			"providerName":      "vaults",
			"providerNamespace": "hashicorp",
			"providerVersion":   "latest",
			"providerDataType":  "resources",
		},
	},
}

var providerResourceDetailsTestCases = []map[string]interface{}{
	{
		"providerName":      "aws",
		"providerNamespace": "hashicorp",
		"providerVersion":   "latest",
	},
	{
		"providerName":      "aws",
		"providerNamespace": "hashicorp",
		"providerVersion":   "latest",
	},
	{
		"providerName":      "aws",
		"providerNamespace": "hashicorp",
		"providerVersion":   "latest",
	},
	{
		"providerName":      "aws",
		"providerNamespace": "hashicorp",
		"providerVersion":   "latest",
	},
	{
		"providerName":      "aws",
		"providerNamespace": "hashicorp",
		"providerVersion":   "latest",
	},
	{
		"providerName":      "aws",
		"providerNamespace": "hashicorp",
		"providerVersion":   "latest",
	},
}
