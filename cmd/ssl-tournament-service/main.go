// Command ssl-tournament-service is the headless (Linux/systemd) artifact.
//
// It contains the full server AND knows how to install itself as a systemd
// service. Run it bare, as root, and it installs *this very binary* as a
// service and starts it — no subcommand to fat-finger, no download step, so
// the version you ran is exactly the version that serves.
//
//	sudo ./ssl-tournament-service            # install + start + enable on boot
//	sudo ./ssl-tournament-service --uninstall            # remove (keeps data)
//	sudo ./ssl-tournament-service --uninstall --purge    # remove + delete data/user
//
// The systemd unit invokes it with the internal --serve flag; humans never do.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/user"

	"github.com/RoboCup-SSL/ssl-tournament-package/internal/server"
)

// version is stamped at build time via -ldflags "-X main.version=...".
var version = "dev"

// The service's identity, derived from one name so nothing can drift. The
// service always binds all interfaces so refs/helpers on the venue Wi-Fi
// reach it from their phones.
const (
	appName  = "ssl-tournament"
	svcUser  = appName
	unitName = appName + ".service"
	unitPath = "/etc/systemd/system/" + unitName
	dataDir  = "/var/lib/" + appName
	binPath  = "/usr/local/bin/ssl-tournament-service"
	bindHost = "0.0.0.0"
)

// unitTemplate is filled with (svcUser, binPath, port, dataDir). StateDirectory
// creates and owns dataDir (the tournament DB lives there from M1 on) and grants
// the service write access to it; the dir survives uninstall unless --purge.
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

# Hardening — the service only serves HTTP and writes its StateDirectory.
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
`

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
		if err := server.Run(bindHost, *port, version); err != nil {
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

func doInstall(port string) error {
	if err := checkSystemd(); err != nil {
		return err
	}

	// Create the locked-down service user if missing.
	if !userExists(svcUser) {
		log.Printf("creating system user %s", svcUser)
		if err := run("useradd", "--system", "--no-create-home",
			"--shell", "/usr/sbin/nologin", svcUser); err != nil {
			return fmt.Errorf("creating user: %w", err)
		}
	}

	// Install this binary at a stable path.
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locating self: %w", err)
	}
	if err := copyFile(self, binPath, 0o755); err != nil {
		return fmt.Errorf("installing binary: %w", err)
	}
	log.Printf("installed %s (%s)", binPath, version)

	// Write the systemd unit.
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

func doUninstall(purge bool) error {
	// Best-effort stop+disable; ignore errors if it was never installed.
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

func requireRoot() {
	if os.Geteuid() != 0 {
		log.Fatal("must run as root (try: sudo ...)")
	}
}

func checkSystemd() error {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return fmt.Errorf("systemd (systemctl) not found — headless install is Linux/systemd only")
	}
	return nil
}

func userExists(name string) bool {
	_, err := user.Lookup(name)
	return err == nil
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %v: %w", name, args, err)
	}
	return nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	if src == dst {
		return nil // already in place
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// Write to a temp file then rename, so we never truncate a running binary.
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
