// Tests that the docs endpoints serve and that routes and openapi.yaml agree.
package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDocsEndpoints(t *testing.T) {
	testServer := newTestServer(t)
	status, body := doRequest(t, testServer, "GET", "/api/openapi.yaml", "")
	if status != http.StatusOK || !strings.Contains(string(body), "openapi:") {
		t.Errorf("openapi.yaml: %d", status)
	}
	status, body = doRequest(t, testServer, "GET", "/api/docs/", "")
	if status != http.StatusOK || !strings.Contains(string(body), "SwaggerUIBundle") {
		t.Errorf("docs index: %d", status)
	}
}

// TestOpenAPIJSON verifies the JSON rendering of the spec and the schema
// annotations the admin UI relies on: required lists and server defaults.
func TestOpenAPIJSON(t *testing.T) {
	testServer := newTestServer(t)
	status, body := doRequest(t, testServer, "GET", "/api/openapi.json", "")
	if status != http.StatusOK {
		t.Fatalf("openapi.json: %d", status)
	}
	var document struct {
		Components struct {
			Schemas map[string]struct {
				Required   []string                  `json:"required"`
				Properties map[string]map[string]any `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(body, &document); err != nil {
		t.Fatalf("openapi.json is not valid JSON: %v", err)
	}
	match, exists := document.Components.Schemas["Match"]
	if !exists {
		t.Fatal("Match schema missing from openapi.json")
	}
	if len(match.Required) != 1 || match.Required[0] != "tournament_id" {
		t.Errorf("Match.required: got %v, want [tournament_id]", match.Required)
	}
	if defaultValue := match.Properties["status"]["default"]; defaultValue != "scheduled" {
		t.Errorf("Match.status default: got %v, want scheduled", defaultValue)
	}
	booking := document.Components.Schemas["FieldBooking"]
	if defaultValue := booking.Properties["kind"]["default"]; defaultValue != "booking" {
		t.Errorf("FieldBooking.kind default: got %v, want booking", defaultValue)
	}
}

func TestRoutesMatchOpenAPI(t *testing.T) {
	var document struct {
		Paths map[string]map[string]any `yaml:"paths"`
	}
	if err := yaml.Unmarshal(openAPISpec, &document); err != nil {
		t.Fatal(err)
	}
	specSet := make(map[string]bool)
	for path, operations := range document.Paths {
		for method := range operations {
			switch method {
			case "get", "post", "patch", "delete":
				specSet[strings.ToUpper(method)+" "+path] = true
			}
		}
	}
	routeSet := make(map[string]bool)
	testHandlers := &handlers{}
	for _, apiRoute := range testHandlers.routes() {
		routeSet[apiRoute.Method+" "+apiRoute.Pattern] = true
	}
	for entry := range routeSet {
		if !specSet[entry] {
			t.Errorf("route %s missing from openapi.yaml", entry)
		}
	}
	for entry := range specSet {
		if !routeSet[entry] {
			t.Errorf("openapi.yaml documents %s but no route exists", entry)
		}
	}
}
