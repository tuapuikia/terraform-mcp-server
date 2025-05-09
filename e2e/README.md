# End To End (e2e) Tests

The purpose of the E2E tests is to have a simple (currently) test that gives maintainers some confidence when adding new resources/tools. It does this by:
 * Building the `terraform-mcp-server` docker image
 * Running the image
 * Interacting with the server via stdio
 * Issuing requests that interact with the existing Resources/Tools

## Running the Tests

A service must be running that supports image building and container creation via the `docker` CLI.

```
HCP_TFE_TOKEN=<YOUR TOKEN> make test-e2e
```

### Note: providing a `HCP_TFE_TOKEN` is only necessary if TFE Resource/Tools are wanting to be tested. If not set they'll be skipped since TFE related tests contain the following:
```go
		if e2eServerToken == "" {
			t.Skip("HCP_TFE_TOKEN environment variable is not set, skipping")
		}
```

Running the tests:

```
➜ HCP_TFE_TOKEN=<INSERT_TOKEN_HERE> make test-e2e
=== RUN   TestE2E
    e2e_test.go:92: Building Docker image for e2e tests...
    e2e_test.go:38: Starting Stdio MCP client...
=== RUN   TestE2E/Initialize
Initialized with server: terraform-mcp-server test-e2e

=== RUN   TestE2E/CallTool_list_providers
    e2e_test.go:83: Raw response content: aws, google, azurerm, kubernetes, github, docker, null, random
--- PASS: TestE2E (2.30s)
    --- PASS: TestE2E/Initialize (0.55s)
    --- PASS: TestE2E/CallTool_list_providers (0.00s)
PASS
ok      terraform-mcp-server/e2e    2.771s
```