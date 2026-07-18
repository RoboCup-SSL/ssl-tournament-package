// Freeform regression tests: the API accepts everything the schema accepts,
// however inconsistent; only vocabulary and dangling references are rejected.
package server

import (
	"net/http"
	"testing"
)

func TestFreeformIsLegalOverHTTP(t *testing.T) {
	testServer := newTestServer(t)
	seedBase(t, testServer)
	accepted := []struct {
		name string
		path string
		body string
	}{
		{"duplicate team name", "/api/teams",
			`{"tournament_id": 1, "name": "Alpha"}`},
		{"team plays itself", "/api/matches",
			`{"tournament_id": 1, "a_team_id": 1, "b_team_id": 1}`},
		{"team referees its own match", "/api/matches",
			`{"tournament_id": 1, "a_team_id": 1, "b_team_id": 2, "referee_team_id": 1}`},
		{"blocked field", "/api/field-bookings",
			`{"tournament_id": 1, "field_id": 1, "kind": "blocked", "label": "maintenance",
			  "starts_at": "2026-07-15T09:00:00", "ends_at": "2026-07-15T18:00:00"}`},
		{"match on the blocked field", "/api/matches",
			`{"tournament_id": 1, "field_id": 1, "scheduled_at": "2026-07-15T10:00:00"}`},
		{"group_rank source carrying a match_id", "/api/matches",
			`{"tournament_id": 1,
			  "a_source": {"kind": "group_rank", "group_id": null, "rank": null, "match_id": 1}}`},
		{"finished match with no winner and equal scores", "/api/matches",
			`{"tournament_id": 1, "status": "finished", "a_score": 2, "b_score": 2}`},
		{"forfeited match with a score", "/api/matches",
			`{"tournament_id": 1, "status": "forfeited", "a_score": 10, "b_score": 0,
			  "winner_team_id": 1, "notes": "walkover"}`},
	}
	for _, testCase := range accepted {
		status, body := doRequest(t, testServer, "POST", testCase.path, testCase.body)
		if status != http.StatusCreated {
			t.Errorf("%s must be accepted: %d %s", testCase.name, status, body)
		}
	}
}
