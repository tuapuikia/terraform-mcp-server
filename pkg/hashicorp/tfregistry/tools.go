// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfregistry

import (
	"net/http"

	"github.com/mark3labs/mcp-go/server"
	log "github.com/sirupsen/logrus"
)

func InitTools(hcServer *server.MCPServer, registryClient *http.Client, logger *log.Logger) {
	hcServer.AddTool(ProviderDetails(registryClient, logger))
	hcServer.AddTool(providerResourceDetails(registryClient, logger))
	hcServer.AddTool(SearchModules(registryClient, logger))
	hcServer.AddTool(ModuleDetails(registryClient, logger))
}
