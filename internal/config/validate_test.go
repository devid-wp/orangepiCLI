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

func containsValidationField(validationErrors ValidationErrors, field string) bool {
	for _, validationError := range validationErrors {
		if validationError.Field == field {
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

func TestValidateDisabledSlotRequiresNoFields(t *testing.T) {
	slot := SlotConfig{
		Slot:             "slot1",
		Enabled:          false,
		WorkingDirectory: "",
		StartCommand:     "",
		LogFile:          "/no/such/path/slot.log",
		Environment:      map[string]string{"API_TOKEN": "value"},
	}
	if errors := Validate(slot); len(errors) != 0 {
		t.Fatalf("Validate() = %v, want no errors for disabled slot", errors)
	}
}

func TestValidateDisabledSlotStillReportsEnvironmentKeys(t *testing.T) {
	slot := SlotConfig{
		Enabled:     false,
		Environment: map[string]string{"1INVALID": "value"},
	}
	errors := Validate(slot)
	if len(errors) != 1 || errors[0].Field != "environment" {
		t.Fatalf("Validate() = %v, want single environment error", errors)
	}
}

func TestValidateWorkingDirectoryMissing(t *testing.T) {
	slot := enabledSlot(t)
	slot.WorkingDirectory = ""

	errors := Validate(slot)
	if !containsValidationField(errors, "working_directory") ||
		!containsValidationMessage(errors, "missing working_directory") {
		t.Fatalf("Validate() = %v, want missing working_directory error", errors)
	}
}

func TestValidateWorkingDirectoryNotADirectory(t *testing.T) {
	slot := enabledSlot(t)
	file := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(file, []byte("hi"), 0o600); err != nil {
		t.Fatal(err)
	}
	slot.WorkingDirectory = file

	errors := Validate(slot)
	if !containsValidationMessage(errors, "invalid working_directory") {
		t.Fatalf("Validate() = %v, want invalid working_directory error", errors)
	}
}

func TestValidateStartCommandMissing(t *testing.T) {
	slot := enabledSlot(t)
	slot.StartCommand = ""

	errors := Validate(slot)
	if !containsValidationMessage(errors, "missing start_command") {
		t.Fatalf("Validate() = %v, want missing start_command error", errors)
	}
}

func TestValidateLogFileEmptyIsAllowed(t *testing.T) {
	slot := enabledSlot(t)
	slot.LogFile = ""

	if errors := Validate(slot); containsValidationField(errors, "log_file") {
		t.Fatalf("Validate() = %v, want no log_file error", errors)
	}
}

func TestValidateEnvFileMissing(t *testing.T) {
	slot := enabledSlot(t)
	slot.EnvFile = filepath.Join(t.TempDir(), "missing.env")

	errors := Validate(slot)
	if !containsValidationMessage(errors, "env_file does not exist") {
		t.Fatalf("Validate() = %v, want env_file missing error", errors)
	}
}

func TestValidateEnvFileRejectsDirectory(t *testing.T) {
	slot := enabledSlot(t)
	slot.EnvFile = t.TempDir()

	if errors := Validate(slot); !containsValidationMessage(errors, "env_file does not exist") {
		t.Fatalf("Validate() = %v, want env_file directory rejection", errors)
	}
}

func TestValidateEnvFileAcceptsSafePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose Unix permission bits")
	}
	slot := enabledSlot(t)
	slot.EnvFile = filepath.Join(t.TempDir(), "secrets.env")
	if err := os.WriteFile(slot.EnvFile, []byte("TOKEN=secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(slot.EnvFile, 0o600); err != nil {
		t.Fatal(err)
	}

	if errors := Validate(slot); containsValidationMessage(errors, "env_file has unsafe permissions") {
		t.Fatalf("Validate() = %v, want no unsafe-permissions error", errors)
	}
}

func TestValidationCodesAreStructured(t *testing.T) {
	slot := enabledSlot(t)
	slot.WorkingDirectory = ""
	slot.StartCommand = ""
	slot.Environment = map[string]string{"1INVALID": "value"}
	slot.EnvFile = filepath.Join(t.TempDir(), "missing.env")
	slot.LogFile = filepath.Join(t.TempDir(), "missing", "logs", "slot.log")

	errors := Validate(slot)
	codes := make(map[string]ValidationCode, len(errors))
	for _, validationError := range errors {
		codes[validationError.Field] = validationError.Code
	}

	want := map[string]ValidationCode{
		"environment":       CodeInvalid,
		"working_directory": CodeMissing,
		"start_command":     CodeMissing,
		"env_file":          CodeNotFound,
		"log_file":          CodeNotFound,
	}
	if len(codes) != len(want) {
		t.Fatalf("Validate() = %+v, want %d entries", errors, len(want))
	}
	for field, wantCode := range want {
		got, ok := codes[field]
		if !ok {
			t.Fatalf("Validate() missing field %q in %+v", field, errors)
		}
		if got != wantCode {
			t.Fatalf("codes[%s] = %s, want %s", field, got, wantCode)
		}
	}
}

func TestValidationCodeForEnvFileUnsafePermissions(t *testing.T) {
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

	errors := Validate(slot)
	var found *ValidationError
	for i := range errors {
		if errors[i].Field == "env_file" {
			found = &errors[i]
		}
	}
	if found == nil {
		t.Fatalf("Validate() = %+v, want env_file entry", errors)
	}
	if found.Code != CodeUnsafePermissions {
		t.Fatalf("env_file code = %s, want %s", found.Code, CodeUnsafePermissions)
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

// TestValidateCoversAllRules is a single table that documents every validation
// rule listed in orangCLIPLAN.md stage 3. It does not replace the per-rule
// tests above; it serves as a reviewable index that one expected error (or
// "no error") exists for each category.
func TestValidateCoversAllRules(t *testing.T) {
	type expect struct {
		wantErr  bool
		field    string
		code     ValidationCode
		message  string
	}

	// expected finds the first ValidationError matching field, or asserts that
	// the slice is empty when wantErr is false.
	expected := func(t *testing.T, errors ValidationErrors, want expect) {
		t.Helper()
		if !want.wantErr {
			for _, validationError := range errors {
				if validationError.Field == want.field {
					t.Fatalf("Validate() = %+v, want no error for field %q", errors, want.field)
				}
			}
			return
		}
		for _, validationError := range errors {
			if validationError.Field == want.field {
				if want.code != "" && validationError.Code != want.code {
					t.Fatalf("Validate()[field=%q].Code = %s, want %s", want.field, validationError.Code, want.code)
				}
				if want.message != "" && validationError.Message != want.message {
					t.Fatalf("Validate()[field=%q].Message = %q, want %q", want.field, validationError.Message, want.message)
				}
				return
			}
		}
		t.Fatalf("Validate() = %+v, want error for field %q", errors, want.field)
	}

	t.Run("disabled slot", func(t *testing.T) {
		slot := SlotConfig{
			Slot:             "slot1",
			Enabled:          false,
			WorkingDirectory: "",
			StartCommand:     "",
			LogFile:          "/no/such/path/slot.log",
			EnvFile:          "/no/such/path/secrets.env",
			Environment:      map[string]string{},
		}
		errors := Validate(slot)
		if len(errors) != 0 {
			t.Fatalf("Validate() = %v, want no errors for disabled slot", errors)
		}
	})

	t.Run("required fields", func(t *testing.T) {
		t.Run("working_directory whitespace", func(t *testing.T) {
			slot := enabledSlot(t)
			slot.WorkingDirectory = " \t\n"
			expected(t, Validate(slot), expect{
				wantErr: true,
				field:   "working_directory",
				code:    CodeMissing,
				message: "missing working_directory",
			})
		})
		t.Run("start_command whitespace", func(t *testing.T) {
			slot := enabledSlot(t)
			slot.StartCommand = " \t\n"
			expected(t, Validate(slot), expect{
				wantErr: true,
				field:   "start_command",
				code:    CodeMissing,
				message: "missing start_command",
			})
		})
	})

	t.Run("working_directory", func(t *testing.T) {
		slot := enabledSlot(t)
		file := filepath.Join(t.TempDir(), "not-a-dir")
		if err := os.WriteFile(file, []byte("hi"), 0o600); err != nil {
			t.Fatal(err)
		}
		slot.WorkingDirectory = file
		expected(t, Validate(slot), expect{
			wantErr: true,
			field:   "working_directory",
			code:    CodeInvalid,
			message: "invalid working_directory",
		})
	})

	t.Run("start_command", func(t *testing.T) {
		slot := enabledSlot(t)
		slot.StartCommand = ""
		expected(t, Validate(slot), expect{
			wantErr: true,
			field:   "start_command",
			code:    CodeMissing,
			message: "missing start_command",
		})
	})

	t.Run("log_file", func(t *testing.T) {
		t.Run("regular file accepted", func(t *testing.T) {
			slot := enabledSlot(t)
			slot.LogFile = filepath.Join(t.TempDir(), "slot.log")
			if err := os.WriteFile(slot.LogFile, []byte("ready\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			expected(t, Validate(slot), expect{field: "log_file"})
		})
		t.Run("missing file with existing parent accepted", func(t *testing.T) {
			slot := enabledSlot(t)
			slot.LogFile = filepath.Join(t.TempDir(), "slot.log")
			expected(t, Validate(slot), expect{field: "log_file"})
		})
		t.Run("missing parent directory rejected", func(t *testing.T) {
			slot := enabledSlot(t)
			slot.LogFile = filepath.Join(t.TempDir(), "missing", "logs", "slot.log")
			expected(t, Validate(slot), expect{
				wantErr: true,
				field:   "log_file",
				code:    CodeNotFound,
				message: "log_file does not exist",
			})
		})
		t.Run("directory rejected", func(t *testing.T) {
			slot := enabledSlot(t)
			slot.LogFile = t.TempDir()
			expected(t, Validate(slot), expect{
				wantErr: true,
				field:   "log_file",
				code:    CodeNotFound,
				message: "log_file does not exist",
			})
		})
		t.Run("symbolic link rejected", func(t *testing.T) {
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
			expected(t, Validate(slot), expect{
				wantErr: true,
				field:   "log_file",
				code:    CodeNotFound,
				message: "log_file does not exist",
			})
		})
	})

	t.Run("env_file", func(t *testing.T) {
		t.Run("missing rejected", func(t *testing.T) {
			slot := enabledSlot(t)
			slot.EnvFile = filepath.Join(t.TempDir(), "missing.env")
			expected(t, Validate(slot), expect{
				wantErr: true,
				field:   "env_file",
				code:    CodeNotFound,
				message: "env_file does not exist",
			})
		})
		t.Run("directory rejected", func(t *testing.T) {
			slot := enabledSlot(t)
			slot.EnvFile = t.TempDir()
			expected(t, Validate(slot), expect{
				wantErr: true,
				field:   "env_file",
				code:    CodeNotFound,
				message: "env_file does not exist",
			})
		})
		t.Run("symbolic link rejected", func(t *testing.T) {
			slot := enabledSlot(t)
			target := filepath.Join(t.TempDir(), "secrets.env")
			if err := os.WriteFile(target, []byte("TOKEN=secret\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			link := filepath.Join(t.TempDir(), "linked.env")
			if err := os.Symlink(target, link); err != nil {
				t.Skipf("symbolic links are unavailable: %v", err)
			}
			slot.EnvFile = link
			expected(t, Validate(slot), expect{
				wantErr: true,
				field:   "env_file",
				code:    CodeNotFound,
				message: "env_file does not exist",
			})
		})
		t.Run("unsafe permissions rejected", func(t *testing.T) {
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
			expected(t, Validate(slot), expect{
				wantErr: true,
				field:   "env_file",
				code:    CodeUnsafePermissions,
				message: "env_file has unsafe permissions",
			})
		})
		t.Run("safe permissions accepted", func(t *testing.T) {
			if runtime.GOOS == "windows" {
				t.Skip("Windows does not expose Unix permission bits")
			}
			slot := enabledSlot(t)
			slot.EnvFile = filepath.Join(t.TempDir(), "secrets.env")
			if err := os.WriteFile(slot.EnvFile, []byte("TOKEN=secret\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(slot.EnvFile, 0o600); err != nil {
				t.Fatal(err)
			}
			expected(t, Validate(slot), expect{field: "env_file"})
		})
	})

	t.Run("environment keys", func(t *testing.T) {
		t.Run("invalid key reported", func(t *testing.T) {
			slot := SlotConfig{
				Environment: map[string]string{"1INVALID": "value"},
			}
			expected(t, Validate(slot), expect{
				wantErr: true,
				field:   "environment",
				code:    CodeInvalid,
				message: "invalid environment key",
			})
		})
		t.Run("valid keys accepted", func(t *testing.T) {
			slot := SlotConfig{
				Environment: map[string]string{"API_TOKEN": "value", "_PRIVATE": "x", "WORKER_2": "y"},
			}
			if errors := Validate(slot); len(errors) != 0 {
				t.Fatalf("Validate() = %v, want no errors for valid keys", errors)
			}
		})
	})

	t.Run("structured codes", func(t *testing.T) {
		slot := enabledSlot(t)
		slot.WorkingDirectory = " \t"
		slot.StartCommand = ""
		slot.Environment = map[string]string{"1INVALID": "x"}
		slot.LogFile = filepath.Join(t.TempDir(), "missing", "logs", "slot.log")
		slot.EnvFile = filepath.Join(t.TempDir(), "missing.env")

		errors := Validate(slot)
		codes := make(map[string]ValidationCode, len(errors))
		for _, validationError := range errors {
			codes[validationError.Field] = validationError.Code
		}
		want := map[string]ValidationCode{
			"environment":       CodeInvalid,
			"working_directory": CodeMissing,
			"start_command":     CodeMissing,
			"log_file":          CodeNotFound,
			"env_file":          CodeNotFound,
		}
		if len(codes) != len(want) {
			t.Fatalf("Validate() = %+v, want %d entries", errors, len(want))
		}
		for field, wantCode := range want {
			got, ok := codes[field]
			if !ok {
				t.Fatalf("Validate() missing field %q in %+v", field, errors)
			}
			if got != wantCode {
				t.Fatalf("codes[%s] = %s, want %s", field, got, wantCode)
			}
		}
	})
}
