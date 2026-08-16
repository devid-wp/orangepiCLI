//go:build windows

package process

import "os/exec"

func configureProcessGroup(_ *exec.Cmd, _ bool) {}
