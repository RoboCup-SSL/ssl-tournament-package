// Tests that the docs endpoints serve and that routes and openapi.yaml agree.
package server

import (
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
