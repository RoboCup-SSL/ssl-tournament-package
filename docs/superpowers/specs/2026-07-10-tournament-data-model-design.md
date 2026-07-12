# Tournament data model — design

- **Date:** 2026-07-10 (revised 2026-07-12)
- **Status:** draft for review
- **Milestone:** M1 (data layer). Domain *logic* (standings computation, reference
  resolution, round-robin generation, ref suggestions, validation) is M3; this spec
  defines only the **model + schema** those will operate on.

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

## Guiding principle: freeform by design

**The data model never enforces logical rules — it only enforces mechanical
integrity.** Real tournaments are too unpredictable for the database to say "no":
an organizer must be able to plan things that look wrong (a team playing against
itself, five matches on one field at the same time, a Div-A match on a Div-B-sized
field, two teams with the same name, a shared 3rd place). All of that is
**valid data**.

- **The DB enforces:** referential integrity (foreign keys — a pointer either
  references a real row or is NULL) and vocabulary (CHECK on enum-like columns such
  as `status` and `kind`, because unknown values aren't creative plans, they're
  unreadable data). Structural shape too: a slot has at most one source, a team has
  at most one rank per group — not because it's a rule, but because the alternative
  is meaningless/ambiguous data.
- **The DB does not enforce:** uniqueness of names, one-match-per-field-per-time,
  distinct opponents, sane scores, complete brackets, rank uniqueness, or anything
  else "logical". A future **advisory validation layer** (M3+) checks invariants and
  *informs* the organizer of violations — it never blocks a write.

Consequence: this schema deliberately has **no UNIQUE constraints on names or
ranks**, and nothing prevents `a_team_id == b_team_id`.

## Key design decisions (deliberate assumptions)

1. **A match has exactly two symmetric participants** — `team_a` / `team_b`. Slot
   order is cosmetic (display only); it confers nothing. There is no home/away, and
   **no color** in the model — team color (blue/yellow) is chosen live by the referee
   and can even swap at halftime (SSL rules §4.3.3), so it belongs to the
   match-running layer (Game Controller), not here. Scores are keyed to the *team*,
   never a color.

2. **The match row stores only resolved facts:** the two teams (when known), their
   scores, the winner. All bracket *wiring* — "this slot will be filled by the winner
   of Upper 1 / by 3rd of Group 2" — lives in a separate `slot_source` table, one row
   per **unresolved** slot. A directly-assigned team needs no `slot_source` row at
   all. This keeps the match row readable ("who plays, score, who won, when, where,
   who refs") and isolates the reference machinery.

3. **Slot references are typed:** `group_rank` (`{group, rank}` — "winner of Group 1"
   = rank 1) or `match_winner` / `match_loser` (`{match}`). Resolution (M3) computes
   the concrete team and writes it into the match row's `a_team_id`/`b_team_id`.
   This one mechanism expresses the whole bracket graph, including
   double-elimination's winner-and-loser routing.

4. **Final placements use the same reference mechanism — all three kinds.** "5th =
   loser of Low 2.1", "1st = winner of Grand Final", and equally "3rd–5th = group
   ranks 3–5" (the German Open shape, where no match decides the lower places). A
   placement references a match outcome or a group rank, just like a slot does.

5. **`division` is a table, not a string** (revised from the earlier free-string
   design). A division row is still freeform — organizers create whatever divisions
   they want; "A"/"B" are merely suggested names — but referencing it by FK prevents
   typos from silently splitting a division, and gives metadata a place to live
   (`notes` now; more later if needed). It carries **no ruleset semantics**: it is
   nothing more than an identifier for "these teams compete against one another."

6. **`field` is orthogonal to division/group** — a thin `{id, name}` a match points
   at. Any match can be placed on any field. Nullable, reassignable.

7. **Every match names two duty teams** (SSL matches are staffed by other teams):
   - `referee_team_id` — referee + game-controller operator
   - `assistant_referee_team_id` — assistant referee + vision operator

8. **Group ranking is auto-computed then human-confirmed.** Live standings are
   computed from results (M3); the *confirmed* ranking is persisted in
   `group_ranking` and is what `group_rank` references resolve against. This is the
   human-in-the-loop checkpoint for ties the algorithm can't break. Ties may even be
   confirmed as ties (two teams at the same rank) — resolution then reports the
   ambiguity to the organizer rather than guessing.

9. **No format generators in the model, no hardcoded SSL structure.** The organizer
   defines the match graph. A "phase" is implicit: a match with a `group_id` is a
   group match; a match without one is an elimination/decider match wired via
   `slot_source`. (The UI should offer *generate all round-robin pairings for this
   group* as a convenience — M3 logic — but the output is ordinary match rows,
   freely editable afterward.)

## How it's used — lifecycle walkthrough

Using RoboCup 2026 Div B as the example:

1. **Create tournament** → one `tournament` row ("RoboCup 2026, Incheon").
2. **Create fields** → `field` rows: "Field A", "Field B0", "Field B1".
3. **Create divisions** → `division` rows: "A", "B" (a regional might create one).
4. **Create teams** → `team` rows, each assigned a `division_id`.
5. **Create groups** → `team_group` rows (G1–G4, division B) + `team_group_member`
   rows. *Which* team goes in *which* group is decided outside (ssl-grouping or a
   human); we record the result.
6. **Create matches**, two flavors:
   - **Group matches:** `group_id` set, `a_team_id`/`b_team_id` filled directly
     (generated as all pairings, or by hand). Field/time assigned freely.
   - **Elimination matches:** `group_id` NULL, teams empty; their `slot_source`
     rows define the wiring — e.g. `LL1` "G1.2–G2.3" = slot a `(group_rank, G1, 2)`,
     slot b `(group_rank, G2, 3)`; `Upper 1` "G3.1–LL1" = slot a
     `(group_rank, G3, 1)`, slot b `(match_winner, LL1)`. Fully fixed showcase
     matches just set teams directly (no `slot_source` rows).
   - **Placements:** rank 1 = winner of Grand Final, rank 5 = loser of Low 2.1, …
7. **Group phase runs** → scores land on group matches (approve-queue or manual).
   Standings are computed live for display. When a group finishes, the organizer
   **confirms the ranking** (auto-computed, editable) → `group_ranking` rows →
   every `group_rank` slot pointing at that group resolves; the bracket populates.
8. **Elimination runs** → each finished match gets `winner_team_id`; slots and
   placements referencing it resolve immediately; losers flow down the lower
   bracket; the Grand Final completes the final standings.

Nothing forces this order. Matches can exist before groups are full, slots can be
re-wired mid-tournament, fields added on day 2 — every step is just rows, and
resolution re-runs whenever an upstream fact changes (with the human-approval gate
in front of anything an external producer pushes).

### Resolution semantics: materialized on write, never computed on read

Resolved teams are **written into `match.a_team_id`/`b_team_id`** by the resolver
(M3); readers never compute anything. The UI reads plain match rows — a NULL slot
renders as its wiring label ("Winner of Upper 1"). Rationale: reads dominate
(every phone on venue Wi-Fi) and the resolved team is itself an organizer-visible
fact, not a derived view.

- **Triggers:** the resolver runs on exactly three mutating events — a match gets
  `winner_team_id` set, a group's ranking is confirmed, a `slot_source` row is
  edited. All writes flow through the single service, so there is one choke point
  and no cache-invalidation problem; the materialized team IDs *are* the cache.
- **One hop, no recursion:** a chain elim→elim→elim→group never needs traversal.
  Each slot resolves the moment its upstream completes (indexed lookups via
  `idx_slot_source_match` / `idx_slot_source_group`); by induction the bracket
  fills itself as the tournament runs. `match_loser` resolves to the participant
  that isn't the winner — if a match was finished without known participants
  (legal, freeform), the slot stays NULL and the advisory layer flags it.
- **Corrections:** if an upstream result changes, the resolver overwrites slots on
  **unfinished** matches only — including **clearing a slot back to NULL** when its
  upstream becomes undecided again (e.g. wiring re-pointed at a not-yet-played
  replay). A downstream match already played with the "wrong" team is recorded
  history — the advisory layer reports the inconsistency; nothing is silently
  rewritten.
- **Manual override:** the organizer may hand-set a slot's team even where wiring
  exists; to deviate permanently, edit or remove the `slot_source` row (otherwise
  re-resolution overwrites the override — exact precedence UX is M3).

## Entities & relationships

```
tournament 1─┬─* division
             ├─* team ──> division?
             ├─* field
             ├─* field_booking ──> field?, team?   (practice slots, closures)
             ├─* team_group ──> division?
             │        ├─*─ team_group_member ──> team
             │        └─*─ group_ranking ──> team        (confirmed order)
             ├─* match ──> division?, field?, team_group?   (all nullable)
             │      ├──> team (a/b resolved, winner, referee, assistant referee)
             │      └─*─ slot_source ──> team_group | match  (bracket wiring)
             ├─* placement ──> division?, match, team
             └─* event ──> match?, user?
user, token   (instance-global auth; not tournament-scoped)
```

## SQL schema (SQLite)

Target driver: `modernc.org/sqlite` (pure Go). **Foreign-key enforcement is off by
default in SQLite and must be enabled per connection** — the data layer runs
`PRAGMA foreign_keys = ON` on every connection.

IDs are `INTEGER PRIMARY KEY AUTOINCREMENT` on externally-referenced entities so a
deleted id is never reused (stable external references, per the architecture's
"stable IDs everywhere"). Timestamps are ISO-8601 UTC `TEXT`.

### Referential-integrity strategy

- **Ownership cascades:** everything tournament-scoped is `ON DELETE CASCADE` from
  `tournament` — deleting a tournament wipes its data cleanly.
- **No `RESTRICT`** anywhere: it would deadlock the tournament cascade (team and
  match are siblings under tournament). References to `team`, `division`, `field`
  from other rows are `ON DELETE SET NULL` — the target vanishing leaves a hole to
  re-point, not corrupt data. The *application* adds confirm dialogs for destructive
  deletes (per the architecture's mistake-proofing); the schema stays permissive.
- **`match.group_id` is `ON DELETE CASCADE`:** a group's round-robin matches belong
  to it (UI confirms before deleting a group).
- Per the freeform principle: **no UNIQUE constraints on names or ranks.**

```sql
PRAGMA foreign_keys = ON;

CREATE TABLE tournament (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  name       TEXT NOT NULL,
  location   TEXT NOT NULL DEFAULT '',
  starts_on  TEXT,                        -- ISO-8601 date, nullable
  ends_on    TEXT,
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

-- A reservation of a field for a time range: practice slots, calibration,
-- maintenance/closure — anything that isn't a match. team_id NULL = non-team
-- booking ("field closed"). Claims are whole-field; "north half" etc. goes in
-- notes. The advisory layer warns when overlapping claims can't fit (3+ teams
-- on one field at once) — never enforced, per the freeform principle.
CREATE TABLE field_booking (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  tournament_id INTEGER NOT NULL REFERENCES tournament(id) ON DELETE CASCADE,
  field_id      INTEGER REFERENCES field(id) ON DELETE SET NULL,
  team_id       INTEGER REFERENCES team(id)  ON DELETE SET NULL,
  label         TEXT NOT NULL DEFAULT '',     -- "practice", "calibration", "closed"
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

## How the example maps (2026 Incheon Div B)

- `LL1` box "G1.2 - G2.3": a `match` row (`division_id=<B>`, `group_id=NULL`,
  teams NULL) + two `slot_source` rows:
  `(LL1,'a','group_rank', ref_group_id=<G1>, ref_rank=2)` and
  `(LL1,'b','group_rank', ref_group_id=<G2>, ref_rank=3)`.
- `Upper 1` "G3.1 - LL1": slot a `(group_rank, G3, 1)`; slot b
  `(match_winner, ref_match_id=<LL1>)`.
- Loser routing (`LU1` = loser of Upper 1) → a slot with
  `kind='match_loser', ref_match_id=<Upper 1>`.
- Placement "5th = loser of Low 2.1" → `placement(rank=5, source_kind='match_loser',
  source_match_id=<Low 2.1>)`. A table-only format's "4th = group rank 4" →
  `placement(rank=4, source_kind='group_rank', source_group_id=<G1>, source_rank=4)`.

Resolution (M3): when a match finishes, set `winner_team_id`; any `slot_source` /
`placement` pointing at it resolves (`match_winner`→winner, `match_loser`→the other
team) and the concrete team is written into `match.a_team_id`/`b_team_id` /
`placement.resolved_team_id`. `group_rank` sources resolve once that group's
`group_ranking` is confirmed.

## Stress-tested scenarios

Real mid-tournament chaos the model was checked against; each reduces to plain
row operations plus the resolver's existing triggers:

1. **Match delayed, field's day shifts** → bulk-update `scheduled_at` for later
   matches on that field (UI convenience). Freeform means transient overlaps are
   legal; the advisory layer re-checks.
2. **Mid-tournament disqualification** → set `team.withdrawn_at`; played matches
   vs. the team → `status='invalidated'` (scores kept as history, excluded from
   standings, which count only `finished`); future ones → `cancelled`. Organizer
   confirms the resulting ranking as usual.
3. **Field becomes unavailable** → re-point `field_id` of its scheduled matches;
   field is a dumb reassignable pointer.
4. **Friendly match on an empty field** → a match with NULL `division_id`/`group_id`,
   teams set directly, nothing referencing it: affects no standings, no bracket.
5. **Group score corrected during eliminations** → fix the group match; re-confirm
   ranking (resolver touches only unfinished matches); mark the wrongly-played
   elimination match `invalidated`; insert the replay with the same wiring;
   `UPDATE slot_source/placement SET ref_match_id=<replay> WHERE ref_match_id=<old>`;
   resolver clears now-undecided downstream slots and re-fills them as the replay
   completes. The voided match stays as unreferenced history.

**Round 2 (adversarial-agent scenarios)** — drove four schema deltas and a set of
documented known-strains:

- **Placements straight from the table** (German Open 3rd–5th) → exposed that
  `placement` lacked the `group_rank` source kind; fixed (same vocabulary as
  `slot_source`).
- **Two GC instances colliding on `dedupe_key`** → the former
  `UNIQUE(source, dedupe_key)` was the one place the DB hard-rejected a legitimate
  producer fact, violating the freeform principle; dropped. Dedupe is app-level
  (drop only while an identical key is still *pending*) + queue-UI grouping.
- **Match suspended overnight** (power/vision failure at 4–2) → `suspended` status;
  `scheduled_at` moves to the resumption slot; partial scores kept.
- **Sticky-note facts** (joint/merged teams, standings penalties, shared robots,
  human referees at small events, suspension details) → `notes` columns on `team`,
  `match`, `team_group`. Structured versions (standings adjustments, inter-team
  scheduling constraints, team merge/alias) are deliberately deferred to M3+, when
  the standings rules and advisory layer they feed are actually designed.
- **Known strains, accepted:** conditional matches — a double-elim *bracket reset*
  ("Grand Final 2 if the lower-bracket team wins GF1") or a Bo3 series game 3 —
  are pre-created and wired normally, then cancelled + re-pointed by hand in the
  branch not taken, with the advisory layer flagging dangling wiring. A
  series/conditional-match entity is not worth its machinery until a real event
  demands it. Walkover cascades and late-arriving teams reduce to existing row ops
  (verified HANDLES).

**Round 3 (committee feedback)** — three deltas:

- **Score-independent winners** (Japan Open: on equal goals, fewer fouls wins) —
  already supported (`winner_team_id` is decoupled from scores by design); added
  per-team `fouls` / `yellow_cards` / `red_cards` columns so the *evidence* for
  such results is visible, not buried in event payloads.
- **Practice slots** → new `field_booking` table (field + optional team + time
  range): covers practice, calibration, and field closures alike. Whole-field
  claims only; "which half" is a note; the advisory layer warns when overlapping
  claims can't fit. Friendlies were already covered (unwired matches).
- **"Did it actually happen?"** → `forfeited` status: counts like `finished` but
  was not (fully) played — a conventional 10:0 walkover is now distinguishable
  from a real 10:0 (gamelog validation knows not to expect a log).
- **Follow-up decisions:** match durations = tournament defaults
  (`default_match_minutes` + `default_gap_minutes`) with per-match
  `duration_minutes` override — stored facts only, the automated scheduler is
  non-MVP. Forfeit scores are organizer-entered (conventional 10:0 or NULL, both
  supported; the choice is per-ruleset). Team self-service practice booking is a
  later API/auth feature — `field_booking` (team + start/stop) already suffices
  as the record.

## Out of scope for this spec

- Reference **resolution**, **standings computation**, and **round-robin pairing
  generation** logic (M3).
- The **advisory validation layer** (violation checking/reporting — M3+). The schema
  deliberately admits "invalid" plans; see the freeform principle.
- Referee **suggestion** algorithm (M3) — the schema only records the chosen duty
  teams.
- Schema **migrations** framework (M1 implementation detail).
- API shape (M2).
- **Automated scheduler** (would consume the duration/gap defaults) — non-MVP.
- **Team self-service practice booking** (request/approval flow, auth) — later;
  the `field_booking` row shape already covers the record itself.
