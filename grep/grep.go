package grep

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
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
	switch {
	case pattern == `\d`:
		return g.matchDigit(line), nil
	case pattern == `\w`:
		return g.matchAlphaNumeric(line), nil
	case strings.HasPrefix(pattern, "[") && strings.HasSuffix(pattern, "]"):
		return g.matchGroup(line, true, strings.Trim(pattern, "[]^")), nil
	case utf8.RuneCountInString(pattern) == 1:
		return bytes.ContainsAny(line, pattern), nil
	default:
		return false, fmt.Errorf("unsupported pattern: %q", pattern)
	}
}

func (g *Grep) matchDigit(line []byte) bool {
	n := 0
	for n < len(line) {
		r, size := utf8.DecodeRune(line[n:])
		if unicode.IsDigit(r) {
			return true
		}
		n += size
	}

	return false
}

func (g *Grep) matchAlphaNumeric(line []byte) bool {
	n := 0
	for n < len(line) {
		r, size := utf8.DecodeRune(line[n:])
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			return true
		}
		n += size
	}

	return false
}

func (g *Grep) matchGroup(line []byte, positive bool, chars string) bool {
	groupChars := make(map[rune]struct{})
	for _, char := range chars {
		groupChars[char] = struct{}{}
	}

	n := 0
	for n < len(line) {
		r, size := utf8.DecodeRune(line[n:])
		_, ok := groupChars[r]
		if ok && positive || !ok && !positive {
			return true
		}
		n += size
	}

	return false
}
