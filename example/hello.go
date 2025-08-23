package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

func main() {
	var uts syscall.Utsname
	syscall.Uname(&uts)
	osVersion := unsafe.String((*byte)(unsafe.Pointer(&uts.Release[0])), len(uts.Release))

	fmt.Printf(
		"Hello from process %d on linux kernel version: %s\n", os.Getpid(), osVersion)

	syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF)
}
