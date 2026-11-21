package main

import (
	"embed"
	"flag"

	"ftu/pkg/cli"
	"ftu/pkg/ui"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	cliMode := flag.Bool("cli", false, "Run in CLI mode")
	flag.Parse()

	if *cliMode {
		cli.Run()
		return
	}

	ui.RunWithAssets(assets)
}
