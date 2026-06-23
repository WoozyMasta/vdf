// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Maxim Levchenko (WoozyMasta)
// Source: github.com/woozymasta/vdf

package vdf

import "fmt"

// Builder constructs a VDF Document using a fluent API.
// Methods return the receiver so calls can be chained.
// The first error encountered is stored and silently skips subsequent calls;
// Document returns it at finalization.
type Builder struct {
	err   error
	doc   *Document
	stack []*Node
}

// NewBuilder creates a Builder with a single root object node.
func NewBuilder(rootKey string) *Builder {
	root := NewObjectNode(rootKey)
	return &Builder{
		doc:   NewDocument(),
		stack: []*Node{root},
	}
}

// Set adds a string key/value node to the current object.
func (b *Builder) Set(key, value string) *Builder {
	if b.err != nil {
		return b
	}

	b.current().Add(NewStringNode(key, value))
	return b
}

// SetUint32 adds a uint32 key/value node to the current object.
func (b *Builder) SetUint32(key string, value uint32) *Builder {
	if b.err != nil {
		return b
	}

	b.current().Add(NewUint32Node(key, value))
	return b
}

// Object opens a nested object scope, invokes fn within it, then closes it.
func (b *Builder) Object(key string, fn func(*Builder)) *Builder {
	if b.err != nil {
		return b
	}

	child := NewObjectNode(key)
	b.current().Add(child)
	b.stack = append(b.stack, child)
	fn(b)
	b.stack = b.stack[:len(b.stack)-1]
	return b
}

// Document finalizes the builder and returns the completed Document.
// Returns an error if any call previously failed or if unclosed objects remain.
// After a successful call, the builder is considered finalized; further calls return an error.
func (b *Builder) Document() (*Document, error) {
	if b.err != nil {
		return nil, b.err
	}

	if len(b.stack) != 1 {
		return nil, fmt.Errorf("%w: %d unclosed object(s)", ErrInvalidNodeState, len(b.stack)-1)
	}

	b.doc.AddRoot(b.stack[0])
	b.stack = b.stack[:0]
	b.err = fmt.Errorf("%w: builder already finalized", ErrInvalidNodeState)
	return b.doc, nil
}

// current returns the top-of-stack object node.
func (b *Builder) current() *Node {
	return b.stack[len(b.stack)-1]
}
