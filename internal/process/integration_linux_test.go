//go:build integration && linux

package process

import "testing"

// TestIntegrationEnvironment marks the suite that must run on a Linux host.
// Lifecycle, procfs, and signal tests are added here so regular unit tests do
// not require /proc or process-group semantics.
func TestIntegrationEnvironment(t *testing.T) {
	if DefaultOperations().Proc == nil {
		t.Fatal("Linux proc reader is unavailable")
	}
}
