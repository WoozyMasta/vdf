// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Maxim Levchenko (WoozyMasta)
// Source: github.com/woozymasta/vdf

package vdf

import (
	"os"
	"path/filepath"
	"strings"
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
	seeds := [][]byte{
		{binaryTypeMapEnd},
		{binaryTypeMapStart, 'r', 0},
		{binaryTypeString, 'k'},
	}

	doc := NewDocument()
	root := NewObjectNode("shortcuts")
	entry := NewObjectNode("0")
	entry.Add(NewStringNode("AppName", "Test Game"))
	entry.Add(NewUint32Node("appid", 0xFF000001))
	root.Add(entry)
	doc.AddRoot(root)

	payload, err := AppendBinary(nil, doc, EncodeOptions{Format: FormatBinary})
	if err != nil {
		return seeds
	}
	seeds = append(seeds, payload)

	longKeyDoc := NewDocument()
	longKeyDoc.AddRoot(NewObjectNode(strings.Repeat("k", 128)))
	longKeyPayload, err := AppendBinary(nil, longKeyDoc, EncodeOptions{Format: FormatBinary})
	if err == nil {
		seeds = append(seeds, longKeyPayload)
	}

	duplicateDoc := NewDocument()
	duplicateRoot := NewObjectNode("root")
	duplicateRoot.Add(NewStringNode("duplicate", "first"))
	duplicateRoot.Add(NewStringNode("duplicate", "second"))
	duplicateDoc.AddRoot(duplicateRoot)
	duplicatePayload, err := AppendBinary(nil, duplicateDoc, EncodeOptions{Format: FormatBinary})
	if err == nil {
		seeds = append(seeds, duplicatePayload)
	}

	return seeds
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

func FuzzStrictDuplicateParser(f *testing.F) {
	for _, seed := range textFuzzSeeds() {
		f.Add(seed, false)
	}
	for _, seed := range binaryFuzzSeeds() {
		f.Add(seed, true)
	}

	f.Fuzz(func(t *testing.T, data []byte, binary bool) {
		format := FormatText
		if binary {
			format = FormatBinary
		}
		doc, err := ParseBytes(data, DecodeOptions{Format: format, Strict: true})
		if err != nil {
			return
		}

		if err := doc.Validate(); err != nil {
			t.Fatalf("strict parser produced invalid document: %v", err)
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
	for _, size := range []int{63, 64, 65} {
		f.Add([]byte(`"` + strings.Repeat("k", size) + `" "v"`))
		f.Add([]byte(`"k" "` + strings.Repeat("v", size) + `"`))
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
