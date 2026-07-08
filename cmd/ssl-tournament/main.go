// Command ssl-tournament runs the tournament server (HTTP API + embedded web
// UI). version is set at build time via -ldflags "-X main.version".
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/RoboCup-SSL/ssl-tournament-package/internal/server"
)

var version = "dev"

// main parses flags and starts the server.
func main() {
	host := flag.String("host", "0.0.0.0", "The host/interface to bind on")
	port := flag.String("port", "8080", "The port to serve on")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	if err := server.Run(*host, *port, version); err != nil {
		log.Fatal(err)
	}
}
