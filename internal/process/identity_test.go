package process

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func verifiedState() (ProcessState, ProcessExpectation, ProcProcess) {
	startedAt := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	state := ProcessState{
		Slot:             "slot1",
		PID:              1234,
		StartedAt:        startedAt,
		WorkingDirectory: "/srv/example",
		ProcessIdentity:  "linux-start-ticks:998877",
	}
	expected := ProcessExpectation{
		Command:          "/bin/sh -c worker",
		WorkingDirectory: "/srv/example",
		UserID:           "1000",
	}
	actual := ProcProcess{
		PID:              state.PID,
		StartedAt:        state.StartedAt,
		Identity:         state.ProcessIdentity,
		Command:          expected.Command,
		WorkingDirectory: state.WorkingDirectory,
		UserID:           expected.UserID,
	}
	return state, expected, actual
}

func TestVerifyProcessIdentityAcceptsExactProcess(t *testing.T) {
	state, expected, actual := verifiedState()
	if err := VerifyProcessIdentity(fakeProcReader{process: actual}, state, expected); err != nil {
		t.Fatalf("VerifyProcessIdentity() error = %v", err)
	}
}

func TestVerifyProcessIdentityRejectsStaleOrReusedPID(t *testing.T) {
	state, expected, actual := verifiedState()
	tests := []struct {
		name  string
		apply func(*ProcProcess)
	}{
		{name: "different PID", apply: func(process *ProcProcess) { process.PID++ }},
		{name: "different start identity", apply: func(process *ProcProcess) { process.Identity = "linux-start-ticks:other" }},
		{name: "different start time", apply: func(process *ProcProcess) { process.StartedAt = process.StartedAt.Add(time.Second) }},
		{name: "different command", apply: func(process *ProcProcess) { process.Command = "/bin/sh -c another" }},
		{name: "different working directory", apply: func(process *ProcProcess) { process.WorkingDirectory = "/srv/other" }},
		{name: "different user", apply: func(process *ProcProcess) { process.UserID = "1001" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := actual
			test.apply(&candidate)
			err := VerifyProcessIdentity(fakeProcReader{process: candidate}, state, expected)
			if !errors.Is(err, ErrProcessIdentityMismatch) {
				t.Fatalf("VerifyProcessIdentity() error = %v, want ErrProcessIdentityMismatch", err)
			}
		})
	}
}

func TestVerifyProcessIdentityReportsMissingProcess(t *testing.T) {
	state, expected, _ := verifiedState()
	err := VerifyProcessIdentity(fakeProcReader{err: ErrProcessNotFound}, state, expected)
	if !errors.Is(err, ErrProcessNotFound) {
		t.Fatalf("VerifyProcessIdentity() error = %v, want ErrProcessNotFound", err)
	}
}

func TestVerifyProcessIdentityDoesNotExposeCommandSecrets(t *testing.T) {
	const secret = "do-not-leak-command-secret"
	state, expected, actual := verifiedState()
	expected.Command = "/bin/sh -c TOKEN=" + secret
	actual.Command = "/bin/sh -c different"
	err := VerifyProcessIdentity(fakeProcReader{process: actual}, state, expected)
	if err == nil {
		t.Fatal("VerifyProcessIdentity() returned no error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("identity error exposed secret: %q", err)
	}
}

func TestVerifyProcessIdentityRejectsIncompleteExpectation(t *testing.T) {
	state, expected, actual := verifiedState()
	expected.UserID = ""
	err := VerifyProcessIdentity(fakeProcReader{process: actual}, state, expected)
	if !errors.Is(err, ErrProcessIdentityMismatch) {
		t.Fatalf("VerifyProcessIdentity() error = %v", err)
	}
}
