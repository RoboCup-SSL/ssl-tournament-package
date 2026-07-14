// Embedded API documentation: the OpenAPI spec and the Swagger UI assets.
package server

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed openapi.yaml
var openAPISpec []byte

//go:embed swaggerui
var swaggerUIFiles embed.FS

// registerDocs adds the documentation routes to mux.
func registerDocs(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/openapi.yaml", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/yaml")
		writer.Write(openAPISpec)
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
