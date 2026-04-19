// Copyright (C) 2026 Rob Augustinus
// SPDX-License-Identifier: MIT

package mcpserver

import (
	"github.com/mark3labs/mcp-go/server"

	"github.com/raugustinus/knoop/internal/store"
)

func New(s *store.Store) *server.MCPServer {
	srv := server.NewMCPServer(
		"knoop",
		"0.1.0",
		server.WithToolCapabilities(false),
	)
	RegisterCaptureTool(srv, s)
	return srv
}
