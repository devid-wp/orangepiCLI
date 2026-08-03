package config

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

func enabledSlot(t *testing.T) SlotConfig {
	t.Helper()
	return SlotConfig{
		Slot:             "slot1",
		Enabled:          true,
		WorkingDirectory: t.TempDir(),
		StartCommand:     "run",
		Environment:      map[string]string{},
	}
}

func TestValidateEnvFileRejectsSymbolicLink(t *testing.T) {
	slot := enabledSlot(t)
	target := filepath.Join(t.TempDir(), "secrets.env")
	if err := os.WriteFile(target, []byte("TOKEN=secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	slot.EnvFile = filepath.Join(t.TempDir(), "linked.env")
	if err := os.Symlink(target, slot.EnvFile); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}

	if errors := Validate(slot); !slices.Contains(errors, "env_file does not exist") {
		t.Fatalf("Validate() = %v, want unsafe env_file error", errors)
	}
}

func TestValidateEnvFileRejectsUnsafePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose Unix permission bits")
	}
	slot := enabledSlot(t)
	slot.EnvFile = filepath.Join(t.TempDir(), "secrets.env")
	if err := os.WriteFile(slot.EnvFile, []byte("TOKEN=secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(slot.EnvFile, 0o644); err != nil {
		t.Fatal(err)
	}

	if errors := Validate(slot); !slices.Contains(errors, "env_file has unsafe permissions") {
		t.Fatalf("Validate() = %v, want unsafe-permissions error", errors)
	}
}
