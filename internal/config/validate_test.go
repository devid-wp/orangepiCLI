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

func TestValidateRejectsWhitespaceOnlyRequiredFields(t *testing.T) {
	tests := []struct {
		name  string
		apply func(*SlotConfig)
		want  string
	}{
		{
			name: "working directory",
			apply: func(slot *SlotConfig) {
				slot.WorkingDirectory = " \t\n"
			},
			want: "missing working_directory",
		},
		{
			name: "start command",
			apply: func(slot *SlotConfig) {
				slot.StartCommand = " \t\n"
			},
			want: "missing start_command",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			slot := enabledSlot(t)
			test.apply(&slot)
			if errors := Validate(slot); !slices.Contains(errors, test.want) {
				t.Fatalf("Validate() = %v, want %q", errors, test.want)
			}
		})
	}
}

func TestValidateEnvironmentKeys(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		invalid bool
	}{
		{name: "letters", key: "API_TOKEN"},
		{name: "leading underscore", key: "_PRIVATE"},
		{name: "digits after first character", key: "WORKER_2"},
		{name: "empty", key: "", invalid: true},
		{name: "starts with digit", key: "2_WORKERS", invalid: true},
		{name: "contains equals", key: "TOKEN=value", invalid: true},
		{name: "contains whitespace", key: "API TOKEN", invalid: true},
		{name: "contains hyphen", key: "API-TOKEN", invalid: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			slot := SlotConfig{Environment: map[string]string{test.key: "secret"}}
			errors := Validate(slot)
			gotInvalid := slices.Contains(errors, "invalid environment key")
			if gotInvalid != test.invalid {
				t.Fatalf("Validate() = %v, invalid key = %t, want %t", errors, gotInvalid, test.invalid)
			}
		})
	}
}
