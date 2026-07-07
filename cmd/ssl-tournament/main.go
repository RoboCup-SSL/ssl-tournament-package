package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/RoboCup-SSL/ssl-tournament-package/frontend"
)

var (
	host = flag.String("host", "0.0.0.0", "The host/interface to bind on")
	port = flag.String("port", "8080", "The port to serve on")
)

func main() {
	flag.Parse()

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	frontend.HandleUi()

	addr := *host + ":" + *port
	log.Printf("ssl-tournament serving on http://%s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
