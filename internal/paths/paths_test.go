package paths

import (
	"path/filepath"
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
