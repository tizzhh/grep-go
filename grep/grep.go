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
	if len(pattern) == 0 {
		return true, nil
	}

	i := 0
	patternIdx := 0

	for i < len(line) {
		r, _ := utf8.DecodeRune(line[i:])

		subMatchFound, err := submatch(r, &i, line, &patternIdx, pattern)
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

	if patternIdx < len(pattern) && pattern[patternIdx] == '$' {
		return true, nil
	}

	return false, nil
}

func submatch(r rune, lineIdx *int, _ []byte, patternIdx *int, pattern []rune) (bool, error) {
	if *patternIdx >= len(pattern) {
		return false, nil
	}

	switch pattern[*patternIdx] {
	case '\\':
		return matchEscape(r, lineIdx, patternIdx, pattern)
	case '[':
		return matchCharGroup(r, lineIdx, patternIdx, pattern)
	case '^':
		return matchStartOfStringAnchor(r, lineIdx, patternIdx)
	case '$':
		return matchEndOfStringAnchor(r, lineIdx)
	default:
		return matchSingleChar(r, lineIdx, patternIdx, pattern)
	}
}

func matchEscape(r rune, lineIdx *int, patternIdx *int, pattern []rune) (bool, error) {
	if *patternIdx+1 >= len(pattern) {
		return false, nil
	}

	defer advanceIndex(r, lineIdx)

	matchFound := false

	switch pattern[*patternIdx+1] {
	case 'd':
		if unicode.IsDigit(r) {
			matchFound = true
		}
	case 'w':
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			matchFound = true
		}
	}

	if matchFound {
		*patternIdx += 2

		return true, nil
	}

	return false, nil
}

func matchCharGroup(r rune, lineIdx *int, patternIdx *int, pattern []rune) (bool, error) {
	defer advanceIndex(r, lineIdx)

	*patternIdx++

	positive := true
	if pattern[*patternIdx] == '^' {
		positive = false
		*patternIdx++
	}

	groupChars := make(map[rune]struct{})
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

func matchStartOfStringAnchor(r rune, lineIdx *int, patternIdx *int) (bool, error) {
	switch {
	case *patternIdx == 0 && *lineIdx == 0:
		*patternIdx++

		return true, nil
	case *patternIdx == 0 && r == '\n':
		*patternIdx++
		advanceIndex(r, lineIdx)

		return true, nil
	default:
		advanceIndex(r, lineIdx)

		return false, nil
	}
}

func matchEndOfStringAnchor(r rune, lineIdx *int) (bool, error) {
	advanceIndex(r, lineIdx)

	return false, nil
}

func matchSingleChar(r rune, lineIdx *int, patternIdx *int, pattern []rune) (bool, error) {
	defer advanceIndex(r, lineIdx)

	if r == pattern[*patternIdx] {
		*patternIdx++

		return true, nil
	}

	return false, nil
}

func advanceIndex(r rune, lineIdx *int) {
	*lineIdx += utf8.RuneLen(r)
}
