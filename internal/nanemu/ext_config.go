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

const cmdDescTmpl = "flag source %s"

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
	Desc   string
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

		lines := strings.Split(string(content), "\n")
		if len(lines) == 0 {
			return fmt.Errorf("can't parse extension file %s: %w", path, err)
		}
		var flags = ExtFlag{}
		desc := fmt.Sprintf(cmdDescTmpl, path)
		for _, l := range lines {
			if strings.HasPrefix(l, "#") || strings.HasPrefix(l, "//") {
				desc = strings.Join([]string{desc, strings.TrimLeft(l, "//#")}, "\n")
				continue
			}
			flags.Tmpls = append(flags.Tmpls, l)
		}

		argsCnt := strings.Count(string(content), "%s")

		if argsCnt == 0 {
			a.fs.Bool(cmdFlagName, false, desc)
		}

		if argsCnt > 0 {
			flags.Values = a.fs.String(cmdFlagName, "none", desc)
		}

		a.extCmdFlags[cmdFlagName] = flags
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
