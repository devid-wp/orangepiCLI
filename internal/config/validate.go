package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var environmentKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

const (
	messageInvalidEnvironmentKey    = "invalid environment key"
	messageMissingWorkingDirectory  = "missing working_directory"
	messageInvalidWorkingDirectory  = "invalid working_directory"
	messageMissingStartCommand      = "missing start_command"
	messageLogFileDoesNotExist      = "log_file does not exist"
	messageEnvFileDoesNotExist      = "env_file does not exist"
	messageEnvFileUnsafePermissions = "env_file has unsafe permissions"
)

type ValidationCode string

const (
	CodeInvalid           ValidationCode = "invalid"
	CodeMissing           ValidationCode = "missing"
	CodeNotFound          ValidationCode = "not_found"
	CodeUnsafePermissions ValidationCode = "unsafe_permissions"
	CodeLoadFailed        ValidationCode = "load_failed"
)

type ValidationError struct {
	Field   string         `json:"field"`
	Code    ValidationCode `json:"code"`
	Message string         `json:"message"`
	Cause   error          `json:"-"`
}

func (validationError ValidationError) Error() string {
	return validationError.Message
}

func (validationError ValidationError) Unwrap() error {
	return validationError.Cause
}

type ValidationErrors []ValidationError

func Validate(slot SlotConfig) ValidationErrors {
	var validationErrors ValidationErrors
	for key := range slot.Environment {
		if !environmentKeyPattern.MatchString(key) {
			validationErrors = append(validationErrors, ValidationError{
				Field: "environment", Code: CodeInvalid, Message: messageInvalidEnvironmentKey,
			})
		}
	}
	if !slot.Enabled {
		return validationErrors
	}
	if strings.TrimSpace(slot.WorkingDirectory) == "" {
		validationErrors = append(validationErrors, ValidationError{
			Field: "working_directory", Code: CodeMissing, Message: messageMissingWorkingDirectory,
		})
	} else if info, err := os.Stat(slot.WorkingDirectory); err != nil || !info.IsDir() {
		validationErrors = append(validationErrors, ValidationError{
			Field: "working_directory", Code: CodeInvalid, Message: messageInvalidWorkingDirectory, Cause: err,
		})
	}
	if strings.TrimSpace(slot.StartCommand) == "" {
		validationErrors = append(validationErrors, ValidationError{
			Field: "start_command", Code: CodeMissing, Message: messageMissingStartCommand,
		})
	}
	if slot.LogFile != "" {
		info, err := os.Lstat(slot.LogFile)
		switch {
		case errors.Is(err, os.ErrNotExist):
			parent := filepath.Dir(slot.LogFile)
			if parentInfo, parentErr := os.Stat(parent); parentErr != nil || !parentInfo.IsDir() {
				validationErrors = append(validationErrors, ValidationError{
					Field: "log_file", Code: CodeNotFound, Message: messageLogFileDoesNotExist, Cause: parentErr,
				})
			}
		case err != nil:
			validationErrors = append(validationErrors, ValidationError{
				Field: "log_file", Code: CodeNotFound, Message: messageLogFileDoesNotExist, Cause: err,
			})
		case info.Mode()&os.ModeSymlink != 0:
			validationErrors = append(validationErrors, ValidationError{
				Field: "log_file", Code: CodeNotFound, Message: messageLogFileDoesNotExist, Cause: err,
			})
		case info.IsDir():
			validationErrors = append(validationErrors, ValidationError{
				Field: "log_file", Code: CodeNotFound, Message: messageLogFileDoesNotExist, Cause: err,
			})
		}
	}
	if slot.EnvFile != "" {
		info, err := os.Lstat(slot.EnvFile)
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			validationErrors = append(validationErrors, ValidationError{
				Field: "env_file", Code: CodeNotFound, Message: messageEnvFileDoesNotExist, Cause: err,
			})
		} else if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
			validationErrors = append(validationErrors, ValidationError{
				Field: "env_file", Code: CodeUnsafePermissions, Message: messageEnvFileUnsafePermissions,
			})
		}
	}
	return validationErrors
}

func LoadAndValidate(name string) (SlotConfig, ValidationErrors) {
	slot, err := Load(name)
	if err != nil {
		return SlotConfig{}, ValidationErrors{{
			Field: "config", Code: CodeLoadFailed, Message: err.Error(), Cause: err,
		}}
	}
	return slot, Validate(slot)
}

func FormatErrors(validationErrors ValidationErrors) string {
	messages := make([]string, 0, len(validationErrors))
	for _, validationError := range validationErrors {
		messages = append(messages, validationError.Error())
	}
	return fmt.Sprintf("%s", strings.Join(messages, "; "))
}

// OpenLogFile opens log_file for appending, creating it with mode 0600 when it
// does not exist. If the path is empty, it returns os.Stdout. The caller is
// responsible for closing the returned file.
func OpenLogFile(path string) (*os.File, error) {
	if path == "" {
		return os.Stdout, nil
	}
	info, err := os.Lstat(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			return nil, fmt.Errorf("create log_file %s: %w", path, err)
		}
		return file, nil
	case err != nil:
		return nil, fmt.Errorf("inspect log_file %s: %w", path, err)
	case info.Mode()&os.ModeSymlink != 0:
		return nil, fmt.Errorf("log_file %s must not be a symbolic link", path)
	case info.IsDir():
		return nil, fmt.Errorf("log_file %s must not be a directory", path)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open log_file %s: %w", path, err)
	}
	if err := file.Chmod(0o600); err != nil && runtime.GOOS != "windows" {
		file.Close()
		return nil, fmt.Errorf("secure log_file %s: %w", path, err)
	}
	return file, nil
}
