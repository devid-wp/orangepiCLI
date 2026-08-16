package process

import (
	"errors"
	"os"
	"testing"
	"time"
)

type fakeStartedProcess struct{ pid int }

func (fakeStartedProcess) PID() int    { return 42 }
func (fakeStartedProcess) Wait() error { return nil }

type fakeLauncher struct {
	command Command
	err     error
}

func (fake *fakeLauncher) Start(command Command) (StartedProcess, error) {
	fake.command = command
	if fake.err != nil {
		return nil, fake.err
	}
	return fakeStartedProcess{}, nil
}

type fakeSignaler struct {
	pid    int
	signal os.Signal
}

func (fake *fakeSignaler) Signal(pid int, signal os.Signal) error {
	fake.pid, fake.signal = pid, signal
	return nil
}

type fakeProcReader struct {
	process ProcProcess
	err     error
}

func (fake fakeProcReader) ReadProcess(int) (ProcProcess, error) {
	return fake.process, fake.err
}

type fakeClock struct{ now time.Time }

func (clock fakeClock) Now() time.Time { return clock.now }

func TestOperationsDependenciesAreReplaceable(t *testing.T) {
	launcher := &fakeLauncher{}
	signaler := &fakeSignaler{}
	proc := fakeProcReader{process: ProcProcess{PID: 42, Identity: "start-42"}}
	clock := fakeClock{now: time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)}
	operations := Operations{Launcher: launcher, Signaler: signaler, Proc: proc, Clock: clock}

	started, err := operations.Launcher.Start(Command{Path: "/bin/echo", Args: []string{"ok"}})
	if err != nil || started.PID() != 42 {
		t.Fatalf("Launcher.Start() = (%v, %v)", started, err)
	}
	if err := operations.Signaler.Signal(42, os.Interrupt); err != nil {
		t.Fatal(err)
	}
	if signaler.pid != 42 || signaler.signal != os.Interrupt {
		t.Fatalf("Signal() recorded (%d, %v)", signaler.pid, signaler.signal)
	}
	got, err := operations.Proc.ReadProcess(42)
	if err != nil || got.Identity != "start-42" {
		t.Fatalf("ReadProcess() = (%+v, %v)", got, err)
	}
	if got := operations.Clock.Now(); !got.Equal(clock.now) {
		t.Fatalf("Clock.Now() = %s, want %s", got, clock.now)
	}
}

func TestFakeLauncherCanReturnFailure(t *testing.T) {
	want := errors.New("cannot start")
	launcher := &fakeLauncher{err: want}
	if _, err := launcher.Start(Command{}); !errors.Is(err, want) {
		t.Fatalf("Start() error = %v, want %v", err, want)
	}
}

func TestDefaultOperationsAreComplete(t *testing.T) {
	operations := DefaultOperations()
	if operations.Launcher == nil || operations.Signaler == nil || operations.Proc == nil || operations.Clock == nil {
		t.Fatalf("DefaultOperations() = %+v, contains a nil dependency", operations)
	}
}
