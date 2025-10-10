//go:build linux

package nanemu

import (
	"os/exec"
	"syscall"
)

func setPlatformProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGTERM,
	}
}
