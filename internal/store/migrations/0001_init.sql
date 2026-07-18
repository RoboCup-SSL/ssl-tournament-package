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
  time_zone   TEXT,                        -- IANA timezone anchor (e.g. Asia/Seoul), nullable
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
