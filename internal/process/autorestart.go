package process

import (
	"fmt"
	"time"
)

const (
	DefaultAutoRestartAttempts = 3
	DefaultAutoRestartDelay    = 5 * time.Second
)

// AutoRestartPolicy bounds recovery after a confirmed crash. A bounded count
// prevents a bad command from creating an infinite restart loop.
type AutoRestartPolicy struct {
	MaxAttempts int
	Delay       time.Duration
}

func DefaultAutoRestartPolicy() AutoRestartPolicy {
	return AutoRestartPolicy{MaxAttempts: DefaultAutoRestartAttempts, Delay: DefaultAutoRestartDelay}
}

// RetryAfterCrash calls restart after each crash until it succeeds, or the
// policy budget is exhausted. The caller supplies crash detection and restart
// mechanics; this keeps the policy testable and usable by a future daemon.
func RetryAfterCrash(policy AutoRestartPolicy, clock Clock, restart func() error) error {
	if policy.MaxAttempts <= 0 || policy.Delay < 0 || clock == nil || restart == nil {
		return fmt.Errorf("invalid auto-restart policy")
	}
	var last error
	for attempt := 0; attempt < policy.MaxAttempts; attempt++ {
		if attempt > 0 {
			clock.Sleep(policy.Delay)
		}
		if err := restart(); err == nil {
			return nil
		} else {
			last = err
		}
	}
	return fmt.Errorf("auto-restart attempts exhausted: %w", last)
}
