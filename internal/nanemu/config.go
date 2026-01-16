package nanemu

import (
	"flag"
	"fmt"
	"github.com/ebirukov/nanemu/internal/initrd"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Arch           string
	QemuBin        string
	QemuCfgArgs    string
	QemuExtCfgArgs *ExtArgs
	KernelArgs     string
	KernelPath     string
	RootFSPath     string
	InitCmdArgs    []string
	ExecTimeout    time.Duration
	FailOnPanic    bool
	Loglevel       string
	Serial         string
	Terminal       bool
	Interactive    bool

	Memory string
	Smp    string

	initrdFile *initrd.ImageFile
	localDir   string
}

func (cfg *Config) Parse(args []string) error {
	if cfg == nil {
		*cfg = Config{}
	}

	fs := flag.NewFlagSet(filepath.Base(args[0]), flag.ExitOnError)
	cfg.QemuExtCfgArgs = NewExtFlags(fs)

	var showHelp bool
	fs.BoolVar(&showHelp, "h", false, "show help")
	fs.StringVar(&cfg.Arch, "arch", runtime.GOARCH, "Platform architecture")
	fs.StringVar(&cfg.KernelPath, "kernel", getEnv("KERNEL_PATH", ""), "Path to linux kernel image (default $KERNEL_PATH)")
	fs.StringVar(&cfg.RootFSPath, "rootfs", "", "Deprecated. Path to initramfs root directory")
	fs.BoolVar(&cfg.FailOnPanic, "fail-on-panic", true, "Fail on kernel panic")
	fs.DurationVar(&cfg.ExecTimeout, "timeout", 0, "Max time of qemu execution")
	fs.StringVar(&cfg.Memory, "memory", "", "Memory limit")
	fs.StringVar(&cfg.Smp, "smp", "", "Cpus limit")
	fs.StringVar(&cfg.Serial, "s", "mon:stdio", "serial port to host device, standard io by default")
	fs.StringVar(&cfg.Loglevel, "loglevel", "3", "kernel log level")
	fs.BoolVar(&cfg.Terminal, "t", false, "create pseudo tty")
	fs.BoolVar(&cfg.Interactive, "i", false, "terminal interaction")

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: \n nanemu [Options] -kernel /path/to/linux/kernel /path/to/rootfs \n")
		fmt.Fprintf(fs.Output(), "Options:\n")
		fs.PrintDefaults()
	}

	if err := cfg.QemuExtCfgArgs.Init(cfg.localDir); err != nil {
		return fmt.Errorf("init qemu args: %w", err)
	}

	if err := fs.Parse(args[1:]); err != nil {
		return fmt.Errorf("can't parse command flags: %w", err)
	}

	if showHelp {
		fs.Usage()
		os.Exit(0)
	}

	switch cfg.Arch {
	case "amd64", "arm64":
	default:
		return fmt.Errorf("unsupported architecture: %s", cfg.Arch)
	}

	if cfg.KernelPath == "" {
		fmt.Fprintf(fs.Output(), "kernel path not specified. set env KERNEL_PATH or use option -kernel \n")

		os.Exit(0)
	}

	kernelArch, _ := checkKernelArch(cfg.KernelPath)
	if kernelArch != "" && kernelArch != cfg.Arch {
		fmt.Fprintf(fs.Output(), "\033[31mperhaps kernel image %s compile for %s; qemu expected arch: %s; use -arch opt\n\033[0m", filepath.Base(cfg.KernelPath), kernelArch, cfg.Arch)
	}

	// -rootfs flag backward support
	if cfg.RootFSPath == "" && len(fs.Args()) == 0 {
		fmt.Fprintf(fs.Output(), "rootfs path not specified\n")

		fs.Usage()

		os.Exit(0)
	}

	if len(fs.Args()) > 0 {
		cfg.RootFSPath, cfg.InitCmdArgs = fs.Args()[0], fs.Args()[1:]
	}

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

func hasField(fields []string, prefix string) bool {
	for _, f := range fields {
		if strings.EqualFold(f, prefix) {
			return true
		}
	}

	return false
}

func (cfg *Config) BuildCmdArgs() ([]string, error) {
	defaultQemuBin := defaultQemuBin[cfg.Arch]
	switch runtime.GOOS {
	case "windows":
		defaultQemuBin = defaultQemuBin + "w.exe"
	}

	cfg.QemuBin = getEnv("QEMU_BIN", defaultQemuBin)
	cfg.KernelArgs = getEnv(KernelArgs, defaultKernelArgs[cfg.Arch])
	cfg.QemuCfgArgs = getEnv("QEMU_ARGS", defaultQemuArgs[cfg.Arch])

	kernelArgs := strings.Fields(cfg.KernelArgs)
	// add console UART for inspect kernel dmesg write to stdout
	if cfg.FailOnPanic && !hasFieldPrefix(kernelArgs, "console=") {
		kernelArgs = append(kernelArgs, defaultKernelArgs[cfg.Arch])
	}

	kernelArgs = append(kernelArgs, "panic=-1")

	if cfg.Loglevel != "" && !hasFieldPrefix(kernelArgs, "loglevel=") && !hasField(kernelArgs, "quiet") {
		kernelArgs = append(kernelArgs, "loglevel="+cfg.Loglevel)
	}

	if !hasFieldPrefix(kernelArgs, "PATH=") {
		kernelArgs = append(kernelArgs, "PATH=/")
	}

	info, _ := os.Stat(cfg.RootFSPath)
	var initFile string
	if !info.IsDir() {
		initFile = filepath.Base(cfg.RootFSPath)
	}

	if info.IsDir() {
		entries, err := os.ReadDir(cfg.RootFSPath)
		if err != nil {
			return nil, fmt.Errorf("can't read rootfs directory: %w", err)
		}
		var fileEntry []os.DirEntry
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			fileEntry = append(fileEntry, entry)
		}
		if len(fileEntry) == 1 && fileEntry[0].Name() != "init" {
			initFile = filepath.Base(fileEntry[0].Name())
		}
	}

	if !hasFieldPrefix(kernelArgs, "rdinit=") && !hasFieldPrefix(kernelArgs, "init=") {
		if len(cfg.InitCmdArgs) > 0 || len(initFile) > 0 {
			rdinit := fmt.Sprintf("%s %s", initFile, strings.Join(cfg.InitCmdArgs, " "))
			kernelArgs = append(kernelArgs, "rdinit="+strings.TrimSpace(rdinit))
		}
	}

	vmArgs := append(cfg.QemuExtCfgArgs.Args(), strings.Fields(cfg.QemuCfgArgs)...)

	if cfg.Memory == "" && cfg.initrdFile.Size() > DefaultQemuMemoryBytes {
		memoryKb := int(cfg.initrdFile.Size()+MinQemuMemoryBytes) / 1024 / 1024
		cfg.Memory = strconv.Itoa(memoryKb+1) + "M"
	}

	if cfg.Memory != "" && !hasField(vmArgs, "-m") {
		vmArgs = append(vmArgs, "-m", cfg.Memory)
	}

	if !hasField(vmArgs, "-serial") {
		vmArgs = append(vmArgs, "-serial", cfg.Serial)
	}

	if cfg.Smp != "" && !hasField(vmArgs, "-smp") {
		vmArgs = append(vmArgs, "-smp", cfg.Smp)
	}

	// accelerate hypervisor by default
	if cfg.Arch == runtime.GOARCH {
		switch runtime.GOOS {
		case "darwin":
			vmArgs = append(vmArgs, "-accel", "hvf")
		case "linux":
			vmArgs = append(vmArgs, "-enable-kvm")
		}
	}

	vmArgs = append(vmArgs,
		"-append", strings.Join(kernelArgs, " "),
		"-kernel", cfg.KernelPath,
		"-initrd", cfg.initrdFile.Path())

	return vmArgs, nil
}

func getEnv(key, defaultVal string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return defaultVal
}
