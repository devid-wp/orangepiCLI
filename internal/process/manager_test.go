package process

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/devid-wp/orangepiCLI/internal/config"
	"github.com/devid-wp/orangepiCLI/internal/paths"
)

func startTestManager(t *testing.T, launcher *fakeLauncher, proc ProcProcess) Manager {
	t.Helper()
	t.Setenv(paths.StateDirEnv, t.TempDir())
	t.Setenv(config.ConfigDirEnv, t.TempDir())
	return Manager{
		Operations: Operations{
			Launcher: launcher,
			Proc:     fakeProcReader{process: proc},
			Clock:    fakeClock{now: time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)},
		},
		StateDirectory: t.TempDir(),
		UserID:         "1000",
	}
}

func enabledStartSlot(t *testing.T) config.SlotConfig {
	t.Helper()
	directory := t.TempDir()
	return config.SlotConfig{
		Slot:             "slot1",
		Enabled:          true,
		WorkingDirectory: directory,
		StartCommand:     "worker --serve",
		LogFile:          filepath.Join(directory, "slot.log"),
		Environment:      map[string]string{"SLOT": "one"},
	}
}

func TestManagerStartLaunchesShellAndWritesState(t *testing.T) {
	launcher := &fakeLauncher{}
	startedAt := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	slot := enabledStartSlot(t)
	proc := ProcProcess{
		PID:              42,
		StartedAt:        startedAt,
		Identity:         "linux-start-ticks:42",
		Command:          shellCommand(slot.StartCommand),
		WorkingDirectory: slot.WorkingDirectory,
		UserID:           "1000",
	}
	manager := startTestManager(t, launcher, proc)

	state, err := manager.Start(slot)
	if err != nil {
		t.Fatal(err)
	}
	if state.PID != 42 || state.ProcessIdentity != proc.Identity {
		t.Fatalf("Start() state = %+v", state)
	}
	if launcher.command.Path != "/bin/sh" || len(launcher.command.Args) != 2 || launcher.command.Args[0] != "-c" || launcher.command.Args[1] != slot.StartCommand {
		t.Fatalf("launcher command = %+v", launcher.command)
	}
	if launcher.command.Dir != slot.WorkingDirectory || !launcher.command.NewProcessGroup {
		t.Fatalf("launcher command = %+v", launcher.command)
	}
	if launcher.command.Stdout == nil || launcher.command.Stderr == nil || launcher.command.Stdout != launcher.command.Stderr {
		t.Fatalf("start did not attach log streams")
	}
	stored, err := ReadState(manager.statePath(slot.Slot))
	if err != nil {
		t.Fatal(err)
	}
	if stored != state {
		t.Fatalf("stored state = %+v, want %+v", stored, state)
	}
	if _, err := os.Stat(slot.LogFile); err != nil {
		t.Fatalf("log file was not created: %v", err)
	}
}

func TestManagerStartRejectsDisabledAndLiveSlots(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		launcher := &fakeLauncher{}
		slot := enabledStartSlot(t)
		slot.Enabled = false
		manager := startTestManager(t, launcher, ProcProcess{})
		if _, err := manager.Start(slot); err == nil {
			t.Fatal("Start() returned no error for disabled slot")
		}
	})
	t.Run("already running", func(t *testing.T) {
		launcher := &fakeLauncher{}
		slot := enabledStartSlot(t)
		startedAt := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
		proc := ProcProcess{PID: 42, StartedAt: startedAt, Identity: "identity", Command: shellCommand(slot.StartCommand), WorkingDirectory: slot.WorkingDirectory, UserID: "1000"}
		manager := startTestManager(t, launcher, proc)
		if err := os.MkdirAll(manager.StateDirectory, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := WriteState(manager.statePath(slot.Slot), ProcessState{Slot: slot.Slot, PID: 42, StartedAt: startedAt, WorkingDirectory: slot.WorkingDirectory, ProcessIdentity: "identity"}); err != nil {
			t.Fatal(err)
		}
		_, err := manager.Start(slot)
		if !errors.Is(err, ErrAlreadyRunning) {
			t.Fatalf("Start() error = %v, want ErrAlreadyRunning", err)
		}
	})
}

func TestManagerStartRemovesStaleStateWithoutSignallingProcess(t *testing.T) {
	launcher := &fakeLauncher{}
	slot := enabledStartSlot(t)
	startedAt := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	proc := ProcProcess{PID: 42, StartedAt: startedAt, Identity: "new", Command: shellCommand(slot.StartCommand), WorkingDirectory: slot.WorkingDirectory, UserID: "1000"}
	manager := startTestManager(t, launcher, proc)
	if err := os.MkdirAll(manager.StateDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := WriteState(manager.statePath(slot.Slot), ProcessState{Slot: slot.Slot, PID: 42, StartedAt: startedAt, WorkingDirectory: slot.WorkingDirectory, ProcessIdentity: "old"}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Start(slot); err != nil {
		t.Fatal(err)
	}
	stored, err := ReadState(manager.statePath(slot.Slot))
	if err != nil {
		t.Fatal(err)
	}
	if stored.ProcessIdentity != "new" {
		t.Fatalf("stale state was not replaced: %+v", stored)
	}
}
