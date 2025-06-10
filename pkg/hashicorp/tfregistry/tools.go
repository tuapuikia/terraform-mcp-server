// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package tfregistry

import (
	"net/http"

	"github.com/mark3labs/mcp-go/server"
	"github.com/posthog/posthog-go"
	log "github.com/sirupsen/logrus"
)

func InitTools(hcServer *server.MCPServer, registryClient *http.Client, phClient posthog.Client, logger *log.Logger) {
	hcServer.AddTool(ResolveProviderDocID(registryClient, phClient, logger))
	hcServer.AddTool(GetProviderDocs(registryClient, phClient, logger))
	hcServer.AddTool(SearchModules(registryClient, phClient, logger))
	hcServer.AddTool(ModuleDetails(registryClient, phClient, logger))
}
