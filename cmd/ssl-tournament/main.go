package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/RoboCup-SSL/ssl-tournament-package/internal/server"
)

// version is stamped at build time via -ldflags "-X main.version=...".
var version = "dev"

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
