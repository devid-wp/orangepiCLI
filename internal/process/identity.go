package process

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrProcessNotFound         = errors.New("process not found")
	ErrProcessIdentityMismatch = errors.New("process identity mismatch")
)

// ProcessExpectation contains independent facts expected of the managed
// process. Command and UserID are intentionally kept out of ProcessState so
// state never needs to duplicate the user's command text.
type ProcessExpectation struct {
	Command          string
	WorkingDirectory string
	UserID           string
}

// VerifyProcessIdentity proves that a stored PID still points at precisely the
// process that was started for a slot. It rejects a PID reuse before callers
// can signal it. Errors do not echo command text or user values, which may
// contain sensitive data.
func VerifyProcessIdentity(reader ProcReader, state ProcessState, expected ProcessExpectation) error {
	if reader == nil {
		return fmt.Errorf("verify process identity: proc reader is nil")
	}
	if state.PID <= 0 || strings.TrimSpace(state.ProcessIdentity) == "" || state.StartedAt.IsZero() || strings.TrimSpace(state.WorkingDirectory) == "" {
		return fmt.Errorf("%w: stored process state is incomplete", ErrProcessIdentityMismatch)
	}
	if strings.TrimSpace(expected.Command) == "" || strings.TrimSpace(expected.UserID) == "" {
		return fmt.Errorf("%w: process expectation is incomplete", ErrProcessIdentityMismatch)
	}
	if expected.WorkingDirectory == "" {
		expected.WorkingDirectory = state.WorkingDirectory
	}
	if expected.WorkingDirectory != state.WorkingDirectory {
		return fmt.Errorf("%w: expected working directory differs from stored state", ErrProcessIdentityMismatch)
	}

	actual, err := reader.ReadProcess(state.PID)
	if err != nil {
		if errors.Is(err, ErrProcessNotFound) {
			return fmt.Errorf("verify process identity: %w", err)
		}
		return fmt.Errorf("inspect tracked process: %w", err)
	}
	if actual.PID != state.PID || actual.Identity != state.ProcessIdentity || !actual.StartedAt.Equal(state.StartedAt) {
		return fmt.Errorf("%w: PID no longer identifies the started process", ErrProcessIdentityMismatch)
	}
	if actual.Command != expected.Command {
		return fmt.Errorf("%w: command differs", ErrProcessIdentityMismatch)
	}
	if actual.WorkingDirectory != state.WorkingDirectory || actual.WorkingDirectory != expected.WorkingDirectory {
		return fmt.Errorf("%w: working directory differs", ErrProcessIdentityMismatch)
	}
	if actual.UserID != expected.UserID {
		return fmt.Errorf("%w: process owner differs", ErrProcessIdentityMismatch)
	}
	return nil
}
