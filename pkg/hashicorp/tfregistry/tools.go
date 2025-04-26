package tfregistry

import (
	"net/http"

	"github.com/mark3labs/mcp-go/server"
)

var DefaultTools = []string{"all"}

func InitTools(hcServer *server.MCPServer, registryClient *http.Client) {
	hcServer.AddTool(ListProviders(registryClient))
}
