// Copyright (C) 2026 Rob Augustinus
// SPDX-License-Identifier: GPL-3.0-or-later

package mcpserver

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/raugustinus/knoop/internal/store"
)

type captureArgs struct {
	Body       string            `json:"body"`
	Source     string            `json:"source"`
	Visibility string            `json:"visibility"`
	Mentions   []captureMention  `json:"mentions"`
	Edges      []captureEdge     `json:"edges"`
}

type captureMention struct {
	Ref  string `json:"ref"`
	Kind string `json:"kind"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type captureEdge struct {
	Src  string `json:"src"`
	Dst  string `json:"dst"`
	Kind string `json:"kind"`
	Data string `json:"data"`
}

func RegisterCaptureTool(s *server.MCPServer, st *store.Store) {
	tool := mcp.NewTool("capture_fragment",
		mcp.WithDescription("Capture a fragment (observation) with its mentioned tokens and typed edges between them."),
		mcp.WithString("body",
			mcp.Required(),
			mcp.Description("The fragment text"),
		),
		mcp.WithString("source",
			mcp.Description("Where this came from (URI-ish)"),
		),
		mcp.WithString("visibility",
			mcp.Description("Fragment visibility; defaults to private"),
			mcp.Enum("private", "team", "public"),
		),
		mcp.WithArray("mentions",
			mcp.Description("Tokens referenced by this fragment"),
			mcp.Items(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"ref":  map[string]any{"type": "string", "description": "Local identifier for this mention"},
					"kind": map[string]any{"type": "string", "description": "Token kind: person, concept, project, etc."},
					"name": map[string]any{"type": "string", "description": "Display name of the token"},
					"role": map[string]any{"type": "string", "description": "Relationship role, defaults to 'mentions'"},
				},
				"required": []string{"ref", "kind", "name"},
			}),
		),
		mcp.WithArray("edges",
			mcp.Description("Typed edges between mentioned tokens"),
			mcp.Items(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"src":  map[string]any{"type": "string", "description": "Local ref of source mention"},
					"dst":  map[string]any{"type": "string", "description": "Local ref of destination mention"},
					"kind": map[string]any{"type": "string", "description": "Edge type from vocabulary"},
					"data": map[string]any{"type": "string", "description": "JSON string with extra context"},
				},
				"required": []string{"src", "dst", "kind"},
			}),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args captureArgs
		if err := request.BindArguments(&args); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
		}

		in := store.CaptureInput{
			Body:       args.Body,
			Source:     args.Source,
			Visibility: args.Visibility,
		}
		for _, m := range args.Mentions {
			in.Mentions = append(in.Mentions, store.Mention{
				Ref:  m.Ref,
				Kind: m.Kind,
				Name: m.Name,
				Role: m.Role,
			})
		}
		for _, e := range args.Edges {
			spec := store.EdgeSpec{
				SrcRef: e.Src,
				DstRef: e.Dst,
				Kind:   e.Kind,
			}
			if e.Data != "" {
				d := e.Data
				spec.Data = &d
			}
			in.Edges = append(in.Edges, spec)
		}

		result, err := st.CaptureFragment(ctx, in)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		msg := fmt.Sprintf(
			"captured fragment id=%d tokens=%d edges=%d",
			result.FragmentID, len(result.TokenIDs), result.EdgeCount,
		)
		return mcp.NewToolResultText(msg), nil
	})
}
