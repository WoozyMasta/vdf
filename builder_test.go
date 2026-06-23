// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Maxim Levchenko (WoozyMasta)
// Source: github.com/woozymasta/vdf

package vdf

import (
	"errors"
	"testing"
)

func TestBuilderSimple(t *testing.T) {
	t.Parallel()

	doc, err := NewBuilder("root").
		Set("name", "server").
		SetUint32("port", 2302).
		Document()
	if err != nil {
		t.Fatalf("Document() error: %v", err)
	}

	if len(doc.Roots) != 1 || doc.Roots[0].Key != "root" {
		t.Fatalf("unexpected roots: %v", doc.Roots)
	}

	root := doc.Roots[0]
	if got := root.First("name"); got == nil || *got.StringValue != "server" {
		t.Fatalf("name = %v", got)
	}

	if got := root.First("port"); got == nil || *got.Uint32Value != 2302 {
		t.Fatalf("port = %v", got)
	}
}

func TestBuilderNested(t *testing.T) {
	t.Parallel()

	doc, err := NewBuilder("config").
		Set("host", "localhost").
		Object("db", func(b *Builder) {
			b.Set("name", "mydb").SetUint32("port", 5432)
		}).
		Document()
	if err != nil {
		t.Fatalf("Document() error: %v", err)
	}

	root := doc.Roots[0]
	db := root.First("db")
	if db == nil || db.Kind != NodeObject {
		t.Fatal("db node missing")
	}

	if got := db.First("name"); got == nil || *got.StringValue != "mydb" {
		t.Fatalf("db.name = %v", got)
	}

	if got := db.First("port"); got == nil || *got.Uint32Value != 5432 {
		t.Fatalf("db.port = %v", got)
	}
}

func TestBuilderDeepNesting(t *testing.T) {
	t.Parallel()

	doc, err := NewBuilder("a").
		Object("b", func(b *Builder) {
			b.Object("c", func(b *Builder) {
				b.Set("leaf", "value")
			})
		}).
		Document()
	if err != nil {
		t.Fatalf("Document() error: %v", err)
	}

	leaf := doc.Roots[0].First("b").First("c").First("leaf")
	if leaf == nil || *leaf.StringValue != "value" {
		t.Fatalf("leaf = %v", leaf)
	}
}

func TestBuilderDocumentIsValid(t *testing.T) {
	t.Parallel()

	doc, err := NewBuilder("root").Set("k", "v").Document()
	if err != nil {
		t.Fatalf("Document() error: %v", err)
	}

	if err := doc.Validate(); err != nil {
		t.Fatalf("Validate() error: %v", err)
	}
}

func TestBuilderDoubleDocumentError(t *testing.T) {
	t.Parallel()

	b := NewBuilder("root")
	_, err := b.Document()
	if err != nil {
		t.Fatalf("first Document() error: %v", err)
	}

	// Stack is empty after finalization; second call must error.
	_, err = b.Document()
	if err == nil {
		t.Fatal("expected error on second Document() call")
	}
}

func TestFromMapSorted(t *testing.T) {
	t.Parallel()

	m := Map{
		"zebra": "last",
		"apple": "first",
		"mango": "middle",
	}

	doc, err := FromMapSorted("root", m)
	if err != nil {
		t.Fatalf("FromMapSorted() error: %v", err)
	}

	children := doc.Roots[0].Children
	if len(children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(children))
	}

	want := []string{"apple", "mango", "zebra"}
	for i, w := range want {
		if children[i].Key != w {
			t.Fatalf("child[%d].Key = %q, want %q", i, children[i].Key, w)
		}
	}
}

func TestFromMapSortedVsFromMap(t *testing.T) {
	t.Parallel()

	m := Map{"b": "2", "a": "1", "c": "3"}

	sorted, err := FromMapSorted("root", m)
	if err != nil {
		t.Fatalf("FromMapSorted() error: %v", err)
	}

	keys := make([]string, len(sorted.Roots[0].Children))
	for i, ch := range sorted.Roots[0].Children {
		keys[i] = ch.Key
	}

	if keys[0] != "a" || keys[1] != "b" || keys[2] != "c" {
		t.Fatalf("sorted keys = %v, want [a b c]", keys)
	}
}

func TestBuilderPropagatesError(t *testing.T) {
	t.Parallel()

	// Simulate an error mid-chain by verifying the builder stops processing
	// after Document() is called with an empty stack (double-call scenario).
	b := NewBuilder("root")
	b.Set("k", "v")
	_, _ = b.Document()

	// After finalization the stack is empty; Set should be a no-op.
	b.Set("after", "finalization")

	// No panic should occur.
	_, err := b.Document()
	if !errors.Is(err, ErrInvalidNodeState) {
		t.Fatalf("expected ErrInvalidNodeState, got %v", err)
	}
}
