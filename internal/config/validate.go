package config

import (
	"fmt"
	"os"
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
		if info, err := os.Stat(slot.LogFile); err != nil || info.IsDir() {
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
