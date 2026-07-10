# Tournament data model — design

- **Date:** 2026-07-10
- **Status:** draft for review
- **Milestone:** M1 (data layer). Domain *logic* (standings computation, reference
  resolution, ref suggestions) is M3; this spec defines only the **model + schema**
  those will operate on.

## Context & scope

The data model is the source of truth behind the API (see `docs/architecture.md`).
It must represent how RoboCup SSL events actually run — and, critically, how they
*vary*. Research across events (RoboCup Worlds, Japan Open, German Open, Schubert
Open, Brazil Open/LARC) shows the only invariants are:

- **teams**,
- **matches** between two teams with a per-team score,
- matches grouped into **phases** (round-robin and/or elimination) whose participants
  are **references** to other results.

Everything else varies per event and even per year:

| Event | Divisions | Group phase | Playoff |
|---|---|---|---|
| Worlds Div A | A (B run separately) | round-robin groups | double-elim |
| Worlds Div B (2026 Incheon) | B | round-robin groups | double-elim |
| Japan Open | single | round-robin, 2 pools | single-elim |
| German Open | single (4–6 teams) | one round-robin | just a final |
| Schubert Open | single (invitational) | one round-robin | placement matches |
| Brazil / LARC | single (+"Entry Level") | small round-robin | ad-hoc / short |

So division count, group count, and playoff style (**none / single-elim /
double-elim**) cannot be hardcoded — they are organizer-defined **data**, expressed
as a graph of matches whose slots reference upstream results.

## Key design decisions (deliberate assumptions)

1. **A match has exactly two symmetric participants** — `team_a` / `team_b`. Slot
   order is cosmetic (display only); it confers nothing. There is no home/away, and
   **no color** in the model — team color (blue/yellow) is chosen live by the referee
   and can even swap at halftime (SSL rules §4.3.3), so it belongs to the
   match-running layer (Game Controller), not here. Scores are keyed to the *team*,
   never a color.

2. **Participants are typed references.** Each slot's `source` is one of:
   `team` (direct), `group_rank` (`{group, rank}` — e.g. "winner of Group 1" =
   rank 1), or `match_winner` / `match_loser` (`{match}`). A `resolved_team_id`
   fills in once the upstream result is known. This one mechanism expresses the whole
   bracket graph, including double-elimination's winner-and-loser routing.

3. **Final placements use the same reference mechanism.** "5th = loser of Low 2.1",
   "1st = winner of Grand Final" — a placement references a match outcome, just like a
   slot does.

4. **`division` is just a free-string label**, not an entity. It carries no ruleset
   meaning (regionals use rule sets matching neither A nor B). It only partitions
   teams into "who competes against whom." UI suggests `A`/`B`; any string is valid.

5. **`field` is orthogonal to division/group** — just a thin `{id, name}` a match
   points at. Any match can be placed on any field (a Div-A match on a Div-B-sized
   field is a real, allowed logistical case). Nullable on the match.

6. **Group ranking is auto-computed then human-confirmed.** Live standings are
   computed from results (M3); the *confirmed* ranking is persisted and is what
   `group_rank` references resolve against. This is the human-in-the-loop checkpoint
   for ties the algorithm can't break.

7. **No format generators, no hardcoded SSL structure.** The organizer defines the
   match graph. A "phase" is implicit: a match with a `group_id` is a group match; a
   match without one is an elimination/decider match wired via references.

## Entities & relationships

```
tournament 1─┬─* team            (team.division = free string)
             ├─* field
             ├─* team_group ──*── team_group_member ──* team
             │        └─── group_ranking ──* team        (confirmed order)
             ├─* match ──> field?          (field_id, nullable)
             │      ├──> team_group?        (group_id; set = group match)
             │      ├──> team (a/b resolved, winner, referee)
             │      └──> match (a/b ref_match_id; self-ref = bracket edges)
             ├─* placement ──> match, team
             └─* event ──> match?, user?
user, token   (instance-global auth; not tournament-scoped)
```

## SQL schema (SQLite)

Target driver: `modernc.org/sqlite` (pure Go). **Foreign-key enforcement is off by
default in SQLite and must be enabled per connection** — the data layer runs
`PRAGMA foreign_keys = ON` on every connection.

IDs are `INTEGER PRIMARY KEY AUTOINCREMENT` on externally-referenced entities so a
deleted id is never reused (stable external references, per the architecture's
"stable IDs everywhere"). Timestamps are ISO-8601 UTC `TEXT`. Booleans are modeled as
nullable timestamps (`*_at`) or `INTEGER` 0/1.

### Referential-integrity strategy

- **Ownership cascades:** everything tournament-scoped is `ON DELETE CASCADE` from
  `tournament` — deleting a tournament wipes its data cleanly.
- **No `RESTRICT`** anywhere: it would deadlock the tournament cascade (team and match
  are siblings under tournament). Instead, references to `team` from `match` are
  `ON DELETE SET NULL`, and the *application* guards against destructive deletes
  (confirm dialogs; "withdraw" a team rather than delete one with recorded results —
  per the architecture's mistake-proofing).
- **Reassignable pointers** (`field_id`, `referee_team_id`, slot `ref_match_id`,
  `ref_group_id`, placement sources) are `ON DELETE SET NULL` — the target vanishing
  leaves a slot to re-wire, not corrupt data.
- **`group_id` on match is `ON DELETE CASCADE`:** a group's round-robin matches belong
  to it (guarded by a confirm dialog in the UI).

```sql
PRAGMA foreign_keys = ON;

CREATE TABLE tournament (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  name       TEXT NOT NULL,
  location   TEXT NOT NULL DEFAULT '',
  starts_on  TEXT,                        -- ISO-8601 date, nullable
  ends_on    TEXT,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE team (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  tournament_id INTEGER NOT NULL REFERENCES tournament(id) ON DELETE CASCADE,
  name          TEXT NOT NULL,
  division      TEXT NOT NULL DEFAULT '',  -- free-string partition label
  country       TEXT NOT NULL DEFAULT '',
  contact       TEXT NOT NULL DEFAULT '',
  created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
  UNIQUE (tournament_id, name)
);

CREATE TABLE field (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  tournament_id INTEGER NOT NULL REFERENCES tournament(id) ON DELETE CASCADE,
  name          TEXT NOT NULL,
  UNIQUE (tournament_id, name)
);

-- Domain concept: "group" (a round-robin pool). Named team_group because GROUP is
-- a reserved SQL keyword.
CREATE TABLE team_group (
  id                   INTEGER PRIMARY KEY AUTOINCREMENT,
  tournament_id        INTEGER NOT NULL REFERENCES tournament(id) ON DELETE CASCADE,
  division             TEXT NOT NULL DEFAULT '',
  name                 TEXT NOT NULL,       -- e.g. "G1"
  ranking_confirmed_at TEXT,                -- null until organizer blesses the order
  UNIQUE (tournament_id, name)
);

CREATE TABLE team_group_member (
  group_id INTEGER NOT NULL REFERENCES team_group(id) ON DELETE CASCADE,
  team_id  INTEGER NOT NULL REFERENCES team(id)       ON DELETE CASCADE,
  PRIMARY KEY (group_id, team_id)
);

-- Organizer-confirmed within-group ranking; what group_rank references resolve to.
CREATE TABLE group_ranking (
  group_id INTEGER NOT NULL REFERENCES team_group(id) ON DELETE CASCADE,
  team_id  INTEGER NOT NULL REFERENCES team(id)       ON DELETE CASCADE,
  rank     INTEGER NOT NULL,                -- 1-based
  PRIMARY KEY (group_id, team_id),
  UNIQUE (group_id, rank)
);

CREATE TABLE match (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  tournament_id   INTEGER NOT NULL REFERENCES tournament(id) ON DELETE CASCADE,
  division        TEXT NOT NULL DEFAULT '',
  group_id        INTEGER REFERENCES team_group(id) ON DELETE CASCADE, -- set = group match
  label           TEXT NOT NULL DEFAULT '',       -- "Upper 1", "Grand Final", "G1"
  field_id        INTEGER REFERENCES field(id)  ON DELETE SET NULL,
  scheduled_at    TEXT,                            -- ISO-8601 datetime, nullable
  status          TEXT NOT NULL DEFAULT 'scheduled'
                    CHECK (status IN ('scheduled','playing','finished','cancelled')),
  referee_team_id INTEGER REFERENCES team(id) ON DELETE SET NULL,

  -- slot A
  a_source       TEXT CHECK (a_source IN ('team','group_rank','match_winner','match_loser')),
  a_team_id      INTEGER REFERENCES team(id)       ON DELETE SET NULL,  -- direct or resolved
  a_ref_group_id INTEGER REFERENCES team_group(id) ON DELETE SET NULL,
  a_ref_rank     INTEGER,
  a_ref_match_id INTEGER REFERENCES match(id)      ON DELETE SET NULL,
  a_score        INTEGER,

  -- slot B
  b_source       TEXT CHECK (b_source IN ('team','group_rank','match_winner','match_loser')),
  b_team_id      INTEGER REFERENCES team(id)       ON DELETE SET NULL,
  b_ref_group_id INTEGER REFERENCES team_group(id) ON DELETE SET NULL,
  b_ref_rank     INTEGER,
  b_ref_match_id INTEGER REFERENCES match(id)      ON DELETE SET NULL,
  b_score        INTEGER,

  -- Authoritative result. Set for finished matches; handles knockout shootouts where
  -- a_score == b_score but a winner is still decided. NULL + equal scores = a draw
  -- (valid in group play). match_winner/match_loser references read this.
  winner_team_id INTEGER REFERENCES team(id) ON DELETE SET NULL,
  created_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

-- Final standings: rank N in a division = winner/loser of some match.
CREATE TABLE placement (
  id               INTEGER PRIMARY KEY AUTOINCREMENT,
  tournament_id    INTEGER NOT NULL REFERENCES tournament(id) ON DELETE CASCADE,
  division         TEXT NOT NULL DEFAULT '',
  rank             INTEGER NOT NULL,
  label            TEXT NOT NULL DEFAULT '',     -- e.g. "Champion"
  source_match_id  INTEGER REFERENCES match(id) ON DELETE SET NULL,
  source_outcome   TEXT CHECK (source_outcome IN ('winner','loser')),
  resolved_team_id INTEGER REFERENCES team(id)  ON DELETE SET NULL,
  UNIQUE (tournament_id, division, rank)
);

-- Pending-event queue (external producers push; organizer approves). See architecture.
CREATE TABLE event (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  tournament_id INTEGER REFERENCES tournament(id) ON DELETE CASCADE,
  type          TEXT NOT NULL,                   -- e.g. "match_ended"
  match_id      INTEGER REFERENCES match(id) ON DELETE SET NULL,
  payload       TEXT NOT NULL DEFAULT '{}',      -- JSON
  source        TEXT NOT NULL DEFAULT '',        -- producer identity
  dedupe_key    TEXT,
  status        TEXT NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending','approved','rejected')),
  received_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
  resolved_at   TEXT,
  resolved_by   INTEGER REFERENCES user(id) ON DELETE SET NULL,
  UNIQUE (source, dedupe_key)                    -- collapses duplicate resends
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
CREATE INDEX idx_team_tournament    ON team(tournament_id);
CREATE INDEX idx_field_tournament   ON field(tournament_id);
CREATE INDEX idx_group_tournament   ON team_group(tournament_id);
CREATE INDEX idx_group_member_team  ON team_group_member(team_id);
CREATE INDEX idx_match_tournament   ON match(tournament_id);
CREATE INDEX idx_match_group        ON match(group_id);
CREATE INDEX idx_match_field        ON match(field_id);
CREATE INDEX idx_match_a_ref_match  ON match(a_ref_match_id);
CREATE INDEX idx_match_b_ref_match  ON match(b_ref_match_id);
CREATE INDEX idx_placement_tournament ON placement(tournament_id);
CREATE INDEX idx_event_status       ON event(status);
CREATE INDEX idx_event_tournament   ON event(tournament_id);
```

## How the example maps (2026 Incheon Div B)

- `LL1` box "G1.2 - G2.3": a match, `division='B'`, `group_id=NULL`,
  `a_source='group_rank'`, `a_ref_group_id=<G1>`, `a_ref_rank=2`;
  `b_source='group_rank'`, `b_ref_group_id=<G2>`, `b_ref_rank=3`.
- `Upper 1` "G3.1 - LL1": `a`=group_rank(G3,1); `b`=match_winner(LL1).
- Loser routing (`LU1` = loser of Upper 1) → some slot with
  `source='match_loser'`, `ref_match_id=<Upper 1>`.
- Placement "5th = loser of Low 2.1" → `placement(rank=5, source_match_id=<Low 2.1>,
  source_outcome='loser')`.

Resolution (M3): when a match finishes, set `winner_team_id`; any slot/placement whose
`ref_match_id` points at it resolves (`match_winner`→winner, `match_loser`→the other
resolved team). `group_rank` slots resolve once `group_ranking` is confirmed.

## Out of scope for this spec

- Reference **resolution** and **standings computation** logic (M3).
- Referee **suggestion** algorithm (M3) — the schema only records the chosen
  `referee_team_id`.
- Schema **migrations** framework (M1 implementation detail).
- API shape (M2).
