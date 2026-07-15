// Package frontend embeds the built web UI and serves it over HTTP.
package frontend

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist
var content embed.FS

// Handler serves the embedded web UI, marked no-store so browsers always
// load the UI matching the running binary.
func Handler() http.Handler {
	dist, err := fs.Sub(content, "dist")
	if err != nil {
		panic(err)
	}
	files := http.FileServer(http.FS(dist))
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Cache-Control", "no-store")
		files.ServeHTTP(writer, request)
	})
}
