package process

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestOpenLogCreatesSecureAppendOnlyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "slot.log")
	file, err := OpenLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fmt.Fprint(file, "first\n"); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	file, err = OpenLog(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fmt.Fprint(file, "second\n"); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(contents), "first\nsecond\n"; got != want {
		t.Fatalf("log contents = %q, want %q", got, want)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("log mode = %o, want 0600", got)
		}
	}
}

func TestOpenLogRejectsUnsafeTargets(t *testing.T) {
	t.Run("directory", func(t *testing.T) {
		if _, err := OpenLog(t.TempDir()); err == nil || !strings.Contains(err.Error(), "regular file") {
			t.Fatalf("OpenLog() error = %v", err)
		}
	})
	t.Run("symbolic link", func(t *testing.T) {
		directory := t.TempDir()
		target := filepath.Join(directory, "target.log")
		if err := os.WriteFile(target, []byte("ready\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(directory, "linked.log")
		if err := os.Symlink(target, link); err != nil {
			t.Skipf("symbolic links are unavailable: %v", err)
		}
		if _, err := OpenLog(link); err == nil || !strings.Contains(err.Error(), "symbolic link") {
			t.Fatalf("OpenLog() error = %v", err)
		}
	})
}

func TestAttachLogsRoutesBothStreamsToOneFile(t *testing.T) {
	command := &Command{}
	path := filepath.Join(t.TempDir(), "slot.log")
	closer, err := AttachLogs(command, path)
	if err != nil {
		t.Fatal(err)
	}
	if command.Stdout == nil || command.Stderr == nil || command.Stdout != command.Stderr {
		t.Fatalf("AttachLogs() did not assign the same log to both streams")
	}
	if _, err := fmt.Fprint(command.Stdout, "stdout\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := fmt.Fprint(command.Stderr, "stderr\n"); err != nil {
		t.Fatal(err)
	}
	if err := closer.Close(); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(contents), "stdout\nstderr\n"; got != want {
		t.Fatalf("log contents = %q, want %q", got, want)
	}
}

func TestAttachLogsRejectsNilCommand(t *testing.T) {
	if _, err := AttachLogs(nil, "slot.log"); err == nil {
		t.Fatal("AttachLogs() returned no error for nil command")
	}
}
