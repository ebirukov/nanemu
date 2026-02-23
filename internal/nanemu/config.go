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
	Version             bool
	Arch                string
	QemuBin             string
	QemuExtCfgArgs      *ExtArgs
	KernelURI           string
	RootFSPath          string
	InitRD              bool
	ExecTimeout         time.Duration
	FailOnPanic         bool
	Serial              string
	Terminal            bool
	Interactive         bool
	KeepDiskImageOnExit bool

	KernelBootParams KernelBootParams

	Memory string
	Smp    string

	localDir string
}

type KernelBootParams struct {
	Env         Environment
	Loglevel    string
	InitCmdArgs []string
}

type Environment []string

func (e *Environment) Set(s string) error {
	if !strings.Contains(s, "=") {
		return fmt.Errorf("invalid format, expected KEY=VALUE")
	}
	*e = append(*e, s)

	return nil
}

func (cfg *Config) Parse(args []string) error {
	if cfg == nil {
		*cfg = Config{}
	}

	opts := flag.NewFlagSet(filepath.Base(args[0]), flag.ExitOnError)
	cfg.QemuExtCfgArgs = NewExtFlags(opts)

	var showHelp bool
	opts.BoolVar(&cfg.Version, "v", false, "version")
	opts.BoolVar(&showHelp, "h", false, "show help")
	opts.StringVar(&cfg.Arch, "arch", runtime.GOARCH, "Platform architecture")
	opts.StringVar(&cfg.KernelURI, "kernel", getEnv("KERNEL_PATH", ""), "Path to linux kernel image (default $KERNEL_PATH)")
	opts.StringVar(&cfg.RootFSPath, "rootfs", "", "Deprecated. Path to initramfs root directory")
	opts.BoolVar(&cfg.InitRD, "initrd", false, "Create initramfs image and use as kernel rootfs")
	opts.BoolVar(&cfg.KeepDiskImageOnExit, "keep-disk", false, "keep disk image file on exit")
	opts.BoolVar(&cfg.FailOnPanic, "fail-on-panic", true, "Fail on kernel panic")
	opts.DurationVar(&cfg.ExecTimeout, "timeout", 0, "Max time of qemu execution")
	opts.StringVar(&cfg.Memory, "memory", "", "Memory limit")
	opts.StringVar(&cfg.Smp, "smp", "", "Cpus limit")
	opts.StringVar(&cfg.Serial, "s", "mon:stdio", "serial port to host device, standard io by default")
	opts.StringVar(&cfg.KernelBootParams.Loglevel, "loglevel", "3", "kernel log level")
	opts.BoolVar(&cfg.Terminal, "t", false, "create pseudo tty")
	opts.BoolVar(&cfg.Interactive, "i", runtime.GOOS != "windows", "terminal interaction")
	opts.Func("e", "set environment variables. format KEY=VALUE", cfg.KernelBootParams.Env.Set)

	opts.Usage = func() {
		fmt.Fprintf(opts.Output(), "Usage: \n nanemu [Options] -kernel /path/to/linux/kernel /path/to/rootfs \n")
		fmt.Fprintf(opts.Output(), "Options:\n")
		opts.PrintDefaults()
	}

	if err := cfg.QemuExtCfgArgs.Init(cfg.localDir); err != nil {
		return fmt.Errorf("init qemu args: %w", err)
	}

	if err := opts.Parse(args[1:]); err != nil {
		return fmt.Errorf("can't parse command flags: %w", err)
	}

	if cfg.Version {
		info, err := BuildInfo()
		if err != nil {
			return fmt.Errorf("can't read build info: %w", err)
		}

		fmt.Printf("Version: %s\n", info.Main.Version)

		os.Exit(0)
	}

	if showHelp {
		opts.Usage()
		os.Exit(0)
	}

	switch cfg.Arch {
	case "amd64", "arm64":
	default:
		fmt.Fprintf(opts.Output(), "unsupported architecture: %s", cfg.Arch)
		opts.Usage()

		os.Exit(0)
	}

	if cfg.KernelURI == "" {
		cfg.KernelURI = FallbackKernelURI
	}

	// -rootfs flag backward support
	if cfg.RootFSPath == "" && len(opts.Args()) == 0 {
		fmt.Fprintf(opts.Output(), "rootfs path not specified\n")

		opts.Usage()

		os.Exit(0)
	}

	if len(opts.Args()) > 0 {
		cfg.RootFSPath, cfg.KernelBootParams.InitCmdArgs = opts.Args()[0], opts.Args()[1:]
	}

	defaultQemuBin := defaultQemuBin[cfg.Arch]
	switch runtime.GOOS {
	case "windows":
		defaultQemuBin = defaultQemuBin + "w.exe"
	}

	cfg.QemuBin = getEnv("QEMU_BIN", defaultQemuBin)

	return nil
}
