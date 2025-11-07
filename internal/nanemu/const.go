package nanemu

import "time"

const DefaultExecTimeout = 30 * time.Minute
const KernelArgs = "KERNEL_ARGS"
const PanicMsg = "Kernel panic"

var (
	defaultQemuArgs   = map[string]string{}
	defaultKernelArgs = map[string]string{
		"amd64": "console=ttyS0",
		"arm64": "console=ttyAMA0",
	}
	defaultQemuBin = map[string]string{
		"amd64": "qemu-system-x86_64",
		"arm64": "qemu-system-aarch64",
	}
	defaultPermBitsMask = map[string]int{
		"windows": 0o110,
		"linux":   0o000,
		"darwin":  0o000,
	}
)
