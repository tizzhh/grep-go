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

type TokenType int

const (
	TokenLiteral TokenType = iota
	TokenEscape
	TokenCharGroup
	TokenAnchorStart
	TokenAnchorEnd
	TokenAny
)

type Grep struct {
	args []string
	r    io.Reader
}

type token struct {
	tType    TokenType
	pattern  []rune
	modifier rune
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
	tokens, err := tokenizePattern(pattern)
	if err != nil {
		return false, err
	}

	for i := 0; i <= len(line); {
		if matchRecursive(i, line, 0, tokens) {
			return true, nil
		}

		if i >= len(line) {
			break
		}

		_, size := utf8.DecodeRune(line[i:])
		i += size
	}

	return false, nil
}

func matchRecursive(lineIdx int, line []byte, tokenIdx int, tokens []*token) bool {
	if tokenIdx >= len(tokens) {
		return true
	}

	if lineIdx >= len(line) {
		if tokens[tokenIdx].tType == TokenAnchorEnd {
			return tokenIdx+1 == len(tokens)
		}

		if tokens[tokenIdx].modifier == 0 {
			return false
		}
	}

	token := tokens[tokenIdx]
	r, _ := utf8.DecodeRune(line[lineIdx:])

	return matchWithModifier(r, lineIdx, line, tokenIdx, token, tokens)
}

func matchWithModifier(
	r rune, lineIdx int, line []byte, tokenIdx int, token *token, tokens []*token,
) bool {
	switch token.modifier {
	case '+':
		return matchOneOrMany(r, lineIdx, line, tokenIdx, tokens)
	case '?':
		return matchZeroOrOne(r, lineIdx, line, tokenIdx, tokens)
	default:
		advance, ok := submatch(r, lineIdx, tokenIdx, tokens)
		if !ok {
			return false
		}

		return matchRecursive(lineIdx+advance, line, tokenIdx+1, tokens)
	}
}

func matchOneOrMany(r rune, lineIdx int, line []byte, tokenIdx int, tokens []*token) bool {
	advance, ok := submatch(r, lineIdx, tokenIdx, tokens)
	if !ok {
		return false
	}

	consumed := lineIdx + advance

	for consumed < len(line) {
		if matchRecursive(consumed, line, tokenIdx+1, tokens) {
			return true
		}
		r, _ = utf8.DecodeRune(line[consumed:])
		advance, ok = submatch(r, consumed, tokenIdx, tokens)
		if !ok {
			break
		}
		consumed += advance
	}

	return false
}

func matchZeroOrOne(r rune, lineIdx int, line []byte, tokenIdx int, tokens []*token) bool {
	advance, ok := submatch(r, lineIdx, tokenIdx, tokens)
	if ok && matchRecursive(lineIdx+advance, line, tokenIdx+1, tokens) {
		return true
	}

	return matchRecursive(lineIdx, line, tokenIdx+1, tokens)
}

//nolint:cyclop,funlen
func tokenizePattern(pattern []rune) ([]*token, error) {
	tokens := []*token{}

	prevToken := &token{}

	for i := 0; i < len(pattern); {
		token := &token{}

		switch pattern[i] {
		case '\\':
			if i+1 >= len(pattern) {
				return nil, fmt.Errorf("incomplete escape sequence")
			}
			token.tType = TokenEscape
			token.pattern = []rune{'\\', pattern[i+1]}
			i += 2
		case '[':
			token.tType = TokenCharGroup
			charGroup := []rune{}

			for ; i < len(pattern) && pattern[i] != ']'; i++ {
				charGroup = append(charGroup, pattern[i])
			}

			if i >= len(pattern) || pattern[i] != ']' {
				return nil, fmt.Errorf("brackets [] not balanced")
			}

			charGroup = append(charGroup, ']')

			i++ // skip ]
			token.pattern = charGroup
		case '^':
			token.tType = TokenAnchorStart
			token.pattern = []rune{'^'}
			i++
		case '$':
			token.tType = TokenAnchorEnd
			token.pattern = []rune{'$'}
			i++
		case '.':
			token.tType = TokenAny
			token.pattern = []rune{'.'}
			i++
		case '+', '?':
			if len(prevToken.pattern) == 0 {
				return nil, fmt.Errorf("repetition-operator operand invalid")
			}

			prevToken.modifier = pattern[i]
			i++

			continue
		default:
			token.tType = TokenLiteral
			token.pattern = []rune{pattern[i]}
			i++
		}

		prevToken = token
		tokens = append(tokens, token)
	}

	return tokens, nil
}

//nolint:cyclop
func submatch(r rune, lineIdx int, tokensIdx int, tokens []*token) (int, bool) {
	if tokensIdx >= len(tokens) || tokens == nil {
		return utf8.RuneLen(r), false
	}

	token := *tokens[tokensIdx]
	if len(token.pattern) == 0 {
		return utf8.RuneLen(r), false
	}

	switch token.tType {
	case TokenEscape:
		return matchEscape(r, token)
	case TokenCharGroup:
		return matchCharGroup(r, token)
	case TokenAnchorStart:
		return matchStartOfStringAnchor(r, lineIdx, tokensIdx)
	case TokenAnchorEnd:
		return matchEndOfStringAnchor(r)
	case TokenAny:
		return matchAny(r)
	case TokenLiteral:
		fallthrough
	default:
		return matchSingleChar(r, token)
	}
}

func matchEscape(r rune, token token) (int, bool) {
	if len(token.pattern) != 2 {
		return utf8.RuneLen(r), false
	}

	matchFound := false

	switch token.pattern[1] {
	case 'd':
		if unicode.IsDigit(r) {
			matchFound = true
		}
	case 'w':
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			matchFound = true
		}
	}

	return utf8.RuneLen(r), matchFound
}

func matchCharGroup(r rune, token token) (int, bool) {
	patternIdx := 0
	patternIdx++ // skip [

	negative := token.pattern[patternIdx] == '^'

	groupChars := make(map[rune]struct{})
	for ; patternIdx < len(token.pattern) && token.pattern[patternIdx] != ']'; patternIdx++ {
		groupChars[token.pattern[patternIdx]] = struct{}{}
	}

	_, ok := groupChars[r]

	if ok != negative {
		return utf8.RuneLen(r), true
	}

	return utf8.RuneLen(r), false
}

func matchStartOfStringAnchor(r rune, lineIdx int, tokensIdx int) (int, bool) {
	switch {
	case tokensIdx == 0 && lineIdx == 0:
		return 0, true
	case tokensIdx == 0 && r == '\n':
		return utf8.RuneLen(r), true
	default:
		return utf8.RuneLen(r), false
	}
}

func matchEndOfStringAnchor(r rune) (int, bool) {
	return utf8.RuneLen(r), false
}

func matchAny(r rune) (int, bool) {
	return utf8.RuneLen(r), r != '\n'
}

func matchSingleChar(r rune, token token) (int, bool) {
	if r == token.pattern[0] {
		return utf8.RuneLen(r), true
	}

	return utf8.RuneLen(r), false
}
