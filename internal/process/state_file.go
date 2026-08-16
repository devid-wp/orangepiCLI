package process

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

var ErrUnsafeStateFile = errors.New("unsafe process state file")

// WriteState atomically replaces a process state file. The temporary file is
// created beside the target, synced, closed, and renamed only after all writes
// have succeeded. Both temporary and final files are restricted to mode 0600.
func WriteState(path string, state ProcessState) error {
	return writeStateAtomically(path, state, os.Rename)
}

func writeStateAtomically(path string, state ProcessState, rename func(string, string) error) (err error) {
	if err := ensureSafeStateTarget(path); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary process state: %w", err)
	}
	temporaryPath := file.Name()
	defer func() {
		_ = file.Close()
		if err != nil {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := file.Chmod(0o600); err != nil && runtime.GOOS != "windows" {
		return fmt.Errorf("secure temporary process state: %w", err)
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(state); err != nil {
		return fmt.Errorf("write process state: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync process state: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close process state: %w", err)
	}
	// Check once more before replacing: a state path must never be used to
	// overwrite through a symbolic link supplied by another actor.
	if err := ensureSafeStateTarget(path); err != nil {
		return err
	}
	if err := rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace process state: %w", err)
	}
	return nil
}

func ensureSafeStateTarget(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect process state: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: symbolic links are not allowed", ErrUnsafeStateFile)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%w: state target must be a regular file", ErrUnsafeStateFile)
	}
	return nil
}

// ReadState loads a state file only when it is a regular non-symlink file.
// It is provided alongside WriteState so status/stop/restart can apply the
// same path safety rule before they inspect a stored PID.
func ReadState(path string) (ProcessState, error) {
	if err := ensureSafeStateTarget(path); err != nil {
		return ProcessState{}, err
	}
	file, err := os.Open(path)
	if err != nil {
		return ProcessState{}, fmt.Errorf("open process state: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var state ProcessState
	if err := decoder.Decode(&state); err != nil {
		return ProcessState{}, fmt.Errorf("decode process state: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return ProcessState{}, fmt.Errorf("decode process state: unexpected data after JSON object")
	}
	return state, nil
}
