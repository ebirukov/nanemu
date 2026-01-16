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
		Prompt:          "# ",
		HistoryFile:     filepath.Join(fs.CfgDir(), ".history"),
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})

	if err != nil {
		return err
	}

	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil {
			if errors.Is(err, readline.ErrInterrupt) {
				if app.qemuCmd != nil && app.qemuCmd.Process != nil {
					app.qemuCmd.Process.Signal(syscall.SIGINT)
				}

				return err
			}
			if err == io.EOF {
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
