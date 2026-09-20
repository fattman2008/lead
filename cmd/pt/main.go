package main

import (
	"os"

	"github.com/fattman2008/lead/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
