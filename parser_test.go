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
		{name: "steam manifest", fixture: "steam_manifest.vdf", wantRoots: 2},
		{name: "long value", fixture: "long_value.vdf", wantRoots: 1},
		{name: "crash brackets", fixture: "crash_brackets.vdf", wantErr: ErrUnexpectedEOFInQuotedString},
		{name: "crash escapes", fixture: "crash_escapes.vdf", wantErr: ErrExpectedValueOrObject},
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

	doc, err := ParseTextFile(filepath.Join("testdata", "valid.vdf"))
	if err != nil {
		t.Fatalf("ParseTextFile() returned error: %v", err)
	}

	if len(doc.Roots) != 1 {
		t.Fatalf("root count = %d, want 1", len(doc.Roots))
	}
}

func TestParseFileDefaultFormat(t *testing.T) {
	t.Parallel()

	doc, err := ParseFile(filepath.Join("testdata", "valid.vdf"))
	if err != nil {
		t.Fatalf("ParseFile(default) returned error: %v", err)
	}

	if doc.Format != FormatText {
		t.Fatalf("default ParseFile format = %v, want %v", doc.Format, FormatText)
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

func TestStringLengthLimits(t *testing.T) {
	t.Parallel()

	longKey := `"` + string(make([]byte, 100)) + `" "v"`
	longVal := `"k" "` + string(make([]byte, 100)) + `"`

	_, err := ParseBytes([]byte(longKey), DecodeOptions{Format: FormatText, MaxKeyBytes: 10})
	if !errors.Is(err, ErrKeyTooLong) {
		t.Fatalf("MaxKeyBytes: error = %v, want ErrKeyTooLong", err)
	}

	_, err = ParseBytes([]byte(longVal), DecodeOptions{Format: FormatText, MaxValueBytes: 10})
	if !errors.Is(err, ErrValueTooLong) {
		t.Fatalf("MaxValueBytes: error = %v, want ErrValueTooLong", err)
	}

	_, err = ParseBytes([]byte(longKey), DecodeOptions{Format: FormatText, MaxStringBytes: 10})
	if !errors.Is(err, ErrKeyTooLong) && !errors.Is(err, ErrValueTooLong) {
		t.Fatalf("MaxStringBytes(key): error = %v, want ErrKeyTooLong or ErrValueTooLong", err)
	}

	_, err = ParseBytes([]byte(longVal), DecodeOptions{Format: FormatText, MaxStringBytes: 10})
	if !errors.Is(err, ErrKeyTooLong) && !errors.Is(err, ErrValueTooLong) {
		t.Fatalf("MaxStringBytes(val): error = %v, want ErrKeyTooLong or ErrValueTooLong", err)
	}

	_, err = ParseBytes([]byte(longVal), DecodeOptions{Format: FormatText})
	if err != nil {
		t.Fatalf("no limits: unexpected error = %v", err)
	}
}

func TestDecoderWalkEvents(t *testing.T) {
	t.Parallel()

	decoder := NewDecoder(strings.NewReader(readFixtureString(t, "duplicates.vdf")), DecodeOptions{Format: FormatText})

	types := make([]EventType, 0)
	for {
		event, err := decoder.WalkEvents()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			t.Fatalf("WalkEvents() returned error: %v", err)
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

func TestNextEventBinaryStreaming(t *testing.T) {
	t.Parallel()

	doc, err := ParseString(`"root" { "k1" "v1" "k2" "v2" }`)
	if err != nil {
		t.Fatalf("ParseString: %v", err)
	}

	bin, err := AppendBinary(nil, doc, EncodeOptions{Format: FormatBinary})
	if err != nil {
		t.Fatalf("AppendBinary: %v", err)
	}

	dec := NewDecoder(strings.NewReader(string(bin)), DecodeOptions{Format: FormatBinary})
	var types []EventType
	for {
		ev, err := dec.NextEvent()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("NextEvent: %v", err)
		}
		types = append(types, ev.Type)
	}

	want := []EventType{EventDocumentStart, EventObjectStart, EventString, EventString, EventObjectEnd, EventDocumentEnd}
	if len(types) != len(want) {
		t.Fatalf("event count = %d, want %d: %v", len(types), len(want), types)
	}
	for i, w := range want {
		if types[i] != w {
			t.Fatalf("event[%d] = %v, want %v", i, types[i], w)
		}
	}
}

func TestNextEventAutoDetect(t *testing.T) {
	t.Parallel()

	dec := NewDecoder(strings.NewReader(`"cfg" { "timeout" "5" }`), DecodeOptions{Format: FormatAuto})

	var types []EventType
	for {
		ev, err := dec.NextEvent()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("NextEvent(auto): %v", err)
		}
		types = append(types, ev.Type)
	}

	want := []EventType{EventDocumentStart, EventObjectStart, EventString, EventObjectEnd, EventDocumentEnd}
	if len(types) != len(want) {
		t.Fatalf("event count = %d, want %d: %v", len(types), len(want), types)
	}
}

func TestNextEventStringValues(t *testing.T) {
	t.Parallel()

	dec := NewDecoder(strings.NewReader(`"root" { "name" "hello" }`), DecodeOptions{Format: FormatText})
	for {
		ev, err := dec.NextEvent()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("NextEvent: %v", err)
		}
		if ev.Type == EventString {
			if ev.Key != "name" {
				t.Fatalf("key = %q, want %q", ev.Key, "name")
			}
			if ev.StringValue == nil || *ev.StringValue != "hello" {
				t.Fatalf("StringValue = %v, want \"hello\"", ev.StringValue)
			}
		}
	}
}

func TestNextEventDepths(t *testing.T) {
	t.Parallel()

	dec := NewDecoder(strings.NewReader(`"root" { "k" "v" }`), DecodeOptions{Format: FormatText})

	wantDepths := map[EventType]int{
		EventDocumentStart: 0,
		EventObjectStart:   1,
		EventString:        2,
		EventObjectEnd:     1,
		EventDocumentEnd:   0,
	}

	for {
		ev, err := dec.NextEvent()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("NextEvent: %v", err)
		}
		if want, ok := wantDepths[ev.Type]; ok {
			if ev.Depth != want {
				t.Errorf("event %v: Depth = %d, want %d", ev.Type, ev.Depth, want)
			}
		}
	}
}

func TestNextEventWalkEventsMutualExclusion(t *testing.T) {
	t.Parallel()

	t.Run("NextAfterWalk", func(t *testing.T) {
		t.Parallel()
		dec := NewDecoder(strings.NewReader(`"r" { "k" "v" }`), DecodeOptions{Format: FormatText})
		if _, err := dec.WalkEvents(); err != nil {
			t.Fatalf("WalkEvents: %v", err)
		}
		_, err := dec.NextEvent()
		if !errors.Is(err, ErrInvalidNodeState) {
			t.Fatalf("NextEvent after WalkEvents: error = %v, want ErrInvalidNodeState", err)
		}
	})

	t.Run("WalkAfterNext", func(t *testing.T) {
		t.Parallel()
		dec := NewDecoder(strings.NewReader(`"r" { "k" "v" }`), DecodeOptions{Format: FormatText})
		if _, err := dec.NextEvent(); err != nil {
			t.Fatalf("NextEvent: %v", err)
		}
		_, err := dec.WalkEvents()
		if !errors.Is(err, ErrInvalidNodeState) {
			t.Fatalf("WalkEvents after NextEvent: error = %v, want ErrInvalidNodeState", err)
		}
	})
}
