// CRUD tests for /api/tournaments — the behavior contract every resource shares.
package server

import (
	"net/http"
	"testing"
)

func TestTournamentCRUD(t *testing.T) {
	testServer := newTestServer(t)

	status, body := doRequest(t, testServer, "GET", "/api/tournaments", "")
	if status != http.StatusOK || string(body) != "[]\n" {
		t.Fatalf("empty list: %d %s", status, body)
	}

	status, body = doRequest(t, testServer, "POST", "/api/tournaments",
		`{"name": "RoboCup 2026", "location": "Incheon", "default_match_minutes": 60}`)
	if status != http.StatusCreated {
		t.Fatalf("create: %d %s", status, body)
	}
	created := asObject(t, body)
	if created["id"] != float64(1) || created["name"] != "RoboCup 2026" ||
		created["starts_on"] != nil || created["created_at"] == "" {
		t.Fatalf("create response: %v", created)
	}

	status, body = doRequest(t, testServer, "GET", "/api/tournaments/1", "")
	if status != http.StatusOK || asObject(t, body)["location"] != "Incheon" {
		t.Fatalf("get: %d %s", status, body)
	}

	status, body = doRequest(t, testServer, "PATCH", "/api/tournaments/1",
		`{"location": "Eindhoven", "default_match_minutes": null}`)
	if status != http.StatusOK {
		t.Fatalf("patch: %d %s", status, body)
	}
	patched := asObject(t, body)
	if patched["location"] != "Eindhoven" || patched["name"] != "RoboCup 2026" ||
		patched["default_match_minutes"] != nil {
		t.Fatalf("tri-state violated: %v", patched)
	}

	status, body = doRequest(t, testServer, "GET", "/api/tournaments", "")
	if status != http.StatusOK || len(asList(t, body)) != 1 {
		t.Fatalf("list: %d %s", status, body)
	}

	status, _ = doRequest(t, testServer, "DELETE", "/api/tournaments/1", "")
	if status != http.StatusNoContent {
		t.Fatalf("delete: %d", status)
	}
	status, _ = doRequest(t, testServer, "GET", "/api/tournaments/1", "")
	if status != http.StatusNotFound {
		t.Fatalf("get after delete: %d", status)
	}
}
