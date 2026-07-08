package frontend

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist
var content embed.FS

// Handler serves the embedded web UI (the built frontend/dist).
func Handler() http.Handler {
	dist, err := fs.Sub(content, "dist")
	if err != nil {
		panic(err)
	}
	return http.FileServer(http.FS(dist))
}
