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
	mustExec(t, s, `INSERT INTO team(id,tournament_id,name) VALUES (999,1,'X')`)
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
