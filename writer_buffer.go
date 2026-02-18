// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Maxim Levchenko (WoozyMasta)
// Source: github.com/woozymasta/vdf

package vdf

// sliceWriter appends encoded bytes into an existing destination slice.
type sliceWriter struct {
	buf []byte
}

// Write appends input bytes into underlying buffer.
func (w *sliceWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	return len(p), nil
}

// WriteString appends one string without intermediate []byte allocation.
func (w *sliceWriter) WriteString(s string) (int, error) {
	w.buf = append(w.buf, s...)
	return len(s), nil
}

// WriteByte appends one byte.
func (w *sliceWriter) WriteByte(b byte) error {
	w.buf = append(w.buf, b)
	return nil
}
