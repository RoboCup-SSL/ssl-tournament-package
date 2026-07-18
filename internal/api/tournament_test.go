// Tests for Tournament CRUD and patch validation.
package api

import (
	"path/filepath"
	"testing"

	"github.com/RoboCup-SSL/ssl-tournament-package/internal/store"
)

// newTestStore opens a fresh temp-dir database for one test.
func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	dataStore, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { dataStore.Close() })
	return dataStore
}

// set builds a present Opt[string] with the given value.
func set(v string) Opt[string] { return Opt[string]{Set: true, Value: v} }

// setEmpty builds a present Opt[string] holding an empty value.
func setEmpty() Opt[string] { return Opt[string]{Set: true, Value: ""} }

func TestTournamentDateTimeValidation(t *testing.T) {
	dataStore := newTestStore(t)
	created, err := CreateTournament(dataStore, TournamentPatch{Name: set("Cup")})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// malformed date rejected with the right field
	_, badErr := UpdateTournament(dataStore, created.ID, TournamentPatch{StartsOn: set("banana")})
	if badErr == nil || badErr.Code != CodeInvalidValue || badErr.Field != "starts_on" {
		t.Fatalf("bad starts_on: got %+v, want INVALID_VALUE/starts_on", badErr)
	}
	// bad zone rejected
	_, zoneErr := UpdateTournament(dataStore, created.ID, TournamentPatch{TimeZone: set("Mars/Phobos")})
	if zoneErr == nil || zoneErr.Field != "time_zone" {
		t.Fatalf("bad zone: got %+v, want time_zone", zoneErr)
	}
	// good values accepted; clock normalizes (8:00 -> 08:00); empty clears
	ok, okErr := UpdateTournament(dataStore, created.ID, TournamentPatch{
		StartsOn: set("2026-07-05"), VenueOpens: set("8:00"), TimeZone: set("Asia/Seoul"),
	})
	if okErr != nil {
		t.Fatalf("good update: %v", okErr)
	}
	if *ok.StartsOn != "2026-07-05" || *ok.VenueOpens != "08:00" || *ok.TimeZone != "Asia/Seoul" {
		t.Fatalf("normalize: %+v", ok)
	}
	cleared, _ := UpdateTournament(dataStore, created.ID, TournamentPatch{StartsOn: setEmpty()})
	if cleared.StartsOn != nil {
		t.Fatalf("empty starts_on should clear, got %v", *cleared.StartsOn)
	}
}
