// Error-envelope tests: every error code observable through the API.
package server

import (
	"net/http"
	"testing"
)

func TestErrorCodes(t *testing.T) {
	testServer := newTestServer(t)
	cases := []struct {
		name   string
		method string
		path   string
		body   string
		status int
		code   string
	}{
		{"invalid json", "POST", "/api/tournaments", `{"name": `, http.StatusBadRequest, "INVALID_JSON"},
		{"unknown field", "POST", "/api/tournaments", `{"colour": "blue"}`, http.StatusBadRequest, "UNKNOWN_FIELD"},
		{"server-set field", "POST", "/api/tournaments", `{"id": 7}`, http.StatusBadRequest, "UNKNOWN_FIELD"},
		{"wrong type", "POST", "/api/tournaments", `{"name": 42}`, http.StatusBadRequest, "INVALID_VALUE"},
		{"null on not-null", "PATCH", "/api/tournaments/1", `{"name": null}`, http.StatusBadRequest, "INVALID_VALUE"},
		{"bad id", "GET", "/api/tournaments/abc", "", http.StatusBadRequest, "INVALID_VALUE"},
		{"missing row", "GET", "/api/tournaments/99", "", http.StatusNotFound, "NOT_FOUND"},
		{"patch missing row", "PATCH", "/api/tournaments/99", `{"name": "x"}`, http.StatusNotFound, "NOT_FOUND"},
		{"delete missing row", "DELETE", "/api/tournaments/99", "", http.StatusNotFound, "NOT_FOUND"},
	}
	if status, _ := doRequest(t, testServer, "POST", "/api/tournaments", `{"name": "T"}`); status != http.StatusCreated {
		t.Fatal("seed tournament failed")
	}
	for _, testCase := range cases {
		status, body := doRequest(t, testServer, testCase.method, testCase.path, testCase.body)
		if status != testCase.status || errorCode(t, body) != testCase.code {
			t.Errorf("%s: got %d %s, want %d %s",
				testCase.name, status, body, testCase.status, testCase.code)
		}
	}
	status, body := doRequest(t, testServer, "GET", "/api/tournaments?colour=1", "")
	if status != http.StatusBadRequest || errorCode(t, body) != "UNKNOWN_FIELD" {
		t.Errorf("unknown query param: %d %s", status, body)
	}
}
