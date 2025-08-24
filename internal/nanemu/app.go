package nanemu

import (
	"context"
	"errors"
	"fmt"
	"github.com/ebirukov/nanemu/internal/text"
	"github.com/ebirukov/nanemu/pkg/cpio"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
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

func NewApp() *App {
	return &App{
		config: &Config{},
		stop:   make(chan struct{}),
	}
}

func (app *App) Init() (err error) {
	if err = app.config.Parse(os.Args); err != nil {
		return fmt.Errorf("can't parse config: %w", err)
	}

	app.ctx, app.cancel = context.WithTimeout(context.Background(), app.config.ExecTimeout)

	initrdFile, err := os.CreateTemp(".", "initramfs.cpio")
	if err != nil {
		return fmt.Errorf("could not create file initramfs.cpio: %v", err)
	}

	defer initrdFile.Close()

	app.onShutdown = append(app.onShutdown, func(app *App) error {
		return os.Remove(initrdFile.Name())
	})

	if err = cpio.Create(initrdFile, app.config.RootFSPath); err != nil {
		return fmt.Errorf("could not create cpio fs %s: %v", initrdFile.Name(), err)
	}

	app.config.initrdFile = initrdFile.Name()

	if app.qemuArgs, err = app.config.BuildCmdArgs(); err != nil {
		return fmt.Errorf("can't build command args: %w", err)
	}

	return nil
}

func (app *App) shutdown() {
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

func (app *App) ExecuteError() error {
	return app.err
}

func (app *App) Run() error {
	defer func() {
		app.cancel()
		<-app.stop
	}()

	app.stopSig = make(chan os.Signal, 1)
	defer close(app.stopSig)

	signal.Notify(app.stopSig, os.Interrupt, syscall.SIGTERM)

	app.qemuCmd = exec.CommandContext(app.ctx, app.config.QemuBin, app.qemuArgs...)

	go func() {
		select {
		case <-app.ctx.Done():
		case sig := <-app.stopSig:
			app.qemuCmd.Process.Signal(sig)
		}

		app.shutdown()
		close(app.stop)
	}()

	app.qemuCmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGTERM,
	}

	app.qemuCmd.Stderr = os.Stderr

	cmdStdOut, err := app.qemuCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("can't get stdout pipe: %w", err)
	}

	defer cmdStdOut.Close()

	if app.config.FailOnPanic {
		cmdStdOut = io.NopCloser(text.NewLineMatchReader(cmdStdOut, PanicMsg, app.HandleKernelPanic))
	}

	go io.Copy(log.Writer(), cmdStdOut)

	log.Printf("executing: %v", app.qemuCmd)

	if err := app.qemuCmd.Start(); err != nil {
		return fmt.Errorf("could not start process: %v", err)
	}

	proc := app.qemuCmd.Process

	log.Printf("process %v started with pid: %d", app.qemuCmd.Path, proc.Pid)

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
