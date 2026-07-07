package frontend

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist
var content embed.FS

// HandleUi registers the embedded web UI on the default ServeMux at "/".
func HandleUi() {
	dist, err := fs.Sub(content, "dist")
	if err != nil {
		panic(err)
	}
	http.Handle("/", http.FileServer(http.FS(dist)))
}
