package main

import (
	"os"

	"github.com/codecrafters-io/grep-starter-go/grep"
)

func main() {
	g := grep.NewGrep(os.Args, os.Stdin)
	g.Run()
}
