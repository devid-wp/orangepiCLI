package config

import (
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
	if err := RequireAllowed("../slot1"); err == nil {
		t.Fatal("expected path-like slot name to be rejected")
	}
}
