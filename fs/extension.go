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

const (
	cfgDirName    = ".nanemu"
	nanemuEnvName = "NANEMU_PATH"
)

var once sync.Once

func CfgDir() string {
	once.Do(func() {
		path := cfgDirName
		var err error
		nanemuPath, ex := os.LookupEnv(nanemuEnvName)
		if ex {
			path = filepath.Join(nanemuPath, cfgDirName)
			err = os.Mkdir(path, 0755)
			if err != nil && !os.IsExist(err) {
				log.Fatal(fmt.Errorf("can't create nanemu config directory: %w", err))
			}
		}
		if !ex {
			homePath, ex := os.LookupEnv("HOME")
			if ex {
				path = filepath.Join(homePath, cfgDirName)
				err = os.Mkdir(path, 0755)
				if err != nil && !os.IsPermission(err) && !os.IsExist(err) {
					log.Fatal(fmt.Errorf("can't create nanemu config directory: %w", err))
				}
			}
		}
		if os.IsPermission(err) {
			path = cfgDirName
			err = os.Mkdir(path, 0755)
			if err != nil && !os.IsExist(err) {
				log.Fatal(fmt.Errorf("can't create nanemu config directory: %w", err))
			}
		}

		exists := os.IsExist(err)

		if cfgDir, err = filepath.Abs(path); err != nil {
			log.Fatal(fmt.Errorf("can't resolve absolute path %s of .nanemu: %w", path, err))
		}

		log.Println("config path:", cfgDir)

		if exists {
			return
		}

		if err := os.CopyFS(cfgDir, Files); err != nil {
			log.Fatal(fmt.Errorf("can't copy extension files to dir %s: %w", cfgDir, err))
		}
	})

	return cfgDir
}
