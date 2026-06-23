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
		if err := writeEscaped(e.w, key); err != nil {
			return err
		}
		_, err := io.WriteString(e.w, " { ")
		e.manualDepth++
		return err
	}

	if _, err := io.WriteString(e.w, indent); err != nil {
		return err
	}
	if err := writeEscaped(e.w, key); err != nil {
		return err
	}
	if _, err := io.WriteString(e.w, "\n"); err != nil {
		return err
	}
	if _, err := io.WriteString(e.w, indent); err != nil {
		return err
	}
	if _, err := io.WriteString(e.w, "{\n"); err != nil {
		return err
	}
	e.manualDepth++
	return nil
}

// endTextObject writes object footer in manual text encoding mode.
func (e *Encoder) endTextObject() error {
	if e.opts.Compact {
		_, err := io.WriteString(e.w, "} ")
		return err
	}

	if _, err := io.WriteString(e.w, strings.Repeat(e.opts.Indent, e.manualDepth)); err != nil {
		return err
	}
	_, err := io.WriteString(e.w, "}\n")
	return err
}

// writeTextLeaf writes one scalar key/value line in manual text mode.
func (e *Encoder) writeTextLeaf(key, value string) error {
	indent := strings.Repeat(e.opts.Indent, e.manualDepth)
	if e.opts.Compact {
		if err := writeEscaped(e.w, key); err != nil {
			return err
		}
		if _, err := io.WriteString(e.w, " "); err != nil {
			return err
		}
		if err := writeEscaped(e.w, value); err != nil {
			return err
		}
		_, err := io.WriteString(e.w, " ")
		return err
	}

	if _, err := io.WriteString(e.w, indent); err != nil {
		return err
	}
	if err := writeEscaped(e.w, key); err != nil {
		return err
	}
	if _, err := io.WriteString(e.w, "\t\t"); err != nil {
		return err
	}
	if err := writeEscaped(e.w, value); err != nil {
		return err
	}
	_, err := io.WriteString(e.w, "\n")
	return err
}

// encodeTextDocument writes the full document in text VDF format.
func encodeTextDocument(w io.Writer, doc *Document, opts EncodeOptions) error {
	roots := orderedNodes(doc.Roots, opts.Deterministic)

	for i, root := range roots {
		if err := encodeTextNode(w, root, opts, ""); err != nil {
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
// indent is the current indentation prefix (accumulated by callers).
func encodeTextNode(w io.Writer, node *Node, opts EncodeOptions, indent string) error {
	switch node.Kind {
	case NodeObject:
		if opts.Compact {
			if err := writeEscaped(w, node.Key); err != nil {
				return err
			}
			if _, err := io.WriteString(w, " { "); err != nil {
				return err
			}

			children := orderedNodes(node.Children, opts.Deterministic)
			childIndent := indent + opts.Indent
			for _, child := range children {
				if err := encodeTextNode(w, child, opts, childIndent); err != nil {
					return err
				}
			}

			_, err := io.WriteString(w, "} ")
			return err
		}

		if _, err := io.WriteString(w, indent); err != nil {
			return err
		}
		if err := writeEscaped(w, node.Key); err != nil {
			return err
		}
		if _, err := io.WriteString(w, "\n"); err != nil {
			return err
		}
		if _, err := io.WriteString(w, indent); err != nil {
			return err
		}
		if _, err := io.WriteString(w, "{\n"); err != nil {
			return err
		}

		children := orderedNodes(node.Children, opts.Deterministic)
		childIndent := indent + opts.Indent
		for _, child := range children {
			if err := encodeTextNode(w, child, opts, childIndent); err != nil {
				return err
			}
		}

		if _, err := io.WriteString(w, indent); err != nil {
			return err
		}
		_, err := io.WriteString(w, "}\n")
		return err

	case NodeString, NodeUint32:
		value, err := textValueForNode(node)
		if err != nil {
			return err
		}

		if opts.Compact {
			if err := writeEscaped(w, node.Key); err != nil {
				return err
			}
			if _, err := io.WriteString(w, " "); err != nil {
				return err
			}
			if err := writeEscaped(w, value); err != nil {
				return err
			}
			_, err := io.WriteString(w, " ")
			return err
		}

		if _, err := io.WriteString(w, indent); err != nil {
			return err
		}
		if err := writeEscaped(w, node.Key); err != nil {
			return err
		}
		if _, err := io.WriteString(w, "\t\t"); err != nil {
			return err
		}
		if err := writeEscaped(w, value); err != nil {
			return err
		}
		_, err = io.WriteString(w, "\n")
		return err

	default:
		return fmt.Errorf("%w: unsupported node kind %d", ErrInvalidNodeState, node.Kind)
	}
}

// writeEscaped writes a quoted, VDF-escaped string directly to w.
// For strings with no special characters no intermediate string is allocated.
func writeEscaped(w io.Writer, s string) error {
	if _, err := io.WriteString(w, `"`); err != nil {
		return err
	}

	start := 0
	for i := 0; i < len(s); i++ {
		var esc string
		switch s[i] {
		case '\\':
			esc = `\\`
		case '"':
			esc = `\"`
		case '\n':
			esc = `\n`
		case '\t':
			esc = `\t`
		case '\r':
			esc = `\r`
		default:
			continue
		}

		if i > start {
			if _, err := io.WriteString(w, s[start:i]); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(w, esc); err != nil {
			return err
		}
		start = i + 1
	}

	if start < len(s) {
		if _, err := io.WriteString(w, s[start:]); err != nil {
			return err
		}
	}

	_, err := io.WriteString(w, `"`)
	return err
}
