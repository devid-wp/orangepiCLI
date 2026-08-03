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

func Validate(slot SlotConfig) []string {
	var errors []string
	for key := range slot.Environment {
		if !environmentKeyPattern.MatchString(key) {
			errors = append(errors, messageInvalidEnvironmentKey)
		}
	}
	if !slot.Enabled {
		return errors
	}
	if strings.TrimSpace(slot.WorkingDirectory) == "" {
		errors = append(errors, messageMissingWorkingDirectory)
	} else if info, err := os.Stat(slot.WorkingDirectory); err != nil || !info.IsDir() {
		errors = append(errors, messageInvalidWorkingDirectory)
	}
	if strings.TrimSpace(slot.StartCommand) == "" {
		errors = append(errors, messageMissingStartCommand)
	}
	if slot.LogFile != "" {
		if info, err := os.Stat(slot.LogFile); err != nil || info.IsDir() {
			errors = append(errors, messageLogFileDoesNotExist)
		}
	}
	if slot.EnvFile != "" {
		info, err := os.Lstat(slot.EnvFile)
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			errors = append(errors, messageEnvFileDoesNotExist)
		} else if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
			errors = append(errors, messageEnvFileUnsafePermissions)
		}
	}
	return errors
}

func LoadAndValidate(name string) (SlotConfig, []string) {
	slot, err := Load(name)
	if err != nil {
		return SlotConfig{}, []string{err.Error()}
	}
	return slot, Validate(slot)
}

func FormatErrors(errors []string) string {
	return fmt.Sprintf("%s", strings.Join(errors, "; "))
}
