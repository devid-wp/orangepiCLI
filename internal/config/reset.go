package config

import (
	"fmt"
	"strings"
)

// Reset replaces one allowed configuration with an empty disabled template,
// preserving a timestamped backup first.
func Reset(name string) (string, error) {
	if err := RequireAllowed(name); err != nil {
		return "", err
	}
	backup, err := Backup(name)
	if err != nil {
		return "", err
	}
	slotNumber := strings.TrimPrefix(name, "slot")
	slot := SlotConfig{Slot: name, DisplayName: "Empty Slot " + slotNumber, Environment: map[string]string{}}
	if err := writeConfigAtomically(Directory()+"/"+name+".json", slot); err != nil {
		return "", fmt.Errorf("reset configuration: %w", err)
	}
	return backup, nil
}
