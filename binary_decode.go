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
	"sync"
)

const (
	// binaryTypeMapStart marks an object value.
	binaryTypeMapStart byte = 0x00
	// binaryTypeString marks a string value.
	binaryTypeString byte = 0x01
	// binaryTypeNumber marks a uint32 value.
	binaryTypeNumber byte = 0x02
	// binaryTypeMapEnd marks end of current object map.
	binaryTypeMapEnd byte = 0x08
)

// binaryStringBufferPool reuses temporary buffers for binary string decoding.
var binaryStringBufferPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 0, 64)
		return &buf
	},
}

// binaryDecoder parses binary VDF stream.
type binaryDecoder struct {
	reader    binaryReadReader // Reader for the input.
	opts      DecodeOptions    // Decode options.
	nodeCount int              // Number of nodes parsed.
}

// binaryReadReader is the binary decode stream contract.
type binaryReadReader interface {
	io.Reader
	ReadByte() (byte, error)
}

// parseBinaryDocument decodes binary VDF from a stream.
func parseBinaryDocument(r io.Reader, opts DecodeOptions) (*Document, error) {
	decoder := &binaryDecoder{
		reader: ensureBinaryReader(r),
		opts:   opts,
	}

	return decoder.decodeDocument()
}

// decodeDocument decodes a full binary document.
func (d *binaryDecoder) decodeDocument() (*Document, error) {
	doc := NewDocumentWithFormat(FormatBinary)

	for {
		typeByte, err := d.readTypeByte()
		if errors.Is(err, io.EOF) {
			if len(doc.Roots) == 0 {
				return doc, nil
			}

			return nil, ErrBufferOverflow
		}

		if err != nil {
			return nil, err
		}

		if typeByte == binaryTypeMapEnd {
			return doc, nil
		}

		node, err := d.decodeEntry(typeByte, 1)
		if err != nil {
			return nil, err
		}

		if d.opts.Strict && containsKey(doc.Roots, node.Key) {
			return nil, fmt.Errorf("%w: root key %q", ErrDuplicateKeyInStrictMode, node.Key)
		}

		doc.AddRoot(node)
	}
}

// decodeEntry decodes one key/value entry based on its type byte.
func (d *binaryDecoder) decodeEntry(typeByte byte, depth int) (*Node, error) {
	if err := d.checkDepth(depth); err != nil {
		return nil, err
	}

	key, err := d.readNullTerminatedString()
	if err != nil {
		return nil, err
	}

	switch typeByte {
	case binaryTypeMapStart:
		node := NewObjectNode(key)
		if err := d.incrementNodeCount(); err != nil {
			return nil, err
		}

		for {
			childType, err := d.readTypeByte()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil, ErrBufferOverflow
				}

				return nil, err
			}

			if childType == binaryTypeMapEnd {
				// End marker closes only the current nested object scope.
				return node, nil
			}

			// Recursively decode each nested entry until map end is reached.
			child, err := d.decodeEntry(childType, depth+1)
			if err != nil {
				return nil, err
			}

			if d.opts.Strict && containsKey(node.Children, child.Key) {
				return nil, fmt.Errorf("%w: key %q in object %q", ErrDuplicateKeyInStrictMode, child.Key, key)
			}

			node.Add(child)
		}
	case binaryTypeString:
		value, err := d.readNullTerminatedString()
		if err != nil {
			return nil, err
		}

		node := NewStringNode(key, value)
		if err := d.incrementNodeCount(); err != nil {
			return nil, err
		}

		return node, nil
	case binaryTypeNumber:
		value, err := d.readUint32()
		if err != nil {
			return nil, err
		}

		node := NewUint32Node(key, value)
		if err := d.incrementNodeCount(); err != nil {
			return nil, err
		}

		return node, nil
	default:
		return nil, fmt.Errorf("%w: 0x%02x", ErrUnrecognizedType, typeByte)
	}
}

// readTypeByte reads one binary type marker byte.
func (d *binaryDecoder) readTypeByte() (byte, error) {
	b, err := d.reader.ReadByte()
	if err != nil {
		return 0, err
	}

	return b, nil
}

// readNullTerminatedString reads one null-terminated string.
func (d *binaryDecoder) readNullTerminatedString() (string, error) {
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
		b, err := d.reader.ReadByte()
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

// readUint32 reads little-endian uint32.
func (d *binaryDecoder) readUint32() (uint32, error) {
	var raw [4]byte
	if _, err := io.ReadFull(d.reader, raw[:]); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return 0, ErrBufferOverflow
		}

		return 0, err
	}

	return binary.LittleEndian.Uint32(raw[:]), nil
}

// checkDepth validates configured maximum nesting depth.
func (d *binaryDecoder) checkDepth(depth int) error {
	if d.opts.MaxDepth > 0 && depth > d.opts.MaxDepth {
		return fmt.Errorf("%w: depth %d > %d", ErrDepthLimitExceeded, depth, d.opts.MaxDepth)
	}

	return nil
}

// incrementNodeCount validates configured maximum node count.
func (d *binaryDecoder) incrementNodeCount() error {
	d.nodeCount++
	if d.opts.MaxNodes > 0 && d.nodeCount > d.opts.MaxNodes {
		return fmt.Errorf("%w: nodes %d > %d", ErrNodeLimitExceeded, d.nodeCount, d.opts.MaxNodes)
	}

	return nil
}

// looksBinaryPrefix checks whether prefix resembles binary VDF payload.
func looksBinaryPrefix(data []byte) bool {
	if len(data) == 0 {
		return false
	}

	first := data[0]
	if first != binaryTypeMapStart && first != binaryTypeString && first != binaryTypeNumber {
		return false
	}

	checkLen := min(50, len(data))
	for i := 1; i < checkLen; i++ {
		if data[i] == 0 {
			return true
		}
	}

	return false
}

// ensureBufferedReader wraps reader into bufio.Reader when needed.
func ensureBufferedReader(r io.Reader) *bufio.Reader {
	if br, ok := r.(*bufio.Reader); ok {
		return br
	}

	return bufio.NewReader(r)
}

// ensureBinaryReader wraps non-byte readers to support efficient byte-oriented decode.
func ensureBinaryReader(r io.Reader) binaryReadReader {
	if br, ok := r.(binaryReadReader); ok {
		return br
	}

	return bufio.NewReader(r)
}
