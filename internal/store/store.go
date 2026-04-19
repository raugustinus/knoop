// Copyright (C) 2026 Rob Augustinus
// SPDX-License-Identifier: MIT

package store

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"

	"github.com/raugustinus/knoop/internal/schema"
)

type Store struct {
	db     *sql.DB
	author string
}

func Open(path, author string) (*Store, error) {
	dsn := path + "?_fk=1&_journal_mode=WAL"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if _, err := db.Exec(schema.Schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	if _, err := db.Exec(schema.EdgeTypeSeed); err != nil {
		db.Close()
		return nil, fmt.Errorf("seed edge types: %w", err)
	}

	return &Store{db: db, author: author}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Author() string {
	return s.author
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) WithTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
