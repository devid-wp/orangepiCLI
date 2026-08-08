package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devid-wp/orangepiCLI/internal/config"
)

func TestHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run() code = %d", code)
	}
	if !strings.Contains(stdout.String(), "orangectl validate") {
		t.Fatalf("help output = %q", stdout.String())
	}
}

func TestValidateDisabledSlot(t *testing.T) {
	directory := t.TempDir()
	t.Setenv(config.ConfigDirEnv, directory)
	data := `{"slot":"slot1","enabled":false,"display_name":"Empty Slot 1","environment":{}}`
	if err := os.WriteFile(filepath.Join(directory, "slot1.json"), []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"validate", "slot1"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "disabled") {
		t.Fatalf("output = %q", stdout.String())
	}
}

func TestRejectsUnknownSlot(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"validate", "other"}, &stdout, &stderr); code != 2 {
		t.Fatalf("Run() code = %d", code)
	}
}

func TestInitCreatesConfigsWithoutOverwriting(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv(config.ConfigDirEnv, configDir)
	t.Setenv("ORANGECTL_STATE_DIR", t.TempDir())
	existing := filepath.Join(configDir, "slot1.json")
	if err := os.WriteFile(existing, []byte("keep me\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"init"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "9 created, 1 kept") {
		t.Fatalf("output = %q", stdout.String())
	}
	contents, err := os.ReadFile(existing)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "keep me\n" {
		t.Fatalf("existing config was overwritten: %q", contents)
	}
}

// TestErrorMessagesGoToStderr checks that every error path produces a
// message on stderr (with the "Error: " prefix) and never on stdout. The
// expected exit code distinguishes usage errors (2) from operational
// failures (1).
func TestErrorMessagesGoToStderr(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		exitCode int
	}{
		{name: "unknown command", args: []string{"bogus"}, exitCode: 2},
		{name: "list with extra arg", args: []string{"list", "slot1"}, exitCode: 2},
		{name: "init with extra arg", args: []string{"init", "now"}, exitCode: 2},
		{name: "validate with too many args", args: []string{"validate", "slot1", "slot2"}, exitCode: 2},
		{name: "validate unknown slot", args: []string{"validate", "other"}, exitCode: 2},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(test.args, &stdout, &stderr)
			if code != test.exitCode {
				t.Fatalf("Run() code = %d, want %d (stderr=%q)", code, test.exitCode, stderr.String())
			}
			if !strings.Contains(stderr.String(), "Error: ") {
				t.Fatalf("stderr does not contain 'Error: ' prefix: %q", stderr.String())
			}
			if strings.Contains(stdout.String(), "Error: ") {
				t.Fatalf("stdout should not contain 'Error: ': %q", stdout.String())
			}
		})
	}
}

// TestListReportsLoadErrorsOnStderr verifies that a broken slot
// configuration surfaces its diagnostic on stderr while the table on
// stdout only carries the slot name and a short status.
func TestListReportsLoadErrorsOnStderr(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv(config.ConfigDirEnv, configDir)
	t.Setenv("ORANGECTL_STATE_DIR", t.TempDir())

	broken := filepath.Join(configDir, "slot1.json")
	if err := os.WriteFile(broken, []byte(`{"slot":"slot1"`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"list"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run() code = %d, want 1 (stderr=%q)", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "Error: ") || !strings.Contains(stderr.String(), "slot1") {
		t.Fatalf("stderr should mention slot1 with Error: prefix: %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "slot1") || !strings.Contains(stdout.String(), "error") {
		t.Fatalf("stdout should show slot1 with 'error' status: %q", stdout.String())
	}
	if strings.Contains(stdout.String(), "Error: ") {
		t.Fatalf("stdout must not contain 'Error: ': %q", stdout.String())
	}
}
