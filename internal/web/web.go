// Copyright (C) 2026 Rob Augustinus
// SPDX-License-Identifier: MIT

package web

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/raugustinus/knoop/internal/store"
)

//go:embed index.html
var indexHTML []byte

type graphJSON struct {
	Nodes []nodeEl `json:"nodes"`
	Edges []edgeEl `json:"edges"`
}

type nodeEl struct {
	Data nodeData `json:"data"`
}

type nodeData struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Kind  string `json:"kind"`
}

type edgeEl struct {
	Data edgeData `json:"data"`
}

type edgeData struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label"`
}

func Serve(addr string, s *store.Store) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexHTML)
	})
	mux.HandleFunc("/graph.json", graphHandler(s))

	log.Printf("knoop: http viewer on %s", addr)
	return http.ListenAndServe(addr, mux)
}

func graphHandler(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		g, err := fetchGraph(r.Context(), s)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(g)
	}
}

func fetchGraph(ctx context.Context, s *store.Store) (*graphJSON, error) {
	g := &graphJSON{Nodes: []nodeEl{}, Edges: []edgeEl{}}

	trows, err := s.DB().QueryContext(ctx, `SELECT id, kind, name FROM tokens ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer trows.Close()
	for trows.Next() {
		var id int64
		var kind, name string
		if err := trows.Scan(&id, &kind, &name); err != nil {
			return nil, err
		}
		g.Nodes = append(g.Nodes, nodeEl{Data: nodeData{
			ID:    strconv.FormatInt(id, 10),
			Label: name,
			Kind:  kind,
		}})
	}
	if err := trows.Err(); err != nil {
		return nil, err
	}

	erows, err := s.DB().QueryContext(ctx, `SELECT src, dst, kind FROM edges ORDER BY src, dst, kind`)
	if err != nil {
		return nil, err
	}
	defer erows.Close()
	for erows.Next() {
		var src, dst int64
		var kind string
		if err := erows.Scan(&src, &dst, &kind); err != nil {
			return nil, err
		}
		g.Edges = append(g.Edges, edgeEl{Data: edgeData{
			ID:     fmt.Sprintf("%d-%d-%s", src, dst, kind),
			Source: strconv.FormatInt(src, 10),
			Target: strconv.FormatInt(dst, 10),
			Label:  kind,
		}})
	}
	if err := erows.Err(); err != nil {
		return nil, err
	}

	return g, nil
}
