//go:build !windows

package process

import (
	"fmt"
	"os"
	"syscall"
)

func signalProcessGroup(pgid int, signal os.Signal) error {
	value, ok := signal.(syscall.Signal)
	if !ok {
		return fmt.Errorf("unsupported signal")
	}
	return syscall.Kill(-pgid, value)
}
func terminationSignal() os.Signal { return syscall.SIGTERM }
func killSignal() os.Signal        { return syscall.SIGKILL }
