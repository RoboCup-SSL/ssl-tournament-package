package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/RoboCup-SSL/ssl-tournament-package/frontend"
)

// version is stamped at build time via -ldflags "-X main.version=...".
// Defaults to "dev" for plain `go build`.
var version = "dev"

var (
	host        = flag.String("host", "0.0.0.0", "The host/interface to bind on")
	port        = flag.String("port", "8080", "The port to serve on")
	showVersion = flag.Bool("version", false, "Print version and exit")
)

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	frontend.HandleUi()

	addr := *host + ":" + *port
	log.Printf("ssl-tournament %s serving on http://%s", version, addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
