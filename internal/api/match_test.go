// Tests for Match patch validation.
package api

import "testing"

func TestMatchScheduledValidation(t *testing.T) {
	store := newTestStore(t)
	tournament, err := CreateTournament(store, TournamentPatch{Name: set("Cup")})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	match, err := CreateMatch(store, MatchPatch{
		TournamentID: Opt[int64]{Set: true, Value: tournament.ID},
	})
	if err != nil {
		t.Fatalf("create match: %v", err)
	}

	if _, e := UpdateMatch(store, match.ID, MatchPatch{ScheduledAt: set("2026-07-15T14:00:00+09:00")}); e == nil || e.Code != CodeInvalidValue || e.Field != "scheduled_at" {
		t.Fatalf("offset should be rejected on scheduled_at, got %+v", e)
	}

	ok, e := UpdateMatch(store, match.ID, MatchPatch{ScheduledAt: set("2026-07-15T14:00")})
	if e != nil {
		t.Fatalf("good update: %v", e)
	}
	if ok.ScheduledAt == nil || *ok.ScheduledAt != "2026-07-15T14:00" {
		t.Fatalf("scheduled_at not stored canonical: %+v", ok)
	}
}
