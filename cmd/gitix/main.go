package main

import (
	"fmt"
	"os"
	"time"

	"github.com/hbeanjarah/gitix/internal/cli"
	"github.com/hbeanjarah/gitix/internal/ui"
)

// Set at build time by GoReleaser.
var (
	version = "dev"
	date    = "unknown"
)

func main() {
	opts := cli.Options{
		In:          os.Stdin,
		Out:         os.Stdout,
		Err:         os.Stderr,
		Interactive: ui.IsTerminal(os.Stdin) && ui.IsTerminal(os.Stdout),
		Paint:       ui.NewPainter(os.Stdout),
		Now:         time.Now,
		Version:     fmt.Sprintf("%s (built %s)", version, date),
	}
	os.Exit(cli.Run(opts, os.Args[1:]))
}
