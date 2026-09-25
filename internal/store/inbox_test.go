// Copyright (C) 2026 Rob Augustinus
// SPDX-License-Identifier: MIT

package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/raugustinus/knoop/internal/store"
)

func seedForInbox(t *testing.T, s *store.Store) {
	t.Helper()
	ctx := context.Background()

	if _, err := s.CaptureFragment(ctx, store.CaptureInput{
		Body: "Useful link for rktlens: concept X applies here because Y.",
		Mentions: []store.Mention{
			{Ref: "r", Kind: "project", Name: "rktlens"},
			{Ref: "x", Kind: "concept", Name: "Concept X"},
		},
	}); err != nil {
		t.Fatalf("seed fragment 1: %v", err)
	}

	if _, err := s.CaptureFragment(ctx, store.CaptureInput{
		Body: "Another rktlens note about telemetry.",
		Mentions: []store.Mention{
			{Ref: "r", Kind: "project", Name: "rktlens"},
			{Ref: "t", Kind: "concept", Name: "Telemetry"},
		},
	}); err != nil {
		t.Fatalf("seed fragment 2: %v", err)
	}

	if _, err := s.CaptureFragment(ctx, store.CaptureInput{
		Body: "Unrelated fragment about knoop itself.",
		Mentions: []store.Mention{
			{Ref: "k", Kind: "project", Name: "knoop"},
		},
	}); err != nil {
		t.Fatalf("seed fragment 3: %v", err)
	}
}

func TestInbox_EmptyTargetErrors(t *testing.T) {
	s := newStore(t)
	_, err := s.Inbox(context.Background(), store.InboxQuery{Target: "  "})
	if !errors.Is(err, store.ErrInboxTargetEmpty) {
		t.Fatalf("want ErrInboxTargetEmpty, got %v", err)
	}
}

func TestInbox_MatchesTokenMentionsOnly(t *testing.T) {
	s := newStore(t)
	seedForInbox(t, s)

	hits, err := s.Inbox(context.Background(), store.InboxQuery{Target: "rktlens"})
	if err != nil {
		t.Fatalf("inbox: %v", err)
	}
	if len(hits) != 2 {
		t.Fatalf("want 2 hits for rktlens, got %d", len(hits))
	}
	for _, h := range hits {
		if h.FragmentID == 3 {
			t.Errorf("fragment 3 (knoop-only) should not appear in rktlens inbox")
		}
	}
}

func TestInbox_CaseInsensitiveTarget(t *testing.T) {
	s := newStore(t)
	seedForInbox(t, s)

	hits, err := s.Inbox(context.Background(), store.InboxQuery{Target: "RKTLENS"})
	if err != nil {
		t.Fatalf("inbox: %v", err)
	}
	if len(hits) != 2 {
		t.Fatalf("want 2 hits for RKTLENS, got %d", len(hits))
	}
}

func TestInbox_DeliveredFragmentsDoNotRepeat(t *testing.T) {
	s := newStore(t)
	seedForInbox(t, s)

	first, err := s.Inbox(context.Background(), store.InboxQuery{Target: "rktlens"})
	if err != nil {
		t.Fatalf("first inbox: %v", err)
	}
	if len(first) != 2 {
		t.Fatalf("first call: want 2, got %d", len(first))
	}

	second, err := s.Inbox(context.Background(), store.InboxQuery{Target: "rktlens"})
	if err != nil {
		t.Fatalf("second inbox: %v", err)
	}
	if len(second) != 0 {
		t.Fatalf("second call: want 0 (already delivered), got %d", len(second))
	}
}

func TestInbox_NewFragmentAfterDeliverySurfaces(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	seedForInbox(t, s)

	if _, err := s.Inbox(ctx, store.InboxQuery{Target: "rktlens"}); err != nil {
		t.Fatalf("drain: %v", err)
	}

	if _, err := s.CaptureFragment(ctx, store.CaptureInput{
		Body: "Third rktlens idea, captured after first delivery.",
		Mentions: []store.Mention{
			{Ref: "r", Kind: "project", Name: "rktlens"},
		},
	}); err != nil {
		t.Fatalf("capture new: %v", err)
	}

	hits, err := s.Inbox(ctx, store.InboxQuery{Target: "rktlens"})
	if err != nil {
		t.Fatalf("second inbox: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("want 1 new hit, got %d", len(hits))
	}
	if hits[0].Body == "" {
		t.Error("expected non-empty body on new hit")
	}
}

func TestInbox_PeekDoesNotMarkDelivered(t *testing.T) {
	s := newStore(t)
	seedForInbox(t, s)

	peeked, err := s.Inbox(context.Background(), store.InboxQuery{Target: "rktlens", Peek: true})
	if err != nil {
		t.Fatalf("peek: %v", err)
	}
	if len(peeked) != 2 {
		t.Fatalf("peek: want 2, got %d", len(peeked))
	}

	again, err := s.Inbox(context.Background(), store.InboxQuery{Target: "rktlens", Peek: true})
	if err != nil {
		t.Fatalf("second peek: %v", err)
	}
	if len(again) != 2 {
		t.Fatalf("second peek: want 2 (peek should not mark delivered), got %d", len(again))
	}
}

func TestInbox_IsolatedBetweenTargets(t *testing.T) {
	s := newStore(t)
	seedForInbox(t, s)

	if _, err := s.Inbox(context.Background(), store.InboxQuery{Target: "rktlens"}); err != nil {
		t.Fatalf("rktlens inbox: %v", err)
	}
	knoopHits, err := s.Inbox(context.Background(), store.InboxQuery{Target: "knoop"})
	if err != nil {
		t.Fatalf("knoop inbox: %v", err)
	}
	if len(knoopHits) != 1 {
		t.Fatalf("knoop inbox: want 1, got %d", len(knoopHits))
	}
}

func TestInbox_LimitRespected(t *testing.T) {
	s := newStore(t)
	seedForInbox(t, s)

	hits, err := s.Inbox(context.Background(), store.InboxQuery{Target: "rktlens", Limit: 1})
	if err != nil {
		t.Fatalf("inbox: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("want 1 under limit, got %d", len(hits))
	}

	hits2, err := s.Inbox(context.Background(), store.InboxQuery{Target: "rktlens"})
	if err != nil {
		t.Fatalf("second inbox: %v", err)
	}
	if len(hits2) != 1 {
		t.Fatalf("want 1 remaining (since only 1 was marked delivered), got %d", len(hits2))
	}
}
