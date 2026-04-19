# knoop

Thinking substrate, not agent memory.

## What it is

Knoop is a personal knowledge graph exposed over MCP. The name is Dutch
for *knot* — the term graph theory uses for a node. Knoop stores
fragments (observations), tokens (people, concepts, projects), and typed
edges between them. SQLite underneath, Go on top, distributed as a
single binary.

## Install

```
go install github.com/raugustinus/knoop/cmd/knoop@latest
```

Knoop uses `mattn/go-sqlite3`, which requires CGo: a C compiler must be
available at build time. On macOS, install the Xcode command line tools
(`xcode-select --install`). On Linux, install `build-essential` or the
equivalent toolchain package for your distribution.

## Claude Desktop configuration

Add knoop to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "knoop": {
      "command": "knoop",
      "args": ["-name", "personal"]
    }
  }
}
```

Each `-name <something>` value maps to its own SQLite file at
`~/.local/share/knoop/<name>.db` (or `$XDG_DATA_HOME/knoop/<name>.db`
if set). Back up a graph by copying that file. Add multiple entries
with different names — `personal`, `work`, `reading` — to keep
separate contexts.

## Worked example

A call Claude would make to `capture_fragment`:

```json
{
  "body": "Jurgen's key point: homoiconicity is elegant but the heavy lifting is provably correct transformations. Rascal treats term rewriting as primary; Racket treats syntax-parse as primary.",
  "source": "chat:claude/knoop-design/2026-04-19",
  "mentions": [
    {"ref": "jurgen", "kind": "person", "name": "Jurgen Vinju"},
    {"ref": "homoiconicity", "kind": "concept", "name": "Homoiconicity"},
    {"ref": "rascal", "kind": "project", "name": "Rascal"},
    {"ref": "racket", "kind": "project", "name": "Racket"}
  ],
  "edges": [
    {"src": "jurgen", "dst": "homoiconicity", "kind": "advised_on"},
    {"src": "rascal", "dst": "racket", "kind": "contrasts_with"}
  ]
}
```

One fragment row, four token rows, two edge rows — committed
atomically, or not at all.

## Schema overview

- `fragments` — the observations themselves, with author and visibility.
- `fragments_fts` — FTS5 virtual table shadowing `fragments.body` for future search.
- `tokens` — people, concepts, projects, etc., with case-insensitive uniqueness per kind.
- `edge_types` — the vocabulary: seeded with stable edges like `advised_on`, `contrasts_with`, `depends_on`.
- `edges` — typed directed edges between tokens.
- `fragment_tokens` — which fragments mention which tokens, and in what role.

## Design notes

- **Append-only vocabulary.** Edge types live in `edge_types` and are
  never deleted. Retired types are marked deprecated and can point at
  their successor via `superseded_by`, so historical queries stay
  honest.
- **Generated `name_norm` column.** Tokens are unique per
  `(kind, lower(trim(name)))` while preserving the original casing of
  `name` for display.
- **Single-transaction ingest.** One fragment, its tokens, and its
  edges commit atomically. A failure anywhere rolls the whole capture
  back.
- **Author and visibility from day one.** Every fragment carries these
  fields from the first row, so a future federation layer does not
  require a data migration.

## License

Knoop is MIT licensed — see [LICENSE](LICENSE). Fork it, embed it,
ship it commercially, keep your changes private: all fine. A mention
is appreciated but not required.

## Roadmap

- Query tool (FTS5 search + neighborhood traversal).
- Co-occurrence analysis across fragments.
- Edge type lifecycle helpers (propose / promote / deprecate).
- Batch import from chat logs.
- Maybe someday, a federation layer across multiple knoop instances.

## Acknowledgements

Knoop grew out of architecture conversations around PQC consultancy
work at Eoncore and FHE research at CodeFactoring.
