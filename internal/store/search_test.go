// Copyright (C) 2026 Rob Augustinus
// SPDX-License-Identifier: MIT

package store_test

import (
	"context"
	"testing"

	"github.com/raugustinus/knoop/internal/store"
)

func seedForSearch(t *testing.T, s *store.Store) {
	t.Helper()
	ctx := context.Background()

	if _, err := s.CaptureFragment(ctx, store.CaptureInput{
		Body:   "Jurgen key point: homoiconicity is elegant but the heavy lifting is provably correct transformations.",
		Source: "chat:design/1",
		Mentions: []store.Mention{
			{Ref: "j", Kind: "person", Name: "Jurgen Vinju"},
			{Ref: "h", Kind: "concept", Name: "Homoiconicity"},
			{Ref: "rascal", Kind: "project", Name: "Rascal"},
			{Ref: "racket", Kind: "project", Name: "Racket"},
		},
		Edges: []store.EdgeSpec{
			{SrcRef: "j", DstRef: "h", Kind: "advised_on"},
			{SrcRef: "rascal", DstRef: "racket", Kind: "contrasts_with"},
		},
	}); err != nil {
		t.Fatalf("seed fragment 1: %v", err)
	}

	if _, err := s.CaptureFragment(ctx, store.CaptureInput{
		Body:   "Rob noted Rascal prioritizes term rewriting as primary.",
		Source: "chat:design/2",
		Mentions: []store.Mention{
			{Ref: "rob", Kind: "person", Name: "Rob"},
			{Ref: "rascal", Kind: "project", Name: "Rascal"},
			{Ref: "tr", Kind: "concept", Name: "Term Rewriting"},
		},
		Edges: []store.EdgeSpec{
			{SrcRef: "rascal", DstRef: "tr", Kind: "uses"},
		},
	}); err != nil {
		t.Fatalf("seed fragment 2: %v", err)
	}

	if _, err := s.CaptureFragment(ctx, store.CaptureInput{
		Body: "Unrelated note about scheduling and coffee.",
		Mentions: []store.Mention{
			{Ref: "c", Kind: "concept", Name: "Coffee"},
		},
	}); err != nil {
		t.Fatalf("seed fragment 3: %v", err)
	}
}

func TestSearch_TextMatch(t *testing.T) {
	s := newStore(t)
	seedForSearch(t, s)

	hits, err := s.Search(context.Background(), store.SearchQuery{Text: "homoiconicity"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits))
	}
	if hits[0].FragmentID != 1 {
		t.Errorf("expected fragment 1, got %d", hits[0].FragmentID)
	}
}

func TestSearch_TokenNameCaseInsensitive(t *testing.T) {
	s := newStore(t)
	seedForSearch(t, s)

	hits, err := s.Search(context.Background(), store.SearchQuery{TokenName: "jurgen vinju"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits))
	}
	if hits[0].FragmentID != 1 {
		t.Errorf("expected fragment 1, got %d", hits[0].FragmentID)
	}
}

func TestSearch_CombinedNarrows(t *testing.T) {
	s := newStore(t)
	seedForSearch(t, s)

	hits, err := s.Search(context.Background(), store.SearchQuery{
		Text:      "rascal",
		TokenName: "Rob",
	})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit (fragment 2), got %d", len(hits))
	}
	if hits[0].FragmentID != 2 {
		t.Errorf("expected fragment 2, got %d", hits[0].FragmentID)
	}
}

func TestSearch_EmptyQueryReturnsRecent(t *testing.T) {
	s := newStore(t)
	seedForSearch(t, s)

	hits, err := s.Search(context.Background(), store.SearchQuery{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 3 {
		t.Fatalf("expected 3 hits, got %d", len(hits))
	}
	if hits[0].FragmentID != 3 {
		t.Errorf("expected most recent (id=3) first, got %d", hits[0].FragmentID)
	}
}

func TestSearch_LimitRespected(t *testing.T) {
	s := newStore(t)
	seedForSearch(t, s)

	hits, err := s.Search(context.Background(), store.SearchQuery{Limit: 2})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 2 {
		t.Fatalf("expected 2 hits under limit, got %d", len(hits))
	}
}

func TestSearch_HitContainsTokensAndEdges(t *testing.T) {
	s := newStore(t)
	seedForSearch(t, s)

	hits, err := s.Search(context.Background(), store.SearchQuery{Text: "homoiconicity"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits))
	}
	hit := hits[0]

	wantTokens := map[string]string{
		"Jurgen Vinju":  "person",
		"Homoiconicity": "concept",
		"Rascal":        "project",
		"Racket":        "project",
	}
	if len(hit.Tokens) != len(wantTokens) {
		t.Errorf("tokens count = %d, want %d", len(hit.Tokens), len(wantTokens))
	}
	for _, tk := range hit.Tokens {
		if kind, ok := wantTokens[tk.Name]; !ok {
			t.Errorf("unexpected token %q", tk.Name)
		} else if kind != tk.Kind {
			t.Errorf("token %q kind = %q, want %q", tk.Name, tk.Kind, kind)
		}
	}

	if len(hit.Edges) != 2 {
		t.Fatalf("expected 2 edges among fragment's tokens, got %d", len(hit.Edges))
	}
	seen := map[string]bool{}
	for _, e := range hit.Edges {
		seen[e.Src+"-"+e.Kind+"->"+e.Dst] = true
	}
	if !seen["Jurgen Vinju-advised_on->Homoiconicity"] {
		t.Error("missing edge Jurgen Vinju advised_on Homoiconicity")
	}
	if !seen["Rascal-contrasts_with->Racket"] {
		t.Error("missing edge Rascal contrasts_with Racket")
	}
}

func TestSearch_EdgesScopedToFragmentTokens(t *testing.T) {
	s := newStore(t)
	seedForSearch(t, s)

	// Fragment 2 has Rob, Rascal, Term Rewriting. The Rascal<->Racket edge
	// from fragment 1 should NOT appear in fragment 2's hit because Racket
	// is not among fragment 2's tokens.
	hits, err := s.Search(context.Background(), store.SearchQuery{Text: "scheduling OR coffee OR rewriting"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	var frag2 *store.SearchHit
	for i := range hits {
		if hits[i].FragmentID == 2 {
			frag2 = &hits[i]
			break
		}
	}
	if frag2 == nil {
		t.Fatalf("expected fragment 2 in results, got ids %v", hitIDs(hits))
	}
	for _, e := range frag2.Edges {
		if e.Dst == "Racket" || e.Src == "Racket" {
			t.Errorf("fragment 2 should not include edges involving Racket: %+v", e)
		}
	}
	if len(frag2.Edges) != 1 || frag2.Edges[0].Kind != "uses" {
		t.Errorf("expected exactly one 'uses' edge for fragment 2, got %+v", frag2.Edges)
	}
}

func TestSearch_UnknownTokenReturnsEmpty(t *testing.T) {
	s := newStore(t)
	seedForSearch(t, s)

	hits, err := s.Search(context.Background(), store.SearchQuery{TokenName: "no such entity"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 0 {
		t.Errorf("expected 0 hits for unknown token, got %d", len(hits))
	}
}

func hitIDs(hits []store.SearchHit) []int64 {
	ids := make([]int64, len(hits))
	for i, h := range hits {
		ids[i] = h.FragmentID
	}
	return ids
}
