package vdf

import (
	"strings"
	"testing"
)

func TestTextLexerTracksLineAndColumn(t *testing.T) {
	t.Parallel()

	lexer := newTextLexer(strings.NewReader("\"a\"\n\"b\""))

	first, err := lexer.nextToken()
	if err != nil {
		t.Fatalf("nextToken(first) returned error: %v", err)
	}

	if first.line != 1 || first.col != 0 {
		t.Fatalf("first token pos = (%d,%d), want (1,0)", first.line, first.col)
	}

	second, err := lexer.nextToken()
	if err != nil {
		t.Fatalf("nextToken(second) returned error: %v", err)
	}

	if second.line != 2 || second.col != 0 {
		t.Fatalf("second token pos = (%d,%d), want (2,0)", second.line, second.col)
	}
}

func TestParseStringErrorContainsPosition(t *testing.T) {
	t.Parallel()

	_, err := ParseString(`"key" }`)
	if err == nil {
		t.Fatalf("ParseString() expected error")
	}

	message := err.Error()
	if !strings.Contains(message, "line") || !strings.Contains(message, "col") {
		t.Fatalf("error message missing position: %q", message)
	}
}
