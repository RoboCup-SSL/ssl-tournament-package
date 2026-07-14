// CRUD and slot-source tests for /api/matches.
package server

import (
	"net/http"
	"testing"
)

func TestMatchCRUD(t *testing.T) {
	testServer := newTestServer(t)
	seedBase(t, testServer)
	if status, body := doRequest(t, testServer, "POST", "/api/groups",
		`{"tournament_id": 1, "division_id": 1, "name": "G1"}`); status != http.StatusCreated {
		t.Fatalf("seed group: %d %s", status, body)
	}

	status, body := doRequest(t, testServer, "POST", "/api/matches",
		`{"tournament_id": 1, "division_id": 1, "label": "SF1", "field_id": 1,
		  "b_team_id": 2,
		  "a_source": {"kind": "group_rank", "group_id": 1, "rank": 1}}`)
	if status != http.StatusCreated {
		t.Fatalf("create: %d %s", status, body)
	}
	created := asObject(t, body)
	if created["status"] != "scheduled" {
		t.Errorf("default status: %v", created["status"])
	}
	sourceA := created["a_source"].(map[string]any)
	if sourceA["kind"] != "group_rank" || sourceA["rank"] != float64(1) {
		t.Errorf("a_source not returned: %v", created)
	}
	if created["b_source"] != nil {
		t.Errorf("b_source must be null: %v", created)
	}

	status, body = doRequest(t, testServer, "PATCH", "/api/matches/1",
		`{"status": "finished", "a_score": 3, "b_score": 1, "winner_team_id": 2,
		  "b_source": {"kind": "match_winner", "match_id": 1}}`)
	if status != http.StatusOK {
		t.Fatalf("patch: %d %s", status, body)
	}
	patched := asObject(t, body)
	if patched["a_score"] != float64(3) || patched["status"] != "finished" {
		t.Errorf("scalar patch: %v", patched)
	}
	sourceA = patched["a_source"].(map[string]any)
	if sourceA["kind"] != "group_rank" {
		t.Errorf("a_source must survive an unrelated patch: %v", patched)
	}
	if patched["b_source"].(map[string]any)["kind"] != "match_winner" {
		t.Errorf("b_source not set: %v", patched)
	}

	status, body = doRequest(t, testServer, "PATCH", "/api/matches/1",
		`{"a_source": null}`)
	if status != http.StatusOK || asObject(t, body)["a_source"] != nil {
		t.Fatalf("null must delete a_source: %d %s", status, body)
	}

	status, body = doRequest(t, testServer, "PATCH", "/api/matches/1",
		`{"a_source": {"kind": "penalty_shootout"}}`)
	if status != http.StatusBadRequest || errorCode(t, body) != "INVALID_VALUE" {
		t.Errorf("bad source kind enum: %d %s", status, body)
	}

	status, body = doRequest(t, testServer, "PATCH", "/api/matches/1",
		`{"status": "postponed"}`)
	if status != http.StatusBadRequest || errorCode(t, body) != "INVALID_VALUE" {
		t.Errorf("bad status enum: %d %s", status, body)
	}

	filterCount(t, testServer, "/api/matches?tournament_id=1&field_id=1", 1)
	filterCount(t, testServer, "/api/matches?group_id=1", 0)

	status, _ = doRequest(t, testServer, "DELETE", "/api/matches/1", "")
	if status != http.StatusNoContent {
		t.Fatalf("delete: %d", status)
	}
	status, _ = doRequest(t, testServer, "GET", "/api/matches/1", "")
	if status != http.StatusNotFound {
		t.Fatalf("get after delete: %d", status)
	}
}
