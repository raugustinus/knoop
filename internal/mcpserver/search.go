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

type searchArgs struct {
	Query     string  `json:"query"`
	TokenName string  `json:"token_name"`
	Limit     float64 `json:"limit"`
}

func RegisterSearchTool(s *server.MCPServer, st *store.Store) {
	tool := mcp.NewTool("search_knoop",
		mcp.WithDescription("Search captured fragments by text (FTS5) and/or token name. Returns matching fragments with their mentioned tokens and edges among those tokens. Use this to recall what has been captured about a topic before answering."),
		mcp.WithString("query",
			mcp.Description("FTS5 expression matched against fragment body (e.g. \"homoiconicity\" or \"rascal OR racket\"). Optional."),
		),
		mcp.WithString("token_name",
			mcp.Description("Only fragments mentioning a token with this name, case-insensitive (e.g. \"Jurgen Vinju\"). Optional."),
		),
		mcp.WithNumber("limit",
			mcp.Description("Max number of results (default 10, max 100)."),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args searchArgs
		if err := request.BindArguments(&args); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
		}

		q := store.SearchQuery{
			Text:      args.Query,
			TokenName: args.TokenName,
			Limit:     int(args.Limit),
		}
		hits, err := st.Search(ctx, q)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(FormatSearchHits(q, hits)), nil
	})
}
