package main

import (
	"github.com/ebirukov/nanemu/fs"
	"github.com/ebirukov/nanemu/internal/config"
	"github.com/ebirukov/nanemu/internal/nanemu"
	"log"
	"os"
)

func main() {
	defaults := config.Default{
		LocalDir:   fs.CfgDir(),
		RootFSPath: "oci://registry-1.docker.io/library/alpine",
		KernelURI:  "oci://registry-1.docker.io/ebirukov/linux-kernel",
	}

	app := nanemu.NewApp(defaults)

	cfg, err := app.ParseCfg(os.Args)
	if err != nil {
		log.Fatalf("can't parse config: %v", err)
	}

	if err := app.Init(cfg); err != nil {
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
