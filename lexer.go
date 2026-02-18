// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Maxim Levchenko (WoozyMasta)
// Source: github.com/woozymasta/vdf

package vdf

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"unicode"
)

// textTokenKind defines internal token categories for text VDF parsing.
type textTokenKind uint8

const (
	// textTokenEOF marks end of input.
	textTokenEOF textTokenKind = iota + 1
	// textTokenString marks quoted or unquoted string tokens.
	textTokenString
	// textTokenLBrace marks '{'.
	textTokenLBrace
	// textTokenRBrace marks '}'.
	textTokenRBrace
)

// textToken stores one lexical token with source position.
type textToken struct {
	value string        // Value of the token.
	line  int           // Line number of the token.
	col   int           // Column number of the token.
	kind  textTokenKind // Type of the token.
}

// runeReader is a minimal rune-scanning reader contract.
type runeReader interface {
	ReadRune() (r rune, size int, err error)
}

// textLexer tokenizes text VDF input.
type textLexer struct {
	reader    runeReader // Reader for the input.
	peeked    rune       // Peeked rune value.
	hasPeeked bool       // Whether peeked rune is set.
	line      int        // Line number of the current position.
	col       int        // Column number of the current position.
}

// newTextLexer creates a text lexer.
func newTextLexer(r io.Reader) *textLexer {
	reader, ok := r.(runeReader)
	if !ok {
		reader = bufio.NewReader(r)
	}

	return &textLexer{
		reader: reader,
		line:   1,
		col:    0,
	}
}

// readRune consumes one rune and updates source position.
func (l *textLexer) readRune() (rune, error) {
	if l.hasPeeked {
		r := l.peeked
		l.hasPeeked = false
		l.advancePosition(r)
		return r, nil
	}

	r, _, err := l.reader.ReadRune()
	if err != nil {
		return 0, err
	}

	l.advancePosition(r)
	return r, nil
}

// advancePosition updates line and column after consuming rune.
func (l *textLexer) advancePosition(r rune) {
	if r == '\n' {
		l.line++
		l.col = 0
		return
	}

	l.col++
}

// peekRune peeks one rune without position changes.
func (l *textLexer) peekRune() (rune, error) {
	if l.hasPeeked {
		return l.peeked, nil
	}

	r, _, err := l.reader.ReadRune()
	if err != nil {
		return 0, err
	}

	l.peeked = r
	l.hasPeeked = true
	return r, nil
}

// skipWhitespace consumes whitespace runes.
func (l *textLexer) skipWhitespace() error {
	for {
		r, err := l.peekRune()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		if !isWhitespace(r) {
			return nil
		}

		if _, err := l.readRune(); err != nil {
			return err
		}
	}
}

// skipLineComment consumes runes until newline or EOF.
func (l *textLexer) skipLineComment() error {
	for {
		r, err := l.readRune()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		if r == '\n' {
			return nil
		}
	}
}

// readQuotedString reads one quoted string and decodes escapes.
func (l *textLexer) readQuotedString() (string, error) {
	if _, err := l.readRune(); err != nil {
		return "", err
	}

	var sb strings.Builder
	for {
		r, err := l.readRune()
		if err == io.EOF {
			return "", ErrUnexpectedEOFInQuotedString
		}

		if err != nil {
			return "", err
		}

		if r == '"' {
			return sb.String(), nil
		}

		if r == '\\' {
			next, err := l.readRune()
			if err == io.EOF {
				return "", ErrUnexpectedEOFInEscapeSequence
			}

			if err != nil {
				return "", err
			}

			switch next {
			case 'n':
				sb.WriteRune('\n')
			case 't':
				sb.WriteRune('\t')
			case 'r':
				sb.WriteRune('\r')
			case '\\':
				sb.WriteRune('\\')
			case '"':
				sb.WriteRune('"')
			default:
				sb.WriteRune('\\')
				sb.WriteRune(next)
			}

			continue
		}

		sb.WriteRune(r)
	}
}

// readUnquotedString reads one unquoted string token.
func (l *textLexer) readUnquotedString() (string, error) {
	var sb strings.Builder
	for {
		r, err := l.peekRune()
		if err == io.EOF {
			break
		}

		if err != nil {
			return "", err
		}

		if isWhitespace(r) || r == '{' || r == '}' || r == '"' {
			break
		}

		if _, err := l.readRune(); err != nil {
			return "", err
		}

		sb.WriteRune(r)
	}

	return sb.String(), nil
}

// isWhitespace is an ASCII-fast whitespace check with Unicode fallback.
func isWhitespace(r rune) bool {
	if r <= 0x7f {
		switch r {
		case ' ', '\t', '\n', '\r', '\v', '\f':
			return true
		default:
			return false
		}
	}

	return unicode.IsSpace(r)
}

// nextToken returns one lexical token.
func (l *textLexer) nextToken() (textToken, error) {
	for {
		if err := l.skipWhitespace(); err != nil {
			return textToken{}, err
		}

		r, err := l.peekRune()
		if err == io.EOF {
			return textToken{kind: textTokenEOF, line: l.line, col: l.col}, nil
		}

		if err != nil {
			return textToken{}, err
		}

		startLine := l.line
		startCol := l.col

		switch r {
		case '/':
			// Slash can start either a line comment or an unquoted token like "/path".
			if _, err := l.readRune(); err != nil {
				return textToken{}, err
			}

			next, err := l.peekRune()
			if err == nil && next == '/' {
				// Consume comment and continue scanning for the next semantic token.
				if err := l.skipLineComment(); err != nil {
					return textToken{}, err
				}

				continue
			}

			rest, err := l.readUnquotedString()
			if err != nil {
				return textToken{}, err
			}

			return textToken{kind: textTokenString, value: "/" + rest, line: startLine, col: startCol}, nil
		case '{':
			if _, err := l.readRune(); err != nil {
				return textToken{}, err
			}

			return textToken{kind: textTokenLBrace, value: "{", line: startLine, col: startCol}, nil
		case '}':
			if _, err := l.readRune(); err != nil {
				return textToken{}, err
			}

			return textToken{kind: textTokenRBrace, value: "}", line: startLine, col: startCol}, nil
		case '"':
			value, err := l.readQuotedString()
			if err != nil {
				return textToken{}, err
			}

			return textToken{kind: textTokenString, value: value, line: startLine, col: startCol}, nil
		default:
			value, err := l.readUnquotedString()
			if err != nil {
				return textToken{}, err
			}

			if value == "" {
				return textToken{}, fmt.Errorf("%w at line %d, col %d", ErrUnexpectedCharacter, startLine, startCol)
			}

			return textToken{kind: textTokenString, value: value, line: startLine, col: startCol}, nil
		}
	}
}
