package tfenterprise

import (
	"fmt"

	"github.com/hashicorp/go-tfe"
	"github.com/mark3labs/mcp-go/server"
)

func Init(hcServer *server.MCPServer, token string, address string) error {
	config := &tfe.Config{
		Address:           address,
		Token:             token,
		RetryServerErrors: true, // Example configuration, adjust as needed
	}

	tfeClient, err := tfe.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to create TFE client: %v", err)
	}

	InitTools(hcServer, tfeClient)
	// TODO: Add InitResources

	return nil
}
