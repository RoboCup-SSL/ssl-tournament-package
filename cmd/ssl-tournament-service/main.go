// Command ssl-tournament-service is the headless (Linux/systemd) artifact: run
// it as root and it installs this binary as a systemd service that starts on
// boot. The systemd unit invokes it with --serve; humans never do. version is
// set at build time via -ldflags "-X main.version".
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

	"github.com/RoboCup-SSL/ssl-tournament-package/internal/datadir"
	"github.com/RoboCup-SSL/ssl-tournament-package/internal/server"
	"github.com/RoboCup-SSL/ssl-tournament-package/internal/store"
)

var version = "dev"

const (
	appName  = "ssl-tournament"
	svcUser  = appName
	unitName = appName + ".service"
	unitPath = "/etc/systemd/system/" + unitName
	dataDir  = "/var/lib/" + appName
	binPath  = "/usr/local/bin/ssl-tournament-service"
	bindHost = "0.0.0.0"
)

// unitTemplate is rendered with (svcUser, binPath, port, dataDir). StateDirectory
// creates dataDir, grants the service write access to it, and preserves it across
// uninstall unless --purge.
const unitTemplate = `[Unit]
Description=SSL Tournament Package
Documentation=https://github.com/RoboCup-SSL/ssl-tournament-package
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=%[1]s
Group=%[1]s
ExecStart=%[2]s --serve --port %[3]s
Restart=always
RestartSec=2
StateDirectory=%[1]s
Environment=SSL_TOURNAMENT_DATA_DIR=%[4]s
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
`

// main dispatches between serve, uninstall, and the default install action.
func main() {
	serve := flag.Bool("serve", false, "Run the server (used by systemd; not for manual use)")
	uninstall := flag.Bool("uninstall", false, "Remove the systemd service")
	purge := flag.Bool("purge", false, "With --uninstall, also delete the data dir and service user")
	showVersion := flag.Bool("version", false, "Print version and exit")
	port := flag.String("port", "8080", "Port to serve on")
	flag.Parse()

	switch {
	case *showVersion:
		fmt.Println(version)
	case *serve:
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
		if err := server.Run(bindHost, *port, version, st); err != nil {
			log.Fatal(err)
		}
	case *uninstall:
		requireRoot()
		if err := doUninstall(*purge); err != nil {
			log.Fatal(err)
		}
	default:
		requireRoot()
		if err := doInstall(*port); err != nil {
			log.Fatal(err)
		}
	}
}

// doInstall creates the service user, installs this binary, writes the systemd
// unit, and starts the service enabled on boot.
func doInstall(port string) error {
	if err := checkSystemd(); err != nil {
		return err
	}

	if !userExists(svcUser) {
		log.Printf("creating system user %s", svcUser)
		if err := run("useradd", "--system", "--no-create-home",
			"--shell", "/usr/sbin/nologin", svcUser); err != nil {
			return fmt.Errorf("creating user: %w", err)
		}
	}

	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locating self: %w", err)
	}
	if err := copyFile(self, binPath, 0o755); err != nil {
		return fmt.Errorf("installing binary: %w", err)
	}
	log.Printf("installed %s (%s)", binPath, version)

	unit := fmt.Sprintf(unitTemplate, svcUser, binPath, port, dataDir)
	if err := os.WriteFile(unitPath, []byte(unit), 0o644); err != nil {
		return fmt.Errorf("writing unit: %w", err)
	}

	if err := run("systemctl", "daemon-reload"); err != nil {
		return err
	}
	if err := run("systemctl", "enable", "--now", unitName); err != nil {
		return err
	}

	log.Print("done. ssl-tournament is running and will start on boot.")
	log.Printf("  status:  systemctl status %s", unitName)
	log.Printf("  logs:    journalctl -u %s -f", unitName)
	log.Printf("  serving on port %s", port)
	return nil
}

// doUninstall stops and removes the service and binary. With purge it also
// deletes the data dir and service user; otherwise the data is kept.
func doUninstall(purge bool) error {
	_ = run("systemctl", "disable", "--now", unitName)
	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing unit: %w", err)
	}
	_ = run("systemctl", "daemon-reload")

	if err := os.Remove(binPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing binary: %w", err)
	}

	if purge {
		log.Print("purging data dir and service user")
		if err := os.RemoveAll(dataDir); err != nil {
			return fmt.Errorf("removing data dir: %w", err)
		}
		if userExists(svcUser) {
			_ = run("userdel", svcUser)
		}
		log.Print("done. everything removed.")
		return nil
	}

	log.Print("done. service and binary removed.")
	log.Printf("data kept at %s (re-run with --uninstall --purge to delete it).", dataDir)
	return nil
}

// requireRoot exits the process unless it is running as root.
func requireRoot() {
	if os.Geteuid() != 0 {
		log.Fatal("must run as root (try: sudo ...)")
	}
}

// checkSystemd errors unless systemctl is available on PATH.
func checkSystemd() error {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return fmt.Errorf("systemd (systemctl) not found — headless install is Linux/systemd only")
	}
	return nil
}

// userExists reports whether a system user with the given name exists.
func userExists(name string) bool {
	_, err := user.Lookup(name)
	return err == nil
}

// run executes a command, forwarding its output to this process's stdout/stderr.
func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %v: %w", name, args, err)
	}
	return nil
}

// copyFile copies src to dst with the given mode via a temp file and rename, so
// a currently-running binary at dst is never truncated.
func copyFile(src, dst string, mode os.FileMode) error {
	if src == dst {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}
