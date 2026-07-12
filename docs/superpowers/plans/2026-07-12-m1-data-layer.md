# M1 Data Layer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** The binary opens (or creates) `tournament.db` in the OS data directory, with the full 14-table schema from the data-model spec applied via embedded migrations, exposed through a typed `internal/store` package.

**Architecture:** A `datadir` package resolves where data lives (env override → OS config dir). A `store` package owns the SQLite database: DSN pragmas (foreign keys, WAL, busy timeout) per connection, embedded `.sql` migrations tracked via `PRAGMA user_version`, and typed accessors (Tournament CRUD sets the pattern; other entities follow in M2 with their API endpoints). The HTTP server gains a mux constructor so `/healthz` can report DB reachability and tests can drive it.

**Tech Stack:** Go 1.26 stdlib + `modernc.org/sqlite` (pure-Go driver — the only new dependency; keeps `CGO_ENABLED=0` cross-compilation working).

## Global Constraints

- Go toolchain is at `/usr/local/go/bin` and NOT on the default PATH: prefix every `go`/`make` invocation with `export PATH=$PATH:/usr/local/go/bin`.
- Module: `github.com/RoboCup-SSL/ssl-tournament-package`. Commits go directly to `main` (no PR flow yet); keep `go vet ./...`, `go test ./...`, `go build ./...` green at every commit.
- `CGO_ENABLED=0` builds must keep working (Makefile exports it). No cgo, no other new dependencies.
- Package installs resolve through the pre-configured internal proxy. Do not add `--registry`/`GOPROXY` flags or edit proxy config; if `go get` fails, stop and report it.
- Comment style: minimal inline comments; short doc comment at the top of each file and on each function (Go convention: starts with the name). Code must read without narration.
- Schema source of truth: `docs/superpowers/specs/2026-07-10-tournament-data-model-design.md` (the ```sql block under "## SQL schema"). The migration file must match it exactly, minus the `PRAGMA foreign_keys = ON;` line (that pragma is applied per-connection via the DSN instead).
- End every commit message with: `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`
- SQLite test gotcha: never use `:memory:` DSNs with `database/sql` (each pooled connection would get its own empty database). Always use a file in `t.TempDir()`.

---

### Task 1: `internal/datadir` — resolve the data directory

**Files:**
- Create: `internal/datadir/datadir.go`
- Test: `internal/datadir/datadir_test.go`

**Interfaces:**
- Consumes: nothing (stdlib only).
- Produces: `datadir.Resolve() (string, error)` — returns the data directory (creating it if missing): `$SSL_TOURNAMENT_DATA_DIR` if non-empty, else `<os.UserConfigDir()>/ssl-tournament`. Also `datadir.EnvVar = "SSL_TOURNAMENT_DATA_DIR"` (the systemd unit in `cmd/ssl-tournament-service/main.go` already sets this env var — the name must match it exactly).

- [ ] **Step 1: Write the failing tests**

```go
// Package datadir tests.
package datadir

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestResolveEnvOverride verifies the env var wins and the directory is created.
func TestResolveEnvOverride(t *testing.T) {
	want := filepath.Join(t.TempDir(), "data")
	t.Setenv(EnvVar, want)

	dir, err := Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if dir != want {
		t.Fatalf("dir = %q, want %q", dir, want)
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Fatalf("directory not created: %v", err)
	}
}

// TestResolveDefault verifies the fallback under the user config dir.
func TestResolveDefault(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("XDG_CONFIG_HOME override is linux-specific")
	}
	t.Setenv(EnvVar, "")
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)

	dir, err := Resolve()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(cfg, "ssl-tournament")
	if dir != want {
		t.Fatalf("dir = %q, want %q", dir, want)
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Fatalf("directory not created: %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `export PATH=$PATH:/usr/local/go/bin && go test ./internal/datadir/ -v`
Expected: FAIL to build with `undefined: EnvVar` / `undefined: Resolve`.

- [ ] **Step 3: Write the implementation**

```go
// Package datadir resolves the per-user directory where the tournament
// database and assets live — never next to the binary.
package datadir

import (
	"os"
	"path/filepath"
)

// EnvVar overrides the data directory when set. The systemd unit installed by
// ssl-tournament-service sets it to /var/lib/ssl-tournament.
const EnvVar = "SSL_TOURNAMENT_DATA_DIR"

// Resolve returns the data directory, creating it if needed: $SSL_TOURNAMENT_DATA_DIR
// when non-empty, otherwise <os.UserConfigDir()>/ssl-tournament.
func Resolve() (string, error) {
	dir := os.Getenv(EnvVar)
	if dir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(base, "ssl-tournament")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `export PATH=$PATH:/usr/local/go/bin && go test ./internal/datadir/ -v`
Expected: both tests PASS (TestResolveDefault may SKIP on non-linux).

- [ ] **Step 5: Commit**

```bash
git add internal/datadir/
git commit -m "feat: datadir package — resolve per-user data directory

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 2: `internal/store` — open SQLite with the right pragmas

**Files:**
- Create: `internal/store/store.go`
- Test: `internal/store/store_test.go`
- Modify: `go.mod` / `go.sum` (via `go get`)

**Interfaces:**
- Consumes: `modernc.org/sqlite` (blank-imported driver, name `"sqlite"`).
- Produces: `store.Open(path string) (*store.Store, error)`, `(*store.Store).Close() error`, `(*store.Store).Ping() error`. `Store` has an unexported `db *sql.DB` field (accessible to same-package tests and later same-package accessors). Every connection gets `foreign_keys=1`, `journal_mode=WAL`, `busy_timeout=5000` via DSN `_pragma` parameters.

- [ ] **Step 1: Add the dependency**

Run: `export PATH=$PATH:/usr/local/go/bin && go get modernc.org/sqlite@latest`
Expected: `go.mod` gains `modernc.org/sqlite` (plus indirect modernc.org/* deps in go.sum). If the module proxy refuses, STOP and report — do not repoint.

- [ ] **Step 2: Write the failing test**

```go
// Package store tests.
package store

import (
	"path/filepath"
	"testing"
)

// openTestStore opens a fresh file-backed store in a temp dir.
func openTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// TestOpenSetsPragmas verifies foreign keys and WAL are active on connections.
func TestOpenSetsPragmas(t *testing.T) {
	s := openTestStore(t)
	if err := s.Ping(); err != nil {
		t.Fatal(err)
	}

	var fk int
	if err := s.db.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatal(err)
	}
	if fk != 1 {
		t.Fatalf("foreign_keys = %d, want 1", fk)
	}

	var mode string
	if err := s.db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if mode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", mode)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `export PATH=$PATH:/usr/local/go/bin && go test ./internal/store/ -v`
Expected: FAIL to build with `undefined: Open`.

- [ ] **Step 4: Write the implementation**

```go
// Package store owns the SQLite tournament database: opening it with the
// right pragmas, applying migrations, and providing typed accessors.
package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Store wraps the tournament database.
type Store struct {
	db *sql.DB
}

// Open opens (or creates) the SQLite database at path. Every pooled connection
// gets foreign-key enforcement, WAL journaling, and a 5s busy timeout.
func Open(path string) (*Store, error) {
	dsn := fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)",
		path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

// Close closes the underlying database.
func (s *Store) Close() error { return s.db.Close() }

// Ping verifies the database is reachable.
func (s *Store) Ping() error { return s.db.Ping() }
```

- [ ] **Step 5: Run test to verify it passes**

Run: `export PATH=$PATH:/usr/local/go/bin && go test ./internal/store/ -v`
Expected: PASS.

- [ ] **Step 6: Verify the static cross-compile still works**

Run: `export PATH=$PATH:/usr/local/go/bin && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./internal/... ./cmd/ssl-tournament && CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build ./internal/... ./cmd/ssl-tournament`
Expected: both succeed (pure-Go driver confirmed). The service binary is excluded — it is a Linux-only artifact (see Makefile `SERVICE_PLATFORMS`).

- [ ] **Step 7: Commit**

```bash
git add go.mod go.sum internal/store/
git commit -m "feat: store package — open SQLite via modernc.org/sqlite with FK/WAL pragmas

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 3: Embedded migrations — the full schema

**Files:**
- Create: `internal/store/migrations/0001_init.sql`
- Create: `internal/store/migrate.go`
- Modify: `internal/store/store.go` (call `migrate` in `Open`)
- Test: `internal/store/migrate_test.go`

**Interfaces:**
- Consumes: `Store`/`Open` from Task 2.
- Produces: `migrate(db *sql.DB) error` (unexported; runs inside `Open`). Migrations are `internal/store/migrations/NNNN_*.sql`, embedded, applied in filename order; file at sorted position i applies as version i+1; `PRAGMA user_version` tracks the current version. Each migration runs in its own transaction.

- [ ] **Step 1: Create the migration SQL**

Create `internal/store/migrations/0001_init.sql` containing the schema below. It is the spec's ```sql block **verbatim minus the `PRAGMA foreign_keys = ON;` line** (that pragma comes from the DSN):

```sql
CREATE TABLE tournament (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  name       TEXT NOT NULL,
  location   TEXT NOT NULL DEFAULT '',
  starts_on  TEXT,                        -- ISO-8601 date, nullable
  ends_on    TEXT,
  -- Daily venue hours, HH:MM local, nullable. Advisory warns about matches/
  -- bookings outside them. Day-specific exceptions (setup day, early close)
  -- are expressed as field_booking kind='blocked' rows.
  venue_opens  TEXT,                      -- e.g. "08:00"
  venue_closes TEXT,                      -- e.g. "22:00"
  -- Schedule-planning defaults, stored as plain facts (the automated scheduler
  -- that would use them is a later, non-MVP feature). RoboCup: 60 + 30;
  -- Japan Open: 45. Per-match override: match.duration_minutes.
  default_match_minutes INTEGER,          -- planned slot length per match
  default_gap_minutes   INTEGER,          -- turnaround between matches on a field
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

-- Freeform partition label ("A", "B", "Entry Level", ...). No ruleset semantics:
-- it only says which teams compete against one another. A table (not a string on
-- team) so references can't typo, and metadata has a home.
CREATE TABLE division (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  tournament_id INTEGER NOT NULL REFERENCES tournament(id) ON DELETE CASCADE,
  name          TEXT NOT NULL,
  notes         TEXT NOT NULL DEFAULT ''
);

CREATE TABLE team (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  tournament_id INTEGER NOT NULL REFERENCES tournament(id) ON DELETE CASCADE,
  division_id   INTEGER REFERENCES division(id) ON DELETE SET NULL,
  name          TEXT NOT NULL,
  country       TEXT NOT NULL DEFAULT '',
  contact       TEXT NOT NULL DEFAULT '',
  notes         TEXT NOT NULL DEFAULT '',  -- freeform ("merged from X+Y", "shares robots with Z", DQ reason)
  withdrawn_at  TEXT,      -- set when the team withdraws / is disqualified;
                           -- teams are withdrawn, never deleted, once they have history
  created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE field (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  tournament_id INTEGER NOT NULL REFERENCES tournament(id) ON DELETE CASCADE,
  name          TEXT NOT NULL
);

-- A reservation of a field for a time range — anything that isn't a match.
--   kind='booking': someone uses the field (practice slot, calibration);
--     claims are whole-field, "north half" etc. goes in notes; the advisory
--     layer warns when overlapping claims can't fit (3+ teams at once).
--   kind='blocked': field wholly unavailable (maintenance, venue closed,
--     announcement); the advisory layer warns on ANY overlapping match/booking.
-- team_id NULL = non-team row (crew bookings, blocks). Never enforced, per the
-- freeform principle — a match on a blocked field is legal data plus a warning.
CREATE TABLE field_booking (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  tournament_id INTEGER NOT NULL REFERENCES tournament(id) ON DELETE CASCADE,
  field_id      INTEGER REFERENCES field(id) ON DELETE SET NULL,
  team_id       INTEGER REFERENCES team(id)  ON DELETE SET NULL,
  kind          TEXT NOT NULL DEFAULT 'booking'
                  CHECK (kind IN ('booking','blocked')),
  label         TEXT NOT NULL DEFAULT '',     -- "practice", "calibration", "venue closed"
  starts_at     TEXT,                         -- ISO-8601 datetime
  ends_at       TEXT,
  notes         TEXT NOT NULL DEFAULT ''
);

-- Domain concept: "group" (a round-robin pool). Named team_group because GROUP is
-- a reserved SQL keyword.
CREATE TABLE team_group (
  id                   INTEGER PRIMARY KEY AUTOINCREMENT,
  tournament_id        INTEGER NOT NULL REFERENCES tournament(id) ON DELETE CASCADE,
  division_id          INTEGER REFERENCES division(id) ON DELETE SET NULL,
  name                 TEXT NOT NULL,       -- e.g. "G1"
  notes                TEXT NOT NULL DEFAULT '',  -- freeform (e.g. why a confirmed ranking deviates)
  ranking_confirmed_at TEXT                 -- null until organizer blesses the order
);

CREATE TABLE team_group_member (
  group_id INTEGER NOT NULL REFERENCES team_group(id) ON DELETE CASCADE,
  team_id  INTEGER NOT NULL REFERENCES team(id)       ON DELETE CASCADE,
  PRIMARY KEY (group_id, team_id)
);

-- Organizer-confirmed within-group ranking; what group_rank references resolve to.
-- One rank per team (structural), but NOT unique per rank: confirmed ties are legal.
CREATE TABLE group_ranking (
  group_id INTEGER NOT NULL REFERENCES team_group(id) ON DELETE CASCADE,
  team_id  INTEGER NOT NULL REFERENCES team(id)       ON DELETE CASCADE,
  rank     INTEGER NOT NULL,                -- 1-based
  PRIMARY KEY (group_id, team_id)
);

-- The match row stores resolved facts only; bracket wiring lives in slot_source.
CREATE TABLE match (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  tournament_id INTEGER NOT NULL REFERENCES tournament(id)  ON DELETE CASCADE,
  division_id   INTEGER          REFERENCES division(id)    ON DELETE SET NULL,
  group_id      INTEGER          REFERENCES team_group(id)  ON DELETE CASCADE, -- set = group match
  label         TEXT NOT NULL DEFAULT '',        -- "Upper 1", "Grand Final", "G1"
  field_id      INTEGER          REFERENCES field(id)       ON DELETE SET NULL,
  scheduled_at  TEXT,                             -- ISO-8601 datetime, nullable
  duration_minutes INTEGER,                       -- overrides tournament.default_match_minutes
  -- cancelled = never happened / won't happen; invalidated = played, result void
  -- (disqualification, score-entry error); suspended = interrupted mid-match
  -- (power/vision failure), partial scores kept, scheduled_at updated to the
  -- resumption slot; forfeited = counts like finished but was not (fully) played
  -- (no-show, walkover, resignation) — distinguishes a real 10:0 from a skipped
  -- match, so e.g. gamelog validation knows not to expect a log.
  -- Standings count 'finished' and 'forfeited'.
  status        TEXT NOT NULL DEFAULT 'scheduled'
                  CHECK (status IN ('scheduled','playing','suspended','finished','forfeited','cancelled','invalidated')),

  -- Duty teams: referee + GC operator, assistant referee + vision operator.
  referee_team_id           INTEGER REFERENCES team(id) ON DELETE SET NULL,
  assistant_referee_team_id INTEGER REFERENCES team(id) ON DELETE SET NULL,

  -- Participants (symmetric; slot order is cosmetic). NULL until known/resolved.
  -- Fouls/cards are final per-team counts: *evidence* for score-independent
  -- results (e.g. Japan Open's fewer-fouls-wins on equal goals) — the decision
  -- itself is always winner_team_id. Richer stats stay in the event payload.
  a_team_id      INTEGER REFERENCES team(id) ON DELETE SET NULL,
  a_score        INTEGER,
  a_fouls        INTEGER,
  a_yellow_cards INTEGER,
  a_red_cards    INTEGER,
  b_team_id      INTEGER REFERENCES team(id) ON DELETE SET NULL,
  b_score        INTEGER,
  b_fouls        INTEGER,
  b_yellow_cards INTEGER,
  b_red_cards    INTEGER,

  -- Authoritative result. Set for finished matches; handles knockout shootouts where
  -- a_score == b_score but a winner is still decided. NULL + equal scores = a draw
  -- (valid in group play). match_winner/match_loser references read this.
  winner_team_id INTEGER REFERENCES team(id) ON DELETE SET NULL,
  notes          TEXT NOT NULL DEFAULT '',  -- freeform ("suspended at 4-2", "walkover", "reffed by Nicolai")
  created_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

-- Bracket wiring: where a slot's team will come from, for slots not directly
-- assigned. At most one source per slot (structural). Resolution (M3) reads this
-- and writes the concrete team into match.a_team_id / b_team_id.
CREATE TABLE slot_source (
  match_id     INTEGER NOT NULL REFERENCES match(id) ON DELETE CASCADE,
  slot         TEXT    NOT NULL CHECK (slot IN ('a','b')),
  kind         TEXT    NOT NULL CHECK (kind IN ('group_rank','match_winner','match_loser')),
  ref_group_id INTEGER REFERENCES team_group(id) ON DELETE SET NULL,  -- for group_rank
  ref_rank     INTEGER,                                               -- for group_rank
  ref_match_id INTEGER REFERENCES match(id)      ON DELETE SET NULL,  -- for match_winner/loser
  PRIMARY KEY (match_id, slot)
);

-- Final standings. Same reference vocabulary as slot_source: a rank can come
-- from a match outcome ("1st = winner of Grand Final") OR straight from a group
-- rank ("3rd-5th = group ranks 3-5", the German Open shape).
-- No unique rank: shared placements (e.g. two 3rds) are legal.
CREATE TABLE placement (
  id               INTEGER PRIMARY KEY AUTOINCREMENT,
  tournament_id    INTEGER NOT NULL REFERENCES tournament(id) ON DELETE CASCADE,
  division_id      INTEGER REFERENCES division(id) ON DELETE SET NULL,
  rank             INTEGER NOT NULL,
  label            TEXT NOT NULL DEFAULT '',     -- e.g. "Champion"
  source_kind      TEXT CHECK (source_kind IN ('group_rank','match_winner','match_loser')),
  source_group_id  INTEGER REFERENCES team_group(id) ON DELETE SET NULL,  -- for group_rank
  source_rank      INTEGER,                                               -- for group_rank
  source_match_id  INTEGER REFERENCES match(id)      ON DELETE SET NULL,  -- for match_winner/loser
  resolved_team_id INTEGER REFERENCES team(id)       ON DELETE SET NULL
);

-- Pending-event queue (external producers push; organizer approves). See architecture.
CREATE TABLE event (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  tournament_id INTEGER REFERENCES tournament(id) ON DELETE CASCADE,
  type          TEXT NOT NULL,                   -- e.g. "match_ended"
  match_id      INTEGER REFERENCES match(id) ON DELETE SET NULL,
  payload       TEXT NOT NULL DEFAULT '{}',      -- JSON
  source        TEXT NOT NULL DEFAULT '',        -- producer identity
  -- Advisory grouping key. Dedupe happens at the APP level (drop an incoming event
  -- only while an identical (source, dedupe_key) row is still pending) and in the
  -- queue UI (group same-key rows). Deliberately NOT a UNIQUE constraint: two GC
  -- instances (or one restarting) can legitimately emit colliding keys, and the DB
  -- must never reject a producer fact — the human-reviewed queue is the last line.
  dedupe_key    TEXT,
  status        TEXT NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending','approved','rejected')),
  received_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
  resolved_at   TEXT,
  resolved_by   INTEGER REFERENCES user(id) ON DELETE SET NULL
);

-- Instance-global auth (not tournament-scoped).
CREATE TABLE user (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  username      TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,                   -- bcrypt
  role          TEXT NOT NULL DEFAULT 'viewer'
                  CHECK (role IN ('organizer','viewer')),
  created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE token (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  name       TEXT NOT NULL,
  token_hash TEXT NOT NULL UNIQUE,               -- hashed bearer token
  scope      TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
  revoked_at TEXT
);

-- Indexes (SQLite does not auto-index foreign keys).
CREATE INDEX idx_division_tournament  ON division(tournament_id);
CREATE INDEX idx_team_tournament      ON team(tournament_id);
CREATE INDEX idx_field_tournament     ON field(tournament_id);
CREATE INDEX idx_booking_tournament   ON field_booking(tournament_id);
CREATE INDEX idx_booking_field        ON field_booking(field_id);
CREATE INDEX idx_group_tournament     ON team_group(tournament_id);
CREATE INDEX idx_group_member_team    ON team_group_member(team_id);
CREATE INDEX idx_match_tournament     ON match(tournament_id);
CREATE INDEX idx_match_group          ON match(group_id);
CREATE INDEX idx_match_field          ON match(field_id);
CREATE INDEX idx_slot_source_group    ON slot_source(ref_group_id);
CREATE INDEX idx_slot_source_match    ON slot_source(ref_match_id);
CREATE INDEX idx_placement_tournament ON placement(tournament_id);
CREATE INDEX idx_event_status         ON event(status);
CREATE INDEX idx_event_tournament     ON event(tournament_id);
```

- [ ] **Step 2: Verify the migration matches the spec**

Run:
```bash
diff <(awk '/^## SQL schema/{f=1} f&&/^```sql$/{inblk=1;next} inblk&&/^```$/{exit} inblk{print}' \
        docs/superpowers/specs/2026-07-10-tournament-data-model-design.md \
      | grep -v '^PRAGMA foreign_keys' | grep -v '^$') \
     <(grep -v '^$' internal/store/migrations/0001_init.sql)
```
Expected: empty output (no differences). If there is a diff, fix the migration file to match the spec, not the other way around.

- [ ] **Step 3: Write the failing tests**

Create `internal/store/migrate_test.go`:

```go
package store

import (
	"path/filepath"
	"reflect"
	"testing"
)

// wantTables is every table the initial migration must create, sorted.
var wantTables = []string{
	"division", "event", "field", "field_booking", "group_ranking",
	"match", "placement", "slot_source", "team", "team_group",
	"team_group_member", "token", "tournament", "user",
}

// TestMigrateCreatesSchema verifies all tables exist and user_version advanced.
func TestMigrateCreatesSchema(t *testing.T) {
	s := openTestStore(t)

	rows, err := s.db.Query(`SELECT name FROM sqlite_master
		WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		got = append(got, name)
	}
	if !reflect.DeepEqual(got, wantTables) {
		t.Fatalf("tables = %v, want %v", got, wantTables)
	}

	var version int
	if err := s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 1 {
		t.Fatalf("user_version = %d, want 1", version)
	}
}

// TestOpenIdempotent verifies reopening an already-migrated database succeeds.
func TestOpenIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	s1, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	s1.Close()

	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	var version int
	if err := s2.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 1 {
		t.Fatalf("user_version after reopen = %d, want 1", version)
	}
}
```

- [ ] **Step 4: Run tests to verify they fail**

Run: `export PATH=$PATH:/usr/local/go/bin && go test ./internal/store/ -v`
Expected: TestMigrateCreatesSchema FAILS (no tables — migrate doesn't exist yet).

- [ ] **Step 5: Write the migration runner**

Create `internal/store/migrate.go`:

```go
// Migration runner: embedded numbered .sql files applied in order, tracked
// via PRAGMA user_version.
package store

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// migrate applies every migrations/*.sql (in filename order) whose position
// exceeds the database's current PRAGMA user_version, each in its own
// transaction, advancing user_version as it goes.
func migrate(db *sql.DB) error {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)

	var current int
	if err := db.QueryRow("PRAGMA user_version").Scan(&current); err != nil {
		return err
	}

	for i, name := range names {
		version := i + 1
		if version <= current {
			continue
		}
		script, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(script)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", version)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s: setting user_version: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
```

Then modify `internal/store/store.go` — in `Open`, after `sql.Open` succeeds, insert:

```go
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
```

(directly before `return &Store{db: db}, nil`).

- [ ] **Step 6: Run tests to verify they pass**

Run: `export PATH=$PATH:/usr/local/go/bin && go test ./internal/store/ -v`
Expected: all PASS (including Task 2's pragma test).

- [ ] **Step 7: Commit**

```bash
git add internal/store/
git commit -m "feat: embedded migrations — full 14-table schema from the data-model spec

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 4: Schema-integrity regression tests

**Files:**
- Test: `internal/store/schema_test.go`

**Interfaces:**
- Consumes: `openTestStore(t)` helper from Task 2's `store_test.go`; the migrated schema from Task 3.
- Produces: nothing new — this task locks the spec's guarantees (FK enforcement, tournament cascade, freeform philosophy, vocabulary CHECKs) as regression tests so future schema changes can't silently violate them.

- [ ] **Step 1: Write the tests**

Create `internal/store/schema_test.go`:

```go
// Regression tests for the data-model spec's guarantees: mechanical integrity
// is enforced (FKs, vocabulary), logical rules are deliberately NOT (freeform,
// warn-don't-block), and deleting a tournament cascades to zero rows.
package store

import (
	"testing"
)

// mustExec fails the test if the statement errors.
func mustExec(t *testing.T, s *Store, q string, args ...any) {
	t.Helper()
	if _, err := s.db.Exec(q, args...); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
}

// seedGraph inserts one tournament with a row in every tournament-owned table.
func seedGraph(t *testing.T, s *Store) {
	t.Helper()
	mustExec(t, s, `INSERT INTO tournament(id,name) VALUES (1,'T')`)
	mustExec(t, s, `INSERT INTO division(id,tournament_id,name) VALUES (1,1,'B')`)
	mustExec(t, s, `INSERT INTO team(id,tournament_id,division_id,name) VALUES (1,1,1,'X'),(2,1,1,'Y')`)
	mustExec(t, s, `INSERT INTO field(id,tournament_id,name) VALUES (1,1,'Field A')`)
	mustExec(t, s, `INSERT INTO field_booking(tournament_id,field_id,team_id) VALUES (1,1,1)`)
	mustExec(t, s, `INSERT INTO team_group(id,tournament_id,division_id,name) VALUES (1,1,1,'G1')`)
	mustExec(t, s, `INSERT INTO team_group_member(group_id,team_id) VALUES (1,1),(1,2)`)
	mustExec(t, s, `INSERT INTO group_ranking(group_id,team_id,rank) VALUES (1,1,1),(1,2,2)`)
	mustExec(t, s, `INSERT INTO match(id,tournament_id,group_id,a_team_id,a_score,b_team_id,b_score,winner_team_id,status)
		VALUES (10,1,1,1,2,2,1,1,'finished')`)
	mustExec(t, s, `INSERT INTO match(id,tournament_id,label) VALUES (11,1,'Upper 1')`)
	mustExec(t, s, `INSERT INTO slot_source(match_id,slot,kind,ref_group_id,ref_rank) VALUES (11,'a','group_rank',1,1)`)
	mustExec(t, s, `INSERT INTO slot_source(match_id,slot,kind,ref_match_id) VALUES (11,'b','match_winner',10)`)
	mustExec(t, s, `INSERT INTO placement(tournament_id,rank,source_kind,source_match_id) VALUES (1,1,'match_winner',11)`)
	mustExec(t, s, `INSERT INTO event(tournament_id,type,match_id,source,dedupe_key) VALUES (1,'match_ended',10,'gc','m-10')`)
}

// TestForeignKeysRejectDangling verifies FK enforcement is on.
func TestForeignKeysRejectDangling(t *testing.T) {
	s := openTestStore(t)
	if _, err := s.db.Exec(`INSERT INTO team(tournament_id,name) VALUES (999,'X')`); err == nil {
		t.Fatal("insert with dangling tournament_id succeeded, want FK error")
	}
}

// TestTournamentCascade verifies deleting a tournament wipes everything it owns.
func TestTournamentCascade(t *testing.T) {
	s := openTestStore(t)
	seedGraph(t, s)
	mustExec(t, s, `DELETE FROM tournament WHERE id = 1`)

	for _, table := range []string{
		"division", "team", "field", "field_booking", "team_group",
		"team_group_member", "group_ranking", "match", "slot_source",
		"placement", "event",
	} {
		var n int
		if err := s.db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("%s: %d rows remain after cascade, want 0", table, n)
		}
	}
}

// TestFreeformIsLegal verifies the warn-don't-block principle: logically
// "wrong" plans are valid data the DB must accept.
func TestFreeformIsLegal(t *testing.T) {
	s := openTestStore(t)
	seedGraph(t, s)

	// Duplicate team name.
	mustExec(t, s, `INSERT INTO team(tournament_id,name) VALUES (1,'X')`)
	// A team playing against itself.
	mustExec(t, s, `INSERT INTO match(tournament_id,a_team_id,b_team_id) VALUES (1,1,1)`)
	// Two matches on one field at the same time.
	mustExec(t, s, `INSERT INTO match(tournament_id,field_id,scheduled_at) VALUES (1,1,'2026-07-16T09:00:00Z')`)
	mustExec(t, s, `INSERT INTO match(tournament_id,field_id,scheduled_at) VALUES (1,1,'2026-07-16T09:00:00Z')`)
	// Colliding (source, dedupe_key) — two GC instances; both rows must land.
	mustExec(t, s, `INSERT INTO event(tournament_id,type,source,dedupe_key) VALUES (1,'match_ended','gc','m-10')`)
	// A confirmed ranking tie and a shared placement rank.
	mustExec(t, s, `INSERT INTO team(id,tournament_id,name) VALUES (3,1,'Z')`)
	mustExec(t, s, `INSERT INTO group_ranking(group_id,team_id,rank) VALUES (1,3,1)`)
	mustExec(t, s, `INSERT INTO placement(tournament_id,rank,source_kind,source_match_id) VALUES (1,1,'match_loser',11)`)
	// A placement straight from a group rank (German Open shape).
	mustExec(t, s, `INSERT INTO placement(tournament_id,rank,source_kind,source_group_id,source_rank) VALUES (1,4,'group_rank',1,4)`)
	// suspended / forfeited / blocked vocabulary in action.
	mustExec(t, s, `INSERT INTO match(tournament_id,status,a_team_id,a_score,notes) VALUES (1,'suspended',1,4,'power cut at 4-2')`)
	mustExec(t, s, `INSERT INTO match(tournament_id,status,a_team_id,a_score,b_team_id,b_score,winner_team_id) VALUES (1,'forfeited',1,10,2,0,1)`)
	mustExec(t, s, `INSERT INTO field_booking(tournament_id,field_id,kind,label) VALUES (1,1,'blocked','venue closed')`)
}

// TestVocabularyIsGuarded verifies CHECK constraints reject unknown enum values.
func TestVocabularyIsGuarded(t *testing.T) {
	s := openTestStore(t)
	seedGraph(t, s)

	bad := []string{
		`INSERT INTO match(tournament_id,status) VALUES (1,'bogus')`,
		`INSERT INTO slot_source(match_id,slot,kind) VALUES (10,'c','group_rank')`,
		`INSERT INTO slot_source(match_id,slot,kind) VALUES (10,'a','bogus')`,
		`INSERT INTO placement(tournament_id,rank,source_kind) VALUES (1,1,'bogus')`,
		`INSERT INTO field_booking(tournament_id,kind) VALUES (1,'bogus')`,
		`INSERT INTO event(tournament_id,type,status) VALUES (1,'x','bogus')`,
		`INSERT INTO user(username,password_hash,role) VALUES ('u','h','bogus')`,
	}
	for _, q := range bad {
		if _, err := s.db.Exec(q); err == nil {
			t.Errorf("accepted bad vocabulary: %s", q)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `export PATH=$PATH:/usr/local/go/bin && go test ./internal/store/ -v`
Expected: all PASS. (These should pass immediately — the schema already guarantees this behavior; the tests exist so it stays guaranteed. If any FAIL, the migration file diverges from the spec: fix the migration, not the test.)

- [ ] **Step 3: Commit**

```bash
git add internal/store/schema_test.go
git commit -m "test: lock schema guarantees — FKs, cascade, freeform legality, vocabulary

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 5: Tournament CRUD — the accessor pattern

**Files:**
- Create: `internal/store/tournament.go`
- Test: `internal/store/tournament_test.go`

**Interfaces:**
- Consumes: `Store` with migrated schema.
- Produces (M2 endpoints and later entity accessors copy this exact pattern):
  - `store.ErrNotFound` (`var ErrNotFound = errors.New("not found")`)
  - `store.Tournament` struct — pointer fields for nullable columns:
    `ID int64; Name, Location string; StartsOn, EndsOn, VenueOpens, VenueCloses *string; DefaultMatchMinutes, DefaultGapMinutes *int64; CreatedAt string`
  - `(*Store).CreateTournament(t *Tournament) error` — inserts, fills `t.ID` + `t.CreatedAt`
  - `(*Store).GetTournament(id int64) (*Tournament, error)` — `ErrNotFound` if absent
  - `(*Store).ListTournaments() ([]Tournament, error)` — newest first
  - `(*Store).DeleteTournament(id int64) error` — `ErrNotFound` if absent; cascades

- [ ] **Step 1: Write the failing tests**

Create `internal/store/tournament_test.go`:

```go
package store

import (
	"errors"
	"testing"
)

// TestTournamentCRUD exercises the full create/get/list/delete cycle.
func TestTournamentCRUD(t *testing.T) {
	s := openTestStore(t)

	mins := int64(60)
	opens := "08:00"
	tr := &Tournament{Name: "RC2026", Location: "Incheon",
		DefaultMatchMinutes: &mins, VenueOpens: &opens}
	if err := s.CreateTournament(tr); err != nil {
		t.Fatal(err)
	}
	if tr.ID == 0 {
		t.Fatal("ID not filled after create")
	}
	if tr.CreatedAt == "" {
		t.Fatal("CreatedAt not filled after create")
	}

	got, err := s.GetTournament(tr.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "RC2026" || got.Location != "Incheon" {
		t.Fatalf("got %q/%q, want RC2026/Incheon", got.Name, got.Location)
	}
	if got.DefaultMatchMinutes == nil || *got.DefaultMatchMinutes != 60 {
		t.Fatalf("DefaultMatchMinutes = %v, want 60", got.DefaultMatchMinutes)
	}
	if got.VenueOpens == nil || *got.VenueOpens != "08:00" {
		t.Fatalf("VenueOpens = %v, want 08:00", got.VenueOpens)
	}
	if got.StartsOn != nil {
		t.Fatalf("StartsOn = %v, want nil", got.StartsOn)
	}

	list, err := s.ListTournaments()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != tr.ID {
		t.Fatalf("list = %+v, want the one created tournament", list)
	}

	if err := s.DeleteTournament(tr.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetTournament(tr.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("get after delete: err = %v, want ErrNotFound", err)
	}
	if err := s.DeleteTournament(tr.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second delete: err = %v, want ErrNotFound", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `export PATH=$PATH:/usr/local/go/bin && go test ./internal/store/ -v -run TestTournamentCRUD`
Expected: FAIL to build with `undefined: Tournament`.

- [ ] **Step 3: Write the implementation**

Create `internal/store/tournament.go`:

```go
// Tournament accessors — the CRUD pattern every entity accessor follows.
package store

import (
	"database/sql"
	"errors"
)

// ErrNotFound is returned when a requested row does not exist.
var ErrNotFound = errors.New("not found")

// Tournament mirrors one row of the tournament table; pointer fields are
// nullable columns.
type Tournament struct {
	ID                  int64
	Name                string
	Location            string
	StartsOn            *string
	EndsOn              *string
	VenueOpens          *string
	VenueCloses         *string
	DefaultMatchMinutes *int64
	DefaultGapMinutes   *int64
	CreatedAt           string
}

// CreateTournament inserts t and fills in its ID and CreatedAt.
func (s *Store) CreateTournament(t *Tournament) error {
	res, err := s.db.Exec(`INSERT INTO tournament
		(name, location, starts_on, ends_on, venue_opens, venue_closes,
		 default_match_minutes, default_gap_minutes)
		VALUES (?,?,?,?,?,?,?,?)`,
		t.Name, t.Location, t.StartsOn, t.EndsOn, t.VenueOpens, t.VenueCloses,
		t.DefaultMatchMinutes, t.DefaultGapMinutes)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	t.ID = id
	return s.db.QueryRow(`SELECT created_at FROM tournament WHERE id = ?`, id).
		Scan(&t.CreatedAt)
}

// GetTournament returns the tournament with the given id, or ErrNotFound.
func (s *Store) GetTournament(id int64) (*Tournament, error) {
	t := Tournament{ID: id}
	err := s.db.QueryRow(`SELECT name, location, starts_on, ends_on, venue_opens,
		venue_closes, default_match_minutes, default_gap_minutes, created_at
		FROM tournament WHERE id = ?`, id).
		Scan(&t.Name, &t.Location, &t.StartsOn, &t.EndsOn, &t.VenueOpens,
			&t.VenueCloses, &t.DefaultMatchMinutes, &t.DefaultGapMinutes, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// ListTournaments returns all tournaments, newest first.
func (s *Store) ListTournaments() ([]Tournament, error) {
	rows, err := s.db.Query(`SELECT id, name, location, starts_on, ends_on,
		venue_opens, venue_closes, default_match_minutes, default_gap_minutes,
		created_at FROM tournament ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tournament
	for rows.Next() {
		var t Tournament
		if err := rows.Scan(&t.ID, &t.Name, &t.Location, &t.StartsOn, &t.EndsOn,
			&t.VenueOpens, &t.VenueCloses, &t.DefaultMatchMinutes,
			&t.DefaultGapMinutes, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// DeleteTournament deletes the tournament and, via cascades, everything it
// owns. Returns ErrNotFound if no such row exists.
func (s *Store) DeleteTournament(id int64) error {
	res, err := s.db.Exec(`DELETE FROM tournament WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `export PATH=$PATH:/usr/local/go/bin && go test ./internal/store/ -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/store/tournament.go internal/store/tournament_test.go
git commit -m "feat: tournament CRUD — the store accessor pattern

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

### Task 6: Wire the store into the server and both binaries

**Files:**
- Modify: `internal/server/server.go` (full replacement below)
- Modify: `cmd/ssl-tournament/main.go` (full replacement below)
- Modify: `cmd/ssl-tournament-service/main.go` (only `main()` and imports change; shown below)
- Test: `internal/server/server_test.go`

**Interfaces:**
- Consumes: `datadir.Resolve()` (Task 1), `store.Open` / `Ping` / `Close` (Tasks 2–3).
- Produces: `server.NewMux(st *store.Store) *http.ServeMux` (routes: `/healthz` pings the DB — 200 `ok` / 503 `db unreachable`; `/` serves the embedded UI) and `server.Run(host, port, version string, st *store.Store) error`. Both binaries resolve the data dir, open `<dir>/tournament.db`, log the path, and pass the store to `Run`. The service binary opens the store **only** in `--serve` mode (install/uninstall never touch the DB).

- [ ] **Step 1: Write the failing test**

Create `internal/server/server_test.go`:

```go
// Package server tests.
package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/RoboCup-SSL/ssl-tournament-package/internal/store"
)

// TestHealthzReportsDB verifies /healthz pings the database.
func TestHealthzReportsDB(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	srv := httptest.NewServer(NewMux(st))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || string(body) != "ok" {
		t.Fatalf("healthz = %d %q, want 200 ok", resp.StatusCode, body)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `export PATH=$PATH:/usr/local/go/bin && go test ./internal/server/ -v`
Expected: FAIL to build with `undefined: NewMux`.

- [ ] **Step 3: Update the server**

Replace `internal/server/server.go` with:

```go
// Package server runs the HTTP API and embedded web UI shared by both binaries.
package server

import (
	"log"
	"net/http"

	"github.com/RoboCup-SSL/ssl-tournament-package/frontend"
	"github.com/RoboCup-SSL/ssl-tournament-package/internal/store"
)

// NewMux returns the HTTP routes: /healthz (reports database reachability)
// and the embedded web UI at /.
func NewMux(st *store.Store) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := st.Ping(); err != nil {
			http.Error(w, "db unreachable", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("/", frontend.Handler())
	return mux
}

// Run starts the HTTP server on host:port and blocks until it exits.
func Run(host, port, version string, st *store.Store) error {
	addr := host + ":" + port
	log.Printf("ssl-tournament %s serving on http://%s", version, addr)
	return http.ListenAndServe(addr, NewMux(st))
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `export PATH=$PATH:/usr/local/go/bin && go test ./internal/server/ -v`
Expected: PASS. (`go build ./...` will now fail — the binaries still call the old `Run` signature; fixed next.)

- [ ] **Step 5: Wire the plain binary**

Replace `cmd/ssl-tournament/main.go` with:

```go
// Command ssl-tournament runs the tournament server (HTTP API + embedded web
// UI). version is set at build time via -ldflags "-X main.version".
package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"github.com/RoboCup-SSL/ssl-tournament-package/internal/datadir"
	"github.com/RoboCup-SSL/ssl-tournament-package/internal/server"
	"github.com/RoboCup-SSL/ssl-tournament-package/internal/store"
)

var version = "dev"

// main parses flags, opens the tournament database, and starts the server.
func main() {
	host := flag.String("host", "0.0.0.0", "The host/interface to bind on")
	port := flag.String("port", "8080", "The port to serve on")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	dir, err := datadir.Resolve()
	if err != nil {
		log.Fatal(err)
	}
	dbPath := filepath.Join(dir, "tournament.db")
	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()
	log.Printf("database: %s", dbPath)

	if err := server.Run(*host, *port, version, st); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 6: Wire the service binary**

In `cmd/ssl-tournament-service/main.go`, add to the imports:

```go
	"path/filepath"

	"github.com/RoboCup-SSL/ssl-tournament-package/internal/datadir"
	"github.com/RoboCup-SSL/ssl-tournament-package/internal/store"
```

and replace the `case *serve:` branch of `main()`'s switch with:

```go
	case *serve:
		dir, err := datadir.Resolve()
		if err != nil {
			log.Fatal(err)
		}
		dbPath := filepath.Join(dir, "tournament.db")
		st, err := store.Open(dbPath)
		if err != nil {
			log.Fatal(err)
		}
		defer st.Close()
		log.Printf("database: %s", dbPath)
		if err := server.Run(bindHost, *port, version, st); err != nil {
			log.Fatal(err)
		}
```

(Install/uninstall branches stay untouched — they never open the DB. The systemd unit already sets `SSL_TOURNAMENT_DATA_DIR=/var/lib/ssl-tournament` and `StateDirectory=` creates it, so `datadir.Resolve()` lands there in service mode.)

- [ ] **Step 7: Full verification**

Run:
```bash
export PATH=$PATH:/usr/local/go/bin
go vet ./... && go test ./... && go build ./...
```
Expected: all clean and green.

Then end-to-end:
```bash
export PATH=$PATH:/usr/local/go/bin
DATA=$(mktemp -d)
go build -o /tmp/ssl-tournament-m1 ./cmd/ssl-tournament
SSL_TOURNAMENT_DATA_DIR=$DATA /tmp/ssl-tournament-m1 --host 127.0.0.1 --port 8137 &
SRV=$!
sleep 1
curl -s http://127.0.0.1:8137/healthz          # expect: ok
sqlite3 $DATA/tournament.db "SELECT count(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%';"  # expect: 14
kill $SRV; rm -rf $DATA /tmp/ssl-tournament-m1
```
Expected: `ok`, `14`, and the startup log shows `database: <tmpdir>/tournament.db`.

- [ ] **Step 8: Commit**

```bash
git add internal/server/ cmd/
git commit -m "feat: wire store into server and binaries — /healthz reports DB, data dir resolved

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```
