PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;

CREATE TABLE IF NOT EXISTS fragments (
  id         INTEGER PRIMARY KEY,
  body       TEXT NOT NULL,
  source     TEXT,
  author     TEXT NOT NULL DEFAULT '',
  visibility TEXT NOT NULL DEFAULT 'private'
             CHECK (visibility IN ('private','team','public')),
  created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE VIRTUAL TABLE IF NOT EXISTS fragments_fts USING fts5(
  body, content='fragments', content_rowid='id'
);

CREATE TRIGGER IF NOT EXISTS fragments_ai AFTER INSERT ON fragments BEGIN
  INSERT INTO fragments_fts(rowid, body) VALUES (new.id, new.body);
END;

CREATE TRIGGER IF NOT EXISTS fragments_ad AFTER DELETE ON fragments BEGIN
  INSERT INTO fragments_fts(fragments_fts, rowid, body) VALUES('delete', old.id, old.body);
END;

CREATE TABLE IF NOT EXISTS tokens (
  id         INTEGER PRIMARY KEY,
  kind       TEXT NOT NULL,
  name       TEXT NOT NULL,
  name_norm  TEXT NOT NULL GENERATED ALWAYS AS (lower(trim(name))) STORED,
  data       TEXT,
  created_at INTEGER NOT NULL DEFAULT (unixepoch()),
  UNIQUE(kind, name_norm)
);

CREATE TABLE IF NOT EXISTS edge_types (
  name          TEXT PRIMARY KEY,
  status        TEXT NOT NULL CHECK (status IN ('proposed','stable','deprecated')),
  description   TEXT NOT NULL,
  domain_kind   TEXT,
  range_kind    TEXT,
  symmetric     INTEGER NOT NULL DEFAULT 0 CHECK (symmetric IN (0,1)),
  superseded_by TEXT REFERENCES edge_types(name),
  created_at    INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE IF NOT EXISTS edges (
  src        INTEGER NOT NULL REFERENCES tokens(id),
  dst        INTEGER NOT NULL REFERENCES tokens(id),
  kind       TEXT NOT NULL REFERENCES edge_types(name),
  data       TEXT,
  created_at INTEGER NOT NULL DEFAULT (unixepoch()),
  PRIMARY KEY (src, dst, kind)
);

CREATE INDEX IF NOT EXISTS edges_dst ON edges(dst, kind);
CREATE INDEX IF NOT EXISTS edges_src_kind ON edges(src, kind);

CREATE TABLE IF NOT EXISTS fragment_tokens (
  fragment_id INTEGER NOT NULL REFERENCES fragments(id) ON DELETE CASCADE,
  token_id    INTEGER NOT NULL REFERENCES tokens(id),
  role        TEXT NOT NULL DEFAULT 'mentions',
  PRIMARY KEY (fragment_id, token_id, role)
);

CREATE INDEX IF NOT EXISTS fragment_tokens_token ON fragment_tokens(token_id);
