package e2e

var providerDetailsTestCases = []map[string]interface{}{
	// Testing with empty payload
	{},
	// Testing without provider version and namespace
	{
		"providerName": "aws",
	},
	// Testing without provider version
	{
		"providerName":      "azurerm",
		"providerNamespace": "hashicorp",
	},
	// Testing without provider namespace, but owned by hashicorp
	{
		"providerName":    "google",
		"providerVersion": "latest",
	},
	// Testing without provider namespace, but not-owned by hashicorp
	{
		"providerName":    "snowflake",
		"providerVersion": "latest",
	},
	// Testing only with required values
	{
		"providerName":      "aws",
		"providerNamespace": "hashicorp",
		"providerVersion":   "latest",
	},
	// Testing data-sources with all values for hashicorp namespace
	{
		"providerName":      "vault",
		"providerNamespace": "hashicorp",
		"providerVersion":   "latest",
		"providerDataType":  "resources",
	},
	// Testing data-sources with all values for non-hashicorp namespace
	{
		"providerName":      "terracurl",
		"providerNamespace": "devops-rob",
		"providerVersion":   "latest",
		"providerDataType":  "data-sources",
	},
	// Testing payload with malformed namespace
	{
		"providerName":      "vault",
		"providerNamespace": "hashicorp-malformed",
		"providerVersion":   "latest",
		"providerDataType":  "resources",
	},
	// Testing payload with malformed name
	{
		"providerName":      "vaults",
		"providerNamespace": "hashicorp",
		"providerVersion":   "latest",
		"providerDataType":  "resources",
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
