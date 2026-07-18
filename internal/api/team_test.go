// Tests for Team patch validation.
package api

import "testing"

func TestTeamWithdrawnValidation(t *testing.T) {
	store := newTestStore(t)
	tournament, err := CreateTournament(store, TournamentPatch{Name: set("Cup")})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}
	team, err := CreateTeam(store, TeamPatch{
		TournamentID: Opt[int64]{Set: true, Value: tournament.ID},
		Name:         set("Tigers"),
	})
	if err != nil {
		t.Fatalf("create team: %v", err)
	}

	if _, e := UpdateTeam(store, team.ID, TeamPatch{WithdrawnAt: set("2026-07-15T14:00:00+09:00")}); e == nil || e.Code != CodeInvalidValue || e.Field != "withdrawn_at" {
		t.Fatalf("offset should be rejected on withdrawn_at, got %+v", e)
	}

	ok, e := UpdateTeam(store, team.ID, TeamPatch{WithdrawnAt: set("2026-07-15T14:00")})
	if e != nil {
		t.Fatalf("good update: %v", e)
	}
	if ok.WithdrawnAt == nil || *ok.WithdrawnAt != "2026-07-15T14:00" {
		t.Fatalf("withdrawn_at not stored canonical: %+v", ok)
	}
}
