// Package process contains the system-facing building blocks for managing a
// configured process slot. Commands use the interfaces in this package rather
// than calling os/exec, os.Process, /proc, or time directly, keeping lifecycle
// rules deterministic in tests.
package process

import (
	"io"
	"os"
	"os/exec"
	"time"
)

// Command describes a process that may be started. It deliberately keeps the
// command and its arguments separate: callers must not build shell text from
// untrusted CLI arguments.
type Command struct {
	Path            string
	Args            []string
	Dir             string
	Env             []string
	Stdout          io.Writer
	Stderr          io.Writer
	NewProcessGroup bool
}

// StartedProcess is the small part of exec.Cmd that lifecycle code needs.
type StartedProcess interface {
	PID() int
	Wait() error
}

// Launcher starts a command. Tests supply a fake Launcher rather than
// executing a real process.
type Launcher interface {
	Start(Command) (StartedProcess, error)
}

// Signaler delivers a signal to a process. A production implementation uses
// os.FindProcess, while tests can record intended signals without affecting
// the host process table.
type Signaler interface {
	Signal(pid int, signal os.Signal) error
}

type GroupSignaler interface {
	SignalGroup(pgid int, signal os.Signal) error
}

// ProcReader reads stable process facts from a proc-compatible filesystem.
// Identity is an OS-provided start token; together with PID it detects reuse
// of a PID by a different process.
type ProcReader interface {
	ReadProcess(pid int) (ProcProcess, error)
}

// Clock abstracts wall time used when a process state is created.
type Clock interface {
	Now() time.Time
	Sleep(time.Duration)
}

// Operations groups all system dependencies used by lifecycle operations.
// A caller may replace any field in tests. Production callers use
// DefaultOperations.
type Operations struct {
	Launcher Launcher
	Signaler Signaler
	Proc     ProcReader
	Clock    Clock
}

// DefaultOperations returns real OS-backed implementations.
func DefaultOperations() Operations {
	return Operations{
		Launcher: execLauncher{},
		Signaler: osSignaler{},
		Proc:     NewProcReader(),
		Clock:    systemClock{},
	}
}

type execLauncher struct{}

func (execLauncher) Start(command Command) (StartedProcess, error) {
	cmd := exec.Command(command.Path, command.Args...)
	cmd.Dir = command.Dir
	cmd.Env = command.Env
	cmd.Stdout = command.Stdout
	cmd.Stderr = command.Stderr
	configureProcessGroup(cmd, command.NewProcessGroup)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return execStartedProcess{cmd: cmd}, nil
}

type execStartedProcess struct{ cmd *exec.Cmd }

func (process execStartedProcess) PID() int    { return process.cmd.Process.Pid }
func (process execStartedProcess) Wait() error { return process.cmd.Wait() }

type osSignaler struct{}

func (osSignaler) Signal(pid int, signal os.Signal) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Signal(signal)
}

func (osSignaler) SignalGroup(pgid int, signal os.Signal) error {
	return signalProcessGroup(pgid, signal)
}

type systemClock struct{}

func (systemClock) Now() time.Time               { return time.Now() }
func (systemClock) Sleep(duration time.Duration) { time.Sleep(duration) }
