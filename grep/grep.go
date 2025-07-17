package grep

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"unicode/utf8"
)

type Grep struct {
	args []string
	r    io.Reader
}

func NewGrep(args []string, r io.Reader) *Grep {
	return &Grep{
		args: args,
		r:    r,
	}
}

func (g *Grep) Run() {
	if len(g.args) < 3 || g.args[1] != "-E" {
		g.reportErrAndExit("usage: mygrep -E <pattern>\n")
	}

	pattern := g.args[2]

	line, err := io.ReadAll(g.r)
	if err != nil {
		g.reportErrAndExit(fmt.Sprintf("Failed to read input text: %v\n", err))
	}

	ok, err := g.matchLine(line, pattern)
	if err != nil {
		g.reportErrAndExit(fmt.Sprintf("Failed to match: %v\n", err))
	}

	if !ok {
		os.Exit(1)
	}
}

func (g *Grep) reportErrAndExit(msg string) {
	fmt.Fprint(os.Stderr, msg)
	os.Exit(2)
}

func (g *Grep) matchLine(line []byte, pattern string) (bool, error) {
	if utf8.RuneCountInString(pattern) != 1 {
		return false, fmt.Errorf("unsupported pattern: %q", pattern)
	}

	return bytes.ContainsAny(line, pattern), nil
}
