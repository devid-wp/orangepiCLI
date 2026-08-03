package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestExplicitDirectoriesTakePriority(t *testing.T) {
	configDir := filepath.Join(t.TempDir(), "config")
	stateDir := filepath.Join(t.TempDir(), "state")
	t.Setenv(ConfigDirEnv, configDir)
	t.Setenv(StateDirEnv, stateDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg-config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(t.TempDir(), "xdg-state"))

	if got := ConfigDir(); got != configDir {
		t.Fatalf("ConfigDir() = %q, want %q", got, configDir)
	}
	if got := StateDir(); got != stateDir {
		t.Fatalf("StateDir() = %q, want %q", got, stateDir)
	}
}

func TestEnsureSecuresApplicationDirectories(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose Unix permission bits")
	}

	configDir := filepath.Join(t.TempDir(), "config")
	stateDir := filepath.Join(t.TempDir(), "state")
	t.Setenv(ConfigDirEnv, configDir)
	t.Setenv(StateDirEnv, stateDir)
	for _, directory := range []string{configDir, stateDir} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	if err := Ensure(); err != nil {
		t.Fatal(err)
	}
	for _, directory := range []string{configDir, stateDir, PIDDir(), LogDir()} {
		info, err := os.Stat(directory)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o700 {
			t.Fatalf("directory %q mode = %o, want 700", directory, got)
		}
	}
}

func TestXDGDirectories(t *testing.T) {
	configBase := filepath.Join(t.TempDir(), "xdg-config")
	stateBase := filepath.Join(t.TempDir(), "xdg-state")
	t.Setenv(ConfigDirEnv, "")
	t.Setenv(StateDirEnv, "")
	t.Setenv("XDG_CONFIG_HOME", configBase)
	t.Setenv("XDG_STATE_HOME", stateBase)

	if got, want := ConfigDir(), filepath.Join(configBase, "orangectl"); got != want {
		t.Fatalf("ConfigDir() = %q, want %q", got, want)
	}
	if got, want := StateDir(), filepath.Join(stateBase, "orangectl"); got != want {
		t.Fatalf("StateDir() = %q, want %q", got, want)
	}
	if got, want := PIDDir(), filepath.Join(stateBase, "orangectl", "pids"); got != want {
		t.Fatalf("PIDDir() = %q, want %q", got, want)
	}
	if got, want := LogDir(), filepath.Join(stateBase, "orangectl", "logs"); got != want {
		t.Fatalf("LogDir() = %q, want %q", got, want)
	}
}
