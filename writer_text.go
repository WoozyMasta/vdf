// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Maxim Levchenko (WoozyMasta)
// Source: github.com/woozymasta/vdf

package vdf

import (
	"fmt"
	"io"
	"strings"
)

// startTextObject writes object header in manual text encoding mode.
func (e *Encoder) startTextObject(key string) error {
	indent := strings.Repeat(e.opts.Indent, e.manualDepth)
	if e.opts.Compact {
		_, err := fmt.Fprintf(e.w, "\"%s\" { ", escapeString(key))
		e.manualDepth++
		return err
	}

	_, err := fmt.Fprintf(e.w, "%s\"%s\"\n%s{\n", indent, escapeString(key), indent)
	if err != nil {
		return err
	}

	e.manualDepth++
	return nil
}

// endTextObject writes object footer in manual text encoding mode.
func (e *Encoder) endTextObject() error {
	indent := strings.Repeat(e.opts.Indent, e.manualDepth)
	if e.opts.Compact {
		_, err := fmt.Fprint(e.w, "} ")
		return err
	}

	_, err := fmt.Fprintf(e.w, "%s}\n", indent)
	return err
}

// writeTextLeaf writes one scalar key/value line in manual text mode.
func (e *Encoder) writeTextLeaf(key, value string) error {
	indent := strings.Repeat(e.opts.Indent, e.manualDepth)
	if e.opts.Compact {
		_, err := fmt.Fprintf(e.w, "\"%s\" \"%s\" ", escapeString(key), escapeString(value))
		return err
	}

	_, err := fmt.Fprintf(e.w, "%s\"%s\"\t\t\"%s\"\n", indent, escapeString(key), escapeString(value))
	return err
}

// encodeTextDocument writes the full document in text VDF format.
func encodeTextDocument(w io.Writer, doc *Document, opts EncodeOptions) error {
	roots := orderedNodes(doc.Roots, opts.Deterministic)

	for i, root := range roots {
		if err := encodeTextNode(w, root, opts, 0); err != nil {
			return err
		}

		if !opts.Compact && i < len(roots)-1 {
			if _, err := io.WriteString(w, "\n"); err != nil {
				return err
			}
		}
	}

	return nil
}

// encodeTextNode writes one AST node in text VDF format.
func encodeTextNode(w io.Writer, node *Node, opts EncodeOptions, depth int) error {
	indent := strings.Repeat(opts.Indent, depth)

	switch node.Kind {
	case NodeObject:
		if opts.Compact {
			if _, err := fmt.Fprintf(w, "\"%s\" { ", escapeString(node.Key)); err != nil {
				return err
			}

			// Reuse the same traversal ordering policy as document-level encode.
			children := orderedNodes(node.Children, opts.Deterministic)
			for _, child := range children {
				if err := encodeTextNode(w, child, opts, depth+1); err != nil {
					return err
				}
			}

			_, err := io.WriteString(w, "} ")
			return err
		}

		if _, err := fmt.Fprintf(w, "%s\"%s\"\n%s{\n", indent, escapeString(node.Key), indent); err != nil {
			return err
		}

		// Keep ordering behavior consistent across compact and pretty branches.
		children := orderedNodes(node.Children, opts.Deterministic)
		for _, child := range children {
			if err := encodeTextNode(w, child, opts, depth+1); err != nil {
				return err
			}
		}

		_, err := fmt.Fprintf(w, "%s}\n", indent)
		return err
	case NodeString, NodeUint32:
		value, err := textValueForNode(node)
		if err != nil {
			return err
		}

		if opts.Compact {
			_, err := fmt.Fprintf(w, "\"%s\" \"%s\" ", escapeString(node.Key), escapeString(value))
			return err
		}

		_, err = fmt.Fprintf(w, "%s\"%s\"\t\t\"%s\"\n", indent, escapeString(node.Key), escapeString(value))
		return err
	default:
		return fmt.Errorf("%w: unsupported node kind %d", ErrInvalidNodeState, node.Kind)
	}
}

// escapeString escapes special runes for text VDF output.
func escapeString(value string) string {
	if !strings.ContainsAny(value, "\\\"\n\t\r") {
		return value
	}

	var sb strings.Builder
	sb.Grow(len(value) + 8)

	for _, r := range value {
		switch r {
		case '\\':
			sb.WriteString("\\\\")
		case '"':
			sb.WriteString("\\\"")
		case '\n':
			sb.WriteString("\\n")
		case '\t':
			sb.WriteString("\\t")
		case '\r':
			sb.WriteString("\\r")
		default:
			sb.WriteRune(r)
		}
	}

	return sb.String()
}
