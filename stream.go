// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Maxim Levchenko (WoozyMasta)
// Source: github.com/woozymasta/vdf

package vdf

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// streamState holds decoder state for true one-pass streaming via NextEvent.
// It reads events directly from the input reader without building an AST.
type streamState struct {
	br           binaryReadReader // non-nil when streaming binary VDF
	lex          *textLexer       // non-nil when streaming text VDF
	opts         DecodeOptions    // decode options passed through from Decoder
	depth        int              // count of currently open object scopes
	format       Format           // resolved format (never FormatAuto after init)
	sentDocStart bool             // whether EventDocumentStart has been emitted
	done         bool             // whether EventDocumentEnd has been emitted (next call returns io.EOF)
}

// newStreamState initialises streaming from a buffered reader.
// Format auto-detection uses Peek without consuming bytes.
func newStreamState(br *bufio.Reader, opts DecodeOptions) (*streamState, error) {
	format := opts.Format
	if format == FormatAuto {
		prefix, err := br.Peek(64)
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, bufio.ErrBufferFull) {
			return nil, err
		}

		if looksBinaryPrefix(prefix) {
			format = FormatBinary
		} else {
			format = FormatText
		}
	}

	s := &streamState{format: format, opts: opts}

	switch format {
	case FormatText:
		// bufio.Reader satisfies runeReader so textLexer uses it directly.
		s.lex = newTextLexer(br, opts)

	case FormatBinary:
		s.br = br

	default:
		return nil, fmt.Errorf("%w: %d", ErrInvalidFormat, format)
	}

	return s, nil
}

// next returns the next streaming event.
// Emits EventDocumentStart on the first call,
// EventDocumentEnd when the underlying stream is exhausted,
// then io.EOF on all subsequent calls.
func (s *streamState) next() (Event, error) {
	if s.done {
		return Event{}, io.EOF
	}

	if !s.sentDocStart {
		s.sentDocStart = true
		return Event{Type: EventDocumentStart, Depth: 0}, nil
	}

	var ev Event
	var err error

	if s.format == FormatText {
		ev, err = s.nextText()
	} else {
		ev, err = s.nextBinary()
	}

	if errors.Is(err, io.EOF) {
		s.done = true
		return Event{Type: EventDocumentEnd, Depth: 0}, nil
	}

	return ev, err
}

// nextText reads one event from the text VDF stream.
// Reads two tokens when a key is encountered to determine whether
// the next token opens a child object or supplies a scalar value.
func (s *streamState) nextText() (Event, error) {
	tok, err := s.lex.nextToken()
	if err != nil {
		return Event{}, err
	}

	switch tok.kind {
	case textTokenEOF:
		return Event{}, io.EOF

	case textTokenRBrace:
		if s.depth == 0 {
			return Event{}, fmt.Errorf("%w: unexpected '}' at line %d", ErrUnexpectedCharacter, tok.line)
		}
		s.depth--
		return Event{Type: EventObjectEnd, Depth: s.depth + 1}, nil

	case textTokenString:
		key := tok.value
		if limit := effectiveKeyLimit(s.opts); limit > 0 && len(key) > limit {
			return Event{}, fmt.Errorf("%w: key %q len=%d limit=%d", ErrKeyTooLong, key, len(key), limit)
		}

		next, err := s.lex.nextToken()
		if err != nil {
			return Event{}, err
		}

		switch next.kind {
		case textTokenLBrace:
			if s.opts.MaxDepth > 0 && s.depth >= s.opts.MaxDepth {
				return Event{}, fmt.Errorf("%w: depth %d exceeds MaxDepth %d",
					ErrDepthLimitExceeded, s.depth+1, s.opts.MaxDepth)
			}
			s.depth++
			return Event{Type: EventObjectStart, Key: key, Depth: s.depth}, nil

		case textTokenString:
			val := next.value
			if limit := effectiveValueLimit(s.opts); limit > 0 && len(val) > limit {
				return Event{}, fmt.Errorf("%w: key %q len=%d limit=%d", ErrValueTooLong, key, len(val), limit)
			}
			return Event{Type: EventString, Key: key, Depth: s.depth + 1, StringValue: &val}, nil

		default:
			return Event{}, fmt.Errorf("%w at line %d col %d",
				ErrExpectedValueOrObject, tok.line, tok.col)
		}

	default:
		return Event{}, fmt.Errorf("%w: unexpected token at line %d", ErrUnexpectedCharacter, tok.line)
	}
}

// nextBinary reads one event from the binary VDF stream.
// Binary type bytes drive the state machine; no lookahead is needed.
func (s *streamState) nextBinary() (Event, error) {
	typeByte, err := s.br.ReadByte()
	if errors.Is(err, io.EOF) {
		return Event{}, io.EOF
	}

	if err != nil {
		return Event{}, err
	}

	switch typeByte {
	case binaryTypeMapEnd:
		if s.depth == 0 {
			// Outer document terminator (bare 0x08 at depth 0).
			return Event{}, io.EOF
		}
		s.depth--
		return Event{Type: EventObjectEnd, Depth: s.depth + 1}, nil

	case binaryTypeMapStart:
		key, err := s.binaryKey()
		if err != nil {
			return Event{}, err
		}

		if s.opts.MaxDepth > 0 && s.depth >= s.opts.MaxDepth {
			return Event{}, fmt.Errorf("%w: depth %d exceeds MaxDepth %d",
				ErrDepthLimitExceeded, s.depth+1, s.opts.MaxDepth)
		}

		s.depth++
		return Event{Type: EventObjectStart, Key: key, Depth: s.depth}, nil

	case binaryTypeString:
		key, err := s.binaryKey()
		if err != nil {
			return Event{}, err
		}

		val, err := s.binaryValue()
		if err != nil {
			return Event{}, err
		}

		return Event{Type: EventString, Key: key, Depth: s.depth + 1, StringValue: &val}, nil

	case binaryTypeNumber:
		key, err := s.binaryKey()
		if err != nil {
			return Event{}, err
		}

		var raw [4]byte
		if _, err := io.ReadFull(s.br, raw[:]); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return Event{}, ErrBufferOverflow
			}

			return Event{}, err
		}

		v := binary.LittleEndian.Uint32(raw[:])
		return Event{Type: EventUint32, Key: key, Depth: s.depth + 1, Uint32Value: &v}, nil

	default:
		return Event{}, fmt.Errorf("%w: 0x%02x", ErrUnrecognizedType, typeByte)
	}
}

// binaryNullTerm reads one null-terminated string from the binary stream.
// Reuses the shared buffer pool from binary_decode.go.
func (s *streamState) binaryNullTerm() (string, error) {
	bufPtr := binaryStringBufferPool.Get().(*[]byte)
	buf := (*bufPtr)[:0]

	defer func() {
		if cap(buf) > 4096 {
			buf = make([]byte, 0, 64)
		}

		*bufPtr = buf
		binaryStringBufferPool.Put(bufPtr)
	}()

	for {
		b, err := s.br.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return "", ErrBufferOverflow
			}

			return "", err
		}

		if b == 0 {
			return string(buf), nil
		}

		buf = append(buf, b)
	}
}

// binaryKey reads a null-terminated key string and enforces key length limits.
func (s *streamState) binaryKey() (string, error) {
	str, err := s.binaryNullTerm()
	if err != nil {
		return "", err
	}

	if limit := effectiveKeyLimit(s.opts); limit > 0 && len(str) > limit {
		return "", fmt.Errorf("%w: len=%d limit=%d", ErrKeyTooLong, len(str), limit)
	}

	return str, nil
}

// binaryValue reads a null-terminated value string and enforces value length limits.
func (s *streamState) binaryValue() (string, error) {
	str, err := s.binaryNullTerm()
	if err != nil {
		return "", err
	}

	if limit := effectiveValueLimit(s.opts); limit > 0 && len(str) > limit {
		return "", fmt.Errorf("%w: len=%d limit=%d", ErrValueTooLong, len(str), limit)
	}

	return str, nil
}
