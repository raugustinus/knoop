// Copyright (C) 2026 Rob Augustinus
// SPDX-License-Identifier: MIT

package mcpserver

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/raugustinus/knoop/internal/store"
)

type inboxArgs struct {
	Target string  `json:"target"`
	Limit  float64 `json:"limit"`
	Peek   bool    `json:"peek"`
}

func RegisterInboxTool(s *server.MCPServer, st *store.Store) {
	tool := mcp.NewTool("inbox_for",
		mcp.WithDescription("Fetch the inbox of fragments routed to a target project. Returns fragments that mention a token matching the target name and haven't been delivered to that target yet, and marks them delivered so they don't repeat. Intended for session-start hooks ('what did Rob capture for this project since I last worked here')."),
		mcp.WithString("target",
			mcp.Required(),
			mcp.Description("Project / token name whose mailbox to drain (case-insensitive, e.g. \"rktlens\")."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Max fragments to return (default 10, max 100)."),
		),
		mcp.WithBoolean("peek",
			mcp.Description("If true, do NOT mark the returned fragments as delivered. Useful for previewing without consuming the inbox."),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args inboxArgs
		if err := request.BindArguments(&args); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
		}
		hits, err := st.Inbox(ctx, store.InboxQuery{
			Target: args.Target,
			Limit:  int(args.Limit),
			Peek:   args.Peek,
		})
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(FormatInboxHits(args.Target, hits)), nil
	})
}
