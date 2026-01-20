package nanemu

import (
	"errors"
	"fmt"
	"github.com/chzyer/readline"
	"github.com/ebirukov/nanemu/fs"
	"io"
	"path/filepath"
	"syscall"
)

func (app *App) interactive(out io.Writer) error {
	rl, err := readline.NewEx(&readline.Config{
		HistoryFile:     filepath.Join(fs.CfgDir(), ".history"),
		InterruptPrompt: "^C",
	})

	if err != nil {
		return err
	}

	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil {
			if errors.Is(err, readline.ErrInterrupt) || errors.Is(err, io.EOF) {
				if app.qemuCmd != nil && app.qemuCmd.Process != nil {
					app.qemuCmd.Process.Signal(syscall.SIGINT)
				}

				return nil
			}

			return err
		}

		if line == "" {
			continue
		}

		fmt.Fprintln(out, line)
	}
}
