// Shared CRUD exercises for the plain (single-table) resources.
package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// seedBase creates tournament 1, division 1, teams 1+2, and field 1 via the API.
func seedBase(t *testing.T, testServer *httptest.Server) {
	t.Helper()
	seeds := []struct{ path, body string }{
		{"/api/tournaments", `{"name": "Seed Cup"}`},
		{"/api/divisions", `{"tournament_id": 1, "name": "A"}`},
		{"/api/teams", `{"tournament_id": 1, "division_id": 1, "name": "Alpha"}`},
		{"/api/teams", `{"tournament_id": 1, "division_id": 1, "name": "Beta"}`},
		{"/api/fields", `{"tournament_id": 1, "name": "Field 1"}`},
	}
	for _, seed := range seeds {
		if status, body := doRequest(t, testServer, "POST", seed.path, seed.body); status != http.StatusCreated {
			t.Fatalf("seed %s: %d %s", seed.path, status, body)
		}
	}
}

// exerciseCRUD runs the uniform create/get/patch/list/delete cycle on one
// resource and verifies the PATCH tri-state expectations.
func exerciseCRUD(t *testing.T, testServer *httptest.Server, path, createBody, patchBody string, wantAfterPatch map[string]any) {
	t.Helper()
	status, body := doRequest(t, testServer, "POST", path, createBody)
	if status != http.StatusCreated {
		t.Fatalf("create %s: %d %s", path, status, body)
	}
	id := int64(asObject(t, body)["id"].(float64))
	itemPath := fmt.Sprintf("%s/%d", path, id)

	if status, body = doRequest(t, testServer, "GET", itemPath, ""); status != http.StatusOK {
		t.Fatalf("get %s: %d %s", itemPath, status, body)
	}
	status, body = doRequest(t, testServer, "PATCH", itemPath, patchBody)
	if status != http.StatusOK {
		t.Fatalf("patch %s: %d %s", itemPath, status, body)
	}
	patched := asObject(t, body)
	for key, want := range wantAfterPatch {
		if patched[key] != want {
			t.Errorf("%s after patch: %s = %v, want %v", itemPath, key, patched[key], want)
		}
	}
	if status, body = doRequest(t, testServer, "GET", path, ""); status != http.StatusOK || len(asList(t, body)) == 0 {
		t.Fatalf("list %s: %d %s", path, status, body)
	}
	if status, _ = doRequest(t, testServer, "DELETE", itemPath, ""); status != http.StatusNoContent {
		t.Fatalf("delete %s: %d", itemPath, status)
	}
	if status, _ = doRequest(t, testServer, "GET", itemPath, ""); status != http.StatusNotFound {
		t.Fatalf("get after delete %s: %d", itemPath, status)
	}
}

// filterCount asserts a filtered list returns exactly want rows.
func filterCount(t *testing.T, testServer *httptest.Server, pathWithQuery string, want int) {
	t.Helper()
	status, body := doRequest(t, testServer, "GET", pathWithQuery, "")
	if status != http.StatusOK || len(asList(t, body)) != want {
		t.Errorf("%s: %d %s, want %d rows", pathWithQuery, status, body, want)
	}
}

func TestDivisionCRUD(t *testing.T) {
	testServer := newTestServer(t)
	seedBase(t, testServer)
	exerciseCRUD(t, testServer, "/api/divisions",
		`{"tournament_id": 1, "name": "B"}`,
		`{"notes": "entry level"}`,
		map[string]any{"name": "B", "notes": "entry level"})
	filterCount(t, testServer, "/api/divisions?tournament_id=1", 1)
	filterCount(t, testServer, "/api/divisions?tournament_id=99", 0)
	status, body := doRequest(t, testServer, "POST", "/api/divisions",
		`{"tournament_id": 99, "name": "ghost"}`)
	if status != http.StatusConflict || errorCode(t, body) != "MISSING_REFERENCE" {
		t.Errorf("dangling FK: %d %s", status, body)
	}
}

func TestTeamCRUD(t *testing.T) {
	testServer := newTestServer(t)
	seedBase(t, testServer)
	exerciseCRUD(t, testServer, "/api/teams",
		`{"tournament_id": 1, "division_id": 1, "name": "Gamma", "country": "NL"}`,
		`{"division_id": null, "withdrawn_at": "2026-07-15T09:00:00Z"}`,
		map[string]any{"name": "Gamma", "division_id": nil,
			"withdrawn_at": "2026-07-15T09:00:00Z"})
	filterCount(t, testServer, "/api/teams?division_id=1", 2)
	filterCount(t, testServer, "/api/teams?tournament_id=1&division_id=1", 2)
	filterCount(t, testServer, "/api/teams?division_id=99", 0)
}

func TestFieldCRUD(t *testing.T) {
	testServer := newTestServer(t)
	seedBase(t, testServer)
	exerciseCRUD(t, testServer, "/api/fields",
		`{"tournament_id": 1, "name": "Field 2"}`,
		`{"name": "Field 2 (renamed)"}`,
		map[string]any{"name": "Field 2 (renamed)"})
	filterCount(t, testServer, "/api/fields?tournament_id=1", 1)
}
