// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Maxim Levchenko (WoozyMasta)
// Source: github.com/woozymasta/vdf

package vdf

// Map represents a generic key-value mapping used by explicit adapters.
// It is inherently lossy for duplicate keys and ordering.
type Map map[string]any

// Event is a streaming traversal event.
type Event struct {
	// StringValue is set for EventString.
	StringValue *string `json:"string_value,omitempty" yaml:"string_value,omitempty"`
	// Uint32Value is set for EventUint32.
	Uint32Value *uint32 `json:"uint32_value,omitempty" yaml:"uint32_value,omitempty"`
	// Key is the node key associated with this event.
	Key string `json:"key,omitempty" yaml:"key,omitempty"`
	// Depth is the traversal depth for this event.
	Depth int `json:"depth" yaml:"depth"`
	// Type is the event kind.
	Type EventType `json:"type" yaml:"type"`
}

// EventType represents a decoded event type from streaming traversal.
type EventType uint8

const (
	// EventDocumentStart marks beginning of a document stream.
	EventDocumentStart EventType = iota + 1
	// EventDocumentEnd marks end of a document stream.
	EventDocumentEnd
	// EventObjectStart marks beginning of an object node.
	EventObjectStart
	// EventObjectEnd marks end of an object node.
	EventObjectEnd
	// EventString marks a string leaf node.
	EventString
	// EventUint32 marks a uint32 leaf node.
	EventUint32
)

// Document represents a complete VDF document.
type Document struct {
	// Roots contains top-level nodes in source order.
	Roots []*Node `json:"roots,omitempty" yaml:"roots,omitempty"`
	// Format is the source or intended encode format.
	Format Format `json:"format,omitempty" yaml:"format,omitempty"`
}

// Node represents a VDF AST node.
type Node struct {
	// StringValue is set for NodeString.
	StringValue *string `json:"string_value,omitempty" yaml:"string_value,omitempty"`
	// Uint32Value is set for NodeUint32.
	Uint32Value *uint32 `json:"uint32_value,omitempty" yaml:"uint32_value,omitempty"`
	// Key is the node key.
	Key string `json:"key" yaml:"key"`
	// Children are set for NodeObject and preserve source order.
	Children []*Node `json:"children,omitempty" yaml:"children,omitempty"`
	// Kind defines the node payload shape.
	Kind NodeKind `json:"kind" yaml:"kind"`
}

// NodeKind defines the value type represented by a node.
type NodeKind uint8

const (
	// NodeObject is a container node with ordered children.
	NodeObject NodeKind = iota + 1
	// NodeString is a leaf node containing a string value.
	NodeString
	// NodeUint32 is a leaf node containing an unsigned 32-bit value.
	NodeUint32
)

// DecodeOptions controls decoder behavior.
type DecodeOptions struct {
	// Format selects expected input format.
	Format Format
	// Strict enables stricter validation paths where available.
	Strict bool
	// MaxDepth limits nested object depth (0 means unlimited).
	MaxDepth int
	// MaxNodes limits total parsed nodes (0 means unlimited).
	MaxNodes int
}

// EncodeOptions controls encoder behavior.
type EncodeOptions struct {
	// Indent sets one indentation level for text format.
	Indent string
	// Format selects output format.
	Format Format
	// Compact enables compact text encoding.
	Compact bool
	// Deterministic enables stable key ordering during encode.
	Deterministic bool
	// Validate enables full document validation before encoding.
	Validate bool
}

// Format defines how encoded/decoded VDF data should be interpreted.
type Format uint8

const (
	// FormatAuto enables format auto-detection for decoding.
	FormatAuto Format = iota
	// FormatText selects text VDF format.
	FormatText
	// FormatBinary selects binary VDF format.
	FormatBinary
)
