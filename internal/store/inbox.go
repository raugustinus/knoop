// Copyright (C) 2026 Rob Augustinus
// SPDX-License-Identifier: MIT

package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

type InboxQuery struct {
	Target string // project / token name to fetch the mailbox for
	Limit  int    // cap on results; default 10, max 100
	Peek   bool   // if true, do NOT mark results as delivered
}

var ErrInboxTargetEmpty = errors.New("inbox target must be non-empty")

func (s *Store) Inbox(ctx context.Context, q InboxQuery) ([]SearchHit, error) {
	target := strings.TrimSpace(q.Target)
	if target == "" {
		return nil, ErrInboxTargetEmpty
	}
	limit := q.Limit
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}
	norm := strings.ToLower(target)

	hits := make([]SearchHit, 0)
	err := s.WithTx(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `
			SELECT DISTINCT ft.fragment_id
			FROM fragment_tokens ft
			JOIN tokens t ON t.id = ft.token_id
			LEFT JOIN deliveries d
			  ON d.fragment_id = ft.fragment_id AND d.target_norm = ?
			WHERE t.name_norm = ?
			  AND d.fragment_id IS NULL
			ORDER BY ft.fragment_id ASC
			LIMIT ?`,
			norm, norm, limit,
		)
		if err != nil {
			return err
		}
		ids, err := collectIDs(rows)
		rows.Close()
		if err != nil {
			return err
		}

		for _, id := range ids {
			hit, err := loadHit(ctx, tx, id)
			if err != nil {
				return err
			}
			if hit != nil {
				hits = append(hits, *hit)
			}
		}

		if q.Peek {
			return nil
		}
		for _, id := range ids {
			if _, err := tx.ExecContext(ctx,
				`INSERT OR IGNORE INTO deliveries (fragment_id, target, target_norm) VALUES (?, ?, ?)`,
				id, target, norm,
			); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return hits, nil
}
