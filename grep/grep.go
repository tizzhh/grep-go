package grep

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"unicode/utf8"
)

const (
	statusCodeOK = iota
	statusCodeNotFound
	statusCodeErr
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
	status, err := g.run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Grep failed: %v\n", err)
		os.Exit(2)
	}

	os.Exit(status)
}

func (g *Grep) run() (int, error) {
	if len(g.args) < 3 || g.args[1] != "-E" {
		return statusCodeErr, fmt.Errorf("usage: mygrep -E <pattern>")
	}

	pattern := g.args[2]

	line, err := io.ReadAll(g.r)
	if err != nil {
		return statusCodeErr, fmt.Errorf("read input text: %w", err)
	}

	ok, err := g.matchLine(line, pattern)
	if err != nil {
		return statusCodeErr, fmt.Errorf("matchLine: %w", err)
	}

	if !ok {
		return statusCodeNotFound, nil
	}

	return statusCodeOK, nil
}

func (g *Grep) matchLine(line []byte, pattern string) (bool, error) {
	var finalPattern string

	switch {
	case pattern == `\d`:
		finalPattern = "0123456789"
	case utf8.RuneCountInString(pattern) == 1:
		finalPattern = pattern
	default:
		return false, fmt.Errorf("unsupported pattern: %q", pattern)
	}

	return bytes.ContainsAny(line, finalPattern), nil
}
