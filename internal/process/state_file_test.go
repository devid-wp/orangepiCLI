package process

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func testState() ProcessState {
	return ProcessState{
		Slot:             "slot1",
		PID:              1234,
		StartedAt:        time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC),
		WorkingDirectory: "/srv/example",
		ProcessIdentity:  "linux-start-ticks:998877",
	}
}

func temporaryStateFiles(t *testing.T, directory string) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(directory, ".*.tmp"))
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestWriteStateAtomicallyCreatesSecureFile(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "slot1.json")
	state := testState()
	if err := WriteState(path, state); err != nil {
		t.Fatal(err)
	}
	got, err := ReadState(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != state {
		t.Fatalf("ReadState() = %+v, want %+v", got, state)
	}
	if temporary := temporaryStateFiles(t, directory); len(temporary) != 0 {
		t.Fatalf("temporary state files remain: %v", temporary)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("state mode = %o, want 0600", got)
		}
	}
}

func TestWriteStateFailurePreservesExistingFileAndCleansTemporaryFile(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "slot1.json")
	original := []byte("user-owned-state\n")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	want := errors.New("rename blocked")
	err := writeStateAtomically(path, testState(), func(string, string) error { return want })
	if !errors.Is(err, want) {
		t.Fatalf("writeStateAtomically() error = %v, want %v", err, want)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != string(original) {
		t.Fatalf("existing state changed after failed rename: %q", contents)
	}
	if temporary := temporaryStateFiles(t, directory); len(temporary) != 0 {
		t.Fatalf("temporary state files remain: %v", temporary)
	}
}

func TestStateFileRejectsSymbolicLink(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "target.json")
	if err := os.WriteFile(target, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "slot1.json")
	if err := os.Symlink(target, path); err != nil {
		t.Skipf("symbolic links are unavailable: %v", err)
	}
	if err := WriteState(path, testState()); !errors.Is(err, ErrUnsafeStateFile) {
		t.Fatalf("WriteState() error = %v, want ErrUnsafeStateFile", err)
	}
	if _, err := ReadState(path); !errors.Is(err, ErrUnsafeStateFile) {
		t.Fatalf("ReadState() error = %v, want ErrUnsafeStateFile", err)
	}
}

func TestReadStateRejectsTrailingData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "slot1.json")
	if err := os.WriteFile(path, []byte(`{"slot":"slot1"}{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadState(path); err == nil || !strings.Contains(err.Error(), "unexpected data") {
		t.Fatalf("ReadState() error = %v", err)
	}
}
