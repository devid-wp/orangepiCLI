package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, name, body string) {
	t.Helper()
	directory := t.TempDir()
	t.Setenv(ConfigDirEnv, directory)
	if err := os.WriteFile(filepath.Join(directory, name+".json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadCompleteSchema(t *testing.T) {
	writeConfig(t, "slot1", `{
		"slot":"slot1","enabled":false,"display_name":"Empty Slot 1",
		"description":"","working_directory":"","start_command":"",
		"stop_command":"","restart_command":"","log_file":"",
		"use_sudo":false,"auto_restart":false,"environment":{},"env_file":""
	}`)

	slot, err := Load("slot1")
	if err != nil {
		t.Fatal(err)
	}
	if slot.Slot != "slot1" || slot.DisplayName != "Empty Slot 1" {
		t.Fatalf("unexpected config: %+v", slot)
	}
}

func TestLoadRejectsUnknownField(t *testing.T) {
	writeConfig(t, "slot1", `{"slot":"slot1","unknown":true}`)
	_, err := Load("slot1")
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown-field error, got %v", err)
	}
}

func TestLoadRejectsMismatchedSlot(t *testing.T) {
	writeConfig(t, "slot1", `{"slot":"slot2"}`)
	_, err := Load("slot1")
	if err == nil || !strings.Contains(err.Error(), "must be") {
		t.Fatalf("expected slot mismatch error, got %v", err)
	}
}

func TestRequireAllowedRejectsPath(t *testing.T) {
	if err := RequireAllowed("../slot1"); !errors.Is(err, ErrInvalidSlot) {
		t.Fatalf("expected invalid-slot error, got %v", err)
	}
}

func TestLoadRejectsDirectory(t *testing.T) {
	directory := t.TempDir()
	t.Setenv(ConfigDirEnv, directory)
	if err := os.Mkdir(filepath.Join(directory, "slot1.json"), 0o700); err != nil {
		t.Fatal(err)
	}

	_, err := Load("slot1")
	if !errors.Is(err, ErrUnsafeConfigFile) || !strings.Contains(err.Error(), "must be a regular file") {
		t.Fatalf("expected non-regular-file error, got %v", err)
	}
}

func TestLoadRejectsSymbolicLink(t *testing.T) {
	directory := t.TempDir()
	t.Setenv(ConfigDirEnv, directory)
	target := filepath.Join(directory, "target.json")
	if err := os.WriteFile(target, []byte(`{"slot":"slot1"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(directory, "slot1.json")); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}

	_, err := Load("slot1")
	if !errors.Is(err, ErrUnsafeConfigFile) || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("expected symbolic-link error, got %v", err)
	}
}

func TestLoadRejectsBrokenJSON(t *testing.T) {
	writeConfig(t, "slot1", `{"slot":"slot1"`)

	_, err := Load("slot1")
	if err == nil {
		t.Fatal("Load() returned no error for broken JSON")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("Load() error = %v, want ErrInvalidConfig", err)
	}
	var syntaxError *json.SyntaxError
	if !errors.As(err, &syntaxError) {
		t.Fatalf("Load() error = %v, want wrapped json.SyntaxError", err)
	}
}

func TestLoadRejectsTrailingData(t *testing.T) {
	writeConfig(t, "slot1", `{"slot":"slot1"}{"slot":"slot2"}`)

	_, err := Load("slot1")
	if err == nil {
		t.Fatal("Load() returned no error for trailing JSON data")
	}
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("Load() error = %v, want ErrInvalidConfig", err)
	}
	if !strings.Contains(err.Error(), "unexpected data after JSON object") {
		t.Fatalf("Load() error = %v, want trailing-data message", err)
	}
}

func TestLoadMissingConfig(t *testing.T) {
	directory := t.TempDir()
	t.Setenv(ConfigDirEnv, directory)

	_, err := Load("slot1")
	if err == nil {
		t.Fatal("Load() returned no error for missing configuration")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Load() error = %v, want os.ErrNotExist", err)
	}
	if errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("Load() error = %v, must not be classified as ErrInvalidConfig", err)
	}
}

// TestLoadErrorClassification enumerates every failure path of Load and
// asserts the expected sentinel via errors.Is. Each subtest is independent
// and uses its own temporary configuration directory.
func TestLoadErrorClassification(t *testing.T) {
	t.Run("path traversal", func(t *testing.T) {
		err := RequireAllowed("../slot1")
		if !errors.Is(err, ErrInvalidSlot) {
			t.Fatalf("RequireAllowed() = %v, want ErrInvalidSlot", err)
		}
	})

	t.Run("broken JSON", func(t *testing.T) {
		writeConfig(t, "slot1", `{`)
		_, err := Load("slot1")
		if !errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("Load() error = %v, want ErrInvalidConfig", err)
		}
	})

	t.Run("trailing data", func(t *testing.T) {
		writeConfig(t, "slot1", `{"slot":"slot1"}{"slot":"slot2"}`)
		_, err := Load("slot1")
		if !errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("Load() error = %v, want ErrInvalidConfig", err)
		}
	})

	t.Run("slot mismatch", func(t *testing.T) {
		writeConfig(t, "slot1", `{"slot":"slot2"}`)
		_, err := Load("slot1")
		if !errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("Load() error = %v, want ErrInvalidConfig", err)
		}
	})

	t.Run("directory", func(t *testing.T) {
		directory := t.TempDir()
		t.Setenv(ConfigDirEnv, directory)
		if err := os.Mkdir(filepath.Join(directory, "slot1.json"), 0o700); err != nil {
			t.Fatal(err)
		}
		_, err := Load("slot1")
		if !errors.Is(err, ErrUnsafeConfigFile) {
			t.Fatalf("Load() error = %v, want ErrUnsafeConfigFile", err)
		}
	})

	t.Run("symbolic link", func(t *testing.T) {
		directory := t.TempDir()
		t.Setenv(ConfigDirEnv, directory)
		target := filepath.Join(directory, "target.json")
		if err := os.WriteFile(target, []byte(`{"slot":"slot1"}`), 0o600); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(directory, "slot1.json")
		if err := os.Symlink(target, link); err != nil {
			t.Skipf("symbolic links are unavailable: %v", err)
		}
		_, err := Load("slot1")
		if !errors.Is(err, ErrUnsafeConfigFile) {
			t.Fatalf("Load() error = %v, want ErrUnsafeConfigFile", err)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		directory := t.TempDir()
		t.Setenv(ConfigDirEnv, directory)
		_, err := Load("slot1")
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("Load() error = %v, want os.ErrNotExist", err)
		}
		if errors.Is(err, ErrInvalidConfig) {
			t.Fatalf("Load() error = %v, must not be classified as ErrInvalidConfig", err)
		}
	})
}
