package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Edit opens only the selected slot's whitelisted configuration file.
func Edit(name string) error {
	if err := RequireAllowed(name); err != nil {
		return err
	}
	editor := os.Getenv("VISUAL")
	if strings.TrimSpace(editor) == "" {
		editor = os.Getenv("EDITOR")
	}
	if strings.TrimSpace(editor) == "" {
		editor = "nano"
	}
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return fmt.Errorf("editor is empty")
	}
	path := filepath.Join(Directory(), name+".json")
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect configuration: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("configuration must be a regular non-symlink file")
	}
	command := exec.Command(parts[0], append(parts[1:], path)...)
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("run editor: %w", err)
	}
	slot, err := Load(name)
	if err != nil {
		return fmt.Errorf("validate edited configuration: %w", err)
	}
	if validationErrors := Validate(slot); len(validationErrors) > 0 {
		return fmt.Errorf("validate edited configuration: %s", FormatErrors(validationErrors))
	}
	return nil
}
