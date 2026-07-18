# Date/time validation + tournament timezone (M2b·dt) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add server-side format validation for every date/time field in the model and give each tournament a single IANA `time_zone` anchor, while keeping the backend timezone-naive and values freeform.

**Architecture:** Pure format validators in `internal/api` parse each field's layout (rejecting malformed input, normalizing to canonical form) and are wired into the existing `applyXPatch` functions. A nullable `time_zone` column is added to `tournament` via a numbered migration. No timezone conversion or math is added anywhere — instants are stored bare (no offset); the frontend (later, M2c) applies the zone for display only.

**Tech Stack:** Go 1.26 stdlib (`time`, `time/tzdata`), SQLite (`modernc.org/sqlite` via the store), embedded numbered migrations.

**Spec:** docs/superpowers/specs/2026-07-18-datetime-validation-timezone-design.md

## Global Constraints

- **`go` is not on the default PATH** — prefix every go command with `export PATH="$PATH:/usr/local/go/bin"`. Verify `go version` → go1.26.x.
- **Freeform values are preserved.** Only *malformed shape* and *non-IANA zones* are rejected. No window/ordering/cross-field checks. A 3 a.m. match, `ends_on` before `starts_on`, open-after-close — all still accepted.
- **Backend stays timezone-naive.** Scheduling instants (`scheduled_at`, booking `starts_at`/`ends_at`, team `withdrawn_at`) are stored as bare local datetimes with **no offset**; an offset or `Z` is **rejected**. No `time.LoadLocation`-based conversion of instants — `LoadLocation` is used only to *validate* the `time_zone` name.
- **Per-field write semantics (uniform):** PATCH key absent → leave unchanged; `null` → clear; empty string `""` → clear (treated as null); non-empty → validate, and store the **canonical re-formatted** value.
- **`created_at` and any server-set/audit timestamp are untouched** (they stay UTC `Z`).
- **Error shape:** `&api.Error{Code: CodeInvalidValue, Field: "<json field name>", Message: ...}` (mirrors existing `errors.go`).
- **`time/tzdata` is embedded** via a blank import in `internal/api/datetime.go` (co-located with the code that needs `LoadLocation`; this guarantees IANA validation works host-independently in both binaries and `go test`). This refines the spec's "add it to the mains" — same effect (embedded once via the api dependency), better locality, and deterministic tests.
- **Branch:** implement on a fresh branch off `main` (e.g. `m2b-dt-validation`) — this slice is pure backend and independent of the M2b shell PR. (The design docs currently live on the `m2b-frontend-shell` branch / PR #5; they merge separately. Confirm branch choice with the human before Task 1.)

---

## File Structure

```
internal/store/migrations/0002_tournament_timezone.sql   # NEW: ALTER TABLE add time_zone
internal/store/tournament.go                             # MODIFY: struct field + 4 SQL statements
internal/store/tournament_test.go (or migrate_test.go)   # MODIFY/NEW: time_zone round-trip
internal/api/datetime.go                                 # NEW: validators + tzdata import
internal/api/datetime_test.go                            # NEW: validator table tests
internal/api/opt.go                                      # MODIFY: applyNullableString helper
internal/api/tournament.go                               # MODIFY: TournamentPatch.TimeZone + wire 5 fields
internal/api/tournament_test.go                          # MODIFY: apply/validation cases
internal/api/booking.go                                  # MODIFY: wire starts_at/ends_at
internal/api/team.go                                     # MODIFY: wire withdrawn_at
internal/api/match.go                                    # MODIFY: wire scheduled_at
internal/api/{booking,team,match}_test.go                # MODIFY: validation cases
internal/server/*_test.go                                # MODIFY: HTTP-level PATCH validation
internal/server/openapi.yaml                             # MODIFY: time_zone + format hints
```

**Note on TDD:** this is backend Go — real red-green TDD applies. Each task writes failing tests first.

---

### Task 1: `time_zone` column (store layer)

**Files:**
- Create: `internal/store/migrations/0002_tournament_timezone.sql`
- Modify: `internal/store/tournament.go` (struct + 4 SQL statements)
- Test: `internal/store/tournament_test.go`

**Interfaces:**
- Produces: `store.Tournament.TimeZone *string` (json `time_zone`), persisted through Create/Get/List/Update. No API exposure yet (that's Task 3).

- [ ] **Step 1: Write the failing store test**

Add to `internal/store/tournament_test.go` (follow the file's existing setup helpers for obtaining a `*Store`):
```go
func TestTournamentTimeZoneRoundTrip(t *testing.T) {
	store := newTestStore(t) // use the file's existing store-construction helper
	zone := "Asia/Seoul"
	tournament := &Tournament{Name: "TZ", TimeZone: &zone}
	if err := store.CreateTournament(tournament); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := store.GetTournament(tournament.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.TimeZone == nil || *got.TimeZone != "Asia/Seoul" {
		t.Fatalf("time_zone = %v, want Asia/Seoul", got.TimeZone)
	}
	got.TimeZone = nil
	if err := store.UpdateTournament(got); err != nil {
		t.Fatalf("update: %v", err)
	}
	again, _ := store.GetTournament(tournament.ID)
	if again.TimeZone != nil {
		t.Fatalf("time_zone = %v, want nil after clear", again.TimeZone)
	}
}
```
(If the test file uses a different store helper name, match it — read the file first.)

- [ ] **Step 2: Run the test to verify it fails**

Run: `export PATH="$PATH:/usr/local/go/bin" && go test ./internal/store/ -run TestTournamentTimeZoneRoundTrip -v`
Expected: FAIL (compile error — `Tournament` has no field `TimeZone`).

- [ ] **Step 3: Add the migration**

Create `internal/store/migrations/0002_tournament_timezone.sql`:
```sql
-- Single IANA timezone anchor per tournament (e.g. Asia/Seoul). Nullable.
-- The backend stays timezone-naive; this is the frontend's display anchor.
ALTER TABLE tournament ADD COLUMN time_zone TEXT;
```

- [ ] **Step 4: Add the struct field and wire the SQL**

In `internal/store/tournament.go`:
- Add to the `Tournament` struct, after `DefaultGapMinutes`:
  ```go
  	TimeZone            *string `json:"time_zone"`
  ```
- `CreateTournament` INSERT: add `time_zone` to the column list and one `?`, and append `tournament.TimeZone` to the args (before the closing `)`).
- `GetTournament` SELECT: add `time_zone` to the selected columns and `&tournament.TimeZone` to the `Scan` (keep column/scan order aligned; place it after `default_gap_minutes`, before `created_at`).
- `ListTournaments` SELECT + `Scan`: same addition, same position.
- `UpdateTournament` UPDATE: add `time_zone=?` to the `SET` list and `tournament.TimeZone` to the args (before `tournament.ID`).

- [ ] **Step 5: Run the test to verify it passes**

Run: `export PATH="$PATH:/usr/local/go/bin" && go test ./internal/store/ -run TestTournamentTimeZoneRoundTrip -v`
Expected: PASS.

- [ ] **Step 6: Run the full suite (catch JSON-shape assertions)**

Run: `export PATH="$PATH:/usr/local/go/bin" && go test ./...`
Expected: PASS. If an existing API/server test asserts the exact Tournament JSON, it will now see `"time_zone":null` — update that expectation to include the new field. Fix and re-run until green.

- [ ] **Step 7: Commit**

```bash
git add internal/store/migrations/0002_tournament_timezone.sql internal/store/tournament.go internal/store/tournament_test.go
git commit -m "feat(store): add nullable time_zone column to tournament"
```

---

### Task 2: Date/time validators

**Files:**
- Create: `internal/api/datetime.go`
- Test: `internal/api/datetime_test.go`

**Interfaces:**
- Produces (all in package `api`):
  - `func validDate(value string) (string, bool)` — layout `2006-01-02`
  - `func validClock(value string) (string, bool)` — layout `15:04`
  - `func validNaiveTime(value string) (string, bool)` — `2006-01-02T15:04[:05]`, no offset
  - `func validZone(value string) (string, bool)` — `time.LoadLocation`
  Each returns the canonical string and `true`, or `"", false` on malformed input.

- [ ] **Step 1: Write the failing tests**

Create `internal/api/datetime_test.go`:
```go
package api

import "testing"

func TestValidators(t *testing.T) {
	cases := []struct {
		name    string
		fn      func(string) (string, bool)
		in      string
		want    string
		wantOK  bool
	}{
		{"date ok", validDate, "2026-07-15", "2026-07-15", true},
		{"date normalizes", validDate, "2026-7-5", "2026-07-05", true},
		{"date bad month", validDate, "2026-13-40", "", false},
		{"date junk", validDate, "banana", "", false},
		{"clock ok", validClock, "08:00", "08:00", true},
		{"clock normalizes", validClock, "8:00", "08:00", true},
		{"clock bad", validClock, "25:99", "", false},
		{"naive minute", validNaiveTime, "2026-07-15T14:00", "2026-07-15T14:00", true},
		{"naive seconds", validNaiveTime, "2026-07-15T14:00:30", "2026-07-15T14:00", true},
		{"naive rejects offset", validNaiveTime, "2026-07-15T14:00:00+09:00", "", false},
		{"naive rejects Z", validNaiveTime, "2026-07-15T14:00:00Z", "", false},
		{"zone ok", validZone, "Asia/Seoul", "Asia/Seoul", true},
		{"zone bad", validZone, "Mars/Phobos", "", false},
	}
	for _, tc := range cases {
		got, ok := tc.fn(tc.in)
		if ok != tc.wantOK || got != tc.want {
			t.Errorf("%s: got (%q,%v), want (%q,%v)", tc.name, got, ok, tc.want, tc.wantOK)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `export PATH="$PATH:/usr/local/go/bin" && go test ./internal/api/ -run TestValidators -v`
Expected: FAIL (compile error — validators undefined).

- [ ] **Step 3: Implement the validators**

Create `internal/api/datetime.go`:
```go
// Format validators for the model's date/time text fields. Each parses its
// layout, rejecting malformed input, and returns the canonical form. The
// backend stays timezone-naive: scheduling instants carry no offset.
package api

import (
	"time"

	_ "time/tzdata" // embed the IANA zone database so validZone works on any host
)

// validDate accepts an ISO calendar date and returns it canonicalized.
func validDate(value string) (string, bool) {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return "", false
	}
	return parsed.Format("2006-01-02"), true
}

// validClock accepts a 24-hour wall-clock time (HH:MM) and canonicalizes it.
func validClock(value string) (string, bool) {
	parsed, err := time.Parse("15:04", value)
	if err != nil {
		return "", false
	}
	return parsed.Format("15:04"), true
}

// validNaiveTime accepts a naive local datetime (optional seconds, no offset)
// and returns canonical minute-precision form. An offset or Z is rejected.
func validNaiveTime(value string) (string, bool) {
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02T15:04"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.Format("2006-01-02T15:04"), true
		}
	}
	return "", false
}

// validZone accepts an IANA timezone name (e.g. Asia/Seoul).
func validZone(value string) (string, bool) {
	if _, err := time.LoadLocation(value); err != nil {
		return "", false
	}
	return value, true
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `export PATH="$PATH:/usr/local/go/bin" && go test ./internal/api/ -run TestValidators -v`
Expected: PASS (all cases).

- [ ] **Step 5: Commit**

```bash
git add internal/api/datetime.go internal/api/datetime_test.go
git commit -m "feat(api): date/time/zone format validators (tzdata embedded)"
```

---

### Task 3: `applyNullableString` helper + Tournament wiring

**Files:**
- Modify: `internal/api/opt.go` (new helper)
- Modify: `internal/api/tournament.go` (`TournamentPatch.TimeZone`, wire 5 fields)
- Test: `internal/api/tournament_test.go`
- Test: an existing `internal/server/*_test.go` covering `PATCH /api/tournaments/{id}`

**Interfaces:**
- Consumes: validators from Task 2; `store.Tournament.TimeZone` from Task 1.
- Produces: `func applyNullableString(field Opt[string], target **string, name string, check func(string) (string, bool)) *Error` (package `api`), reused by Task 4.

- [ ] **Step 1: Write the failing API test**

Add to `internal/api/tournament_test.go` (use the file's existing store/test helpers):
```go
func TestTournamentDateTimeValidation(t *testing.T) {
	store := newTestStore(t) // match the file's helper
	created, err := CreateTournament(store, TournamentPatch{Name: set("Cup")})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// malformed date rejected with the right field
	_, badErr := UpdateTournament(store, created.ID, TournamentPatch{StartsOn: set("banana")})
	if badErr == nil || badErr.Code != CodeInvalidValue || badErr.Field != "starts_on" {
		t.Fatalf("bad starts_on: got %+v, want INVALID_VALUE/starts_on", badErr)
	}
	// bad zone rejected
	_, zoneErr := UpdateTournament(store, created.ID, TournamentPatch{TimeZone: set("Mars/Phobos")})
	if zoneErr == nil || zoneErr.Field != "time_zone" {
		t.Fatalf("bad zone: got %+v, want time_zone", zoneErr)
	}
	// good values normalize; empty clears
	ok, okErr := UpdateTournament(store, created.ID, TournamentPatch{
		StartsOn: set("2026-7-5"), VenueOpens: set("8:00"), TimeZone: set("Asia/Seoul"),
	})
	if okErr != nil {
		t.Fatalf("good update: %v", okErr)
	}
	if *ok.StartsOn != "2026-07-05" || *ok.VenueOpens != "08:00" || *ok.TimeZone != "Asia/Seoul" {
		t.Fatalf("normalize: %+v", ok)
	}
	cleared, _ := UpdateTournament(store, created.ID, TournamentPatch{StartsOn: setEmpty()})
	if cleared.StartsOn != nil {
		t.Fatalf("empty starts_on should clear, got %v", *cleared.StartsOn)
	}
}
```
Helpers `set`/`setEmpty` construct an `Opt[string]`: if the test file lacks them, add
```go
func set(v string) Opt[string]  { return Opt[string]{Set: true, Value: v} }
func setEmpty() Opt[string]      { return Opt[string]{Set: true, Value: ""} }
```
(reuse existing equivalents if the package already has them — read the file first).

- [ ] **Step 2: Run to verify it fails**

Run: `export PATH="$PATH:/usr/local/go/bin" && go test ./internal/api/ -run TestTournamentDateTimeValidation -v`
Expected: FAIL (compile — `TournamentPatch` has no `TimeZone`; `applyNullableString` undefined).

- [ ] **Step 3: Add the helper to `opt.go`**

Add beside `applyNullable`:
```go
// applyNullableString applies a nullable text field: absent leaves it, null or
// "" clears it, and a non-empty value is validated by check (which returns the
// canonical form). Malformed input yields an INVALID_VALUE error on name.
func applyNullableString(field Opt[string], target **string, name string,
	check func(string) (string, bool)) *Error {
	if !field.Set {
		return nil
	}
	if field.Null || field.Value == "" {
		*target = nil
		return nil
	}
	canonical, ok := check(field.Value)
	if !ok {
		return &Error{Code: CodeInvalidValue, Field: name, Message: "invalid " + name}
	}
	*target = &canonical
	return nil
}
```

- [ ] **Step 4: Wire the Tournament fields**

In `internal/api/tournament.go`:
- Add to `TournamentPatch`, after `DefaultGapMinutes`:
  ```go
  	TimeZone            Opt[string] `json:"time_zone"`
  ```
- In `applyTournamentPatch`, replace the four `applyNullable` calls for `StartsOn`/`EndsOn`/`VenueOpens`/`VenueCloses`, and add `TimeZone`, keeping the `name`/`location`/`*_minutes` lines unchanged:
  ```go
  	if applyError := applyNullableString(patch.StartsOn, &tournament.StartsOn, "starts_on", validDate); applyError != nil {
  		return applyError
  	}
  	if applyError := applyNullableString(patch.EndsOn, &tournament.EndsOn, "ends_on", validDate); applyError != nil {
  		return applyError
  	}
  	if applyError := applyNullableString(patch.VenueOpens, &tournament.VenueOpens, "venue_opens", validClock); applyError != nil {
  		return applyError
  	}
  	if applyError := applyNullableString(patch.VenueCloses, &tournament.VenueCloses, "venue_closes", validClock); applyError != nil {
  		return applyError
  	}
  	if applyError := applyNullableString(patch.TimeZone, &tournament.TimeZone, "time_zone", validZone); applyError != nil {
  		return applyError
  	}
  ```

- [ ] **Step 5: Run the API test to verify it passes**

Run: `export PATH="$PATH:/usr/local/go/bin" && go test ./internal/api/ -run TestTournamentDateTimeValidation -v`
Expected: PASS.

- [ ] **Step 6: Add an HTTP-level test**

In the existing `internal/server` test file that covers tournament PATCH (find it: `grep -rl "PATCH.*tournaments\|patchTournament\|/api/tournaments/" internal/server/*_test.go`), add a case: `PATCH /api/tournaments/{id}` with body `{"starts_on":"banana"}` → expect HTTP 400 and a JSON envelope whose `error.field == "starts_on"`; and `{"time_zone":"Asia/Seoul"}` → 200. Mirror the assertion style already used in that file (do not invent a new harness).

- [ ] **Step 7: Run the full suite**

Run: `export PATH="$PATH:/usr/local/go/bin" && go test ./...`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/api/opt.go internal/api/tournament.go internal/api/tournament_test.go internal/server
git commit -m "feat(api): validate + normalize tournament date/time/zone fields"
```

---

### Task 4: Wire the instant fields (booking, team, match)

**Files:**
- Modify: `internal/api/booking.go` (`starts_at`, `ends_at`)
- Modify: `internal/api/team.go` (`withdrawn_at`)
- Modify: `internal/api/match.go` (`scheduled_at`)
- Test: `internal/api/booking_test.go`, `internal/api/team_test.go`, `internal/api/match_test.go`

**Interfaces:**
- Consumes: `applyNullableString` (Task 3) and `validNaiveTime` (Task 2).

- [ ] **Step 1: Write failing tests (one per entity)**

For each of `booking_test.go`, `team_test.go`, `match_test.go`, add a test that:
- creates the entity (reuse the file's existing create helper / parent fixtures),
- updates it with a bad instant (`"2026-07-15T14:00:00+09:00"`) and asserts `Error.Code == CodeInvalidValue` with `Field` equal to the field's json name (`starts_at` / `withdrawn_at` / `scheduled_at`),
- updates with a good naive value (`"2026-07-15T14:00"`) and asserts it is stored canonicalized.

Example (booking; adapt names/fixtures per file):
```go
func TestBookingScheduleValidation(t *testing.T) {
	store := newTestStore(t)
	tid := mustTournament(t, store) // reuse existing fixture helper
	b, err := CreateBooking(store, BookingPatch{TournamentID: setInt(tid)})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, e := UpdateBooking(store, b.ID, BookingPatch{StartsAt: set("2026-07-15T14:00:00+09:00")}); e == nil || e.Field != "starts_at" {
		t.Fatalf("offset should be rejected on starts_at, got %+v", e)
	}
	ok, e := UpdateBooking(store, b.ID, BookingPatch{StartsAt: set("2026-07-15T14:00")})
	if e != nil || *ok.StartsAt != "2026-07-15T14:00" {
		t.Fatalf("good starts_at: %+v err=%+v", ok, e)
	}
}
```
(Use the actual create/update function names and fixture helpers from each file — read them first.)

- [ ] **Step 2: Run to verify they fail**

Run: `export PATH="$PATH:/usr/local/go/bin" && go test ./internal/api/ -run 'ScheduleValidation|WithdrawnValidation|ScheduledValidation' -v`
Expected: FAIL (the fields still use plain `applyNullable`, so offsets are accepted → assertions fail).

- [ ] **Step 3: Wire the three entities**

- `internal/api/booking.go` — replace:
  ```go
  	applyNullable(patch.StartsAt, &booking.StartsAt)
  	applyNullable(patch.EndsAt, &booking.EndsAt)
  ```
  with:
  ```go
  	if e := applyNullableString(patch.StartsAt, &booking.StartsAt, "starts_at", validNaiveTime); e != nil {
  		return e
  	}
  	if e := applyNullableString(patch.EndsAt, &booking.EndsAt, "ends_at", validNaiveTime); e != nil {
  		return e
  	}
  ```
- `internal/api/team.go` — replace `applyNullable(patch.WithdrawnAt, &team.WithdrawnAt)` with:
  ```go
  	if e := applyNullableString(patch.WithdrawnAt, &team.WithdrawnAt, "withdrawn_at", validNaiveTime); e != nil {
  		return e
  	}
  ```
- `internal/api/match.go` — replace `applyNullable(patch.ScheduledAt, &match.ScheduledAt)` with:
  ```go
  	if e := applyNullableString(patch.ScheduledAt, &match.ScheduledAt, "scheduled_at", validNaiveTime); e != nil {
  		return e
  	}
  ```
  (These `applyXPatch` functions already return `*Error`; the new early-returns fit the existing chain.)

- [ ] **Step 4: Run to verify they pass**

Run: `export PATH="$PATH:/usr/local/go/bin" && go test ./internal/api/ -run 'ScheduleValidation|WithdrawnValidation|ScheduledValidation' -v`
Expected: PASS.

- [ ] **Step 5: Run the full suite**

Run: `export PATH="$PATH:/usr/local/go/bin" && go test ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/api/booking.go internal/api/team.go internal/api/match.go internal/api/booking_test.go internal/api/team_test.go internal/api/match_test.go
git commit -m "feat(api): validate naive-datetime fields on bookings, teams, matches"
```

---

### Task 5: OpenAPI documentation

**Files:**
- Modify: `internal/server/openapi.yaml`

**Interfaces:** documentation only — no runtime behavior. Validation already lives in the api layer.

- [ ] **Step 1: Add `time_zone` and format hints to the spec**

In `internal/server/openapi.yaml`, `components.schemas`:
- `Tournament`: add a `time_zone` property after `default_gap_minutes`:
  ```yaml
        time_zone:
          type: string
          nullable: true
          description: IANA timezone (e.g. Asia/Seoul); the single anchor for interpreting this tournament's local times.
  ```
  and set `starts_on`/`ends_on` → add `format: date`; ensure `venue_opens`/`venue_closes` keep `description: HH:MM local`.
- `FieldBooking` `starts_at`/`ends_at`, `Team` `withdrawn_at`, `Match` `scheduled_at`: add
  `description: naive local datetime YYYY-MM-DDTHH:MM (no offset; interpreted in the tournament time_zone).`

- [ ] **Step 2: Verify the spec still parses and routes still sync**

Run: `export PATH="$PATH:/usr/local/go/bin" && go test ./internal/server/ -v`
Expected: PASS (the route-sync test checks paths, not schema fields; this confirms the YAML is still valid and embeds).

- [ ] **Step 3: Sanity-check the served spec**

Run:
```bash
export PATH="$PATH:/usr/local/go/bin"
go build -o /tmp/dt-check ./cmd/ssl-tournament && /tmp/dt-check --port 8091 & sleep 2
curl -s http://localhost:8091/api/openapi.json | grep -o '"time_zone"' | head -1
kill %1 2>/dev/null
```
Expected: prints `"time_zone"` (the field is present in the served spec).

- [ ] **Step 4: Commit**

```bash
git add internal/server/openapi.yaml
git commit -m "docs(api): openapi time_zone + date/time format hints"
```

---

## Self-Review

**Spec coverage** (against `2026-07-18-datetime-validation-timezone-design.md`):
- `time_zone` column + struct + SQL → Task 1. ✓
- Validators (date/clock/naive/zone) with normalization → Task 2. ✓
- Empty→null, null clears, absent leaves, malformed→INVALID_VALUE+field → Task 3 (helper) + Tasks 3/4 (wiring/tests). ✓
- Applied across Tournament, FieldBooking, Team, Match → Tasks 3, 4. ✓
- Offset/`Z` rejected on instants → Task 2 test + Task 4 tests. ✓
- `time/tzdata` embedded → Task 2 (`datetime.go`). ✓
- OpenAPI updated → Task 5. ✓
- `created_at` untouched → not modified in any task. ✓
- HTTP-level validation surfaced → Task 3 Step 6. ✓
- Migration applies via numbered runner → Task 1. ✓

**Placeholder scan:** test bodies note "match the file's existing helper" where a fixture name must be read from the target file — this is guidance to reuse existing infrastructure, not a missing value; the assertions and production code are complete. No TBD/TODO.

**Type consistency:** `applyNullableString(Opt[string], **string, string, func(string)(string,bool)) *Error` is defined in Task 3 and consumed in Tasks 3 & 4; validators `validDate/validClock/validNaiveTime/validZone` defined in Task 2 match their uses; `store.Tournament.TimeZone *string` (Task 1) matches `applyNullableString`'s `**string` target and `TournamentPatch.TimeZone Opt[string]` (Task 3). Field json names (`starts_on`, `venue_opens`, `time_zone`, `starts_at`, `ends_at`, `withdrawn_at`, `scheduled_at`) match the `Error.Field` values and the store columns. ✓
