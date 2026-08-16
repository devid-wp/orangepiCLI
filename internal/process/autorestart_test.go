package process

import (
	"errors"
	"testing"
)

func TestRetryAfterCrashIsBounded(t *testing.T) {
	calls := 0
	err := RetryAfterCrash(AutoRestartPolicy{MaxAttempts: 3, Delay: 1}, fakeClock{}, func() error { calls++; return errors.New("crash") })
	if err == nil || calls != 3 {
		t.Fatalf("RetryAfterCrash() = %v after %d calls", err, calls)
	}
}
func TestRetryAfterCrashStopsAfterRecovery(t *testing.T) {
	calls := 0
	err := RetryAfterCrash(DefaultAutoRestartPolicy(), fakeClock{}, func() error {
		calls++
		if calls == 2 {
			return nil
		}
		return errors.New("crash")
	})
	if err != nil || calls != 2 {
		t.Fatalf("RetryAfterCrash() = %v after %d calls", err, calls)
	}
}
