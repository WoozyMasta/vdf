// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Maxim Levchenko (WoozyMasta)
// Source: github.com/woozymasta/vdf

package vdf

import (
	"os"
	"path/filepath"
	"testing"
)

// textFuzzSeeds returns byte slices from all text VDF fixture files.
func textFuzzSeeds() [][]byte {
	names := []string{
		"valid.vdf",
		"consolesample.vdf",
		"empty.vdf",
		"duplicates.vdf",
		"steam_manifest.vdf",
		"long_value.vdf",
		"corrupted.vdf",
		"no_brace.vdf",
		"broken_comment.vdf",
		"crash_brackets.vdf",
		"crash_escapes.vdf",
	}

	seeds := make([][]byte, 0, len(names))
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err == nil {
			seeds = append(seeds, data)
		}
	}

	return seeds
}

// binaryFuzzSeeds returns a pre-encoded binary VDF payload as a seed.
func binaryFuzzSeeds() [][]byte {
	doc := NewDocument()
	root := NewObjectNode("shortcuts")
	entry := NewObjectNode("0")
	entry.Add(NewStringNode("AppName", "Test Game"))
	entry.Add(NewUint32Node("appid", 0xFF000001))
	root.Add(entry)
	doc.AddRoot(root)

	payload, err := AppendBinary(nil, doc, EncodeOptions{Format: FormatBinary})
	if err != nil {
		return nil
	}

	return [][]byte{payload}
}

func FuzzTextParser(f *testing.F) {
	for _, seed := range textFuzzSeeds() {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		doc, err := ParseBytes(data, DecodeOptions{Format: FormatText})
		if err != nil {
			return
		}

		if err := doc.Validate(); err != nil {
			t.Fatalf("parsed document failed Validate(): %v", err)
		}
	})
}

func FuzzBinaryParser(f *testing.F) {
	for _, seed := range binaryFuzzSeeds() {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		doc, err := ParseBytes(data, DecodeOptions{Format: FormatBinary})
		if err != nil {
			return
		}

		if err := doc.Validate(); err != nil {
			t.Fatalf("parsed document failed Validate(): %v", err)
		}
	})
}

func FuzzWriterRoundtrip(f *testing.F) {
	for _, seed := range textFuzzSeeds() {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		doc, err := ParseBytes(data, DecodeOptions{Format: FormatText})
		if err != nil {
			return
		}

		encoded, err := AppendText(nil, doc, EncodeOptions{Format: FormatText})
		if err != nil {
			t.Fatalf("AppendText() error after successful parse: %v", err)
		}

		doc2, err := ParseBytes(encoded, DecodeOptions{Format: FormatText})
		if err != nil {
			t.Fatalf("re-parse after encode failed: %v", err)
		}

		if err := doc2.Validate(); err != nil {
			t.Fatalf("re-parsed document Validate() failed: %v", err)
		}
	})
}

func FuzzParseAuto(f *testing.F) {
	for _, seed := range textFuzzSeeds() {
		f.Add(seed)
	}

	for _, seed := range binaryFuzzSeeds() {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ParseAuto(data)
	})
}

func FuzzTextParserLimits(f *testing.F) {
	for _, seed := range textFuzzSeeds() {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		opts := DecodeOptions{
			Format:         FormatText,
			MaxDepth:       10,
			MaxNodes:       50,
			MaxStringBytes: 64,
		}
		_, _ = ParseBytes(data, opts)
	})
}
