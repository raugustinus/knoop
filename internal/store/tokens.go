// Copyright (C) 2026 Rob Augustinus
// SPDX-License-Identifier: GPL-3.0-or-later

package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

type TokenRef struct {
	Kind string
	Name string
	Data *string
}

func ResolveOrCreateToken(ctx context.Context, tx *sql.Tx, ref TokenRef) (int64, error) {
	norm := strings.ToLower(strings.TrimSpace(ref.Name))

	var id int64
	var existing sql.NullString
	err := tx.QueryRowContext(ctx,
		`SELECT id, data FROM tokens WHERE kind = ? AND name_norm = ?`,
		ref.Kind, norm,
	).Scan(&id, &existing)
	switch {
	case err == nil:
		if ref.Data != nil && !existing.Valid {
			if _, err := tx.ExecContext(ctx,
				`UPDATE tokens SET data = ? WHERE id = ?`,
				*ref.Data, id,
			); err != nil {
				return 0, err
			}
		}
		return id, nil
	case errors.Is(err, sql.ErrNoRows):
		// fall through to insert
	default:
		return 0, err
	}

	var dataArg any
	if ref.Data != nil {
		dataArg = *ref.Data
	}
	res, err := tx.ExecContext(ctx,
		`INSERT INTO tokens (kind, name, data) VALUES (?, ?, ?)`,
		ref.Kind, ref.Name, dataArg,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
