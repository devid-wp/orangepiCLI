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
	_, err := os.Lstat(path)
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("inspect configuration %s: %w", path, err)
	}

	slotNumber := strings.TrimPrefix(name, "slot")
	slot := SlotConfig{
		Slot:        name,
		DisplayName: "Empty Slot " + slotNumber,
		Environment: map[string]string{},
	}
	if err := writeConfigAtomically(path, slot); err != nil {
		return false, err
	}
	return true, nil
}

func writeConfigAtomically(path string, slot SlotConfig) (err error) {
	file, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary configuration for %s: %w", path, err)
	}
	temporaryPath := file.Name()
	defer func() {
		_ = file.Close()
		if err != nil {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := file.Chmod(0o600); err != nil {
		return fmt.Errorf("secure temporary configuration for %s: %w", path, err)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(slot); err != nil {
		return fmt.Errorf("write configuration %s: %w", path, err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync configuration %s: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close configuration %s: %w", path, err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace configuration %s: %w", path, err)
	}
	return nil
}
