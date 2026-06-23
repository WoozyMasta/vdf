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
	events    *eventIterator // Event iterator for WalkEvents (AST-based).
	stream    *streamState   // Streaming state for NextEvent (true streaming).
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

// WalkEvents returns the next DFS traversal event for the decoded document.
// The full document is decoded into an AST on the first call;
// subsequent calls traverse that AST in depth-first order.
// Returns io.EOF when all events have been emitted.
//
// Use WalkEvents when you need the full document available for further access after iteration.
// For a one-pass, lower-memory alternative see NextEvent.
func (d *Decoder) WalkEvents() (Event, error) {
	if d.stream != nil {
		return Event{}, fmt.Errorf("%w: NextEvent streaming is already active on this decoder", ErrInvalidNodeState)
	}

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

// NextEvent returns the next event from the input stream without building an AST.
// On the first call it initialises a streaming reader;
// subsequent calls continue from that position.
// Returns io.EOF when all events have been consumed.
//
// NextEvent is a true streaming decoder:
// it reads and yields events one at a time with no intermediate AST,
// making it suitable for large inputs or constrained memory environments.
// The decoded document is not available after iteration;
// use WalkEvents when post-iteration document access is required.
func (d *Decoder) NextEvent() (Event, error) {
	if d.events != nil {
		return Event{}, fmt.Errorf("%w: WalkEvents is already active on this decoder", ErrInvalidNodeState)
	}

	if d.stream == nil {
		s, err := newStreamState(d.bufferedReader(), d.opts)
		if err != nil {
			return Event{}, err
		}

		d.stream = s
	}

	return d.stream.next()
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

// ParseFile decodes VDF from file path.
// Without options it decodes as text format.
func ParseFile(path string, opts ...DecodeOptions) (doc *Document, err error) {
	effective := DecodeOptions{Format: FormatText}
	if len(opts) > 0 {
		effective = opts[0]
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close file: %w", cerr)
		}
	}()

	return NewDecoder(f, effective).DecodeDocument()
}

// ParseTextFile decodes text VDF from file path.
func ParseTextFile(path string) (*Document, error) {
	return ParseFile(path, DecodeOptions{Format: FormatText})
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

	if opts.MaxDepth < 0 {
		opts.MaxDepth = 0
	}

	if opts.MaxNodes < 0 {
		opts.MaxNodes = 0
	}

	if opts.MaxKeyBytes < 0 {
		opts.MaxKeyBytes = 0
	}

	if opts.MaxValueBytes < 0 {
		opts.MaxValueBytes = 0
	}

	if opts.MaxStringBytes < 0 {
		opts.MaxStringBytes = 0
	}

	return opts
}

// effectiveKeyLimit returns the active byte limit for keys (0 = unlimited).
func effectiveKeyLimit(opts DecodeOptions) int {
	if opts.MaxKeyBytes > 0 {
		return opts.MaxKeyBytes
	}

	return opts.MaxStringBytes
}

// effectiveValueLimit returns the active byte limit for string values (0 = unlimited).
func effectiveValueLimit(opts DecodeOptions) int {
	if opts.MaxValueBytes > 0 {
		return opts.MaxValueBytes
	}

	return opts.MaxStringBytes
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
