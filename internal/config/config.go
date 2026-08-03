package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/devid-wp/orangepiCLI/internal/paths"
)

const ConfigDirEnv = paths.ConfigDirEnv

var (
	ErrInvalidSlot        = errors.New("invalid slot")
	ErrInvalidConfig      = errors.New("invalid configuration")
	ErrUnsafeConfigFile   = errors.New("unsafe configuration file")
)

var AllowedSlots = []string{
	"slot1", "slot2", "slot3", "slot4", "slot5",
	"slot6", "slot7", "slot8", "slot9", "slot10",
}

type SlotConfig struct {
	Slot             string            `json:"slot"`
	Enabled          bool              `json:"enabled"`
	DisplayName      string            `json:"display_name"`
	Description      string            `json:"description"`
	WorkingDirectory string            `json:"working_directory"`
	StartCommand     string            `json:"start_command"`
	StopCommand      string            `json:"stop_command"`
	RestartCommand   string            `json:"restart_command"`
	LogFile          string            `json:"log_file"`
	UseSudo          bool              `json:"use_sudo"`
	AutoRestart      bool              `json:"auto_restart"`
	Environment      map[string]string `json:"environment"`
	EnvFile          string            `json:"env_file,omitempty"`
}

func IsAllowed(name string) bool {
	for _, slot := range AllowedSlots {
		if name == slot {
			return true
		}
	}
	return false
}

func RequireAllowed(name string) error {
	if !IsAllowed(name) {
		return fmt.Errorf("%w: slot %q does not exist; available slots: %s", ErrInvalidSlot, name, strings.Join(AllowedSlots, ", "))
	}
	return nil
}

func Directory() string {
	return paths.ConfigDir()
}

func Load(name string) (SlotConfig, error) {
	if err := RequireAllowed(name); err != nil {
		return SlotConfig{}, fmt.Errorf("load configuration: %w", err)
	}

	path := filepath.Join(Directory(), name+".json")
	info, err := os.Lstat(path)
	if err != nil {
		return SlotConfig{}, fmt.Errorf("inspect configuration %s: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return SlotConfig{}, fmt.Errorf("%w: configuration %s must be a regular file, not a symbolic link", ErrUnsafeConfigFile, path)
	}
	if !info.Mode().IsRegular() {
		return SlotConfig{}, fmt.Errorf("%w: configuration %s must be a regular file", ErrUnsafeConfigFile, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return SlotConfig{}, fmt.Errorf("read configuration %s: %w", path, err)
	}

	var slot SlotConfig
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&slot); err != nil {
		return SlotConfig{}, fmt.Errorf("%w: invalid JSON in %s: %w", ErrInvalidConfig, path, err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err != nil {
			return SlotConfig{}, fmt.Errorf("%w: invalid JSON in %s: unexpected data after JSON object: %w", ErrInvalidConfig, path, err)
		}
		return SlotConfig{}, fmt.Errorf("%w: invalid JSON in %s: unexpected data after JSON object", ErrInvalidConfig, path)
	}
	if slot.Slot != name {
		return SlotConfig{}, fmt.Errorf("%w: field %q in %s must be %q", ErrInvalidConfig, "slot", path, name)
	}
	if slot.Environment == nil {
		slot.Environment = map[string]string{}
	}
	return slot, nil
}
