package main

import (
	"debug/buildinfo"
	"fmt"
	"github.com/ebirukov/nanemu/internal/nanemu"
	"log"
	"os"
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		path, err := os.Executable()
		if err != nil {
			log.Fatal(err)
		}

		info, err := buildinfo.ReadFile(path)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Version: %s\n", info.Main.Version)

		return
	}

	app := nanemu.NewApp()

	if err := app.Init(); err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}

	if err := app.Run(); err != nil {
		log.Fatalf("failed to run app: %v", err)
	}

	if app.ExecuteError() != nil {
		log.Fatalf("complete with error: %v", app.ExecuteError())
	}
}
