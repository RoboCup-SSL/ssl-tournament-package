// Package datadir tests.
package datadir

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestResolveEnvOverride verifies the env var wins and the directory is created.
func TestResolveEnvOverride(t *testing.T) {
	want := filepath.Join(t.TempDir(), "data")
	t.Setenv(EnvVar, want)

	dir, err := Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if dir != want {
		t.Fatalf("dir = %q, want %q", dir, want)
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Fatalf("directory not created: %v", err)
	}
}

// TestResolveDefault verifies the fallback under the user config dir.
func TestResolveDefault(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("XDG_CONFIG_HOME override is linux-specific")
	}
	t.Setenv(EnvVar, "")
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)

	dir, err := Resolve()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(cfg, "ssl-tournament")
	if dir != want {
		t.Fatalf("dir = %q, want %q", dir, want)
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Fatalf("directory not created: %v", err)
	}
}
