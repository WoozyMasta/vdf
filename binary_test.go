package vdf

import (
	"bytes"
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
