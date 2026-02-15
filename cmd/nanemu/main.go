package main

import (
	"github.com/ebirukov/nanemu/fs"
	"github.com/ebirukov/nanemu/internal/nanemu"
	"github.com/pilat/go-ext4fs"
	"log"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "create-disk" {
		if len(os.Args) < 3 {
			log.Fatalf("Usage: %s create-disk <path> [size MB]\n", os.Args[0])
		}

		var err error
		sizeMB := 512
		if len(os.Args) > 3 {
			sizeMB, err = strconv.Atoi(os.Args[3])
			if err != nil {
				log.Fatalf("can't parse size MB: %v", err)
			}
		}

		img, err := ext4fs.New(
			ext4fs.WithImagePath(os.Args[2]),
			ext4fs.WithSizeInMB(sizeMB),
		)
		if err != nil {
			log.Fatalf("can't create hard disk image file %s: %s\n", os.Args[2], err)
		}

		defer img.Close()

		return
	}

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
