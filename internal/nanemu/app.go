package nanemu

import (
	"context"
	"errors"
	"fmt"
	"github.com/ebirukov/nanemu/internal/diskimg"
	"github.com/ebirukov/nanemu/internal/resource"
	"github.com/ebirukov/nanemu/internal/text"
	"golang.org/x/term"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

type App struct {
	config     *Config
	qemuArgs   []string
	qemuCmd    *exec.Cmd
	ctx        context.Context
	cancel     context.CancelFunc
	stopSig    chan os.Signal
	stop       chan struct{}
	onShutdown []func(app *App) error
	err        error
}

func NewApp(cfgDir string) *App {
	return &App{
		config: &Config{
			localDir: cfgDir,
		},
		stop: make(chan struct{}),
	}
}

func (app *App) ParseCfg(args []string) (cfg *Config, err error) {
	if err := app.config.Parse(args); err != nil {
		return nil, err
	}

	return app.config, nil
}

func (app *App) Init() (err error) {
	if app.config.ExecTimeout > 0 {
		app.ctx, app.cancel = context.WithTimeout(context.Background(), app.config.ExecTimeout)
	} else {
		app.ctx, app.cancel = context.WithCancel(context.Background())
	}

	r := resource.DefaultFetcher
	r.AddFetcher("oci", resource.NewOCIResolver(app.config.Arch))
	r.AddFetcher("https", resource.NewHTTPFetcher(app.config.Arch))
	r.AddFetcher("docker", resource.NewDockerResolver(app.config.Arch))

	_, err = os.Stat(app.config.RootFSPath)
	if err != nil {
		app.config.RootFSPath, err = r.FetchPath(app.config.RootFSPath)
		if err != nil {
			return fmt.Errorf("can't resolve rootfs path: %w", err)
		}
	}

	if app.config.InitRD {
		app.config.rootFSFile, err = diskimg.CreateInitRDImage(app.config.RootFSPath)
		if err != nil {
			return fmt.Errorf("could not create initrd: %w", err)
		}
	} else {
		app.config.rootFSFile, err = diskimg.CreateHardDiskImage(app.config.RootFSPath)
		if err != nil {
			return fmt.Errorf("could not create initrd: %w", err)
		}
	}

	defer app.config.rootFSFile.Close()

	if !app.config.KeepDiskImageOnExit {
		app.onShutdown = append(app.onShutdown, func(app *App) error {
			return app.config.rootFSFile.Remove()
		})
	}

	if app.qemuArgs, err = app.config.BuildCmdArgs(); err != nil {
		return fmt.Errorf("can't build command args: %w", err)
	}

	kernelPath, err := r.FetchPath(app.config.KernelURI)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("can't resolve kernel path: %w", err)
		}

		fmt.Fprintf(os.Stderr, "\u001B[31mcan't get kernel from uri: %s\n\u001B[0m", err)
	}

	if kernelPath == "" {
		return fmt.Errorf("kernel path not specified. set env KERNEL_PATH or use option -kernel")
	}

	kernelPath, err = r.FetchPath(kernelPath)
	if err != nil {
		return fmt.Errorf("can't resolve kernel path '%s': %w", kernelPath, err)
	}

	kernelArch, _ := checkKernelArch(kernelPath)
	if kernelArch != "" && kernelArch != app.config.Arch {
		fmt.Fprintf(os.Stderr, "\033[31mperhaps kernel image %s compile for %s; qemu expected arch: %s; use -arch opt\n\033[0m", filepath.Base(kernelPath), kernelArch, app.config.Arch)
	}

	app.qemuArgs = append(app.qemuArgs, "-kernel", kernelPath)

	return nil
}

func (app *App) Shutdown() {
	for _, shutdown := range app.onShutdown {
		if err := shutdown(app); err != nil {
			log.Printf("WARNING: shutdown app failed: %v", err)
		}
	}
}

func (app *App) HandleKernelPanic(mention string) {
	app.qemuCmd.Process.Signal(syscall.SIGINT)

	app.err = fmt.Errorf("kernel panic: %s", mention)
}

func (app *App) Error() error {
	return app.err
}

func (app *App) Run() error {
	defer func() {
		app.cancel()
		<-app.stop
	}()

	app.qemuCmd = exec.CommandContext(app.ctx, app.config.QemuBin, app.qemuArgs...)

	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		app.stopSig = make(chan os.Signal, 1)
		defer close(app.stopSig)

		signal.Notify(app.stopSig, os.Interrupt, syscall.SIGTERM, syscall.SIGABRT)

		go func() {
			select {
			case sig := <-app.stopSig:
				app.qemuCmd.Process.Signal(sig)
			}

			app.cancel()
		}()

		setPlatformProcAttr(app.qemuCmd)
	}

	go func() {
		<-app.ctx.Done()
		app.Shutdown()
		close(app.stop)
	}()

	app.qemuCmd.Stderr = os.Stderr

	if !app.config.Terminal {
		cmdStdIn, err := app.qemuCmd.StdinPipe()
		if err != nil {
			return fmt.Errorf("can't get stdin pipe: %w", err)
		}

		defer cmdStdIn.Close()

		if !app.config.Interactive {
			go io.Copy(cmdStdIn, os.Stdin)
		} else {
			fd := int(os.Stdin.Fd())
			oldState, err := term.GetState(fd)
			if err != nil {
				return fmt.Errorf("cannot get terminal state: %w", err)
			}

			defer term.Restore(fd, oldState)

			go app.interactive(cmdStdIn)
		}

		cmdStdOut, err := app.qemuCmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("can't get stdout pipe: %w", err)
		}

		defer cmdStdOut.Close()

		if app.config.FailOnPanic {
			cmdStdOut = io.NopCloser(text.NewLineMatchReader(cmdStdOut, PanicMsg, app.HandleKernelPanic))
		}

		go io.Copy(log.Writer(), cmdStdOut)
	}

	if app.config.Terminal {
		pty, err := app.openPty()
		if err != nil {
			return fmt.Errorf("failed to open pty: %w", err)
		}

		if app.config.Interactive {
			go app.interactive(pty)
		} else {
			go io.Copy(pty, os.Stdin)
		}

		go io.Copy(os.Stdout, pty)
	}

	printCmd(app.qemuCmd)

	if err := app.qemuCmd.Start(); err != nil {
		return fmt.Errorf("could not start process: %v", err)
	}

	proc := app.qemuCmd.Process

	state, err := proc.Wait()
	if err != nil {
		return fmt.Errorf("could not complete process: %v; state %s", err, state)
	}

	switch state := state.Sys().(type) {
	case syscall.WaitStatus:
		if state.Signaled() {
			if errors.Is(app.ctx.Err(), context.DeadlineExceeded) {
				return fmt.Errorf("process %s [%d] terminated by timeout. For more details to use qemu arg '-serial mon:stdio'", app.qemuCmd, proc.Pid)
			}

			return fmt.Errorf("process %s [%d] terminated by signal: %s", app.qemuCmd, proc.Pid, state.Signal())
		}
	}

	log.Printf("exit with code: %d", state.ExitCode())

	return nil
}

func printCmd(cmd *exec.Cmd) {
	var b strings.Builder
	b.WriteString(cmd.Path)
	for _, a := range cmd.Args[1:] {
		if strings.HasPrefix(a, "-") {
			b.WriteString(" \\")
			b.WriteByte('\n')
		}
		b.WriteByte('\t')
		b.WriteString(a)
	}

	log.Printf("start: %v", b.String())
}
