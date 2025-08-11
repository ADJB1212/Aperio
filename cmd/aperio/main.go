package main

import (
	"os"

	"github.com/ADJB1212/Aperio/internal/run"
)

var version = "dev" // will be overridden at build time

func main() {
	os.Exit(run.Run(version))
}
