# HCP Terraform MCP Server

### Build from source

If you don't have Docker, you can use `go build` to build the binary in the
`cmd/hcp-terraform-mcp-server` directory, and use the `hcp-terraform-mcp-server stdio` command with the `HCP_TFE_TOKEN` environment variable set to your token. To specify the output location of the build, use the `-o` flag. You should configure your server to use the built executable as its `command`. For example:

```JSON
{
  "mcp": {
    "servers": {
      "github": {
        "command": "/path/to/hcp-terraform-mcp-server",
        "args": ["stdio"],
        "env": {
          "HCP_TFE_TOKEN": "<YOUR_TOKEN>"
        }
      }
    }
  }
}
```