// Package datadir resolves the per-user directory where the tournament
// database and assets live — never next to the binary.
package datadir

import (
	"os"
	"path/filepath"
)

// EnvVar overrides the data directory when set. The systemd unit installed by
// ssl-tournament-service sets it to /var/lib/ssl-tournament.
const EnvVar = "SSL_TOURNAMENT_DATA_DIR"

// Resolve returns the data directory, creating it if needed: $SSL_TOURNAMENT_DATA_DIR
// when non-empty, otherwise <os.UserConfigDir()>/ssl-tournament.
func Resolve() (string, error) {
	dir := os.Getenv(EnvVar)
	if dir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(base, "ssl-tournament")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}
