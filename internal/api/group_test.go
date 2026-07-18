// Tests for Group patch validation.
package api

import "testing"

func TestGroupRankingConfirmedAtValidation(t *testing.T) {
	dataStore := newTestStore(t)
	tournament, err := CreateTournament(dataStore, TournamentPatch{Name: set("T")})
	if err != nil {
		t.Fatalf("create tournament: %v", err)
	}

	_, badErr := CreateGroup(dataStore, GroupPatch{
		TournamentID:       Opt[int64]{Set: true, Value: tournament.ID},
		Name:               set("A"),
		RankingConfirmedAt: set("2026-07-15T14:00:00+09:00"),
	})
	if badErr == nil || badErr.Code != CodeInvalidValue || badErr.Field != "ranking_confirmed_at" {
		t.Fatalf("bad ranking_confirmed_at: got %+v, want INVALID_VALUE/ranking_confirmed_at", badErr)
	}

	good, goodErr := CreateGroup(dataStore, GroupPatch{
		TournamentID:       Opt[int64]{Set: true, Value: tournament.ID},
		Name:               set("B"),
		RankingConfirmedAt: set("2026-07-15T14:00"),
	})
	if goodErr != nil {
		t.Fatalf("good ranking_confirmed_at: %v", goodErr)
	}
	if good.RankingConfirmedAt == nil || *good.RankingConfirmedAt != "2026-07-15T14:00" {
		t.Fatalf("expected canonical naive datetime, got %+v", good.RankingConfirmedAt)
	}
}
