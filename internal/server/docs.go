// Embedded API documentation: the OpenAPI spec and the Swagger UI assets.
package server

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"

	"gopkg.in/yaml.v3"
)

//go:embed openapi.yaml
var openAPISpec []byte

//go:embed swaggerui
var swaggerUIFiles embed.FS

// specAsJSON converts the embedded YAML spec to JSON for clients without a
// YAML parser, such as the admin UI's schema tables.
func specAsJSON() []byte {
	var document any
	if err := yaml.Unmarshal(openAPISpec, &document); err != nil {
		panic(err)
	}
	converted, err := json.Marshal(document)
	if err != nil {
		panic(err)
	}
	return converted
}

// registerDocs adds the documentation routes to mux.
func registerDocs(mux *http.ServeMux) {
	openAPISpecJSON := specAsJSON()
	mux.HandleFunc("GET /api/openapi.yaml", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/yaml")
		writer.Header().Set("Cache-Control", "no-store")
		writer.Write(openAPISpec)
	})
	mux.HandleFunc("GET /api/openapi.json", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Cache-Control", "no-store")
		writer.Write(openAPISpecJSON)
	})
	assets, err := fs.Sub(swaggerUIFiles, "swaggerui")
	if err != nil {
		panic(err)
	}
	mux.Handle("GET /api/docs/", http.StripPrefix("/api/docs/", http.FileServerFS(assets)))
	mux.HandleFunc("GET /api/docs", func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, "/api/docs/", http.StatusMovedPermanently)
	})
}
