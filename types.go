// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Maxim Levchenko (WoozyMasta)
// Source: github.com/woozymasta/vdf

package vdf

import (
	"fmt"
	"math"
	"strconv"
)

// NewDocument creates an empty document with auto format marker.
func NewDocument() *Document {
	return &Document{
		Format: FormatAuto,
		Roots:  make([]*Node, 0, 1),
	}
}

// NewDocumentWithFormat creates an empty document with explicit format marker.
func NewDocumentWithFormat(format Format) *Document {
	doc := NewDocument()
	doc.Format = format

	return doc
}

// NewObjectNode creates an object node with the provided key.
func NewObjectNode(key string) *Node {
	return &Node{
		Key:      key,
		Kind:     NodeObject,
		Children: make([]*Node, 0, 4),
	}
}

// NewStringNode creates a string node with the provided key and value.
func NewStringNode(key, value string) *Node {
	return &Node{
		Key:         key,
		Kind:        NodeString,
		StringValue: &value,
	}
}

// NewUint32Node creates a uint32 node with the provided key and value.
func NewUint32Node(key string, value uint32) *Node {
	return &Node{
		Key:         key,
		Kind:        NodeUint32,
		Uint32Value: &value,
	}
}

// Add appends a child node to an object node.
func (n *Node) Add(child *Node) {
	if n == nil || n.Kind != NodeObject || child == nil {
		return
	}

	n.Children = append(n.Children, child)
}

// First returns the first child with the given key.
func (n *Node) First(key string) *Node {
	if n == nil || n.Kind != NodeObject {
		return nil
	}

	for _, child := range n.Children {
		if child != nil && child.Key == key {
			return child
		}
	}

	return nil
}

// All returns all children with the given key in source order.
func (n *Node) All(key string) []*Node {
	if n == nil || n.Kind != NodeObject {
		return nil
	}

	matches := make([]*Node, 0)
	for _, child := range n.Children {
		if child != nil && child.Key == key {
			matches = append(matches, child)
		}
	}

	return matches
}

// AddRoot appends a root node to the document.
func (d *Document) AddRoot(node *Node) {
	if d == nil || node == nil {
		return
	}

	d.Roots = append(d.Roots, node)
}

// Validate ensures document and node invariants are satisfied.
func (d *Document) Validate() error {
	if d == nil {
		return fmt.Errorf("%w: nil document", ErrInvalidNodeState)
	}

	seen := make(map[*Node]struct{})
	for i, root := range d.Roots {
		if err := validateNode(root, seen); err != nil {
			return fmt.Errorf("root[%d]: %w", i, err)
		}
	}

	return nil
}

// ToMapStrict converts document to map and fails on duplicate keys.
func (d *Document) ToMapStrict() (Map, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}

	out := Map{}
	for _, root := range d.Roots {
		if _, exists := out[root.Key]; exists {
			return nil, fmt.Errorf("%w: root key %q", ErrDuplicateKeyInStrictMode, root.Key)
		}

		value, err := nodeToStrictValue(root)
		if err != nil {
			return nil, err
		}

		out[root.Key] = value
	}

	return out, nil
}

// ToMapLossy converts document to map using last-write-wins for duplicate keys.
func (d *Document) ToMapLossy() Map {
	out := Map{}
	if d == nil {
		return out
	}

	for _, root := range d.Roots {
		if root == nil {
			continue
		}

		out[root.Key] = nodeToLossyValue(root)
	}

	return out
}

// FromMap builds a document with one object root from a map.
func FromMap(rootKey string, m Map) (*Document, error) {
	doc := NewDocumentWithFormat(FormatAuto)
	root := NewObjectNode(rootKey)

	children, err := mapToNodeChildren(m)
	if err != nil {
		return nil, err
	}

	root.Children = append(root.Children, children...)
	doc.AddRoot(root)

	if err := doc.Validate(); err != nil {
		return nil, err
	}

	return doc, nil
}

// validateNode validates a node recursively and detects cycles.
func validateNode(node *Node, seen map[*Node]struct{}) error {
	if node == nil {
		return fmt.Errorf("%w: nil node", ErrInvalidNodeState)
	}

	if _, exists := seen[node]; exists {
		return fmt.Errorf("%w: cyclic node %q", ErrInvalidNodeState, node.Key)
	}
	seen[node] = struct{}{}
	defer delete(seen, node)

	switch node.Kind {
	case NodeObject:
		if node.StringValue != nil || node.Uint32Value != nil {
			return fmt.Errorf("%w: object %q has scalar payload", ErrInvalidNodeState, node.Key)
		}

		for i, child := range node.Children {
			if err := validateNode(child, seen); err != nil {
				return fmt.Errorf("child[%d]: %w", i, err)
			}
		}

	case NodeString:
		if node.StringValue == nil {
			return fmt.Errorf("%w: string node %q missing value", ErrInvalidNodeState, node.Key)
		}

		if node.Uint32Value != nil || len(node.Children) != 0 {
			return fmt.Errorf("%w: string node %q has invalid extra data", ErrInvalidNodeState, node.Key)
		}

	case NodeUint32:
		if node.Uint32Value == nil {
			return fmt.Errorf("%w: uint32 node %q missing value", ErrInvalidNodeState, node.Key)
		}

		if node.StringValue != nil || len(node.Children) != 0 {
			return fmt.Errorf("%w: uint32 node %q has invalid extra data", ErrInvalidNodeState, node.Key)
		}

	default:
		return fmt.Errorf("%w: unknown node kind %d", ErrInvalidNodeState, node.Kind)
	}

	return nil
}

// nodeToStrictValue converts a node to map-friendly value with duplicate detection.
func nodeToStrictValue(node *Node) (any, error) {
	switch node.Kind {
	case NodeString:
		return *node.StringValue, nil

	case NodeUint32:
		return *node.Uint32Value, nil

	case NodeObject:
		m := Map{}
		for _, child := range node.Children {
			if _, exists := m[child.Key]; exists {
				return nil, fmt.Errorf("%w: key %q", ErrDuplicateKeyInStrictMode, child.Key)
			}

			value, err := nodeToStrictValue(child)
			if err != nil {
				return nil, err
			}

			m[child.Key] = value
		}
		return m, nil

	default:
		return nil, fmt.Errorf("%w: unknown node kind %d", ErrInvalidNodeState, node.Kind)
	}
}

// nodeToLossyValue converts a node to map-friendly value with last-write-wins semantics.
func nodeToLossyValue(node *Node) any {
	switch node.Kind {
	case NodeString:
		return *node.StringValue

	case NodeUint32:
		return *node.Uint32Value

	case NodeObject:
		m := Map{}
		for _, child := range node.Children {
			m[child.Key] = nodeToLossyValue(child)
		}
		return m

	default:
		return nil
	}
}

// mapToNodeChildren converts map entries to ordered node children.
func mapToNodeChildren(m Map) ([]*Node, error) {
	children := make([]*Node, 0, len(m))
	for key, value := range m {
		node, err := mapValueToNode(key, value)
		if err != nil {
			return nil, err
		}

		children = append(children, node)
	}

	return children, nil
}

// mapValueToNode converts a single map value to a node.
func mapValueToNode(key string, value any) (*Node, error) {
	switch val := value.(type) {
	case string:
		return NewStringNode(key, val), nil

	case uint32:
		return NewUint32Node(key, val), nil

	case int:
		if val < 0 || val > math.MaxUint32 {
			return nil, fmt.Errorf("%w: key %q int=%d", ErrIntOutOfRange, key, val)
		}
		return NewUint32Node(key, uint32(val)), nil

	case int64:
		if val < 0 || val > math.MaxUint32 {
			return nil, fmt.Errorf("%w: key %q int64=%d", ErrIntOutOfRange, key, val)
		}
		return NewUint32Node(key, uint32(val)), nil

	case Map:
		obj := NewObjectNode(key)
		children, err := mapToNodeChildren(val)
		if err != nil {
			return nil, err
		}
		obj.Children = append(obj.Children, children...)
		return obj, nil

	case map[string]any:
		return mapValueToNode(key, Map(val))

	case float64:
		if val < 0 || val > math.MaxUint32 || val != math.Trunc(val) {
			return nil, fmt.Errorf("%w: key %q float64=%v", ErrIntOutOfRange, key, val)
		}
		return NewUint32Node(key, uint32(val)), nil

	default:
		return nil, fmt.Errorf("%w: key %q type=%T", ErrUnsupportedMapValueType, key, value)
	}
}

// textValueForNode converts leaf nodes to textual value for text format writer.
func textValueForNode(node *Node) (string, error) {
	switch node.Kind {
	case NodeString:
		if node.StringValue == nil {
			return "", fmt.Errorf("%w: string node %q missing value", ErrInvalidNodeState, node.Key)
		}
		return *node.StringValue, nil

	case NodeUint32:
		if node.Uint32Value == nil {
			return "", fmt.Errorf("%w: uint32 node %q missing value", ErrInvalidNodeState, node.Key)
		}
		return strconv.FormatUint(uint64(*node.Uint32Value), 10), nil

	default:
		return "", fmt.Errorf("%w: node %q kind=%d cannot be formatted as text leaf", ErrInvalidNodeState, node.Key, node.Kind)
	}
}
