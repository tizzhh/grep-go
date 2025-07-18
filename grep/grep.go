package grep

import (
	"fmt"
	"io"
	"os"
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

type matchFunc func(r rune, patternIdx *int, pattern []rune) (bool, error)

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

	ok, err := g.matchLine(line, []rune(pattern))
	if err != nil {
		return statusCodeErr, fmt.Errorf("matchLine: %w", err)
	}

	if !ok {
		return statusCodeNotFound, nil
	}

	return statusCodeOK, nil
}

func (g *Grep) matchLine(line []byte, pattern []rune) (bool, error) {
	i := 0
	patternIdx := 0

	for i < len(line) {
		r, size := utf8.DecodeRune(line[i:])
		i += size

		subMatchFound, err := submatch(r, &patternIdx, pattern)
		if err != nil {
			return false, err
		}

		if !subMatchFound {
			patternIdx = 0

			continue
		}

		if patternIdx >= len(pattern) {
			return true, nil
		}
	}

	return false, nil
}

func submatch(r rune, patternIdx *int, pattern []rune) (bool, error) {
	if *patternIdx >= len(pattern) {
		return false, nil
	}

	var matchFunc matchFunc

	switch pattern[*patternIdx] {
	case '\\':
		matchFunc = matchEscape
	case '[':
		matchFunc = matchCharGroup
	default:
		matchFunc = matchSingleChar
	}

	return matchFunc(r, patternIdx, pattern)
}

func matchEscape(r rune, patternIdx *int, pattern []rune) (bool, error) {
	if *patternIdx+1 >= len(pattern) {
		return false, nil
	}

	switch pattern[*patternIdx+1] {
	case 'd':
		if unicode.IsDigit(r) {
			*patternIdx += 2

			return true, nil
		}
	case 'w':
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			*patternIdx += 2

			return true, nil
		}
	}

	return false, nil
}

func matchCharGroup(r rune, patternIdx *int, pattern []rune) (bool, error) {
	groupChars := make(map[rune]struct{})

	*patternIdx++

	positive := true
	if pattern[*patternIdx] == '^' {
		positive = false
		*patternIdx++
	}

	for ; *patternIdx < len(pattern) && pattern[*patternIdx] != ']'; *patternIdx++ {
		groupChars[pattern[*patternIdx]] = struct{}{}
	}

	if *patternIdx >= len(pattern) || pattern[*patternIdx] != ']' {
		return false, fmt.Errorf("brackets [] not balanced")
	}
	*patternIdx++

	_, ok := groupChars[r]

	return ok == positive, nil
}

func matchSingleChar(r rune, patternIdx *int, pattern []rune) (bool, error) {
	if r == pattern[*patternIdx] {
		*patternIdx++

		return true, nil
	}

	return false, nil
}
