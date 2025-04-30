package tfregistry

import (
	"net/http"

	"github.com/mark3labs/mcp-go/server"
	log "github.com/sirupsen/logrus"
)

var DefaultTools = []string{"all"}

func InitTools(hcServer *server.MCPServer, registryClient *http.Client, logger *log.Logger) {
	hcServer.AddTool(ProviderDetails(registryClient, logger))
	hcServer.AddTool(providerResourceDetails(registryClient, logger))
	hcServer.AddTool(ListModules(registryClient, logger))
}
