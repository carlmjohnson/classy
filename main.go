package main

import (
	"os"

	"github.com/carlmjohnson/classy/extract"
	"github.com/carlmjohnson/exitcode"
)

func main() {
	exitcode.Exit(extract.CLI(os.Args[1:]))
}
