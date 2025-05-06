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
		TestDescription: "Testing without providerNamespace and providerVersion",
		TestPayload:     map[string]interface{}{"ProviderName": "google"},
	},
	{
		TestShouldFail:  false,
		TestDescription: "Testing without providerVersion",
		TestPayload: map[string]interface{}{
			"providerName":      "azurerm",
			"providerNamespace": "hashicorp",
		},
	},
	{
		TestShouldFail:  false,
		TestDescription: "Testing without providerNamespace, but owned by hashicorp",
		TestPayload: map[string]interface{}{
			"providerName":    "google",
			"providerVersion": "latest",
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
		TestPayload: map[string]interface{}{
			"providerName":      "aws",
			"providerNamespace": "hashicorp",
		},
	},
	{
		TestShouldFail:  false,
		TestDescription: "Testing resources with all values for hashicorp providerNamespace",
		TestPayload: map[string]interface{}{
			"providerName":      "vault",
			"providerNamespace": "hashicorp",
			"providerVersion":   "latest",
			"providerDataType":  "resources",
		},
	},
	{
		TestShouldFail:  false,
		TestDescription: "Testing data-sources with all values for non-hashicorp providerNamespace",
		TestPayload: map[string]interface{}{
			"providerName":      "terracurl",
			"providerNamespace": "devops-rob",
			"providerVersion":   "latest",
		},
	},
	{
		TestShouldFail:  false,
		TestDescription: "Testing resources payload with malformed providerNamespace",
		TestPayload: map[string]interface{}{
			"providerName":      "vault",
			"providerNamespace": "hashicorp-malformed",
			"providerVersion":   "latest",
		},
	},
	{
		TestShouldFail:  true,
		TestDescription: "Testing resources payload with malformed providerName",
		TestPayload: map[string]interface{}{
			"providerName":      "vaults",
			"providerNamespace": "hashicorp",
			"providerVersion":   "latest",
		},
	},
}
