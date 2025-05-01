package tfenterprise

import (
	"github.com/hashicorp/go-tfe"
	"github.com/mark3labs/mcp-go/server"
)

var DefaultTools = []string{"all"}

func InitTools(hcServer *server.MCPServer, tfeClient *tfe.Client) {
	// TODO: Uncomment on phase 2
	// hcServer.AddTool(ListWorkspaces(tfeClient))
}
