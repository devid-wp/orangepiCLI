package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/devid-wp/orangepiCLI/internal/paths"
)

type InitializeResult struct {
	Created  []string
	Existing []string
}

func Initialize() (InitializeResult, error) {
	var result InitializeResult
	if err := paths.Ensure(); err != nil {
		return result, fmt.Errorf("create application directories: %w", err)
	}

	for _, name := range AllowedSlots {
		created, err := createEmptyConfig(name)
		if err != nil {
			return result, err
		}
		if created {
			result.Created = append(result.Created, name)
		} else {
			result.Existing = append(result.Existing, name)
		}
	}
	return result, nil
}

func createEmptyConfig(name string) (bool, error) {
	path := filepath.Join(Directory(), name+".json")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("create configuration %s: %w", path, err)
	}

	slotNumber := strings.TrimPrefix(name, "slot")
	slot := SlotConfig{
		Slot:        name,
		DisplayName: "Empty Slot " + slotNumber,
		Environment: map[string]string{},
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(slot); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return false, fmt.Errorf("write configuration %s: %w", path, err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return false, fmt.Errorf("close configuration %s: %w", path, err)
	}
	return true, nil
}
