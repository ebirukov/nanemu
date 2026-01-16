package nanemu

import (
	"fmt"
	"github.com/creack/pty"
	"golang.org/x/term"
	"os"
)

func (app *App) openPty() (*os.File, error) {
	fd := int(os.Stdin.Fd())
	oldState, err := term.GetState(fd)
	if err != nil {
		return nil, fmt.Errorf("cannot get terminal state: %w", err)
	}

	pty, tty, err := pty.Open()
	if err != nil {
		return nil, err
	}

	app.qemuCmd.Stdin, app.qemuCmd.Stdout = tty, tty

	app.onShutdown = append(app.onShutdown, func(app *App) error {
		pty.Close()
		tty.Close()
		return term.Restore(fd, oldState)
	})

	return pty, nil

}
