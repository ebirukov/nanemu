package nanemu

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const cmdDescTmpl = "additional arguments from file %s"

type ExtArgs struct {
	extCmdFlags map[string]ExtFlag
	fs          *flag.FlagSet
}

func NewExtFlags(fs *flag.FlagSet) *ExtArgs {
	return &ExtArgs{
		extCmdFlags: map[string]ExtFlag{},
		fs:          fs,
	}
}

type ExtFlag struct {
	Tmpls  []string
	Values *string
}

func (a *ExtArgs) Init(dir string) error {
	dir = filepath.Join(dir, "extension")
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("can't read dir %s: %w", dir, err)
	}

	for _, file := range files {
		path := filepath.Join(dir, file.Name())
		if file.IsDir() {
			continue
		}
		cmdFlagName := file.Name()

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("can't open extension file %s: %w", path, err)
		}

		tmpls := strings.Split(string(content), "\n")
		argsCnt := strings.Count(string(content), "%s")

		if argsCnt == 0 {
			a.fs.Bool(cmdFlagName, false, fmt.Sprintf(cmdDescTmpl, path))
			a.extCmdFlags[cmdFlagName] = ExtFlag{
				Tmpls: tmpls,
			}
		}

		if argsCnt > 0 {
			a.extCmdFlags[cmdFlagName] = ExtFlag{
				Tmpls:  tmpls,
				Values: a.fs.String(cmdFlagName, "none", fmt.Sprintf(cmdDescTmpl, path)),
			}
		}
	}

	return nil
}

func (a *ExtArgs) Args() (args []string) {
	if !a.fs.Parsed() {
		log.Fatalf("arguments not parsed")
	}
	a.fs.VisitAll(func(f *flag.Flag) {
		if f.Value.String() == "none" ||
			f.Value.String() == strconv.FormatBool(false) {
			return
		}
		flags, ok := a.extCmdFlags[f.Name]
		if !ok {
			return
		}
		var flagVal []any
		for _, val := range []*string{flags.Values} {
			if val != nil {
				flagVal = append(flagVal, *val)
			}
		}
		n := 0
		for _, tmpl := range flags.Tmpls {
			cnt := strings.Count(tmpl, "%s")
			if len(flagVal) >= n+cnt {
				tmpl = fmt.Sprintf(tmpl, flagVal[n:n+cnt]...)
			}
			n = n + cnt
			args = append(args, strings.Fields(tmpl)...)
		}

	})

	return args
}
