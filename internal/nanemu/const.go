package nanemu

import "time"

const DefaultExecTimeout = 10 * time.Second
const KernelArgs = "KERNEL_ARGS"

var (
	defaultQemuArgs = map[string]string{
		"amd64": "-enable-kvm -nodefaults -serial mon:stdio -display none -no-reboot",
		"arm64": "-nodefaults -serial mon:stdio -machine virt -cpu cortex-a53 -nographic -no-reboot",
	}
	defaultKernelArgs = map[string]string{
		"amd64": "console=ttyS0 quiet",
		"arm64": "console=ttyAMA0 cma=0 audit=0 nosmp maxcpus=1 ipv6.disable=1 net.ifnames=0 lsm= acpi=off ima_appraise=off quiet",
	}
)
