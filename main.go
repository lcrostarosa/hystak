package main

import "github.com/lcrostarosa/hystak/internal/cli"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cli.Execute(version, commit, date)
}
