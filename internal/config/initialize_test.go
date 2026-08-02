package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/devid-wp/orangepiCLI/internal/paths"
)

func TestInitializeCreatesDirectoriesAndConfigs(t *testing.T) {
	configDir := filepath.Join(t.TempDir(), "config")
	stateDir := filepath.Join(t.TempDir(), "state")
	t.Setenv(ConfigDirEnv, configDir)
	t.Setenv(paths.StateDirEnv, stateDir)

	result, err := Initialize()
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Created) != len(AllowedSlots) || len(result.Existing) != 0 {
		t.Fatalf("Initialize() result = %+v", result)
	}

	for _, directory := range []string{configDir, stateDir, paths.PIDDir(), paths.LogDir()} {
		info, err := os.Stat(directory)
		if err != nil || !info.IsDir() {
			t.Fatalf("directory %q was not created", directory)
		}
	}
	for _, name := range AllowedSlots {
		slot, err := Load(name)
		if err != nil {
			t.Fatalf("Load(%q): %v", name, err)
		}
		if slot.Enabled || slot.Slot != name || slot.Environment == nil {
			t.Fatalf("unexpected empty slot: %+v", slot)
		}
		if runtime.GOOS != "windows" {
			info, err := os.Stat(filepath.Join(configDir, name+".json"))
			if err != nil {
				t.Fatal(err)
			}
			if got := info.Mode().Perm(); got != 0o600 {
				t.Fatalf("config mode = %o, want 600", got)
			}
		}
	}
}

func TestInitializePreservesExistingConfig(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv(ConfigDirEnv, configDir)
	t.Setenv(paths.StateDirEnv, t.TempDir())
	original := []byte("user-owned-content\n")
	path := filepath.Join(configDir, "slot1.json")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := Initialize()
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Created) != len(AllowedSlots)-1 || len(result.Existing) != 1 || result.Existing[0] != "slot1" {
		t.Fatalf("Initialize() result = %+v", result)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != string(original) {
		t.Fatalf("existing config was overwritten: %q", contents)
	}
}
