package nanemu

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Config struct {
	Arch        string
	QemuBin     string
	QemuCfgArgs string
	KernelArgs  string
	KernelPath  string
	RootFSPath  string
	ExecTimeout time.Duration
	FailOnPanic bool

	initrdFile string
}

func (cfg *Config) Parse(args []string) error {
	if cfg == nil {
		*cfg = Config{}
	}

	fs := flag.NewFlagSet(filepath.Base(args[0]), flag.ExitOnError)

	fs.StringVar(&cfg.Arch, "arch", runtime.GOARCH, "Platform architecture")
	fs.StringVar(&cfg.KernelPath, "kernel", getEnv("KERNEL_PATH", ""), "Path to linux kernel image")
	fs.StringVar(&cfg.RootFSPath, "rootfs", "", "Path to initramfs root directory")
	fs.BoolVar(&cfg.FailOnPanic, "fail-on-panic", true, "Fail on kernel panic")
	fs.DurationVar(&cfg.ExecTimeout, "timeout", DefaultExecTimeout, "Max time of qemu execution")

	if err := fs.Parse(args[1:]); err != nil {
		return fmt.Errorf("can't parse command flags: %w", err)
	}

	if cfg.RootFSPath == "" || cfg.KernelPath == "" {
		fs.Usage()

		os.Exit(1)
	}

	switch cfg.Arch {
	case "amd64", "arm64":
	default:
		return fmt.Errorf("unsupported architecture: %s", cfg.Arch)
	}

	cfg.QemuBin = getEnv("QEMU_BIN", fmt.Sprintf("qemu-system-%s", cfg.Arch))
	cfg.KernelArgs = getEnv(KernelArgs, defaultKernelArgs[cfg.Arch])
	cfg.QemuCfgArgs = getEnv("QEMU_ARGS", defaultQemuArgs[cfg.Arch])

	return nil
}

func hasFieldPrefix(fields []string, prefix string) bool {
	for _, f := range fields {
		if strings.Contains(f, prefix) {
			return true
		}
	}

	return false
}

func (cfg *Config) BuildCmdArgs() ([]string, error) {
	kernelArgs := strings.Fields(cfg.KernelArgs)
	// add console UART for inspect kernel dmesg write to stdout
	if cfg.FailOnPanic && !hasFieldPrefix(kernelArgs, "console=") {
		kernelArgs = append(kernelArgs, defaultKernelArgs[cfg.Arch])
	}

	info, _ := os.Stat(cfg.RootFSPath)
	if !info.IsDir() {
		kernelArgs = append(kernelArgs, "rdinit="+filepath.Base(cfg.RootFSPath))
	}

	vmArgs := strings.Fields(cfg.QemuCfgArgs)
	switch runtime.GOOS {
	case "darwin":
		vmArgs = append(vmArgs, "-accel", "hvf")
	case "linux":
		vmArgs = append(vmArgs, "-enable-kvm")
	}

	vmArgs = append(vmArgs,
		"-append", strings.Join(kernelArgs, " "),
		"-kernel", cfg.KernelPath,
		"-initrd", cfg.initrdFile)

	return vmArgs, nil
}

func getEnv(key, defaultVal string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return defaultVal
}
