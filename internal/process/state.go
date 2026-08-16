package process

import (
	"fmt"
	"strings"
	"time"
)

// ProcessState is the durable state for one managed process. PID alone is
// never proof of ownership because an operating system may reuse it after the
// original process exits. ProcessIdentity is the /proc start token captured
// for this exact process instance and is verified before later lifecycle
// operations act on the PID.
type ProcessState struct {
	Slot             string    `json:"slot"`
	PID              int       `json:"pid"`
	StartedAt        time.Time `json:"started_at"`
	WorkingDirectory string    `json:"working_directory"`
	ProcessIdentity  string    `json:"process_identity"`
}

// NewProcessState creates state only when a process instance has a durable
// identity. If procfs cannot supply an identity, callers must not persist a
// PID that could later point to an unrelated process.
func NewProcessState(slot string, process ProcProcess, workingDirectory string, clock Clock) (ProcessState, error) {
	if strings.TrimSpace(slot) == "" {
		return ProcessState{}, fmt.Errorf("process state requires a slot")
	}
	if process.PID <= 0 {
		return ProcessState{}, fmt.Errorf("process state requires a positive PID")
	}
	if strings.TrimSpace(process.Identity) == "" {
		return ProcessState{}, fmt.Errorf("process state requires a process identity")
	}
	if strings.TrimSpace(workingDirectory) == "" {
		return ProcessState{}, fmt.Errorf("process state requires a working directory")
	}
	if clock == nil {
		return ProcessState{}, fmt.Errorf("process state requires a clock")
	}
	startedAt := process.StartedAt
	if startedAt.IsZero() {
		startedAt = clock.Now()
	}
	return ProcessState{
		Slot:             slot,
		PID:              process.PID,
		StartedAt:        startedAt.UTC(),
		WorkingDirectory: workingDirectory,
		ProcessIdentity:  process.Identity,
	}, nil
}
