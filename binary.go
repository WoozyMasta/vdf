// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Maxim Levchenko (WoozyMasta)
// Source: github.com/woozymasta/vdf

package vdf

import (
	"fmt"
	"os"
)

// ParseAuto decodes VDF bytes with automatic format detection.
func ParseAuto(data []byte) (*Document, error) {
	return ParseBytes(data, DecodeOptions{Format: FormatAuto})
}

// ParseAutoFile decodes VDF file with automatic format detection.
func ParseAutoFile(path string) (doc *Document, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close file: %w", cerr)
		}
	}()

	return NewDecoder(f, DecodeOptions{Format: FormatAuto}).DecodeDocument()
}
