// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Maxim Levchenko (WoozyMasta)
// Source: github.com/woozymasta/vdf

package vdf

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strconv"
)

// Encoder encodes VDF documents to an output stream.
type Encoder struct {
	w                    io.Writer     // Writer for the output.
	opts                 EncodeOptions // Encode options.
	manualDepth          int           // Current depth for manual streaming.
	manualBinaryUsed     bool          // Whether binary mode is used for manual streaming.
	manualBinaryFinished bool          // Whether binary mode is finished for manual streaming.
}

// NewEncoder creates a VDF encoder.
func NewEncoder(w io.Writer, opts EncodeOptions) *Encoder {
	return &Encoder{
		w:    w,
		opts: normalizeEncodeOptions(opts),
	}
}

// EncodeDocument encodes a complete document in selected output format.
func (e *Encoder) EncodeDocument(doc *Document) error {
	if doc == nil {
		return fmt.Errorf("%w: nil document", ErrInvalidNodeState)
	}

	if e.opts.Validate {
		if err := doc.Validate(); err != nil {
			return err
		}
	}

	format := e.opts.Format
	if format == FormatAuto {
		if doc.Format == FormatBinary || doc.Format == FormatText {
			format = doc.Format
		} else {
			format = FormatText
		}
	}

	switch format {
	case FormatText:
		return encodeTextDocument(e.w, doc, e.opts)
	case FormatBinary:
		return encodeBinaryDocument(e.w, doc, e.opts)
	default:
		return fmt.Errorf("%w: %d", ErrInvalidFormat, format)
	}
}

// StartObject begins an object in manual streaming mode.
func (e *Encoder) StartObject(key string) error {
	switch e.manualFormat() {
	case FormatText:
		return e.startTextObject(key)

	case FormatBinary:
		e.manualBinaryUsed = true
		e.manualDepth++
		if err := writeBinaryByte(e.w, binaryTypeMapStart); err != nil {
			return err
		}
		return writeNullTerminatedString(e.w, key)

	default:
		return fmt.Errorf("%w: %d", ErrInvalidFormat, e.opts.Format)
	}
}

// WriteString writes a string leaf in manual streaming mode.
func (e *Encoder) WriteString(key, value string) error {
	switch e.manualFormat() {
	case FormatText:
		return e.writeTextLeaf(key, value)

	case FormatBinary:
		e.manualBinaryUsed = true
		if err := writeBinaryByte(e.w, binaryTypeString); err != nil {
			return err
		}
		if err := writeNullTerminatedString(e.w, key); err != nil {
			return err
		}
		return writeNullTerminatedString(e.w, value)

	default:
		return fmt.Errorf("%w: %d", ErrInvalidFormat, e.opts.Format)
	}
}

// WriteUint32 writes an unsigned numeric leaf in manual streaming mode.
func (e *Encoder) WriteUint32(key string, value uint32) error {
	switch e.manualFormat() {
	case FormatText:
		return e.writeTextLeaf(key, strconv.FormatUint(uint64(value), 10))

	case FormatBinary:
		e.manualBinaryUsed = true
		if err := writeBinaryByte(e.w, binaryTypeNumber); err != nil {
			return err
		}
		if err := writeNullTerminatedString(e.w, key); err != nil {
			return err
		}

		var raw [4]byte
		binary.LittleEndian.PutUint32(raw[:], value)
		_, err := e.w.Write(raw[:])
		return err

	default:
		return fmt.Errorf("%w: %d", ErrInvalidFormat, e.opts.Format)
	}
}

// EndObject ends an object in manual streaming mode.
func (e *Encoder) EndObject() error {
	switch e.manualFormat() {
	case FormatText:
		if e.manualDepth <= 0 {
			return fmt.Errorf("%w: no open object", ErrInvalidNodeState)
		}
		e.manualDepth--
		return e.endTextObject()

	case FormatBinary:
		if e.manualDepth <= 0 {
			return fmt.Errorf("%w: no open object", ErrInvalidNodeState)
		}
		e.manualDepth--
		return writeBinaryByte(e.w, binaryTypeMapEnd)

	default:
		return fmt.Errorf("%w: %d", ErrInvalidFormat, e.opts.Format)
	}
}

// Close finalizes manual streaming state.
func (e *Encoder) Close() error {
	if e.manualFormat() != FormatBinary || !e.manualBinaryUsed || e.manualBinaryFinished {
		return nil
	}

	if e.manualDepth != 0 {
		return fmt.Errorf("%w: %d unclosed objects", ErrInvalidNodeState, e.manualDepth)
	}

	e.manualBinaryFinished = true
	return writeBinaryByte(e.w, binaryTypeMapEnd)
}

// Write encodes document as text VDF with default options.
func Write(w io.Writer, doc *Document) error {
	return NewEncoder(w, EncodeOptions{Format: FormatText}).EncodeDocument(doc)
}

// WriteString encodes document as text VDF string.
func WriteString(doc *Document) (string, error) {
	out, err := AppendText(nil, doc, EncodeOptions{Format: FormatText})
	if err != nil {
		return "", err
	}

	return string(out), nil
}

// WriteFile encodes document as text VDF file.
func WriteFile(path string, doc *Document) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close file: %w", cerr)
		}
	}()

	return Write(f, doc)
}

// AppendText appends text VDF output to destination byte slice.
func AppendText(dst []byte, doc *Document, opts EncodeOptions) ([]byte, error) {
	writer := &sliceWriter{buf: dst}
	opts.Format = FormatText

	if err := NewEncoder(writer, opts).EncodeDocument(doc); err != nil {
		return nil, err
	}

	return writer.buf, nil
}

// AppendBinary appends binary VDF output to destination byte slice.
func AppendBinary(dst []byte, doc *Document, opts EncodeOptions) ([]byte, error) {
	extra := estimateBinaryDocumentSize(doc, opts.Deterministic)
	dst = reserveAppendCapacity(dst, extra)

	writer := &sliceWriter{buf: dst}
	opts.Format = FormatBinary

	if err := NewEncoder(writer, opts).EncodeDocument(doc); err != nil {
		return nil, err
	}

	return writer.buf, nil
}

// normalizeEncodeOptions applies default encoder options.
func normalizeEncodeOptions(opts EncodeOptions) EncodeOptions {
	if opts.Indent == "" {
		opts.Indent = "\t"
	}
	if opts.Format == FormatAuto {
		opts.Format = FormatText
	}

	return opts
}

// reserveAppendCapacity grows destination capacity for append-heavy writers.
func reserveAppendCapacity(dst []byte, extra int) []byte {
	if extra <= 0 || cap(dst)-len(dst) >= extra {
		return dst
	}

	out := make([]byte, len(dst), len(dst)+extra)
	copy(out, dst)
	return out
}

// manualFormat resolves effective format for manual streaming calls.
func (e *Encoder) manualFormat() Format {
	if e.opts.Format == FormatAuto {
		return FormatText
	}

	return e.opts.Format
}
