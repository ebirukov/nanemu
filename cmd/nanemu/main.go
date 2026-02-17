package main

import (
	"github.com/ebirukov/nanemu/fs"
	"github.com/ebirukov/nanemu/internal/nanemu"
	"log"
	"os"
)

func main() {
	app := nanemu.NewApp(fs.CfgDir())

	_, err := app.ParseCfg(os.Args)
	if err != nil {
		log.Fatalf("can't parse config: %v", err)
	}

	if err := app.Init(); err != nil {
		app.Shutdown()

		log.Fatalf("failed to initialize app: %v", err)
	}

	if err := app.Run(); err != nil {
		log.Fatalf("failed to run app: %v", err)
	}

	if err := app.Error(); err != nil {
		log.Fatalf("complete with error: %v", err)
	}
}
