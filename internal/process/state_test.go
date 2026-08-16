package process

import (
	"strings"
	"testing"
	"time"
)

func TestNewProcessStateIncludesIdentityBeyondPID(t *testing.T) {
	startedAt := time.Date(2026, 8, 16, 10, 30, 0, 0, time.FixedZone("UTC+3", 3*60*60))
	clock := fakeClock{now: time.Date(2026, 8, 16, 9, 0, 0, 0, time.UTC)}
	state, err := NewProcessState("slot1", ProcProcess{
		PID:       1234,
		StartedAt: startedAt,
		Identity:  "linux-start-ticks:998877",
	}, "/srv/example", clock)
	if err != nil {
		t.Fatal(err)
	}
	if state.PID != 1234 || state.ProcessIdentity != "linux-start-ticks:998877" {
		t.Fatalf("state = %+v", state)
	}
	if got, want := state.StartedAt, startedAt.UTC(); !got.Equal(want) {
		t.Fatalf("StartedAt = %s, want %s", got, want)
	}
}

func TestNewProcessStateUsesClockWhenProcStartTimeIsUnavailable(t *testing.T) {
	now := time.Date(2026, 8, 16, 9, 0, 0, 0, time.UTC)
	state, err := NewProcessState("slot1", ProcProcess{PID: 1234, Identity: "start-1"}, "/srv/example", fakeClock{now: now})
	if err != nil {
		t.Fatal(err)
	}
	if !state.StartedAt.Equal(now) {
		t.Fatalf("StartedAt = %s, want %s", state.StartedAt, now)
	}
}

func TestNewProcessStateRejectsUnsafeIdentityMaterial(t *testing.T) {
	tests := []struct {
		name    string
		slot    string
		process ProcProcess
		dir     string
		clock   Clock
	}{
		{name: "empty slot", process: ProcProcess{PID: 1, Identity: "id"}, dir: "/srv", clock: fakeClock{}},
		{name: "zero PID", slot: "slot1", process: ProcProcess{Identity: "id"}, dir: "/srv", clock: fakeClock{}},
		{name: "missing identity", slot: "slot1", process: ProcProcess{PID: 1}, dir: "/srv", clock: fakeClock{}},
		{name: "empty directory", slot: "slot1", process: ProcProcess{PID: 1, Identity: "id"}, clock: fakeClock{}},
		{name: "nil clock", slot: "slot1", process: ProcProcess{PID: 1, Identity: "id"}, dir: "/srv"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewProcessState(test.slot, test.process, test.dir, test.clock)
			if err == nil || !strings.Contains(err.Error(), "requires") {
				t.Fatalf("NewProcessState() error = %v", err)
			}
		})
	}
}
