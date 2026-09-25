// Copyright (C) 2026 Rob Augustinus
// SPDX-License-Identifier: MIT

package mcpserver

import (
	"fmt"
	"strings"
	"time"

	"github.com/raugustinus/knoop/internal/store"
)

func FormatSearchHits(q store.SearchQuery, hits []store.SearchHit) string {
	desc := describeSearchQuery(q)
	if len(hits) == 0 {
		return fmt.Sprintf("No fragments matched %s.", desc)
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d fragment%s %s:\n\n", len(hits), plural(len(hits)), desc)
	writeHits(&sb, hits)
	return strings.TrimRight(sb.String(), "\n")
}

func FormatInboxHits(target string, hits []store.SearchHit) string {
	if len(hits) == 0 {
		return fmt.Sprintf("Inbox for %q is empty.", target)
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "Inbox for %q — %d new fragment%s:\n\n", target, len(hits), plural(len(hits)))
	writeHits(&sb, hits)
	return strings.TrimRight(sb.String(), "\n")
}

func writeHits(sb *strings.Builder, hits []store.SearchHit) {
	for i, h := range hits {
		when := time.Unix(h.CreatedAt, 0).UTC().Format("2006-01-02")
		fmt.Fprintf(sb, "### [%d] fragment #%d · %s", i+1, h.FragmentID, when)
		if h.Source != "" {
			fmt.Fprintf(sb, " · %s", h.Source)
		}
		if h.Author != "" {
			fmt.Fprintf(sb, " · by %s", h.Author)
		}
		sb.WriteString("\n")
		sb.WriteString(h.Body)
		sb.WriteString("\n\n")
		if len(h.Tokens) > 0 {
			sb.WriteString("Tokens: ")
			for j, t := range h.Tokens {
				if j > 0 {
					sb.WriteString(", ")
				}
				fmt.Fprintf(sb, "%s (%s)", t.Name, t.Kind)
			}
			sb.WriteString("\n")
		}
		if len(h.Edges) > 0 {
			sb.WriteString("Edges: ")
			for j, e := range h.Edges {
				if j > 0 {
					sb.WriteString("; ")
				}
				fmt.Fprintf(sb, "%s --%s--> %s", e.Src, e.Kind, e.Dst)
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}
}

func describeSearchQuery(q store.SearchQuery) string {
	text := strings.TrimSpace(q.Text)
	tok := strings.TrimSpace(q.TokenName)
	switch {
	case text != "" && tok != "":
		return fmt.Sprintf("matching %q AND mentioning %q", text, tok)
	case text != "":
		return fmt.Sprintf("matching %q", text)
	case tok != "":
		return fmt.Sprintf("mentioning %q", tok)
	default:
		return "(most recent)"
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
