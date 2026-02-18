// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Maxim Levchenko (WoozyMasta)
// Source: github.com/woozymasta/vdf

package vdf

import "errors"

var (
	// ErrInvalidFormat indicates unsupported format selection.
	ErrInvalidFormat = errors.New("invalid VDF format")
	// ErrUnrecognizedType indicates that an unknown binary VDF type byte was encountered.
	ErrUnrecognizedType = errors.New("unrecognized VDF type")
	// ErrBufferOverflow indicates that parsing attempted to read past available input bytes.
	ErrBufferOverflow = errors.New("buffer overflow")
	// ErrNullInString indicates that a binary VDF string contains an embedded null byte.
	ErrNullInString = errors.New("null byte found in string")
	// ErrUnsupportedMapValueType indicates that map conversion encountered an unsupported value type.
	ErrUnsupportedMapValueType = errors.New("unsupported map value type")
	// ErrIntOutOfRange indicates an integer cannot be represented as uint32.
	ErrIntOutOfRange = errors.New("integer out of uint32 range")
	// ErrDuplicateKeyInStrictMode indicates strict map conversion encountered duplicate keys.
	ErrDuplicateKeyInStrictMode = errors.New("duplicate key in strict map conversion")
	// ErrInvalidNodeState indicates AST node fields do not match node kind invariants.
	ErrInvalidNodeState = errors.New("invalid node state")
	// ErrDepthLimitExceeded indicates decode exceeded configured max depth.
	ErrDepthLimitExceeded = errors.New("maximum depth exceeded")
	// ErrNodeLimitExceeded indicates decode exceeded configured max node count.
	ErrNodeLimitExceeded = errors.New("maximum node count exceeded")
	// ErrUnexpectedEOFInQuotedString indicates that a quoted text token ended before its closing quote.
	ErrUnexpectedEOFInQuotedString = errors.New("unexpected EOF in quoted string")
	// ErrUnexpectedEOFInEscapeSequence indicates that an escape sequence ended before its escaped rune.
	ErrUnexpectedEOFInEscapeSequence = errors.New("unexpected EOF in escape sequence")
	// ErrUnexpectedCharacter indicates that the lexer found an invalid token start.
	ErrUnexpectedCharacter = errors.New("unexpected character")
	// ErrExpectedStringKey indicates that the parser expected a string token for a node key.
	ErrExpectedStringKey = errors.New("expected string key")
	// ErrExpectedValueOrObject indicates that the parser expected either a string value or an object start.
	ErrExpectedValueOrObject = errors.New("expected value or '{'")
	// ErrExpectedObjectStart indicates that the parser expected an opening object brace.
	ErrExpectedObjectStart = errors.New("expected '{'")
	// ErrUnexpectedEOFInObject indicates that the parser reached EOF before closing an object.
	ErrUnexpectedEOFInObject = errors.New("unexpected EOF, expected '}'")
)
