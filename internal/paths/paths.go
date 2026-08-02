package paths

import (
	"os"
	"path/filepath"
)

const (
	ConfigDirEnv = "ORANGECTL_CONFIG_DIR"
	StateDirEnv  = "ORANGECTL_STATE_DIR"
)

func ConfigDir() string {
	if directory := os.Getenv(ConfigDirEnv); directory != "" {
		return directory
	}
	if base := os.Getenv("XDG_CONFIG_HOME"); base != "" {
		return filepath.Join(base, "orangectl")
	}
	return filepath.Join(homeDir(), ".config", "orangectl")
}

func StateDir() string {
	if directory := os.Getenv(StateDirEnv); directory != "" {
		return directory
	}
	if base := os.Getenv("XDG_STATE_HOME"); base != "" {
		return filepath.Join(base, "orangectl")
	}
	return filepath.Join(homeDir(), ".local", "state", "orangectl")
}

func PIDDir() string {
	return filepath.Join(StateDir(), "pids")
}

func LogDir() string {
	return filepath.Join(StateDir(), "logs")
}

func homeDir() string {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return home
	}
	return "."
}
