package main

import (
	"github.com/ebirukov/nanemu/internal/nanemu"
	"log"
)

func main() {
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
