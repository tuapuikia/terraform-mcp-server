package tfenterprise

import (
	"github.com/hashicorp/go-tfe"
	"github.com/mark3labs/mcp-go/server"
)

var DefaultTools = []string{"all"}

func InitTools(hcServer *server.MCPServer, tfeClient *tfe.Client) {
	hcServer.AddTool(ListWorkspaces(tfeClient))
}
