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

// encoderMode identifies the logical document operation selected by an encoder.
type encoderMode uint8

const (
	// encoderModeUnused means no output operation has started.
	encoderModeUnused encoderMode = iota
	// encoderModeManual means manual streaming owns the encoder.
	encoderModeManual
	// encoderModeDocument means full-document encoding owns the encoder.
	encoderModeDocument
)

// Encoder encodes VDF documents to an output stream.
type Encoder struct {
	w           io.Writer     // Writer for the output.
	writeErr    error         // Terminal error after output may have been partially written.
	opts        EncodeOptions // Encode options.
	manualDepth int           // Current depth for manual streaming.
	mode        encoderMode   // Output mode selected by the first successful operation.
	closed      bool          // Whether the logical document was finalized.
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
	if err := e.available(); err != nil {
		return err
	}
	if e.mode != encoderModeUnused {
		return fmt.Errorf("%w: encoder already used for manual or full-document output", ErrInvalidNodeState)
	}

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

	if format != FormatText && format != FormatBinary {
		return fmt.Errorf("%w: %d", ErrInvalidFormat, format)
	}

	e.mode = encoderModeDocument
	var err error

	switch format {
	case FormatText:
		err = encodeTextDocument(e.w, doc, e.opts)
	case FormatBinary:
		err = encodeBinaryDocument(e.w, doc, e.opts)
	}
	if err != nil {
		return e.latchWriteError(err)
	}

	e.closed = true
	return nil
}

// StartObject begins an object in manual streaming mode.
func (e *Encoder) StartObject(key string) error {
	format, err := e.beginManual()
	if err != nil {
		return err
	}

	e.mode = encoderModeManual

	switch format {
	case FormatText:
		err = e.startTextObject(key)

	case FormatBinary:
		if err = writeBinaryByte(e.w, binaryTypeMapStart); err == nil {
			err = writeNullTerminatedString(e.w, key)
		}
		if err == nil {
			e.manualDepth++
		}

	default:
		return fmt.Errorf("%w: %d", ErrInvalidFormat, e.opts.Format)
	}

	return e.latchWriteError(err)
}

// WriteString writes a string leaf in manual streaming mode.
func (e *Encoder) WriteString(key, value string) error {
	format, err := e.beginManual()
	if err != nil {
		return err
	}

	e.mode = encoderModeManual

	switch format {
	case FormatText:
		err = e.writeTextLeaf(key, value)

	case FormatBinary:
		if err = writeBinaryByte(e.w, binaryTypeString); err == nil {
			err = writeNullTerminatedString(e.w, key)
		}
		if err == nil {
			err = writeNullTerminatedString(e.w, value)
		}

	default:
		return fmt.Errorf("%w: %d", ErrInvalidFormat, e.opts.Format)
	}

	return e.latchWriteError(err)
}

// WriteUint32 writes an unsigned numeric leaf in manual streaming mode.
func (e *Encoder) WriteUint32(key string, value uint32) error {
	format, err := e.beginManual()
	if err != nil {
		return err
	}

	e.mode = encoderModeManual

	switch format {
	case FormatText:
		err = e.writeTextLeaf(key, strconv.FormatUint(uint64(value), 10))

	case FormatBinary:
		if err = writeBinaryByte(e.w, binaryTypeNumber); err == nil {
			err = writeNullTerminatedString(e.w, key)
		}
		if err == nil {
			var raw [4]byte
			binary.LittleEndian.PutUint32(raw[:], value)
			_, err = e.w.Write(raw[:])
		}

	default:
		return fmt.Errorf("%w: %d", ErrInvalidFormat, e.opts.Format)
	}

	return e.latchWriteError(err)
}

// EndObject ends an object in manual streaming mode.
func (e *Encoder) EndObject() error {
	format, err := e.beginManual()
	if err != nil {
		return err
	}
	if e.manualDepth <= 0 {
		return fmt.Errorf("%w: no open object", ErrInvalidNodeState)
	}

	e.mode = encoderModeManual

	switch format {
	case FormatText:
		err = e.endTextObject()

	case FormatBinary:
		err = writeBinaryByte(e.w, binaryTypeMapEnd)

	default:
		return fmt.Errorf("%w: %d", ErrInvalidFormat, e.opts.Format)
	}

	if err != nil {
		return e.latchWriteError(err)
	}

	e.manualDepth--
	return nil
}

// Close finalizes manual streaming state.
func (e *Encoder) Close() error {
	if err := e.available(); err != nil {
		return err
	}
	if e.closed {
		return nil
	}
	if e.mode == encoderModeDocument {
		return fmt.Errorf("%w: full document already owns encoder", ErrInvalidNodeState)
	}

	if e.manualDepth != 0 {
		return fmt.Errorf("%w: %d unclosed objects", ErrInvalidNodeState, e.manualDepth)
	}

	format := e.manualFormat()
	if format != FormatText && format != FormatBinary {
		return fmt.Errorf("%w: %d", ErrInvalidFormat, e.opts.Format)
	}

	e.mode = encoderModeManual
	if format == FormatBinary {
		if err := writeBinaryByte(e.w, binaryTypeMapEnd); err != nil {
			return e.latchWriteError(err)
		}
	}

	e.closed = true
	return nil
}

// available returns the terminal writer error, if one was latched.
func (e *Encoder) available() error {
	if e.writeErr != nil {
		return e.writeErr
	}

	return nil
}

// latchWriteError preserves an output error after a potentially partial write.
func (e *Encoder) latchWriteError(err error) error {
	if err != nil && e.writeErr == nil {
		e.writeErr = err
	}

	return e.writeErr
}

// beginManual validates the encoder lifecycle and selects manual mode.
func (e *Encoder) beginManual() (Format, error) {
	if err := e.available(); err != nil {
		return FormatAuto, err
	}
	if e.closed || e.mode == encoderModeDocument {
		return FormatAuto, fmt.Errorf("%w: encoder is finalized or used for full-document output", ErrInvalidNodeState)
	}

	format := e.manualFormat()
	if format != FormatText && format != FormatBinary {
		return FormatAuto, fmt.Errorf("%w: %d", ErrInvalidFormat, e.opts.Format)
	}

	return format, nil
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

// WriteFile encodes document to file.
// Without options it writes text format.
func WriteFile(path string, doc *Document, opts ...EncodeOptions) (err error) {
	effective := EncodeOptions{Format: FormatText}
	if len(opts) > 0 {
		effective = opts[0]
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close file: %w", cerr)
		}
	}()

	return NewEncoder(f, effective).EncodeDocument(doc)
}

// WriteTextFile encodes document as text VDF file.
func WriteTextFile(path string, doc *Document) error {
	return WriteFile(path, doc, EncodeOptions{Format: FormatText})
}

// WriteBinaryFile encodes document as binary VDF file.
func WriteBinaryFile(path string, doc *Document) error {
	return WriteFile(path, doc, EncodeOptions{Format: FormatBinary})
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
	if doc == nil {
		return nil, fmt.Errorf("%w: nil document", ErrInvalidNodeState)
	}
	if opts.Validate {
		if err := doc.Validate(); err != nil {
			return nil, err
		}
	}

	extra := estimateBinaryDocumentSize(doc)
	dst = reserveAppendCapacity(dst, extra)

	writer := &sliceWriter{buf: dst}
	opts.Format = FormatBinary
	opts.Validate = false

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
