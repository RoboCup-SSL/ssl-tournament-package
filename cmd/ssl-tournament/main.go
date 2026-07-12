// Command ssl-tournament runs the tournament server (HTTP API + embedded web
// UI). version is set at build time via -ldflags "-X main.version".
package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"github.com/RoboCup-SSL/ssl-tournament-package/internal/datadir"
	"github.com/RoboCup-SSL/ssl-tournament-package/internal/server"
	"github.com/RoboCup-SSL/ssl-tournament-package/internal/store"
)

var version = "dev"

// main parses flags, opens the tournament database, and starts the server.
func main() {
	host := flag.String("host", "0.0.0.0", "The host/interface to bind on")
	port := flag.String("port", "8080", "The port to serve on")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	dir, err := datadir.Resolve()
	if err != nil {
		log.Fatal(err)
	}
	dbPath := filepath.Join(dir, "tournament.db")
	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()
	log.Printf("database: %s", dbPath)

	if err := server.Run(*host, *port, version, st); err != nil {
		log.Fatal(err)
	}
}
