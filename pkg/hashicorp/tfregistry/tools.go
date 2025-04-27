package tfregistry

import (
	"net/http"

	"github.com/mark3labs/mcp-go/server"
	log "github.com/sirupsen/logrus"
)

var DefaultTools = []string{"all"}

func InitTools(hcServer *server.MCPServer, registryClient *http.Client, logger *log.Logger) {
	hcServer.AddTool(ListProviders(registryClient))
	hcServer.AddTool(ListModules(registryClient, logger))
}
