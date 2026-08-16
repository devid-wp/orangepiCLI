package process

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/devid-wp/orangepiCLI/internal/config"
	"github.com/devid-wp/orangepiCLI/internal/paths"
)

var ErrAlreadyRunning = errors.New("slot process is already running")
var ErrStopTimeout = errors.New("process stop timed out")

const DefaultStopTimeout = 10 * time.Second

type Status string

const (
	StatusStopped Status = "stopped"
	StatusRunning Status = "running"
	StatusStale   Status = "stale"
)

// SlotStatus is a safe status projection; it contains no command or
// environment values.
type SlotStatus struct {
	Slot  string       `json:"slot"`
	State Status       `json:"state"`
	PID   int          `json:"pid,omitempty"`
	Info  ProcessState `json:"info,omitempty"`
}

// Manager owns lifecycle dependencies and persistent process state. It is
// deliberately configured with interfaces so tests can use no real processes.
type Manager struct {
	Operations     Operations
	StateDirectory string
	UserID         string
	StopTimeout    time.Duration
	PollInterval   time.Duration
}

func NewManager(operations Operations, stateDirectory string) Manager {
	if stateDirectory == "" {
		stateDirectory = paths.PIDDir()
	}
	return Manager{Operations: operations, StateDirectory: stateDirectory, UserID: currentUserID(), StopTimeout: DefaultStopTimeout, PollInterval: 100 * time.Millisecond}
}

func DefaultManager() Manager { return NewManager(DefaultOperations(), paths.PIDDir()) }

func currentUserID() string {
	current, err := user.Current()
	if err != nil {
		return ""
	}
	return current.Uid
}

func (manager Manager) statePath(slot string) string {
	return filepath.Join(manager.StateDirectory, slot+".json")
}

func (manager Manager) expected(slot config.SlotConfig, command string) ProcessExpectation {
	return ProcessExpectation{
		Command:          shellCommand(command),
		WorkingDirectory: slot.WorkingDirectory,
		UserID:           manager.UserID,
	}
}

func shellCommand(command string) string { return "/bin/sh -c " + command }

// Start validates an enabled slot, refuses a live state file, starts its shell
// command in a new process group, attaches its log, and persists identity.
func (manager Manager) Start(slot config.SlotConfig) (ProcessState, error) {
	if err := manager.valid(); err != nil {
		return ProcessState{}, err
	}
	if err := config.RequireAllowed(slot.Slot); err != nil {
		return ProcessState{}, err
	}
	if validationErrors := config.Validate(slot); len(validationErrors) > 0 {
		return ProcessState{}, fmt.Errorf("validate slot: %s", config.FormatErrors(validationErrors))
	}
	if !slot.Enabled {
		return ProcessState{}, fmt.Errorf("slot %q is disabled", slot.Slot)
	}
	if err := manager.ensureNoRunningState(slot); err != nil {
		return ProcessState{}, err
	}
	if err := os.MkdirAll(manager.StateDirectory, 0o700); err != nil {
		return ProcessState{}, fmt.Errorf("prepare process state directory: %w", err)
	}
	if err := paths.Ensure(); err != nil {
		return ProcessState{}, fmt.Errorf("prepare application directories: %w", err)
	}
	environment, err := BuildEnvironment(slot.EnvFile, slot.Environment)
	if err != nil {
		return ProcessState{}, err
	}
	command := Command{
		Path:            "/bin/sh",
		Args:            []string{"-c", slot.StartCommand},
		Dir:             slot.WorkingDirectory,
		Env:             environment,
		NewProcessGroup: true,
	}
	log, err := AttachLogs(&command, manager.logPath(slot))
	if err != nil {
		return ProcessState{}, err
	}
	defer log.Close()
	started, err := manager.Operations.Launcher.Start(command)
	if err != nil {
		return ProcessState{}, fmt.Errorf("start slot process: %w", err)
	}
	proc, err := manager.Operations.Proc.ReadProcess(started.PID())
	if err != nil {
		return ProcessState{}, fmt.Errorf("inspect started process: %w", err)
	}
	state, err := NewProcessState(slot.Slot, proc, slot.WorkingDirectory, manager.Operations.Clock)
	if err != nil {
		return ProcessState{}, err
	}
	if err := WriteState(manager.statePath(slot.Slot), state); err != nil {
		return ProcessState{}, err
	}
	return state, nil
}

// Status reports stopped when no state exists, running only after full
// identity verification, and stale when the PID file refers to an exited or
// replaced process. A status lookup never signals or deletes anything.
func (manager Manager) Status(slot config.SlotConfig) (SlotStatus, error) {
	if err := manager.valid(); err != nil {
		return SlotStatus{}, err
	}
	if err := config.RequireAllowed(slot.Slot); err != nil {
		return SlotStatus{}, err
	}
	state, err := ReadState(manager.statePath(slot.Slot))
	if errors.Is(err, os.ErrNotExist) {
		return SlotStatus{Slot: slot.Slot, State: StatusStopped}, nil
	}
	if err != nil {
		return SlotStatus{}, err
	}
	err = VerifyProcessIdentity(manager.Operations.Proc, state, manager.expected(slot, slot.StartCommand))
	if err == nil {
		return SlotStatus{Slot: slot.Slot, State: StatusRunning, PID: state.PID, Info: state}, nil
	}
	if errors.Is(err, ErrProcessNotFound) || errors.Is(err, ErrProcessIdentityMismatch) {
		return SlotStatus{Slot: slot.Slot, State: StatusStale, PID: state.PID, Info: state}, nil
	}
	return SlotStatus{}, err
}

func (manager Manager) Stop(slot config.SlotConfig) error {
	return manager.stop(slot, false)
}

// ForceStop sends SIGKILL only after a timed-out graceful stop and a fresh
// identity check. It never kills a PID solely because it appears in state.
func (manager Manager) ForceStop(slot config.SlotConfig) error {
	return manager.stop(slot, true)
}

func (manager Manager) stop(slot config.SlotConfig, force bool) error {
	if err := manager.valid(); err != nil {
		return err
	}
	state, err := ReadState(manager.statePath(slot.Slot))
	if err != nil {
		return err
	}
	if err := VerifyProcessIdentity(manager.Operations.Proc, state, manager.expected(slot, slot.StartCommand)); err != nil {
		return err
	}
	if slot.StopCommand != "" {
		env, err := BuildEnvironment(slot.EnvFile, slot.Environment)
		if err != nil {
			return err
		}
		command := Command{Path: "/bin/sh", Args: []string{"-c", slot.StopCommand}, Dir: slot.WorkingDirectory, Env: env}
		started, err := manager.Operations.Launcher.Start(command)
		if err != nil {
			return fmt.Errorf("run stop command: %w", err)
		}
		if err := started.Wait(); err != nil {
			return fmt.Errorf("wait for stop command: %w", err)
		}
	} else {
		group, ok := manager.Operations.Signaler.(GroupSignaler)
		if !ok {
			return fmt.Errorf("process manager cannot signal process groups")
		}
		if err := group.SignalGroup(state.PID, terminationSignal()); err != nil {
			return fmt.Errorf("send termination signal: %w", err)
		}
	}
	err = manager.waitForExit(slot, state)
	if !force || !errors.Is(err, ErrStopTimeout) {
		return err
	}
	if err := VerifyProcessIdentity(manager.Operations.Proc, state, manager.expected(slot, slot.StartCommand)); err != nil {
		return err
	}
	group, ok := manager.Operations.Signaler.(GroupSignaler)
	if !ok {
		return fmt.Errorf("process manager cannot signal process groups")
	}
	if err := group.SignalGroup(state.PID, killSignal()); err != nil {
		return fmt.Errorf("send kill signal: %w", err)
	}
	return manager.waitForExit(slot, state)
}

func (manager Manager) waitForExit(slot config.SlotConfig, state ProcessState) error {
	timeout := manager.StopTimeout
	if timeout <= 0 {
		timeout = DefaultStopTimeout
	}
	poll := manager.PollInterval
	if poll <= 0 {
		poll = 100 * time.Millisecond
	}
	deadline := manager.Operations.Clock.Now().Add(timeout)
	for {
		err := VerifyProcessIdentity(manager.Operations.Proc, state, manager.expected(slot, slot.StartCommand))
		if errors.Is(err, ErrProcessNotFound) {
			return RemoveState(manager.statePath(slot.Slot))
		}
		if err != nil {
			return err
		}
		if !manager.Operations.Clock.Now().Before(deadline) {
			return ErrStopTimeout
		}
		manager.Operations.Clock.Sleep(poll)
	}
}

func (manager Manager) logPath(slot config.SlotConfig) string {
	if slot.LogFile != "" {
		return slot.LogFile
	}
	return filepath.Join(paths.LogDir(), slot.Slot+".log")
}

func (manager Manager) ensureNoRunningState(slot config.SlotConfig) error {
	path := manager.statePath(slot.Slot)
	state, err := ReadState(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read existing process state: %w", err)
	}
	err = VerifyProcessIdentity(manager.Operations.Proc, state, manager.expected(slot, slot.StartCommand))
	if err == nil {
		return fmt.Errorf("%w: %s", ErrAlreadyRunning, slot.Slot)
	}
	if !errors.Is(err, ErrProcessNotFound) && !errors.Is(err, ErrProcessIdentityMismatch) {
		return err
	}
	if err := RemoveState(path); err != nil {
		return err
	}
	return nil
}

func (manager Manager) valid() error {
	if manager.Operations.Launcher == nil || manager.Operations.Proc == nil || manager.Operations.Clock == nil {
		return fmt.Errorf("process manager has incomplete operations")
	}
	if strings.TrimSpace(manager.UserID) == "" {
		return fmt.Errorf("process manager has no current user identity")
	}
	return nil
}
