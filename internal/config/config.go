package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const ConfigDirEnv = "ORANGECTL_CONFIG_DIR"

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
		return fmt.Errorf("slot %q does not exist; available slots: %s", name, strings.Join(AllowedSlots, ", "))
	}
	return nil
}

func Directory() string {
	if directory := os.Getenv(ConfigDirEnv); directory != "" {
		return directory
	}
	return "configs"
}

func Load(name string) (SlotConfig, error) {
	if err := RequireAllowed(name); err != nil {
		return SlotConfig{}, err
	}

	path := filepath.Join(Directory(), name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return SlotConfig{}, fmt.Errorf("read configuration %s: %w", path, err)
	}

	var slot SlotConfig
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&slot); err != nil {
		return SlotConfig{}, fmt.Errorf("invalid JSON in %s: %w", path, err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return SlotConfig{}, fmt.Errorf("invalid JSON in %s: unexpected data after JSON object", path)
	}
	if slot.Slot != name {
		return SlotConfig{}, fmt.Errorf("field %q in %s must be %q", "slot", path, name)
	}
	if slot.Environment == nil {
		slot.Environment = map[string]string{}
	}
	return slot, nil
}
