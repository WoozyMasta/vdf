// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Maxim Levchenko (WoozyMasta)
// Source: github.com/woozymasta/vdf

package vdf

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

// binaryZeroByte is a zero byte.
var binaryZeroByte = [1]byte{0}

// byteWriter is the binary encode stream contract.
type byteWriter interface {
	WriteByte(byte) error
}

// encodeBinaryDocument writes document in binary VDF format.
func encodeBinaryDocument(w io.Writer, doc *Document, opts EncodeOptions) error {
	roots := orderedNodes(doc.Roots, opts.Deterministic)
	for _, root := range roots {
		if err := encodeBinaryNode(w, root, opts); err != nil {
			return err
		}
	}

	if err := writeBinaryByte(w, binaryTypeMapEnd); err != nil {
		return err
	}

	return nil
}

// encodeBinaryNode writes a single AST node as binary entry.
func encodeBinaryNode(w io.Writer, node *Node, opts EncodeOptions) error {
	switch node.Kind {
	case NodeObject:
		if err := writeBinaryByte(w, binaryTypeMapStart); err != nil {
			return err
		}

		if err := writeNullTerminatedString(w, node.Key); err != nil {
			return err
		}

		children := orderedNodes(node.Children, opts.Deterministic)
		for _, child := range children {
			if err := encodeBinaryNode(w, child, opts); err != nil {
				return err
			}
		}

		if err := writeBinaryByte(w, binaryTypeMapEnd); err != nil {
			return err
		}

		return nil
	case NodeString:
		if err := writeBinaryByte(w, binaryTypeString); err != nil {
			return err
		}

		if err := writeNullTerminatedString(w, node.Key); err != nil {
			return err
		}

		if node.StringValue == nil {
			return fmt.Errorf("%w: nil string value for key %q", ErrInvalidNodeState, node.Key)
		}

		return writeNullTerminatedString(w, *node.StringValue)
	case NodeUint32:
		if err := writeBinaryByte(w, binaryTypeNumber); err != nil {
			return err
		}

		if err := writeNullTerminatedString(w, node.Key); err != nil {
			return err
		}

		if node.Uint32Value == nil {
			return fmt.Errorf("%w: nil uint32 value for key %q", ErrInvalidNodeState, node.Key)
		}

		var raw [4]byte
		binary.LittleEndian.PutUint32(raw[:], *node.Uint32Value)
		_, err := w.Write(raw[:])
		return err
	default:
		return fmt.Errorf("%w: unknown node kind %d", ErrInvalidNodeState, node.Kind)
	}
}

// writeBinaryByte writes one byte to output stream.
func writeBinaryByte(w io.Writer, b byte) error {
	if bw, ok := w.(byteWriter); ok {
		return bw.WriteByte(b)
	}

	var one [1]byte
	one[0] = b
	_, err := w.Write(one[:])
	return err
}

// writeNullTerminatedString writes one zero-terminated string.
func writeNullTerminatedString(w io.Writer, value string) error {
	if strings.IndexByte(value, 0) >= 0 {
		return ErrNullInString
	}

	if _, err := io.WriteString(w, value); err != nil {
		return err
	}

	if _, err := w.Write(binaryZeroByte[:]); err != nil {
		return err
	}

	return nil
}

// estimateBinaryDocumentSize returns an approximate encoded byte size.
func estimateBinaryDocumentSize(doc *Document, deterministic bool) int {
	if doc == nil {
		return 0
	}

	size := 1 // trailing root map-end byte
	roots := orderedNodes(doc.Roots, deterministic)
	for _, root := range roots {
		size += estimateBinaryNodeSize(root, deterministic)
	}

	return size
}

// estimateBinaryNodeSize returns encoded byte size for one AST node.
func estimateBinaryNodeSize(node *Node, deterministic bool) int {
	if node == nil {
		return 0
	}

	size := 1 + len(node.Key) + 1 // type byte + key + null

	switch node.Kind {
	case NodeObject:
		children := orderedNodes(node.Children, deterministic)
		for _, child := range children {
			size += estimateBinaryNodeSize(child, deterministic)
		}

		size++ // object end byte
	case NodeString:
		if node.StringValue != nil {
			size += len(*node.StringValue) + 1
		}

	case NodeUint32:
		size += 4
	}

	return size
}
