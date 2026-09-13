package vdf

import (
	"bytes"
	"errors"
	"io"
	"path/filepath"
	"testing"
)

func TestBinaryEncodeDecodeRoundtrip(t *testing.T) {
	t.Parallel()

	doc := NewDocumentWithFormat(FormatBinary)
	root := NewObjectNode("shortcuts")
	entry := NewObjectNode("0")
	entry.Add(NewStringNode("AppName", "Test Game"))
	entry.Add(NewUint32Node("appid", 0xFF000001))
	root.Add(entry)
	doc.AddRoot(root)

	binaryData, err := AppendBinary(nil, doc, EncodeOptions{Format: FormatBinary})
	if err != nil {
		t.Fatalf("AppendBinary() returned error: %v", err)
	}

	decoded, err := ParseBytes(binaryData, DecodeOptions{Format: FormatBinary})
	if err != nil {
		t.Fatalf("ParseBytes(binary) returned error: %v", err)
	}

	if decoded.Format != FormatBinary {
		t.Fatalf("decoded format = %v, want %v", decoded.Format, FormatBinary)
	}

	if err := decoded.Validate(); err != nil {
		t.Fatalf("decoded Validate() returned error: %v", err)
	}
}

func TestBinaryDocumentFraming(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		wantErr error
	}{
		{name: "empty document", input: []byte{binaryTypeMapEnd}},
		{name: "empty input is truncated", input: []byte{}, wantErr: ErrBufferOverflow},
		{
			name:  "scalar root with terminator",
			input: []byte{binaryTypeString, 'k', 0, 'v', 0, binaryTypeMapEnd},
		},
		{
			name:  "object root with terminators",
			input: []byte{binaryTypeMapStart, 'r', 0, binaryTypeMapEnd, binaryTypeMapEnd},
		},
		{
			name:    "missing document terminator",
			input:   []byte{binaryTypeString, 'k', 0, 'v', 0},
			wantErr: ErrBufferOverflow,
		},
		{
			name:    "missing nested object terminator",
			input:   []byte{binaryTypeMapStart, 'r', 0, binaryTypeString, 'k', 0, 'v', 0},
			wantErr: ErrUnexpectedEOFInObject,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, fullErr := ParseBytes(test.input, DecodeOptions{Format: FormatBinary})
			if test.wantErr != nil {
				if !errors.Is(fullErr, test.wantErr) {
					t.Fatalf("DecodeDocument() error = %v, want %v", fullErr, test.wantErr)
				}

				streamErr := drainNextEvent(t, NewDecoder(bytes.NewReader(test.input), DecodeOptions{Format: FormatBinary}))
				if !errors.Is(streamErr, test.wantErr) {
					t.Fatalf("NextEvent() error = %v, want %v", streamErr, test.wantErr)
				}

				return
			}

			doc, err := ParseBytes(test.input, DecodeOptions{Format: FormatBinary})
			if err != nil {
				t.Fatalf("DecodeDocument() error = %v", err)
			}
			if doc.Format != FormatBinary {
				t.Fatalf("document format = %v, want %v", doc.Format, FormatBinary)
			}

			streamErr := drainNextEvent(t, NewDecoder(bytes.NewReader(test.input), DecodeOptions{Format: FormatBinary}))
			if streamErr != nil {
				t.Fatalf("NextEvent() error = %v", streamErr)
			}
		})
	}
}

func TestBinaryUnknownTypeRejectedBeforeKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []byte
	}{
		{name: "root type", input: []byte{0xff}},
		{name: "nested type", input: []byte{binaryTypeMapStart, 'r', 0, 0xff}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ParseBytes(test.input, DecodeOptions{Format: FormatBinary})
			if !errors.Is(err, ErrUnrecognizedType) {
				t.Fatalf("DecodeDocument() error = %v, want ErrUnrecognizedType", err)
			}

			dec := NewDecoder(bytes.NewReader(test.input), DecodeOptions{Format: FormatBinary})
			streamErr := drainNextEvent(t, dec)
			if !errors.Is(streamErr, ErrUnrecognizedType) {
				t.Fatalf("NextEvent() error = %v, want ErrUnrecognizedType", streamErr)
			}
		})
	}
}

func TestBinaryDocumentEndEventOnce(t *testing.T) {
	t.Parallel()

	dec := NewDecoder(bytes.NewReader([]byte{binaryTypeMapEnd}), DecodeOptions{Format: FormatBinary})
	if event, err := dec.NextEvent(); err != nil || event.Type != EventDocumentStart {
		t.Fatalf("first event = (%v, %v), want document start", event.Type, err)
	}
	if event, err := dec.NextEvent(); err != nil || event.Type != EventDocumentEnd {
		t.Fatalf("second event = (%v, %v), want document end", event.Type, err)
	}
	if event, err := dec.NextEvent(); !errors.Is(err, io.EOF) || event.Type != 0 {
		t.Fatalf("third event = (%v, %v), want zero event and EOF", event.Type, err)
	}
}

func TestParseAuto(t *testing.T) {
	t.Parallel()

	textDoc, err := ParseAuto(readFixtureBytes(t, "valid.vdf"))
	if err != nil {
		t.Fatalf("ParseAuto(text) returned error: %v", err)
	}

	if textDoc.Format != FormatText {
		t.Fatalf("text auto format = %v, want %v", textDoc.Format, FormatText)
	}

	binaryDoc := NewDocumentWithFormat(FormatBinary)
	binaryRoot := NewObjectNode("r")
	binaryRoot.Add(NewStringNode("k", "v"))
	binaryDoc.AddRoot(binaryRoot)

	payload, err := AppendBinary(nil, binaryDoc, EncodeOptions{Format: FormatBinary})
	if err != nil {
		t.Fatalf("AppendBinary() returned error: %v", err)
	}

	autoDoc, err := ParseAuto(payload)
	if err != nil {
		t.Fatalf("ParseAuto(binary) returned error: %v", err)
	}

	if autoDoc.Format != FormatBinary {
		t.Fatalf("binary auto format = %v, want %v", autoDoc.Format, FormatBinary)
	}
}

func TestParseAutoFile(t *testing.T) {
	t.Parallel()

	doc, err := ParseAutoFile(filepath.Join("testdata", "valid.vdf"))
	if err != nil {
		t.Fatalf("ParseAutoFile() returned error: %v", err)
	}

	if doc.Format != FormatText {
		t.Fatalf("auto file format = %v, want %v", doc.Format, FormatText)
	}
}

func TestDeterministicBinaryEncoding(t *testing.T) {
	t.Parallel()

	doc := NewDocumentWithFormat(FormatBinary)
	root := NewObjectNode("root")
	root.Add(NewStringNode("b", "2"))
	root.Add(NewStringNode("a", "1"))
	doc.AddRoot(root)

	first, err := AppendBinary(nil, doc, EncodeOptions{Format: FormatBinary, Deterministic: true})
	if err != nil {
		t.Fatalf("AppendBinary(first) returned error: %v", err)
	}

	second, err := AppendBinary(nil, doc, EncodeOptions{Format: FormatBinary, Deterministic: true})
	if err != nil {
		t.Fatalf("AppendBinary(second) returned error: %v", err)
	}

	if !bytes.Equal(first, second) {
		t.Fatalf("deterministic binary output differs")
	}
}

func TestBinaryStringLengthLimits(t *testing.T) {
	t.Parallel()

	doc := NewDocumentWithFormat(FormatBinary)
	root := NewObjectNode("root")
	root.Add(NewStringNode("longkey_that_exceeds_limit", "v"))
	doc.AddRoot(root)

	payload, err := AppendBinary(nil, doc, EncodeOptions{Format: FormatBinary})
	if err != nil {
		t.Fatalf("AppendBinary() returned error: %v", err)
	}

	_, err = ParseBytes(payload, DecodeOptions{Format: FormatBinary, MaxKeyBytes: 3})
	if !errors.Is(err, ErrKeyTooLong) {
		t.Fatalf("MaxKeyBytes: error = %v, want ErrKeyTooLong", err)
	}

	doc2 := NewDocumentWithFormat(FormatBinary)
	root2 := NewObjectNode("r")
	root2.Add(NewStringNode("k", "longvalue_that_exceeds_limit"))
	doc2.AddRoot(root2)

	payload2, err := AppendBinary(nil, doc2, EncodeOptions{Format: FormatBinary})
	if err != nil {
		t.Fatalf("AppendBinary() returned error: %v", err)
	}

	_, err = ParseBytes(payload2, DecodeOptions{Format: FormatBinary, MaxValueBytes: 3})
	if !errors.Is(err, ErrValueTooLong) {
		t.Fatalf("MaxValueBytes: error = %v, want ErrValueTooLong", err)
	}
}

func TestBinaryStringLimitEnforcedDuringRead(t *testing.T) {
	t.Parallel()

	// Craft a binary payload
	// MapStart + key "k\0" + String type + key "k\0" + a value that is limit+1 bytes long with NO null terminator.
	// With early enforcement the decoder stops accumulating as soon as len > limit.
	const limit = 4

	buildPayload := func(valueLen int) []byte {
		var b []byte
		b = append(b, binaryTypeString)
		b = append(b, 'k', 0)                                 // null-terminated key
		b = append(b, bytes.Repeat([]byte("X"), valueLen)...) // value without terminator
		return b
	}

	payload := buildPayload(limit + 1)
	_, err := ParseBytes(payload, DecodeOptions{Format: FormatBinary, MaxValueBytes: limit})
	if !errors.Is(err, ErrValueTooLong) {
		t.Fatalf("early limit: error = %v, want ErrValueTooLong", err)
	}
}

func TestManualBinaryEncoder(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	enc := NewEncoder(&buf, EncodeOptions{Format: FormatBinary})
	if err := enc.StartObject("root"); err != nil {
		t.Fatalf("StartObject() returned error: %v", err)
	}

	if err := enc.WriteString("k", "v"); err != nil {
		t.Fatalf("WriteString() returned error: %v", err)
	}

	if err := enc.EndObject(); err != nil {
		t.Fatalf("EndObject() returned error: %v", err)
	}

	if err := enc.Close(); err != nil {
		t.Fatalf("Close() returned error: %v", err)
	}

	decoded, err := ParseBytes(buf.Bytes(), DecodeOptions{Format: FormatBinary})
	if err != nil {
		t.Fatalf("ParseBytes() returned error: %v", err)
	}

	if err := decoded.Validate(); err != nil {
		t.Fatalf("Validate() returned error: %v", err)
	}
}
