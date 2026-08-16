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
func TestLastLinesReturnsSuffix(t *testing.T) {
	path := filepath.Join(t.TempDir(), "slot.log")
	if err := os.WriteFile(path, []byte("one\ntwo\nthree\nfour\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := LastLines(path, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "three" || got[1] != "four" {
		t.Fatalf("LastLines=%v", got)
	}
}
