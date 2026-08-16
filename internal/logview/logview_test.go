package logview

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenRejectsSymbolicLink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real.log")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.log")
	if err := os.Symlink(target, link); err != nil {
		t.Skip(err)
	}
	if _, err := Open(link); !errors.Is(err, ErrUnsafeLogFile) {
		t.Fatalf("Open() error=%v", err)
	}
}
