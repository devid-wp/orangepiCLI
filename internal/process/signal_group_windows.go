//go:build windows

package process

import (
	"fmt"
	"os"
)

func signalProcessGroup(int, os.Signal) error {
	return fmt.Errorf("process groups are unavailable on Windows")
}
func terminationSignal() os.Signal { return os.Interrupt }
func killSignal() os.Signal        { return os.Kill }
