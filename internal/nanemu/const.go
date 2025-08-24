package nanemu

import "time"

const DefaultExecTimeout = 10 * time.Second
const KernelArgs = "KERNEL_ARGS"
const PanicMsg = "Kernel panic"

var (
	defaultQemuArgs = map[string]string{
		"amd64": "-serial mon:stdio -machine pc -cpu max -display none",
		"arm64": "-serial mon:stdio -machine virt -cpu cortex-a53 -display none",
	}
	defaultKernelArgs = map[string]string{
		"amd64": "console=ttyS0",
		"arm64": "console=ttyAMA0",
	}
)
