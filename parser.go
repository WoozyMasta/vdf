// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Maxim Levchenko (WoozyMasta)
// Source: github.com/woozymasta/vdf

package vdf

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// Decoder decodes VDF data from an input stream.
type Decoder struct {
	decodeErr error          // Error from last decode operation.
	reader    io.Reader      // Source input reader.
	buffered  *bufio.Reader  // Lazy buffered reader for auto-detect and generic streams.
	decoded   *Document      // Decoded document.
	events    *eventIterator // Event iterator.
	opts      DecodeOptions  // Decode options.
}

// NewDecoder creates a decoder with normalized options.
func NewDecoder(r io.Reader, opts DecodeOptions) *Decoder {
	return &Decoder{
		reader: r,
		opts:   normalizeDecodeOptions(opts),
	}
}

// DecodeDocument decodes the full input stream into a document.
func (d *Decoder) DecodeDocument() (*Document, error) {
	if d.decoded != nil || d.decodeErr != nil {
		return d.decoded, d.decodeErr
	}

	if err := validateDecodeFormat(d.opts.Format); err != nil {
		d.decodeErr = err
		return nil, err
	}

	format := d.opts.Format
	source := d.reader

	if format == FormatAuto {
		br := d.bufferedReader()
		detected, err := detectStreamFormat(br)
		if err != nil {
			d.decodeErr = err
			return nil, err
		}

		format = detected
		source = br
	}

	var (
		doc *Document
		err error
	)

	switch format {
	case FormatText:
		doc, err = parseTextDocument(source, d.opts)
	case FormatBinary:
		doc, err = parseBinaryDocument(source, d.opts)
	default:
		err = fmt.Errorf("%w: %d", ErrInvalidFormat, format)
	}

	if err != nil {
		d.decodeErr = err
		return nil, err
	}

	doc.Format = format
	d.decoded = doc
	return doc, nil
}

// NextEvent returns the next DFS event for the decoded document.
func (d *Decoder) NextEvent() (Event, error) {
	if d.events == nil {
		doc, err := d.DecodeDocument()
		if err != nil {
			return Event{}, err
		}

		d.events = newEventIterator(doc)
	}

	event, ok := d.events.next()
	if !ok {
		return Event{}, io.EOF
	}

	return event, nil
}

// Parse decodes text VDF from reader.
func Parse(r io.Reader) (*Document, error) {
	return NewDecoder(r, DecodeOptions{Format: FormatText}).DecodeDocument()
}

// ParseBytes decodes VDF from bytes using the given options.
func ParseBytes(data []byte, opts DecodeOptions) (*Document, error) {
	return NewDecoder(bytes.NewReader(data), opts).DecodeDocument()
}

// ParseString decodes text VDF from a string.
func ParseString(s string) (*Document, error) {
	return NewDecoder(strings.NewReader(s), DecodeOptions{Format: FormatText}).DecodeDocument()
}

// ParseFile decodes text VDF from file path.
func ParseFile(path string) (doc *Document, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close file: %w", cerr)
		}
	}()

	return Parse(f)
}

// detectStreamFormat peeks a short prefix and infers format heuristically.
func detectStreamFormat(r *bufio.Reader) (Format, error) {
	prefix, err := r.Peek(64)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, bufio.ErrBufferFull) {
		return FormatAuto, err
	}

	if len(prefix) == 0 {
		return FormatText, nil
	}

	if looksBinaryPrefix(prefix) {
		return FormatBinary, nil
	}

	return FormatText, nil
}

// normalizeDecodeOptions fills default values for decode options.
func normalizeDecodeOptions(opts DecodeOptions) DecodeOptions {
	if opts.Format == 0 {
		opts.Format = FormatAuto
	}

	return opts
}

// validateDecodeFormat checks whether decode format value is supported.
func validateDecodeFormat(format Format) error {
	if format < FormatAuto || format > FormatBinary {
		return fmt.Errorf("%w: %d", ErrInvalidFormat, format)
	}

	return nil
}

// bufferedReader returns one shared buffered reader instance for the decoder.
func (d *Decoder) bufferedReader() *bufio.Reader {
	if d.buffered != nil {
		return d.buffered
	}

	d.buffered = ensureBufferedReader(d.reader)
	return d.buffered
}
