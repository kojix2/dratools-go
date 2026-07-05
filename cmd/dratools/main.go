package main

import (
	"os"

	"github.com/kojix2/dratools-go/internal/dratools"
)

func main() {
	os.Exit(dratools.Main(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
