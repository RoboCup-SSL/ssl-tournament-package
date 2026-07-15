// Package server tests.
package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/RoboCup-SSL/ssl-tournament-package/internal/store"
)

// TestHealthzReportsDB verifies /healthz pings the database.
func TestHealthzReportsDB(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	srv := httptest.NewServer(NewMux("test-version", st))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || string(body) != "ok" {
		t.Fatalf("healthz = %d %q, want 200 ok", resp.StatusCode, body)
	}
}

// TestVersionEndpoint verifies /api/version reports the build version.
func TestVersionEndpoint(t *testing.T) {
	testServer := newTestServer(t)
	status, body := doRequest(t, testServer, "GET", "/api/version", "")
	if status != http.StatusOK || asObject(t, body)["version"] != "test-version" {
		t.Errorf("version endpoint: %d %s", status, body)
	}
}

// TestNoStoreHeaders verifies the web UI and spec endpoints forbid browser
// caching, so a binary upgrade is visible on the next page load.
func TestNoStoreHeaders(t *testing.T) {
	testServer := newTestServer(t)
	for _, path := range []string{"/", "/api/version", "/api/openapi.yaml", "/api/openapi.json"} {
		response, err := testServer.Client().Get(testServer.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if header := response.Header.Get("Cache-Control"); header != "no-store" {
			t.Errorf("%s: Cache-Control = %q, want no-store", path, header)
		}
	}
}
