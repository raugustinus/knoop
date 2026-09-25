// Copyright (C) 2026 Rob Augustinus
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/server"

	"github.com/raugustinus/knoop/internal/mcpserver"
	"github.com/raugustinus/knoop/internal/store"
	"github.com/raugustinus/knoop/internal/web"
)

func main() {
	name := flag.String("name", "global", "graph name (selects DB file)")
	dbPath := flag.String("db", "", "explicit DB path (overrides -name)")
	author := flag.String("author", defaultAuthor(), "author recorded on each fragment")
	httpAddr := flag.String("http", "", "serve graph viewer on addr (e.g. :8080); if set, stdio MCP is disabled")
	inbox := flag.String("inbox", "", "drain inbox for a target project and print markdown to stdout (for use in SessionStart hooks)")
	inboxCwd := flag.Bool("inbox-cwd", false, "like -inbox, but derive the target from the git repo root (or cwd) basename; for user-scope SessionStart hooks")
	limit := flag.Int("limit", 10, "max fragments returned for -inbox")
	peek := flag.Bool("peek", false, "with -inbox: don't mark fragments as delivered")
	flag.Parse()

	if *inboxCwd && *inbox == "" {
		target := deriveInboxTarget()
		if target == "" {
			// no derivable target (e.g. bogus cwd); silent no-op so the
			// hook never blocks CC startup.
			return
		}
		*inbox = target
	}

	path := *dbPath
	if path == "" {
		path = resolveDBPath(*name)
	}

	if err := ensureParentDir(path); err != nil {
		log.Fatalf("ensure dir: %v", err)
	}

	s, err := store.Open(path, *author)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer s.Close()

	if *inbox != "" {
		hits, err := s.Inbox(context.Background(), store.InboxQuery{
			Target: *inbox,
			Limit:  *limit,
			Peek:   *peek,
		})
		if err != nil {
			log.Fatalf("inbox: %v", err)
		}
		fmt.Println(mcpserver.FormatInboxHits(*inbox, hits))
		return
	}

	if *httpAddr != "" {
		if err := web.Serve(*httpAddr, s); err != nil {
			log.Fatalf("serve http: %v", err)
		}
		return
	}

	srv := mcpserver.New(s)
	if err := server.ServeStdio(srv); err != nil {
		log.Fatalf("serve stdio: %v", err)
	}
}

func defaultAuthor() string {
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	return "unknown"
}

func resolveDBPath(name string) string {
	if base := os.Getenv("XDG_DATA_HOME"); base != "" {
		return filepath.Join(base, "knoop", name+".db")
	}
	return filepath.Join(os.Getenv("HOME"), ".local", "share", "knoop", name+".db")
}

func ensureParentDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}

func deriveInboxTarget() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	if out, err := exec.Command("git", "-C", cwd, "rev-parse", "--show-toplevel").Output(); err == nil {
		if root := strings.TrimSpace(string(out)); root != "" {
			return filepath.Base(root)
		}
	}
	return filepath.Base(cwd)
}
