package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/devid-wp/orangepiCLI/internal/paths"
)

func TestInitializeCreatesDirectoriesAndConfigs(t *testing.T) {
	configDir := filepath.Join(t.TempDir(), "config")
	stateDir := filepath.Join(t.TempDir(), "state")
	t.Setenv(ConfigDirEnv, configDir)
	t.Setenv(paths.StateDirEnv, stateDir)

	result, err := Initialize()
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Created) != len(AllowedSlots) || len(result.Existing) != 0 {
		t.Fatalf("Initialize() result = %+v", result)
	}

	for _, directory := range []string{configDir, stateDir, paths.PIDDir(), paths.LogDir()} {
		info, err := os.Stat(directory)
		if err != nil || !info.IsDir() {
			t.Fatalf("directory %q was not created", directory)
		}
	}
	for _, name := range AllowedSlots {
		slot, err := Load(name)
		if err != nil {
			t.Fatalf("Load(%q): %v", name, err)
		}
		if slot.Enabled || slot.Slot != name || slot.Environment == nil {
			t.Fatalf("unexpected empty slot: %+v", slot)
		}
		if runtime.GOOS != "windows" {
			info, err := os.Stat(filepath.Join(configDir, name+".json"))
			if err != nil {
				t.Fatal(err)
			}
			if got := info.Mode().Perm(); got != 0o600 {
				t.Fatalf("config mode = %o, want 600", got)
			}
		}
	}
	temporaryFiles, err := filepath.Glob(filepath.Join(configDir, ".*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(temporaryFiles) != 0 {
		t.Fatalf("temporary configuration files remain: %v", temporaryFiles)
	}
}

func TestInitializePreservesExistingConfig(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv(ConfigDirEnv, configDir)
	t.Setenv(paths.StateDirEnv, t.TempDir())
	original := []byte("user-owned-content\n")
	path := filepath.Join(configDir, "slot1.json")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := Initialize()
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Created) != len(AllowedSlots)-1 || len(result.Existing) != 1 || result.Existing[0] != "slot1" {
		t.Fatalf("Initialize() result = %+v", result)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != string(original) {
		t.Fatalf("existing config was overwritten: %q", contents)
	}
}

// requireConfigDir returns a fresh config directory and points the package
// at it via ORANGECTL_CONFIG_DIR. Tests that exercise writeConfigAtomically
// directly use this helper to keep their setup in one line.
func requireConfigDir(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	t.Setenv(ConfigDirEnv, directory)
	return directory
}

// listTemporaryConfigs returns every leftover `*.tmp` file in directory.
func listTemporaryConfigs(t *testing.T, directory string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(directory, ".*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	return matches
}

func TestWriteConfigAtomicallyCreatesTempInTargetDirectory(t *testing.T) {
	configDir := requireConfigDir(t)
	path := filepath.Join(configDir, "slot1.json")

	if err := writeConfigAtomically(path, SlotConfig{
		Slot:        "slot1",
		DisplayName: "Empty Slot 1",
		Environment: map[string]string{},
	}); err != nil {
		t.Fatal(err)
	}

	// Temporary file must have been in the target directory and must have
	// been renamed away by the time we observe the directory.
	if leftover := listTemporaryConfigs(t, configDir); len(leftover) != 0 {
		t.Fatalf("temporary files remain after atomic write: %v", leftover)
	}

	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("Lstat(%s): %v", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("config file is a symbolic link: %s", path)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("config file is not a regular file: mode=%s", info.Mode())
	}
	if filepath.Dir(path) != configDir {
		t.Fatalf("writeConfigAtomically wrote outside target directory: dir=%s", filepath.Dir(path))
	}
}

func TestWriteConfigAtomicallySetsSecurePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not expose Unix permission bits")
	}
	configDir := requireConfigDir(t)
	path := filepath.Join(configDir, "slot1.json")

	if err := writeConfigAtomically(path, SlotConfig{
		Slot:        "slot1",
		DisplayName: "Empty Slot 1",
		Environment: map[string]string{},
	}); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("config mode = %o, want 0o600", got)
	}
}

func TestWriteConfigAtomicallyRemainsAtomicOnRenameFailure(t *testing.T) {
	configDir := requireConfigDir(t)
	path := filepath.Join(configDir, "slot1.json")

	// Force os.Rename to fail by placing a directory at the target path.
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}

	err := writeConfigAtomically(path, SlotConfig{
		Slot:        "slot1",
		DisplayName: "Empty Slot 1",
		Environment: map[string]string{},
	})
	if err == nil {
		t.Fatal("writeConfigAtomically() returned no error when rename target was a directory")
	}

	// The deferred cleanup must have removed the temporary file even though
	// the rename failed.
	if leftover := listTemporaryConfigs(t, configDir); len(leftover) != 0 {
		t.Fatalf("temporary files remain after failed rename: %v", leftover)
	}
}

func TestWriteConfigAtomicallyOverwritesExistingConfig(t *testing.T) {
	configDir := requireConfigDir(t)
	path := filepath.Join(configDir, "slot1.json")

	original := []byte(`{"slot":"slot1","display_name":"user"}`)
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := writeConfigAtomically(path, SlotConfig{
		Slot:        "slot1",
		DisplayName: "Empty Slot 1",
		Environment: map[string]string{},
	}); err != nil {
		t.Fatal(err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) == string(original) {
		t.Fatalf("config was not overwritten; contents=%q", contents)
	}

	loaded, err := Load("slot1")
	if err != nil {
		t.Fatalf("Load() after atomic write: %v", err)
	}
	if loaded.DisplayName != "Empty Slot 1" {
		t.Fatalf("overwritten config has unexpected display_name: %q", loaded.DisplayName)
	}

	if leftover := listTemporaryConfigs(t, configDir); len(leftover) != 0 {
		t.Fatalf("temporary files remain after overwriting existing config: %v", leftover)
	}
}
