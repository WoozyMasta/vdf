// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Maxim Levchenko (WoozyMasta)
// Source: github.com/woozymasta/vdf

package vdf

import (
	"fmt"
	"io"
)

// textParser parses text-lexer tokens into AST nodes.
type textParser struct {
	lexer     *textLexer    // Lexer for the input.
	peeked    textToken     // Peeked token value.
	hasPeeked bool          // Whether peek token is set.
	opts      DecodeOptions // Decode options.
	nodeCount int           // Number of nodes parsed.
}

// parseTextDocument parses one full text VDF stream.
func parseTextDocument(r io.Reader, opts DecodeOptions) (*Document, error) {
	parser := &textParser{
		lexer: newTextLexer(r),
		opts:  opts,
	}

	doc := NewDocumentWithFormat(FormatText)

	for {
		tok, err := parser.peekToken()
		if err != nil {
			return nil, err
		}

		if tok.kind == textTokenEOF {
			return doc, nil
		}

		node, err := parser.parseNode(1)
		if err != nil {
			return nil, err
		}

		if parser.opts.Strict && containsKey(doc.Roots, node.Key) {
			return nil, fmt.Errorf("%w: root key %q", ErrDuplicateKeyInStrictMode, node.Key)
		}

		doc.AddRoot(node)
	}
}

// parseNode parses either a scalar key/value entry or object entry.
func (p *textParser) parseNode(depth int) (*Node, error) {
	if err := p.checkDepth(depth); err != nil {
		return nil, err
	}

	keyTok, err := p.nextToken()
	if err != nil {
		return nil, err
	}

	if keyTok.kind != textTokenString {
		return nil, fmt.Errorf("%w at line %d, col %d", ErrExpectedStringKey, keyTok.line, keyTok.col)
	}

	nextTok, err := p.peekToken()
	if err != nil {
		return nil, err
	}

	switch nextTok.kind {
	case textTokenString:
		valueTok, err := p.nextToken()
		if err != nil {
			return nil, err
		}

		node := NewStringNode(keyTok.value, valueTok.value)
		if err := p.incrementNodeCount(); err != nil {
			return nil, err
		}

		return node, nil
	case textTokenLBrace:
		return p.parseObject(keyTok.value, depth)
	default:
		return nil, fmt.Errorf("%w at line %d, col %d", ErrExpectedValueOrObject, nextTok.line, nextTok.col)
	}
}

// parseObject parses an object body until a matching closing brace.
func (p *textParser) parseObject(key string, depth int) (*Node, error) {
	lbrace, err := p.nextToken()
	if err != nil {
		return nil, err
	}

	if lbrace.kind != textTokenLBrace {
		return nil, fmt.Errorf("%w at line %d, col %d", ErrExpectedObjectStart, lbrace.line, lbrace.col)
	}

	node := NewObjectNode(key)
	if err := p.incrementNodeCount(); err != nil {
		return nil, err
	}

	for {
		tok, err := p.peekToken()
		if err != nil {
			return nil, err
		}

		// Closing brace completes the current object scope.
		if tok.kind == textTokenRBrace {
			if _, err := p.nextToken(); err != nil {
				return nil, err
			}

			return node, nil
		}

		if tok.kind == textTokenEOF {
			return nil, fmt.Errorf("%w for object %q", ErrUnexpectedEOFInObject, key)
		}

		child, err := p.parseNode(depth + 1)
		if err != nil {
			return nil, err
		}

		// Strict mode rejects duplicate keys at the same object depth.
		if p.opts.Strict && containsKey(node.Children, child.Key) {
			return nil, fmt.Errorf("%w: key %q in object %q", ErrDuplicateKeyInStrictMode, child.Key, key)
		}

		node.Add(child)
	}
}

// nextToken consumes one token from parser stream.
func (p *textParser) nextToken() (textToken, error) {
	if p.hasPeeked {
		tok := p.peeked
		p.hasPeeked = false
		return tok, nil
	}

	return p.lexer.nextToken()
}

// peekToken peeks one token without consuming it.
func (p *textParser) peekToken() (textToken, error) {
	if p.hasPeeked {
		return p.peeked, nil
	}

	tok, err := p.lexer.nextToken()
	if err != nil {
		return textToken{}, err
	}

	p.peeked = tok
	p.hasPeeked = true
	return tok, nil
}

// checkDepth validates configured nesting depth limits.
func (p *textParser) checkDepth(depth int) error {
	if p.opts.MaxDepth > 0 && depth > p.opts.MaxDepth {
		return fmt.Errorf("%w: depth %d > %d", ErrDepthLimitExceeded, depth, p.opts.MaxDepth)
	}

	return nil
}

// incrementNodeCount validates configured total node count limits.
func (p *textParser) incrementNodeCount() error {
	p.nodeCount++
	if p.opts.MaxNodes > 0 && p.nodeCount > p.opts.MaxNodes {
		return fmt.Errorf("%w: nodes %d > %d", ErrNodeLimitExceeded, p.nodeCount, p.opts.MaxNodes)
	}

	return nil
}

// containsKey checks whether a node list already contains a key.
func containsKey(nodes []*Node, key string) bool {
	for _, node := range nodes {
		if node != nil && node.Key == key {
			return true
		}
	}

	return false
}
