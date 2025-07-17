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

//nolint:cyclop
func (g *Grep) matchLine(line []byte, pattern string) (bool, error) {
	var (
		matchFunc  func(r rune) bool
		groupChars = make(map[rune]struct{})
	)

	switch {
	case pattern == `\d`:
		matchFunc = unicode.IsDigit
	case pattern == `\w`:
		matchFunc = func(r rune) bool {
			return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
		}
	case strings.HasPrefix(pattern, "[") && strings.HasSuffix(pattern, "]"):
		for _, char := range strings.Trim(pattern, "[]^") {
			groupChars[char] = struct{}{}
		}

		positive := true

		if strings.HasPrefix(pattern, "[^") {
			positive = false
		}

		matchFunc = func(r rune) bool {
			_, ok := groupChars[r]

			return ok == positive
		}
	case utf8.RuneCountInString(pattern) == 1:
		return bytes.ContainsAny(line, pattern), nil
	default:
		return false, fmt.Errorf("unsupported pattern: %q", pattern)
	}

	n := 0
	for n < len(line) {
		r, size := utf8.DecodeRune(line[n:])
		if matchFunc(r) {
			return true, nil
		}
		n += size
	}

	return false, nil
}
