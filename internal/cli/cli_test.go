package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devid-wp/orangepiCLI/internal/config"
)

func TestHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run() code = %d", code)
	}
	if !strings.Contains(stdout.String(), "orangectl validate") {
		t.Fatalf("help output = %q", stdout.String())
	}
}

func TestValidateDisabledSlot(t *testing.T) {
	directory := t.TempDir()
	t.Setenv(config.ConfigDirEnv, directory)
	data := `{"slot":"slot1","enabled":false,"display_name":"Empty Slot 1","environment":{}}`
	if err := os.WriteFile(filepath.Join(directory, "slot1.json"), []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"validate", "slot1"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "disabled") {
		t.Fatalf("output = %q", stdout.String())
	}
}

func TestRejectsUnknownSlot(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"validate", "other"}, &stdout, &stderr); code != 2 {
		t.Fatalf("Run() code = %d", code)
	}
}
