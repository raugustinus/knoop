// Copyright (C) 2026 Rob Augustinus
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/server"

	"github.com/raugustinus/knoop/internal/mcpserver"
	"github.com/raugustinus/knoop/internal/store"
)

func main() {
	name := flag.String("name", "default", "graph name (selects DB file)")
	dbPath := flag.String("db", "", "explicit DB path (overrides -name)")
	author := flag.String("author", defaultAuthor(), "author recorded on each fragment")
	flag.Parse()

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
