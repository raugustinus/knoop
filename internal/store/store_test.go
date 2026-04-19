// Copyright (C) 2026 Rob Augustinus
// SPDX-License-Identifier: MIT

package store_test

import (
	"context"
	"testing"

	"github.com/raugustinus/knoop/internal/store"
)

func newStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(":memory:", "testuser")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func baseInput() store.CaptureInput {
	return store.CaptureInput{
		Body:   "Jurgen contrasted Rascal with Racket.",
		Source: "chat:test/1",
		Mentions: []store.Mention{
			{Ref: "rascal", Kind: "project", Name: "Rascal"},
			{Ref: "racket", Kind: "project", Name: "Racket"},
		},
		Edges: []store.EdgeSpec{
			{SrcRef: "rascal", DstRef: "racket", Kind: "contrasts_with"},
		},
	}
}

func countRows(t *testing.T, s *store.Store, query string, args ...any) int {
	t.Helper()
	var n int
	if err := s.DB().QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("count query %q: %v", query, err)
	}
	return n
}

func TestCapture_AllTablesPopulated(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	res, err := s.CaptureFragment(ctx, baseInput())
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	if res.FragmentID == 0 {
		t.Fatalf("expected non-zero fragment id")
	}
	if len(res.TokenIDs) != 2 {
		t.Fatalf("expected 2 token ids, got %d", len(res.TokenIDs))
	}
	if res.EdgeCount != 1 {
		t.Fatalf("expected edge count 1, got %d", res.EdgeCount)
	}

	var author, visibility string
	if err := s.DB().QueryRow(
		`SELECT author, visibility FROM fragments WHERE id = ?`, res.FragmentID,
	).Scan(&author, &visibility); err != nil {
		t.Fatalf("read fragment: %v", err)
	}
	if author != "testuser" {
		t.Errorf("author = %q, want %q", author, "testuser")
	}
	if visibility != "private" {
		t.Errorf("visibility = %q, want %q", visibility, "private")
	}
	if got := countRows(t, s, `SELECT count(*) FROM tokens`); got != 2 {
		t.Errorf("tokens count = %d, want 2", got)
	}
	if got := countRows(t, s, `SELECT count(*) FROM fragment_tokens WHERE fragment_id = ?`, res.FragmentID); got != 2 {
		t.Errorf("fragment_tokens count = %d, want 2", got)
	}
	if got := countRows(t, s, `SELECT count(*) FROM edges`); got != 1 {
		t.Errorf("edges count = %d, want 1", got)
	}
}

func TestCapture_TokenReuseAcrossFragments(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	first, err := s.CaptureFragment(ctx, baseInput())
	if err != nil {
		t.Fatalf("first capture: %v", err)
	}
	second, err := s.CaptureFragment(ctx, store.CaptureInput{
		Body: "Another note about Rascal.",
		Mentions: []store.Mention{
			{Ref: "rascal", Kind: "project", Name: "Rascal"},
		},
	})
	if err != nil {
		t.Fatalf("second capture: %v", err)
	}

	if first.TokenIDs["rascal"] != second.TokenIDs["rascal"] {
		t.Fatalf("expected same token id across captures: %d vs %d",
			first.TokenIDs["rascal"], second.TokenIDs["rascal"])
	}
	if got := countRows(t, s,
		`SELECT count(*) FROM fragment_tokens WHERE token_id = ?`,
		first.TokenIDs["rascal"],
	); got != 2 {
		t.Errorf("fragment_tokens for rascal = %d, want 2", got)
	}
}

func TestCapture_UnknownEdgeKindRollsBack(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	in := baseInput()
	in.Edges[0].Kind = "no_such_kind"

	if _, err := s.CaptureFragment(ctx, in); err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := countRows(t, s, `SELECT count(*) FROM fragments`); got != 0 {
		t.Errorf("fragments after failed capture = %d, want 0", got)
	}
	if got := countRows(t, s, `SELECT count(*) FROM tokens`); got != 0 {
		t.Errorf("tokens after failed capture = %d, want 0", got)
	}
	if got := countRows(t, s, `SELECT count(*) FROM edges`); got != 0 {
		t.Errorf("edges after failed capture = %d, want 0", got)
	}
}

func TestCapture_MissingMentionRefRejected(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	in := baseInput()
	in.Edges[0].DstRef = "ghost"

	if _, err := s.CaptureFragment(ctx, in); err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := countRows(t, s, `SELECT count(*) FROM fragments`); got != 0 {
		t.Errorf("fragments after rejected capture = %d, want 0", got)
	}
	if got := countRows(t, s, `SELECT count(*) FROM tokens`); got != 0 {
		t.Errorf("tokens after rejected capture = %d, want 0", got)
	}
}

func TestCapture_TokenNameNormalization(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	first, err := s.CaptureFragment(ctx, store.CaptureInput{
		Body: "First mention.",
		Mentions: []store.Mention{
			{Ref: "j", Kind: "person", Name: "Jurgen Vinju"},
		},
	})
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := s.CaptureFragment(ctx, store.CaptureInput{
		Body: "Second mention.",
		Mentions: []store.Mention{
			{Ref: "j", Kind: "person", Name: "jurgen vinju"},
		},
	})
	if err != nil {
		t.Fatalf("second: %v", err)
	}

	if first.TokenIDs["j"] != second.TokenIDs["j"] {
		t.Fatalf("token ids differ: %d vs %d", first.TokenIDs["j"], second.TokenIDs["j"])
	}

	var stored string
	if err := s.DB().QueryRow(
		`SELECT name FROM tokens WHERE id = ?`, first.TokenIDs["j"],
	).Scan(&stored); err != nil {
		t.Fatalf("read token: %v", err)
	}
	if stored != "Jurgen Vinju" {
		t.Errorf("token.name = %q, want original casing %q", stored, "Jurgen Vinju")
	}
	if got := countRows(t, s, `SELECT count(*) FROM tokens`); got != 1 {
		t.Errorf("tokens count = %d, want 1", got)
	}
}

func TestCapture_DuplicateEdgeIgnored(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	if _, err := s.CaptureFragment(ctx, baseInput()); err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := s.CaptureFragment(ctx, baseInput())
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if second.EdgeCount != 0 {
		t.Errorf("second capture EdgeCount = %d, want 0 (duplicate ignored)", second.EdgeCount)
	}
	if got := countRows(t, s, `SELECT count(*) FROM edges`); got != 1 {
		t.Errorf("edges count = %d, want 1", got)
	}
}

func TestCapture_DeprecatedEdgeKindRejected(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	if _, err := s.DB().Exec(
		`UPDATE edge_types SET status='deprecated' WHERE name='uses'`,
	); err != nil {
		t.Fatalf("deprecate edge kind: %v", err)
	}

	_, err := s.CaptureFragment(ctx, store.CaptureInput{
		Body: "Rascal uses term rewriting.",
		Mentions: []store.Mention{
			{Ref: "rascal", Kind: "project", Name: "Rascal"},
			{Ref: "tr", Kind: "concept", Name: "Term Rewriting"},
		},
		Edges: []store.EdgeSpec{
			{SrcRef: "rascal", DstRef: "tr", Kind: "uses"},
		},
	})
	if err == nil {
		t.Fatal("expected error for deprecated edge kind, got nil")
	}
	if got := countRows(t, s, `SELECT count(*) FROM fragments`); got != 0 {
		t.Errorf("fragments after rejected deprecated kind = %d, want 0", got)
	}
}

func TestCapture_ExplicitVisibilityHonored(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	in := baseInput()
	in.Visibility = "team"

	res, err := s.CaptureFragment(ctx, in)
	if err != nil {
		t.Fatalf("capture: %v", err)
	}

	var visibility string
	if err := s.DB().QueryRow(
		`SELECT visibility FROM fragments WHERE id = ?`, res.FragmentID,
	).Scan(&visibility); err != nil {
		t.Fatalf("read fragment: %v", err)
	}
	if visibility != "team" {
		t.Errorf("visibility = %q, want %q", visibility, "team")
	}
}

func TestCapture_InvalidVisibilityRejected(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	in := baseInput()
	in.Visibility = "nope"

	if _, err := s.CaptureFragment(ctx, in); err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := countRows(t, s, `SELECT count(*) FROM fragments`); got != 0 {
		t.Errorf("fragments after rejected visibility = %d, want 0", got)
	}
	if got := countRows(t, s, `SELECT count(*) FROM tokens`); got != 0 {
		t.Errorf("tokens after rejected visibility = %d, want 0", got)
	}
}
