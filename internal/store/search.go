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

type SearchQuery struct {
	Text      string
	TokenName string
	Limit     int
}

type SearchHit struct {
	FragmentID int64
	Body       string
	Source     string
	Author     string
	CreatedAt  int64
	Tokens     []SearchToken
	Edges      []SearchEdge
}

type SearchToken struct {
	ID   int64
	Kind string
	Name string
}

type SearchEdge struct {
	Src  string
	Dst  string
	Kind string
}

const (
	defaultSearchLimit = 10
	maxSearchLimit     = 100
)

func (s *Store) Search(ctx context.Context, q SearchQuery) ([]SearchHit, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}

	text := strings.TrimSpace(q.Text)
	tokenName := strings.TrimSpace(q.TokenName)

	var ids []int64
	var err error
	switch {
	case text != "" && tokenName != "":
		ids, err = s.searchTextAndToken(ctx, text, tokenName, limit)
	case text != "":
		ids, err = s.searchText(ctx, text, limit)
	case tokenName != "":
		ids, err = s.searchToken(ctx, tokenName, limit)
	default:
		ids, err = s.searchRecent(ctx, limit)
	}
	if err != nil {
		return nil, err
	}

	hits := make([]SearchHit, 0, len(ids))
	for _, id := range ids {
		hit, err := loadHit(ctx, s.db, id)
		if err != nil {
			return nil, err
		}
		if hit != nil {
			hits = append(hits, *hit)
		}
	}
	return hits, nil
}

type querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (s *Store) searchText(ctx context.Context, text string, limit int) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT rowid FROM fragments_fts WHERE fragments_fts MATCH ? ORDER BY rank LIMIT ?`,
		text, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("fts5 query: %w", err)
	}
	defer rows.Close()
	return collectIDs(rows)
}

func (s *Store) searchToken(ctx context.Context, tokenName string, limit int) ([]int64, error) {
	tokenID, err := resolveTokenIDByName(ctx, s.db, tokenName)
	if err != nil {
		return nil, err
	}
	if tokenID == 0 {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT fragment_id FROM fragment_tokens
		 WHERE token_id = ?
		 ORDER BY fragment_id DESC
		 LIMIT ?`,
		tokenID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectIDs(rows)
}

func (s *Store) searchTextAndToken(ctx context.Context, text, tokenName string, limit int) ([]int64, error) {
	tokenID, err := resolveTokenIDByName(ctx, s.db, tokenName)
	if err != nil {
		return nil, err
	}
	if tokenID == 0 {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT f.rowid FROM fragments_fts f
		 WHERE f.fragments_fts MATCH ?
		   AND f.rowid IN (SELECT fragment_id FROM fragment_tokens WHERE token_id = ?)
		 ORDER BY rank
		 LIMIT ?`,
		text, tokenID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("fts5 query: %w", err)
	}
	defer rows.Close()
	return collectIDs(rows)
}

func (s *Store) searchRecent(ctx context.Context, limit int) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id FROM fragments ORDER BY id DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectIDs(rows)
}

func resolveTokenIDByName(ctx context.Context, db *sql.DB, tokenName string) (int64, error) {
	norm := strings.ToLower(strings.TrimSpace(tokenName))
	if norm == "" {
		return 0, nil
	}
	var id int64
	err := db.QueryRowContext(ctx,
		`SELECT id FROM tokens WHERE name_norm = ? LIMIT 1`, norm,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return id, nil
}

func collectIDs(rows *sql.Rows) ([]int64, error) {
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func loadHit(ctx context.Context, q querier, fragmentID int64) (*SearchHit, error) {
	hit := &SearchHit{FragmentID: fragmentID}
	var source sql.NullString
	err := q.QueryRowContext(ctx,
		`SELECT body, source, author, created_at FROM fragments WHERE id = ?`,
		fragmentID,
	).Scan(&hit.Body, &source, &hit.Author, &hit.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if source.Valid {
		hit.Source = source.String
	}

	trows, err := q.QueryContext(ctx,
		`SELECT t.id, t.kind, t.name
		 FROM tokens t
		 JOIN fragment_tokens ft ON ft.token_id = t.id
		 WHERE ft.fragment_id = ?
		 ORDER BY t.id`,
		fragmentID,
	)
	if err != nil {
		return nil, err
	}
	defer trows.Close()
	tokenIDs := []int64{}
	for trows.Next() {
		var tk SearchToken
		if err := trows.Scan(&tk.ID, &tk.Kind, &tk.Name); err != nil {
			return nil, err
		}
		hit.Tokens = append(hit.Tokens, tk)
		tokenIDs = append(tokenIDs, tk.ID)
	}
	if err := trows.Err(); err != nil {
		return nil, err
	}

	if len(tokenIDs) >= 2 {
		placeholders := strings.TrimRight(strings.Repeat("?,", len(tokenIDs)), ",")
		query := fmt.Sprintf(
			`SELECT s.name, d.name, e.kind
			 FROM edges e
			 JOIN tokens s ON s.id = e.src
			 JOIN tokens d ON d.id = e.dst
			 WHERE e.src IN (%s) AND e.dst IN (%s)
			 ORDER BY s.id, d.id, e.kind`,
			placeholders, placeholders,
		)
		args := make([]any, 0, len(tokenIDs)*2)
		for _, id := range tokenIDs {
			args = append(args, id)
		}
		for _, id := range tokenIDs {
			args = append(args, id)
		}
		erows, err := q.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		defer erows.Close()
		for erows.Next() {
			var e SearchEdge
			if err := erows.Scan(&e.Src, &e.Dst, &e.Kind); err != nil {
				return nil, err
			}
			hit.Edges = append(hit.Edges, e)
		}
		if err := erows.Err(); err != nil {
			return nil, err
		}
	}

	return hit, nil
}
