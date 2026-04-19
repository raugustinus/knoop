// Copyright (C) 2026 Rob Augustinus
// SPDX-License-Identifier: MIT

package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type Mention struct {
	Ref  string
	Kind string
	Name string
	Role string
}

type EdgeSpec struct {
	SrcRef string
	DstRef string
	Kind   string
	Data   *string
}

type CaptureInput struct {
	Body       string
	Source     string
	Visibility string
	Mentions   []Mention
	Edges      []EdgeSpec
}

type CaptureResult struct {
	FragmentID int64
	TokenIDs   map[string]int64
	EdgeCount  int
}

var allowedVisibilities = map[string]bool{
	"private": true,
	"team":    true,
	"public":  true,
}

func (s *Store) CaptureFragment(ctx context.Context, in CaptureInput) (*CaptureResult, error) {
	if strings.TrimSpace(in.Body) == "" {
		return nil, errors.New("body must not be empty")
	}
	visibility := in.Visibility
	if visibility == "" {
		visibility = "private"
	}
	if !allowedVisibilities[visibility] {
		return nil, fmt.Errorf("invalid visibility: %q (allowed: private, team, public)", visibility)
	}

	refSet := make(map[string]struct{}, len(in.Mentions))
	for i, m := range in.Mentions {
		if m.Ref == "" {
			return nil, fmt.Errorf("mentions[%d]: ref must be non-empty", i)
		}
		if m.Kind == "" {
			return nil, fmt.Errorf("mentions[%d]: kind must be non-empty", i)
		}
		if m.Name == "" {
			return nil, fmt.Errorf("mentions[%d]: name must be non-empty", i)
		}
		refSet[m.Ref] = struct{}{}
	}
	for i, e := range in.Edges {
		if _, ok := refSet[e.SrcRef]; !ok {
			return nil, fmt.Errorf("edges[%d]: src %q is not a declared mention ref", i, e.SrcRef)
		}
		if _, ok := refSet[e.DstRef]; !ok {
			return nil, fmt.Errorf("edges[%d]: dst %q is not a declared mention ref", i, e.DstRef)
		}
		if e.Kind == "" {
			return nil, fmt.Errorf("edges[%d]: kind must be non-empty", i)
		}
	}

	result := &CaptureResult{TokenIDs: make(map[string]int64, len(in.Mentions))}

	err := s.WithTx(ctx, func(tx *sql.Tx) error {
		for _, e := range in.Edges {
			if err := ValidateEdgeKind(ctx, tx, e.Kind); err != nil {
				return err
			}
		}

		for _, m := range in.Mentions {
			id, err := ResolveOrCreateToken(ctx, tx, TokenRef{Kind: m.Kind, Name: m.Name})
			if err != nil {
				return fmt.Errorf("resolve token %q: %w", m.Ref, err)
			}
			result.TokenIDs[m.Ref] = id
		}

		var sourceArg any
		if in.Source != "" {
			sourceArg = in.Source
		}
		res, err := tx.ExecContext(ctx,
			`INSERT INTO fragments (body, source, author, visibility) VALUES (?, ?, ?, ?)`,
			in.Body, sourceArg, s.author, visibility,
		)
		if err != nil {
			return fmt.Errorf("insert fragment: %w", err)
		}
		fragID, err := res.LastInsertId()
		if err != nil {
			return err
		}
		result.FragmentID = fragID

		for _, m := range in.Mentions {
			role := m.Role
			if role == "" {
				role = "mentions"
			}
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO fragment_tokens (fragment_id, token_id, role) VALUES (?, ?, ?)`,
				fragID, result.TokenIDs[m.Ref], role,
			); err != nil {
				return fmt.Errorf("insert fragment_tokens: %w", err)
			}
		}

		for _, e := range in.Edges {
			var dataArg any
			if e.Data != nil {
				dataArg = *e.Data
			}
			res, err := tx.ExecContext(ctx,
				`INSERT OR IGNORE INTO edges (src, dst, kind, data) VALUES (?, ?, ?, ?)`,
				result.TokenIDs[e.SrcRef], result.TokenIDs[e.DstRef], e.Kind, dataArg,
			)
			if err != nil {
				return fmt.Errorf("insert edge: %w", err)
			}
			n, err := res.RowsAffected()
			if err != nil {
				return err
			}
			result.EdgeCount += int(n)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
