package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func containsValidationMessage(validationErrors ValidationErrors, message string) bool {
	for _, validationError := range validationErrors {
		if validationError.Message == message {
			return true
		}
	}
	return false
}

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

	if errors := Validate(slot); !containsValidationMessage(errors, "env_file does not exist") {
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

	if errors := Validate(slot); !containsValidationMessage(errors, "env_file has unsafe permissions") {
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
			if errors := Validate(slot); !containsValidationMessage(errors, test.want) {
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
			gotInvalid := containsValidationMessage(errors, "invalid environment key")
			if gotInvalid != test.invalid {
				t.Fatalf("Validate() = %v, invalid key = %t, want %t", errors, gotInvalid, test.invalid)
			}
		})
	}
}

func TestValidateReturnsStructuredErrors(t *testing.T) {
	slot := enabledSlot(t)
	slot.WorkingDirectory = " \t"
	slot.StartCommand = ""

	validationErrors := Validate(slot)
	if len(validationErrors) != 2 {
		t.Fatalf("Validate() = %+v, want two errors", validationErrors)
	}
	if got := validationErrors[0]; got.Field != "working_directory" || got.Code != CodeMissing {
		t.Fatalf("first validation error = %+v", got)
	}
	if got := validationErrors[1]; got.Field != "start_command" || got.Code != CodeMissing {
		t.Fatalf("second validation error = %+v", got)
	}
	if got := FormatErrors(validationErrors); got != "missing working_directory; missing start_command" {
		t.Fatalf("FormatErrors() = %q", got)
	}
}

func TestValidationErrorsDoNotExposeEnvironmentSecrets(t *testing.T) {
	const secret = "do-not-leak-this-secret"
	slot := enabledSlot(t)
	slot.Environment = map[string]string{
		"INVALID=" + secret: secret,
	}
	slot.EnvFile = filepath.Join(t.TempDir(), "missing.env")

	formatted := FormatErrors(Validate(slot))
	if strings.Contains(formatted, secret) {
		t.Fatalf("validation error exposed environment secret: %q", formatted)
	}
}

func TestValidationErrorsDoNotExposeEnvFileContents(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unsafe Unix permissions are not represented on Windows")
	}
	const secret = "env-file-secret-value"
	slot := enabledSlot(t)
	slot.EnvFile = filepath.Join(t.TempDir(), "secrets.env")
	if err := os.WriteFile(slot.EnvFile, []byte("TOKEN="+secret+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(slot.EnvFile, 0o644); err != nil {
		t.Fatal(err)
	}

	formatted := FormatErrors(Validate(slot))
	if strings.Contains(formatted, secret) {
		t.Fatalf("validation error exposed env_file contents: %q", formatted)
	}
}

func TestValidateLogFileAcceptsExistingRegularFile(t *testing.T) {
	slot := enabledSlot(t)
	slot.LogFile = filepath.Join(t.TempDir(), "slot.log")
	if err := os.WriteFile(slot.LogFile, []byte("ready\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if errors := Validate(slot); containsValidationMessage(errors, "log_file does not exist") {
		t.Fatalf("Validate() = %v, want no log_file error", errors)
	}
}

func TestValidateLogFileAcceptsMissingFileWithExistingParent(t *testing.T) {
	slot := enabledSlot(t)
	slot.LogFile = filepath.Join(t.TempDir(), "slot.log")
	if errors := Validate(slot); containsValidationMessage(errors, "log_file does not exist") {
		t.Fatalf("Validate() = %v, want no log_file error", errors)
	}
}

func TestValidateLogFileRejectsMissingParentDirectory(t *testing.T) {
	slot := enabledSlot(t)
	slot.LogFile = filepath.Join(t.TempDir(), "missing", "logs", "slot.log")
	if errors := Validate(slot); !containsValidationMessage(errors, "log_file does not exist") {
		t.Fatalf("Validate() = %v, want missing log_file error", errors)
	}
}

func TestValidateLogFileRejectsDirectory(t *testing.T) {
	slot := enabledSlot(t)
	slot.LogFile = t.TempDir()
	if errors := Validate(slot); !containsValidationMessage(errors, "log_file does not exist") {
		t.Fatalf("Validate() = %v, want directory rejection", errors)
	}
}

func TestValidateLogFileRejectsSymbolicLink(t *testing.T) {
	slot := enabledSlot(t)
	directory := t.TempDir()
	target := filepath.Join(directory, "real.log")
	if err := os.WriteFile(target, []byte("ready\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(directory, "linked.log")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}
	slot.LogFile = link

	if errors := Validate(slot); !containsValidationMessage(errors, "log_file does not exist") {
		t.Fatalf("Validate() = %v, want symbolic-link rejection", errors)
	}
}

func TestOpenLogFileCreatesFileWithSecurePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose Unix permission bits")
	}
	path := filepath.Join(t.TempDir(), "logs", "slot.log")
	file, err := OpenLogFile(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { file.Close() })

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("log_file mode = %o, want 0o600", perm)
	}
}

func TestOpenLogFileRejectsSymbolicLink(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "real.log")
	if err := os.WriteFile(target, []byte("ready\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(directory, "linked.log")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}

	if _, err := OpenLogFile(link); err == nil {
		t.Fatal("OpenLogFile() returned no error for a symbolic link")
	}
}

func TestOpenLogFileRejectsDirectory(t *testing.T) {
	if _, err := OpenLogFile(t.TempDir()); err == nil {
		t.Fatal("OpenLogFile() returned no error for a directory")
	}
}

func TestOpenLogFileEmptyPathReturnsStdout(t *testing.T) {
	file, err := OpenLogFile("")
	if err != nil {
		t.Fatal(err)
	}
	if file != os.Stdout {
		t.Fatalf("OpenLogFile(\"\") = %v, want os.Stdout", file)
	}
}
