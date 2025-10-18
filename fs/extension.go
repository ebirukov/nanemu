package fs

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

//go:embed extension/**
var Files embed.FS
var cfgDir string

const cfgDirName = ".nanemu"

var once sync.Once

func CfgDir() string {
	once.Do(func() {
		err := os.Mkdir(cfgDirName, 0755)
		if err != nil && !os.IsExist(err) {
			log.Fatal(fmt.Errorf("can't create nanemu directory: %w", err))
		}

		exists := os.IsExist(err)

		if cfgDir, err = filepath.Abs(cfgDirName); err != nil {
			log.Fatal(fmt.Errorf("can't resolve absolute path of .nanemu: %w", err))
		}

		if exists {
			return
		}

		if err := os.CopyFS(cfgDir, Files); err != nil {
			log.Fatal(fmt.Errorf("can't copy extension files to dir %s: %w", cfgDir, err))
		}
	})

	return cfgDir
}
