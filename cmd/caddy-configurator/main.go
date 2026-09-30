package main

import (
	"os"

	"caddy-configurator/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
