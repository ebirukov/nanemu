package nanemu

import "time"

const (
	DefaultExecTimeout     = 30 * time.Minute
	KernelArgs             = "KERNEL_ARGS"
	PanicMsg               = "Kernel panic"
	DefaultQemuMemoryBytes = 1 << 27
	MinQemuMemoryBytes     = 1 << 29
	FallbackKernelURI      = "oci://docker.io/ebirukov/linux-kernel"
)

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
