package vdf

import (
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFixtures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fixture   string
		wantRoots int
		wantErr   error
	}{
		{name: "valid", fixture: "valid.vdf", wantRoots: 1},
		{name: "console sample", fixture: "consolesample.vdf", wantRoots: 1},
		{name: "empty", fixture: "empty.vdf", wantRoots: 0},
		{name: "corrupted quote", fixture: "corrupted.vdf", wantErr: ErrUnexpectedEOFInQuotedString},
		{name: "missing object braces", fixture: "no_brace.vdf", wantErr: ErrExpectedValueOrObject},
		{name: "broken comment", fixture: "broken_comment.vdf", wantErr: ErrExpectedValueOrObject},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc, err := ParseString(readFixtureString(t, tt.fixture))
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ParseString(%q) error = %v, want errors.Is(_, %v)", tt.fixture, err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("ParseString(%q) returned error: %v", tt.fixture, err)
			}

			if err := doc.Validate(); err != nil {
				t.Fatalf("Validate(%q) returned error: %v", tt.fixture, err)
			}

			if doc.Format != FormatText {
				t.Fatalf("document format = %v, want %v", doc.Format, FormatText)
			}

			if len(doc.Roots) != tt.wantRoots {
				t.Fatalf("root count = %d, want %d", len(doc.Roots), tt.wantRoots)
			}
		})
	}
}

func TestParseStringPreservesDuplicates(t *testing.T) {
	t.Parallel()

	doc, err := ParseString(readFixtureString(t, "duplicates.vdf"))
	if err != nil {
		t.Fatalf("ParseString() returned error: %v", err)
	}

	if doc.Format != FormatText {
		t.Fatalf("document format = %v, want %v", doc.Format, FormatText)
	}

	root := doc.Roots[0]
	vals := root.All("dup")
	if len(vals) != 2 {
		t.Fatalf("dup values len = %d, want 2", len(vals))
	}

	if got := *vals[0].StringValue; got != "a" {
		t.Fatalf("first duplicate = %q, want %q", got, "a")
	}

	if got := *vals[1].StringValue; got != "b" {
		t.Fatalf("second duplicate = %q, want %q", got, "b")
	}
}

func TestParseStringStrictRejectsDuplicates(t *testing.T) {
	t.Parallel()

	_, err := ParseBytes(readFixtureBytes(t, "duplicates.vdf"), DecodeOptions{Format: FormatText, Strict: true})
	if !errors.Is(err, ErrDuplicateKeyInStrictMode) {
		t.Fatalf("ParseBytes(strict) error = %v, want ErrDuplicateKeyInStrictMode", err)
	}
}

func TestParseFileFixture(t *testing.T) {
	t.Parallel()

	doc, err := ParseFile(filepath.Join("testdata", "valid.vdf"))
	if err != nil {
		t.Fatalf("ParseFile() returned error: %v", err)
	}

	if len(doc.Roots) != 1 {
		t.Fatalf("root count = %d, want 1", len(doc.Roots))
	}
}

func TestDecodeOptionsLimits(t *testing.T) {
	t.Parallel()

	input := []byte(`"root" { "child" { "leaf" "x" } }`)

	_, err := ParseBytes(input, DecodeOptions{Format: FormatText, MaxDepth: 1})
	if !errors.Is(err, ErrDepthLimitExceeded) {
		t.Fatalf("ParseBytes(MaxDepth) error = %v, want ErrDepthLimitExceeded", err)
	}

	_, err = ParseBytes(input, DecodeOptions{Format: FormatText, MaxNodes: 2})
	if !errors.Is(err, ErrNodeLimitExceeded) {
		t.Fatalf("ParseBytes(MaxNodes) error = %v, want ErrNodeLimitExceeded", err)
	}
}

func TestDecoderNextEvent(t *testing.T) {
	t.Parallel()

	decoder := NewDecoder(strings.NewReader(readFixtureString(t, "duplicates.vdf")), DecodeOptions{Format: FormatText})

	types := make([]EventType, 0)
	for {
		event, err := decoder.NextEvent()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			t.Fatalf("NextEvent() returned error: %v", err)
		}

		types = append(types, event.Type)
	}

	want := []EventType{EventDocumentStart, EventObjectStart, EventString, EventString, EventObjectEnd, EventDocumentEnd}
	if len(types) != len(want) {
		t.Fatalf("event count = %d, want %d", len(types), len(want))
	}

	for i := range want {
		if types[i] != want[i] {
			t.Fatalf("event[%d] = %v, want %v", i, types[i], want[i])
		}
	}
}
