package vdf

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

// failOnWriteWriter returns a configured error on one write call.
type failOnWriteWriter struct {
	writes int   // Number of writes received.
	failAt int   // Write call that should fail.
	err    error // Error returned by the failing write.
}

// Write counts calls and fails at the configured structural write.
func (w *failOnWriteWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writes == w.failAt {
		return 0, w.err
	}

	return len(p), nil
}

func TestWriteStringRoundtripFixture(t *testing.T) {
	t.Parallel()

	doc, err := ParseString(readFixtureString(t, "consolesample.vdf"))
	if err != nil {
		t.Fatalf("ParseString(fixture) returned error: %v", err)
	}

	text, err := WriteString(doc)
	if err != nil {
		t.Fatalf("WriteString() returned error: %v", err)
	}

	roundtrip, err := ParseString(text)
	if err != nil {
		t.Fatalf("ParseString(roundtrip) returned error: %v", err)
	}

	if err := roundtrip.Validate(); err != nil {
		t.Fatalf("roundtrip Validate() returned error: %v", err)
	}

	if len(roundtrip.Roots) != len(doc.Roots) {
		t.Fatalf("roundtrip roots = %d, want %d", len(roundtrip.Roots), len(doc.Roots))
	}
}

func TestWriteStringAndRoundtrip(t *testing.T) {
	t.Parallel()

	doc := NewDocumentWithFormat(FormatText)
	root := NewObjectNode("root")
	root.Add(NewStringNode("name", `value with "quotes"`))
	root.Add(NewUint32Node("id", 42))
	doc.AddRoot(root)

	text, err := WriteString(doc)
	if err != nil {
		t.Fatalf("WriteString() returned error: %v", err)
	}

	if !strings.Contains(text, `"name"`) || !strings.Contains(text, `\"quotes\"`) {
		t.Fatalf("encoded text missing expected fragments:\n%s", text)
	}

	roundtrip, err := ParseString(text)
	if err != nil {
		t.Fatalf("ParseString(roundtrip) returned error: %v", err)
	}

	if err := roundtrip.Validate(); err != nil {
		t.Fatalf("roundtrip Validate() returned error: %v", err)
	}
}

func TestEncoderCompactMode(t *testing.T) {
	t.Parallel()

	doc := NewDocumentWithFormat(FormatText)
	root := NewObjectNode("root")
	root.Add(NewStringNode("k", "v"))
	doc.AddRoot(root)

	var buf bytes.Buffer
	enc := NewEncoder(&buf, EncodeOptions{Format: FormatText, Compact: true})
	if err := enc.EncodeDocument(doc); err != nil {
		t.Fatalf("EncodeDocument(compact) returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `"root" {`) {
		t.Fatalf("compact output missing object compact form: %s", out)
	}
}

func TestEncodeDocumentFormatAuto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		docFormat  Format
		wantFormat Format
	}{
		{name: "document text", docFormat: FormatText, wantFormat: FormatText},
		{name: "document binary", docFormat: FormatBinary, wantFormat: FormatBinary},
		{name: "document auto defaults to text", docFormat: FormatAuto, wantFormat: FormatText},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			doc := NewDocumentWithFormat(test.docFormat)
			root := NewObjectNode("root")
			root.Add(NewStringNode("key", "value"))
			doc.AddRoot(root)

			var buf bytes.Buffer
			enc := NewEncoder(&buf, EncodeOptions{Format: FormatAuto})
			if err := enc.EncodeDocument(doc); err != nil {
				t.Fatalf("EncodeDocument() error = %v", err)
			}

			encoded := buf.Bytes()
			if test.wantFormat == FormatBinary && encoded[0] != binaryTypeMapStart {
				t.Fatalf("binary output first byte = %#02x, want %#02x", encoded[0], binaryTypeMapStart)
			}

			if _, err := ParseBytes(encoded, DecodeOptions{Format: test.wantFormat}); err != nil {
				t.Fatalf("ParseBytes(format=%v) error = %v", test.wantFormat, err)
			}
		})
	}
}

func TestManualEncoderFormatAutoUsesText(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	enc := NewEncoder(&buf, EncodeOptions{Format: FormatAuto})
	if err := enc.StartObject("root"); err != nil {
		t.Fatalf("StartObject() error = %v", err)
	}
	if err := enc.WriteString("key", "value"); err != nil {
		t.Fatalf("WriteString() error = %v", err)
	}
	if err := enc.EndObject(); err != nil {
		t.Fatalf("EndObject() error = %v", err)
	}
	if err := enc.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if _, err := ParseBytes(buf.Bytes(), DecodeOptions{Format: FormatText}); err != nil {
		t.Fatalf("ParseBytes(format=text) error = %v", err)
	}
}

func TestManualEncoderText(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	enc := NewEncoder(&buf, EncodeOptions{Format: FormatText})

	if err := enc.StartObject("root"); err != nil {
		t.Fatalf("StartObject() returned error: %v", err)
	}

	if err := enc.WriteString("name", "srv"); err != nil {
		t.Fatalf("WriteString() returned error: %v", err)
	}

	if err := enc.EndObject(); err != nil {
		t.Fatalf("EndObject() returned error: %v", err)
	}

	if err := enc.Close(); err != nil {
		t.Fatalf("Close() returned error: %v", err)
	}

	if !strings.Contains(buf.String(), `"name"`) {
		t.Fatalf("manual encoded output mismatch:\n%s", buf.String())
	}
}

func TestEncodeDocumentValidateOption(t *testing.T) {
	t.Parallel()

	value := "broken"
	doc := NewDocumentWithFormat(FormatText)
	doc.AddRoot(&Node{
		Key:         "bad",
		Kind:        NodeString,
		StringValue: nil,
		Uint32Value: nil,
		Children:    nil,
	})

	_, err := AppendText(nil, doc, EncodeOptions{Format: FormatText})
	if !errors.Is(err, ErrInvalidNodeState) {
		t.Fatalf("AppendText(validate=false) error = %v, want ErrInvalidNodeState", err)
	}

	doc.Roots[0].StringValue = &value
	doc.Roots[0].Kind = NodeObject

	_, err = AppendText(nil, doc, EncodeOptions{Format: FormatText, Validate: true})
	if !errors.Is(err, ErrInvalidNodeState) {
		t.Fatalf("AppendText(validate=true) error = %v, want ErrInvalidNodeState", err)
	}
}

func TestEncodeNilNodesWithoutValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		format Format
		roots  []*Node
	}{
		{
			name:   "text nil root",
			format: FormatText,
			roots:  []*Node{nil}},
		{
			name:   "binary nil root",
			format: FormatBinary,
			roots:  []*Node{nil}},
		{
			name:   "text nil child",
			format: FormatText,
			roots:  []*Node{{Key: "root", Kind: NodeObject, Children: []*Node{nil}}},
		},
		{
			name:   "binary nil child",
			format: FormatBinary,
			roots:  []*Node{{Key: "root", Kind: NodeObject, Children: []*Node{nil}}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			doc := &Document{Format: test.format, Roots: test.roots}
			var err error
			if test.format == FormatText {
				_, err = AppendText(nil, doc, EncodeOptions{Validate: false})
			} else {
				_, err = AppendBinary(nil, doc, EncodeOptions{Validate: false})
			}
			if !errors.Is(err, ErrInvalidNodeState) {
				t.Fatalf("encode error = %v, want ErrInvalidNodeState", err)
			}
		})
	}
}

func TestAppendBinaryValidatesBeforeEstimating(t *testing.T) {
	t.Parallel()

	root := NewObjectNode("root")
	root.Children = append(root.Children, root)
	doc := &Document{Roots: []*Node{root}}
	_, err := AppendBinary(nil, doc, EncodeOptions{Validate: true})
	if !errors.Is(err, ErrInvalidNodeState) {
		t.Fatalf("AppendBinary(cyclic doc, validate=true) error = %v, want ErrInvalidNodeState", err)
	}
}

func TestManualEncoderLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("empty binary close writes terminator", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		enc := NewEncoder(&buf, EncodeOptions{Format: FormatBinary})
		if err := enc.Close(); err != nil {
			t.Fatalf("Close(): %v", err)
		}
		if !bytes.Equal(buf.Bytes(), []byte{binaryTypeMapEnd}) {
			t.Fatalf("Close() output = %v, want [%#02x]", buf.Bytes(), binaryTypeMapEnd)
		}
	})

	t.Run("close is idempotent and rejects later writes", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		enc := NewEncoder(&buf, EncodeOptions{Format: FormatText})
		if err := enc.WriteString("key", "value"); err != nil {
			t.Fatalf("WriteString(): %v", err)
		}
		if err := enc.Close(); err != nil {
			t.Fatalf("Close(): %v", err)
		}
		before := buf.String()
		if err := enc.Close(); err != nil {
			t.Fatalf("second Close(): %v", err)
		}
		if got := buf.String(); got != before {
			t.Fatalf("second Close() changed output to %q, want %q", got, before)
		}
		if err := enc.WriteString("later", "value"); !errors.Is(err, ErrInvalidNodeState) {
			t.Fatalf("WriteString() after Close() error = %v, want ErrInvalidNodeState", err)
		}
		if err := enc.StartObject("later"); !errors.Is(err, ErrInvalidNodeState) {
			t.Fatalf("StartObject() after Close() error = %v, want ErrInvalidNodeState", err)
		}
		if err := enc.EndObject(); !errors.Is(err, ErrInvalidNodeState) {
			t.Fatalf("EndObject() after Close() error = %v, want ErrInvalidNodeState", err)
		}
	})

	t.Run("manual and full document modes cannot mix", func(t *testing.T) {
		t.Parallel()
		doc := NewDocument()
		var manualFirst bytes.Buffer
		enc := NewEncoder(&manualFirst, EncodeOptions{Format: FormatText})
		if err := enc.WriteString("key", "value"); err != nil {
			t.Fatalf("WriteString(): %v", err)
		}
		if err := enc.EncodeDocument(doc); !errors.Is(err, ErrInvalidNodeState) {
			t.Fatalf("EncodeDocument after manual write error = %v, want ErrInvalidNodeState", err)
		}

		var documentFirst bytes.Buffer
		enc = NewEncoder(&documentFirst, EncodeOptions{Format: FormatText})
		if err := enc.EncodeDocument(doc); err != nil {
			t.Fatalf("EncodeDocument(): %v", err)
		}
		if err := enc.WriteString("key", "value"); !errors.Is(err, ErrInvalidNodeState) {
			t.Fatalf("WriteString after EncodeDocument error = %v, want ErrInvalidNodeState", err)
		}
		if err := enc.EncodeDocument(doc); !errors.Is(err, ErrInvalidNodeState) {
			t.Fatalf("repeated EncodeDocument error = %v, want ErrInvalidNodeState", err)
		}
	})
}

func TestManualEncoderLatchesStructuralWriterError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("structural write failed")
	w := &failOnWriteWriter{failAt: 4, err: wantErr}
	enc := NewEncoder(w, EncodeOptions{Format: FormatBinary})
	if err := enc.StartObject("root"); err != nil {
		t.Fatalf("StartObject(): %v", err)
	}
	if enc.manualDepth != 1 {
		t.Fatalf("manualDepth after StartObject() = %d, want 1", enc.manualDepth)
	}

	err := enc.EndObject()
	if !errors.Is(err, wantErr) {
		t.Fatalf("EndObject() error = %v, want %v", err, wantErr)
	}
	if enc.manualDepth != 1 {
		t.Fatalf("manualDepth after failed EndObject() = %d, want 1", enc.manualDepth)
	}
	if got := enc.EndObject(); got != err {
		t.Fatalf("repeated EndObject() error = %v, want latched error %v", got, err)
	}
	if got := enc.Close(); got != err {
		t.Fatalf("Close() after writer failure = %v, want latched error %v", got, err)
	}
}

func TestEncodeDocumentLatchesWriterError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("write failed")
	w := writerFunc(func(p []byte) (int, error) { return 0, wantErr })
	enc := NewEncoder(w, EncodeOptions{Format: FormatText})
	doc := NewDocument()
	doc.AddRoot(NewObjectNode("root"))
	if err := enc.EncodeDocument(doc); !errors.Is(err, wantErr) {
		t.Fatalf("EncodeDocument() error = %v, want %v", err, wantErr)
	}
	if err := enc.EncodeDocument(doc); err == nil || !errors.Is(err, wantErr) {
		t.Fatalf("repeated EncodeDocument() error = %v, want latched %v", err, wantErr)
	}
}

// writerFunc adapts a function to io.Writer for focused I/O failure tests.
type writerFunc func([]byte) (int, error)

// Write invokes the adapted writer function.
func (f writerFunc) Write(p []byte) (int, error) {
	return f(p)
}

func TestManualEncoderCloseUnclosed(t *testing.T) {
	t.Parallel()

	t.Run("text unclosed", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		enc := NewEncoder(&buf, EncodeOptions{Format: FormatText})
		if err := enc.StartObject("root"); err != nil {
			t.Fatalf("StartObject: %v", err)
		}
		// Deliberately skip EndObject.
		if err := enc.Close(); !errors.Is(err, ErrInvalidNodeState) {
			t.Fatalf("Close() with unclosed text object: error = %v, want ErrInvalidNodeState", err)
		}
	})

	t.Run("text closed", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		enc := NewEncoder(&buf, EncodeOptions{Format: FormatText})
		if err := enc.StartObject("root"); err != nil {
			t.Fatalf("StartObject: %v", err)
		}
		if err := enc.EndObject(); err != nil {
			t.Fatalf("EndObject: %v", err)
		}
		if err := enc.Close(); err != nil {
			t.Fatalf("Close() with properly closed text object: %v", err)
		}
	})

	t.Run("binary unclosed", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		enc := NewEncoder(&buf, EncodeOptions{Format: FormatBinary})
		if err := enc.StartObject("root"); err != nil {
			t.Fatalf("StartObject: %v", err)
		}
		// Deliberately skip EndObject.
		if err := enc.Close(); !errors.Is(err, ErrInvalidNodeState) {
			t.Fatalf("Close() with unclosed binary object: error = %v, want ErrInvalidNodeState", err)
		}
	})
}

func TestWriteFileWrappers(t *testing.T) {
	t.Parallel()

	doc := NewDocumentWithFormat(FormatText)
	root := NewObjectNode("root")
	root.Add(NewStringNode("name", "srv"))
	doc.AddRoot(root)

	dir := t.TempDir()
	defaultPath := filepath.Join(dir, "default.vdf")
	if err := WriteFile(defaultPath, doc); err != nil {
		t.Fatalf("WriteFile(default) returned error: %v", err)
	}

	defaultDoc, err := ParseTextFile(defaultPath)
	if err != nil {
		t.Fatalf("ParseTextFile(default) returned error: %v", err)
	}

	if defaultDoc.Format != FormatText {
		t.Fatalf("default WriteFile format = %v, want %v", defaultDoc.Format, FormatText)
	}

	textPath := filepath.Join(dir, "sample.vdf")
	if err := WriteTextFile(textPath, doc); err != nil {
		t.Fatalf("WriteTextFile() returned error: %v", err)
	}

	textDoc, err := ParseTextFile(textPath)
	if err != nil {
		t.Fatalf("ParseTextFile() returned error: %v", err)
	}

	if textDoc.Format != FormatText {
		t.Fatalf("text file format = %v, want %v", textDoc.Format, FormatText)
	}

	binPath := filepath.Join(dir, "sample.bin")
	if err := WriteBinaryFile(binPath, doc); err != nil {
		t.Fatalf("WriteBinaryFile() returned error: %v", err)
	}

	binDoc, err := ParseAutoFile(binPath)
	if err != nil {
		t.Fatalf("ParseAutoFile(binary) returned error: %v", err)
	}

	if binDoc.Format != FormatBinary {
		t.Fatalf("binary file format = %v, want %v", binDoc.Format, FormatBinary)
	}
}
