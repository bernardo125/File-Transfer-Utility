package main

import (
	"flag"

	"ftu/pkg/cli"
	"ftu/pkg/ui"
)

func main() {
	cliMode := flag.Bool("cli", false, "Run in CLI mode")
	flag.Parse()

	if *cliMode {
		cli.Run()
		return
	}

	ui.Run()
}
