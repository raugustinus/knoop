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

func ValidateEdgeKind(ctx context.Context, tx *sql.Tx, kind string) error {
	var status string
	err := tx.QueryRowContext(ctx,
		`SELECT status FROM edge_types WHERE name = ?`, kind,
	).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return rejectEdgeKind(ctx, tx, kind, "unknown edge kind")
	}
	if err != nil {
		return err
	}
	if status == "deprecated" {
		return rejectEdgeKind(ctx, tx, kind, "edge kind is deprecated")
	}
	return nil
}

func rejectEdgeKind(ctx context.Context, tx *sql.Tx, kind, prefix string) error {
	allowed, listErr := allowedEdgeKinds(ctx, tx)
	if listErr != nil {
		return fmt.Errorf("%s: %q", prefix, kind)
	}
	return fmt.Errorf("%s: %q (allowed: %s)", prefix, kind, strings.Join(allowed, ", "))
}

func allowedEdgeKinds(ctx context.Context, tx *sql.Tx) ([]string, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT name FROM edge_types WHERE status IN ('stable','proposed') ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		names = append(names, n)
	}
	return names, rows.Err()
}
