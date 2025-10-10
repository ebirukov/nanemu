//go:build !linux

package nanemu

import (
	"os/exec"
)

func setPlatformProcAttr(cmd *exec.Cmd) {}
