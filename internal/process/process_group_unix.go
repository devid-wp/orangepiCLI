//go:build !windows

package process

import (
	"os/exec"
	"syscall"
)

func configureProcessGroup(command *exec.Cmd, enabled bool) {
	if enabled {
		command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}
}
