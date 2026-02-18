package vdf

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

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
