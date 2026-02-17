package nanemu

import "time"

const (
	DefaultExecTimeout     = 30 * time.Minute
	KernelArgs             = "KERNEL_ARGS"
	PanicMsg               = "Kernel panic"
	DefaultQemuMemoryBytes = 1 << 27
	MinQemuMemoryBytes     = 1 << 29
	FallbackKernelURI      = "oci://registry-1.docker.io/ebirukov/linux-kernel"
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
)
