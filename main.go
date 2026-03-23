package main

import (
	"os"

	"github.com/hystak/hystak/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
