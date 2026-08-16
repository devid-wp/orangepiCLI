package process

import "time"

// ProcProcess is the process identity material read from /proc. Identity and
// StartedAt must describe the same process instance, not merely a PID.
type ProcProcess struct {
	PID              int
	StartedAt        time.Time
	Identity         string
	Command          string
	WorkingDirectory string
	UserID           string
}
