package config

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

func Validate(slot SlotConfig) []string {
	var errors []string
	if !slot.Enabled {
		return errors
	}
	if strings.TrimSpace(slot.WorkingDirectory) == "" {
		errors = append(errors, "missing working_directory")
	} else if info, err := os.Stat(slot.WorkingDirectory); err != nil || !info.IsDir() {
		errors = append(errors, "invalid working_directory")
	}
	if strings.TrimSpace(slot.StartCommand) == "" {
		errors = append(errors, "missing start_command")
	}
	if slot.LogFile != "" {
		if info, err := os.Stat(slot.LogFile); err != nil || info.IsDir() {
			errors = append(errors, "log_file does not exist")
		}
	}
	if slot.EnvFile != "" {
		info, err := os.Lstat(slot.EnvFile)
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			errors = append(errors, "env_file does not exist")
		} else if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
			errors = append(errors, "env_file has unsafe permissions")
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
