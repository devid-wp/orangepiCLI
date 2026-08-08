package cli

import (
	"bytes"
	"encoding/json"
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

func TestRunRejectsUnknownGlobalFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "flag before command", args: []string{"--xml", "list"}},
		{name: "flag after command", args: []string{"list", "--xml"}},
		{name: "flag alone", args: []string{"--xml"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(test.args, &stdout, &stderr)
			if code != 2 {
				t.Fatalf("Run() code = %d, want 2 (stderr=%q)", code, stderr.String())
			}
			if !strings.Contains(stderr.String(), "unknown global flag") {
				t.Fatalf("stderr should mention unknown global flag: %q", stderr.String())
			}
		})
	}
}

func TestRunJSONFlagInAnyPosition(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "before command", args: []string{"--json", "help"}},
		{name: "after command", args: []string{"help", "--json"}},
		{name: "alone", args: []string{"--json", "help"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(test.args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("Run() code = %d, stderr=%q", code, stderr.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr should be empty in JSON mode: %q", stderr.String())
			}
			var probe jsonHelpResult
			if err := json.Unmarshal(stdout.Bytes(), &probe); err != nil {
				t.Fatalf("stdout is not valid JSON: %v (raw=%q)", err, stdout.String())
			}
		})
	}
}

func TestJSONInitShape(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv(config.ConfigDirEnv, configDir)
	t.Setenv("ORANGECTL_STATE_DIR", t.TempDir())

	existing := filepath.Join(configDir, "slot1.json")
	if err := os.WriteFile(existing, []byte("keep me\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--json", "init"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr=%q", code, stderr.String())
	}

	var got jsonInitResult
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("init --json output is not valid JSON: %v (raw=%q)", err, stdout.String())
	}
	if got.ConfigDir != configDir {
		t.Fatalf("config_dir = %q, want %q", got.ConfigDir, configDir)
	}
	if len(got.Existing) != 1 || got.Existing[0] != "slot1" {
		t.Fatalf("existing = %v, want [slot1]", got.Existing)
	}
	if len(got.Created) != len(config.AllowedSlots)-1 {
		t.Fatalf("created = %v, want %d entries", got.Created, len(config.AllowedSlots)-1)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr should be empty in JSON mode: %q", stderr.String())
	}
}

func TestJSONListShape(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv(config.ConfigDirEnv, configDir)
	t.Setenv("ORANGECTL_STATE_DIR", t.TempDir())

	broken := filepath.Join(configDir, "slot1.json")
	if err := os.WriteFile(broken, []byte(`{"slot":"slot1"`), 0o600); err != nil {
		t.Fatal(err)
	}
	healthy := filepath.Join(configDir, "slot2.json")
	if err := os.WriteFile(healthy, []byte(`{"slot":"slot2","enabled":false,"display_name":"Empty Slot 2","environment":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"list", "--json"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run() code = %d, want 1 (stderr=%q)", code, stderr.String())
	}

	var got jsonListResult
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("list --json output is not valid JSON: %v (raw=%q)", err, stdout.String())
	}
	if len(got.Slots) != len(config.AllowedSlots) {
		t.Fatalf("slots length = %d, want %d", len(got.Slots), len(config.AllowedSlots))
	}
	if len(got.Errors) != 1 || got.Errors[0].Slot != "slot1" {
		t.Fatalf("errors = %+v, want one slot1 error", got.Errors)
	}
	if stderr.Len() != 0 {
		t.Fatalf("errors should be in JSON, not stderr: %q", stderr.String())
	}
}

func TestJSONListAllHealthy(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv(config.ConfigDirEnv, configDir)
	t.Setenv("ORANGECTL_STATE_DIR", t.TempDir())

	if _, err := config.Initialize(); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--json", "list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr=%q", code, stderr.String())
	}
	var got jsonListResult
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("list --json output is not valid JSON: %v", err)
	}
	if len(got.Errors) != 0 {
		t.Fatalf("expected no errors, got %+v", got.Errors)
	}
	if len(got.Slots) != len(config.AllowedSlots) {
		t.Fatalf("slots length = %d, want %d", len(got.Slots), len(config.AllowedSlots))
	}
}

func TestJSONValidateShape(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv(config.ConfigDirEnv, configDir)
	t.Setenv("ORANGECTL_STATE_DIR", t.TempDir())

	// Healthy disabled slot.
	healthy := filepath.Join(configDir, "slot1.json")
	if err := os.WriteFile(healthy, []byte(`{"slot":"slot1","enabled":false,"display_name":"Empty Slot 1","environment":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	// Broken slot: missing working_directory in an enabled context.
	broken := filepath.Join(configDir, "slot2.json")
	if err := os.WriteFile(broken, []byte(`{"slot":"slot2","enabled":true,"environment":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"validate", "--json"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run() code = %d, want 1 (stderr=%q)", code, stderr.String())
	}

	var got jsonValidateResult
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("validate --json output is not valid JSON: %v (raw=%q)", err, stdout.String())
	}
	if len(got.Results) != 2 {
		t.Fatalf("results length = %d, want 2", len(got.Results))
	}
	results := map[string]jsonValidateSlot{}
	for _, entry := range got.Results {
		results[entry.Slot] = entry
	}
	if entry := results["slot1"]; entry.Result != "disabled" {
		t.Fatalf("slot1 result = %q, want disabled", entry.Result)
	}
	if entry := results["slot2"]; entry.Result != "errors" {
		t.Fatalf("slot2 result = %q, want errors", entry.Result)
	}
	foundField := false
	for _, e := range results["slot2"].Errors {
		if e.Field == "working_directory" && e.Code == "missing" {
			foundField = true
			break
		}
	}
	if !foundField {
		t.Fatalf("slot2 errors do not mention working_directory/missing: %+v", results["slot2"].Errors)
	}
}

func TestJSONHelpShape(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"help", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr=%q", code, stderr.String())
	}

	var got jsonHelpResult
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("help --json output is not valid JSON: %v (raw=%q)", err, stdout.String())
	}
	names := make(map[string]bool, len(got.Commands))
	for _, c := range got.Commands {
		names[c.Name] = true
	}
	for _, want := range []string{"init", "list", "validate", "help"} {
		if !names[want] {
			t.Fatalf("commands missing %q in %+v", want, got.Commands)
		}
	}
	if !strings.Contains(got.Usage, "orangectl validate") {
		t.Fatalf("usage missing 'orangectl validate': %q", got.Usage)
	}
}
