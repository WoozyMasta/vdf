// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Maxim Levchenko (WoozyMasta)
// Source: github.com/woozymasta/vdf

package vdf

// ParseAuto decodes VDF bytes with automatic format detection.
func ParseAuto(data []byte) (*Document, error) {
	return ParseBytes(data, DecodeOptions{Format: FormatAuto})
}

// ParseAutoFile decodes VDF file with automatic format detection.
func ParseAutoFile(path string) (*Document, error) {
	return ParseFile(path, DecodeOptions{Format: FormatAuto})
}
