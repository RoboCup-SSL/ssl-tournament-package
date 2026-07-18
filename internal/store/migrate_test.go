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
