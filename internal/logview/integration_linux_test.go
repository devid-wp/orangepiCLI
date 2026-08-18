//go:build integration && linux

package logview

import "testing"

func TestIntegrationEnvironment(t *testing.T) {
	if testing.Short() {
		t.Skip("integration suite is disabled in short mode")
	}
}
