package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Backup creates a private timestamped copy of one whitelisted config.
func Backup(name string) (string, error) {
	if err := RequireAllowed(name); err != nil {
		return "", err
	}
	source := filepath.Join(Directory(), name+".json")
	data, err := os.ReadFile(source)
	if err != nil {
		return "", fmt.Errorf("read configuration backup source: %w", err)
	}
	directory := filepath.Join(Directory(), "backups")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(directory, fmt.Sprintf("%s-%s.json", name, time.Now().UTC().Format("20060102T150405.000000000Z")))
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("write configuration backup: %w", err)
	}
	return path, nil
}
