package main

import (
	"os"

	"feedctl/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
