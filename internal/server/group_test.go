// CRUD and composite-list tests for /api/groups.
package server

import (
	"net/http"
	"testing"
)

func TestGroupCRUD(t *testing.T) {
	testServer := newTestServer(t)
	seedBase(t, testServer)

	status, body := doRequest(t, testServer, "POST", "/api/groups",
		`{"tournament_id": 1, "division_id": 1, "name": "G1", "member_team_ids": [1, 2]}`)
	if status != http.StatusCreated {
		t.Fatalf("create: %d %s", status, body)
	}
	created := asObject(t, body)
	if members, _ := created["member_team_ids"].([]any); len(members) != 2 {
		t.Fatalf("members not returned: %v", created)
	}
	if ranking, isList := created["ranking"].([]any); !isList || len(ranking) != 0 {
		t.Fatalf("ranking must be [] not null: %v", created)
	}

	status, body = doRequest(t, testServer, "PATCH", "/api/groups/1",
		`{"ranking": [{"team_id": 2, "rank": 1}, {"team_id": 1, "rank": 1}],
		  "ranking_confirmed_at": "2026-07-16T18:00:00"}`)
	if status != http.StatusOK {
		t.Fatalf("patch ranking: %d %s", status, body)
	}
	patched := asObject(t, body)
	ranking := patched["ranking"].([]any)
	if len(ranking) != 2 {
		t.Fatalf("tie ranking must be stored: %v", patched)
	}
	first := ranking[0].(map[string]any)
	if first["team_id"] != float64(1) || first["rank"] != float64(1) {
		t.Errorf("ranking order (rank, then team_id): %v", ranking)
	}

	status, body = doRequest(t, testServer, "PATCH", "/api/groups/1",
		`{"member_team_ids": [1]}`)
	if status != http.StatusOK {
		t.Fatalf("replace members: %d %s", status, body)
	}
	patched = asObject(t, body)
	if members := patched["member_team_ids"].([]any); len(members) != 1 {
		t.Errorf("member replace failed: %v", patched)
	}
	if ranking := patched["ranking"].([]any); len(ranking) != 2 {
		t.Errorf("ranking must be untouched by member patch: %v", patched)
	}

	status, body = doRequest(t, testServer, "PATCH", "/api/groups/1",
		`{"ranking": null}`)
	if status != http.StatusOK || len(asObject(t, body)["ranking"].([]any)) != 0 {
		t.Fatalf("null must clear ranking: %d %s", status, body)
	}

	status, body = doRequest(t, testServer, "PATCH", "/api/groups/1",
		`{"ranking": [{"team_id": 1, "rank": 1}, {"team_id": 1, "rank": 2}]}`)
	if status != http.StatusBadRequest || errorCode(t, body) != "INVALID_VALUE" {
		t.Errorf("duplicate team in ranking: %d %s", status, body)
	}

	status, body = doRequest(t, testServer, "GET", "/api/groups?division_id=1", "")
	if status != http.StatusOK {
		t.Fatalf("filtered list: %d %s", status, body)
	}
	listed := asList(t, body)
	if len(listed) != 1 {
		t.Fatalf("filter: %v", listed)
	}
	if members := listed[0].(map[string]any)["member_team_ids"].([]any); len(members) != 1 {
		t.Errorf("list must include members: %v", listed)
	}

	status, _ = doRequest(t, testServer, "DELETE", "/api/groups/1", "")
	if status != http.StatusNoContent {
		t.Fatalf("delete: %d", status)
	}
	status, _ = doRequest(t, testServer, "GET", "/api/groups/1", "")
	if status != http.StatusNotFound {
		t.Fatalf("get after delete: %d", status)
	}
}
