package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kingpin/v2"

	"github.com/MaxCaribe/library-go/internal/config"
	"github.com/MaxCaribe/library-go/internal/server"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.InitConfig()
	kingpin.Parse()

	application, err := server.New(*cfg)
	if err != nil {
		return err
	}

	return application.Start()
}
