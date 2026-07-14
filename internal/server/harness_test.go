// Shared HTTP test harness: a real server over a fresh temp-dir database.
package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/RoboCup-SSL/ssl-tournament-package/internal/store"
)

// newTestServer returns an httptest server backed by a fresh database.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	dataStore, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { dataStore.Close() })
	testServer := httptest.NewServer(NewMux(dataStore))
	t.Cleanup(testServer.Close)
	return testServer
}

// doRequest issues one HTTP request with a JSON body and returns the status
// and raw response body.
func doRequest(t *testing.T, testServer *httptest.Server, method, path, body string) (int, []byte) {
	t.Helper()
	request, err := http.NewRequest(method, testServer.URL+path, bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatal(err)
	}
	response, err := testServer.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, responseBody
}

// asObject decodes a JSON object response.
func asObject(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var object map[string]any
	if err := json.Unmarshal(body, &object); err != nil {
		t.Fatalf("decode %s: %v", body, err)
	}
	return object
}

// asList decodes a JSON array response.
func asList(t *testing.T, body []byte) []any {
	t.Helper()
	var list []any
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("decode %s: %v", body, err)
	}
	return list
}

// errorCode extracts the code from an error envelope.
func errorCode(t *testing.T, body []byte) string {
	t.Helper()
	envelope := asObject(t, body)
	errorObject, isObject := envelope["error"].(map[string]any)
	if !isObject {
		t.Fatalf("no error envelope in %s", body)
	}
	code, _ := errorObject["code"].(string)
	return code
}
