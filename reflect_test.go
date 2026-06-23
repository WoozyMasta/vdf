// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Maxim Levchenko (WoozyMasta)
// Source: github.com/woozymasta/vdf

package vdf

import (
	"errors"
	"fmt"
	"testing"
)

type simpleStruct struct {
	Name    string `vdf:"name"`
	Version uint32 `vdf:"version"`
	Active  bool   `vdf:"active"`
}

type nestedStruct struct {
	Title string       `vdf:"title"`
	Sub   simpleStruct `vdf:"sub"`
}

type omitStruct struct {
	Name  string `vdf:"name,omitempty"`
	Score uint32 `vdf:"score,omitempty"`
}

type inlineParent struct {
	Extra string       `vdf:"extra"`
	Base  simpleStruct `vdf:",inline"`
}

type repeatedStruct struct {
	Tags []string `vdf:"tag,repeated"`
}

type indexedStruct struct {
	Items []string `vdf:"items,indexed"`
}

type skipStruct struct {
	Keep   string `vdf:"keep"`
	Hidden string `vdf:"-"`
}

// textMarshalerField implements encoding.TextMarshaler / TextUnmarshaler.
type textMarshalerField struct{ V string }

func (t textMarshalerField) MarshalText() ([]byte, error) { return []byte("TM:" + t.V), nil }
func (t *textMarshalerField) UnmarshalText(b []byte) error {
	if len(b) < 3 || string(b[:3]) != "TM:" {
		return fmt.Errorf("bad prefix")
	}
	t.V = string(b[3:])
	return nil
}

type tmStruct struct {
	Field textMarshalerField `vdf:"field"`
}

func TestMarshalSimpleStruct(t *testing.T) {
	t.Parallel()

	v := simpleStruct{Name: "server", Version: 2, Active: true}
	doc, err := Marshal("cfg", v)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	if len(doc.Roots) != 1 || doc.Roots[0].Key != "cfg" {
		t.Fatalf("unexpected root: %v", doc.Roots)
	}

	root := doc.Roots[0]
	if got := root.First("name"); got == nil || *got.StringValue != "server" {
		t.Fatalf("name = %v, want %q", got, "server")
	}

	if got := root.First("version"); got == nil || *got.Uint32Value != 2 {
		t.Fatalf("version = %v, want 2", got)
	}

	if got := root.First("active"); got == nil || *got.StringValue != "1" {
		t.Fatalf("active = %v, want %q", got, "1")
	}
}

func TestMarshalNested(t *testing.T) {
	t.Parallel()

	v := nestedStruct{Title: "top", Sub: simpleStruct{Name: "inner"}}
	doc, err := Marshal("root", v)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	sub := doc.Roots[0].First("sub")
	if sub == nil || sub.Kind != NodeObject {
		t.Fatalf("sub node missing or wrong kind")
	}

	if got := sub.First("name"); got == nil || *got.StringValue != "inner" {
		t.Fatalf("sub.name = %v, want %q", got, "inner")
	}
}

func TestMarshalOmitempty(t *testing.T) {
	t.Parallel()

	doc, err := Marshal("root", omitStruct{})
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	root := doc.Roots[0]
	if len(root.Children) != 0 {
		t.Fatalf("expected no children with omitempty zeros, got %d", len(root.Children))
	}
}

func TestMarshalInline(t *testing.T) {
	t.Parallel()

	v := inlineParent{
		Extra: "bonus",
		Base:  simpleStruct{Name: "base", Version: 1},
	}
	doc, err := Marshal("root", v)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	root := doc.Roots[0]
	if root.First("extra") == nil {
		t.Fatal("extra field missing")
	}

	if root.First("name") == nil {
		t.Fatal("inlined name field missing at root level")
	}

	if root.First("sub") != nil {
		t.Fatal("inlined struct should not appear as sub-node")
	}
}

func TestMarshalRepeated(t *testing.T) {
	t.Parallel()

	v := repeatedStruct{Tags: []string{"alpha", "beta", "gamma"}}
	doc, err := Marshal("root", v)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	tags := doc.Roots[0].All("tag")
	if len(tags) != 3 {
		t.Fatalf("expected 3 tag nodes, got %d", len(tags))
	}

	for i, want := range []string{"alpha", "beta", "gamma"} {
		if *tags[i].StringValue != want {
			t.Fatalf("tag[%d] = %q, want %q", i, *tags[i].StringValue, want)
		}
	}
}

func TestMarshalIndexed(t *testing.T) {
	t.Parallel()

	v := indexedStruct{Items: []string{"x", "y"}}
	doc, err := Marshal("root", v)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	items := doc.Roots[0].First("items")
	if items == nil || items.Kind != NodeObject {
		t.Fatal("items node missing or wrong kind")
	}

	if got := items.First("0"); got == nil || *got.StringValue != "x" {
		t.Fatalf("items[0] = %v, want %q", got, "x")
	}

	if got := items.First("1"); got == nil || *got.StringValue != "y" {
		t.Fatalf("items[1] = %v, want %q", got, "y")
	}
}

func TestMarshalSkipTag(t *testing.T) {
	t.Parallel()

	v := skipStruct{Keep: "yes", Hidden: "secret"}
	doc, err := Marshal("root", v)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	root := doc.Roots[0]
	if root.First("keep") == nil {
		t.Fatal("keep field missing")
	}

	if root.First("hidden") != nil || root.First("-") != nil {
		t.Fatal("hidden/skipped field should not appear in output")
	}
}

func TestMarshalTextMarshaler(t *testing.T) {
	t.Parallel()

	v := tmStruct{Field: textMarshalerField{V: "hello"}}
	doc, err := Marshal("root", v)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	node := doc.Roots[0].First("field")
	if node == nil || *node.StringValue != "TM:hello" {
		t.Fatalf("field = %v, want %q", node, "TM:hello")
	}
}

func TestMarshalUnsupportedType(t *testing.T) {
	t.Parallel()

	type bad struct {
		Ch chan int `vdf:"ch"`
	}

	_, err := Marshal("root", bad{Ch: make(chan int)})
	if !errors.Is(err, ErrReflectUnsupportedType) {
		t.Fatalf("expected ErrReflectUnsupportedType, got %v", err)
	}
}

func TestUnmarshalSimpleStruct(t *testing.T) {
	t.Parallel()

	doc, _ := ParseString(`"cfg" { "name" "server" "version" "3" "active" "1" }`)

	var v simpleStruct
	if err := Unmarshal(doc, "cfg", &v); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}

	if v.Name != "server" {
		t.Fatalf("Name = %q, want %q", v.Name, "server")
	}

	if v.Version != 3 {
		t.Fatalf("Version = %d, want 3", v.Version)
	}

	if !v.Active {
		t.Fatalf("Active = false, want true")
	}
}

func TestUnmarshalNested(t *testing.T) {
	t.Parallel()

	doc, _ := ParseString(`"root" { "title" "top" "sub" { "name" "inner" "version" "5" "active" "0" } }`)

	var v nestedStruct
	if err := Unmarshal(doc, "root", &v); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}

	if v.Title != "top" {
		t.Fatalf("Title = %q, want %q", v.Title, "top")
	}

	if v.Sub.Name != "inner" || v.Sub.Version != 5 {
		t.Fatalf("Sub = %+v", v.Sub)
	}
}

func TestUnmarshalRoundtrip(t *testing.T) {
	t.Parallel()

	orig := simpleStruct{Name: "roundtrip", Version: 42, Active: true}
	doc, err := Marshal("item", orig)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	var out simpleStruct
	if err := Unmarshal(doc, "item", &out); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}

	if out != orig {
		t.Fatalf("roundtrip: got %+v, want %+v", out, orig)
	}
}

func TestUnmarshalInline(t *testing.T) {
	t.Parallel()

	doc, _ := ParseString(`"root" { "extra" "bonus" "name" "base" "version" "1" "active" "0" }`)

	var v inlineParent
	if err := Unmarshal(doc, "root", &v); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}

	if v.Extra != "bonus" {
		t.Fatalf("Extra = %q, want %q", v.Extra, "bonus")
	}

	if v.Base.Name != "base" {
		t.Fatalf("Base.Name = %q, want %q", v.Base.Name, "base")
	}
}

func TestUnmarshalRepeated(t *testing.T) {
	t.Parallel()

	doc, _ := ParseString(`"root" { "tag" "alpha" "tag" "beta" "tag" "gamma" }`)

	var v repeatedStruct
	if err := Unmarshal(doc, "root", &v); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}

	if len(v.Tags) != 3 || v.Tags[0] != "alpha" || v.Tags[1] != "beta" || v.Tags[2] != "gamma" {
		t.Fatalf("Tags = %v", v.Tags)
	}
}

func TestUnmarshalIndexed(t *testing.T) {
	t.Parallel()

	doc, _ := ParseString(`"root" { "items" { "0" "x" "1" "y" } }`)

	var v indexedStruct
	if err := Unmarshal(doc, "root", &v); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}

	if len(v.Items) != 2 || v.Items[0] != "x" || v.Items[1] != "y" {
		t.Fatalf("Items = %v", v.Items)
	}
}

func TestUnmarshalTextUnmarshaler(t *testing.T) {
	t.Parallel()

	doc, _ := ParseString(`"root" { "field" "TM:world" }`)

	var v tmStruct
	if err := Unmarshal(doc, "root", &v); err != nil {
		t.Fatalf("Unmarshal() error: %v", err)
	}

	if v.Field.V != "world" {
		t.Fatalf("Field.V = %q, want %q", v.Field.V, "world")
	}
}

func TestUnmarshalMissingRoot(t *testing.T) {
	t.Parallel()

	doc, _ := ParseString(`"other" { "k" "v" }`)

	var v simpleStruct
	err := Unmarshal(doc, "missing", &v)
	if !errors.Is(err, ErrReflectFieldMismatch) {
		t.Fatalf("expected ErrReflectFieldMismatch, got %v", err)
	}
}

func TestUnmarshalRequiresPointer(t *testing.T) {
	t.Parallel()

	doc, _ := ParseString(`"r" { }`)

	var v simpleStruct
	err := Unmarshal(doc, "r", v) // not a pointer
	if !errors.Is(err, ErrReflectUnsupportedType) {
		t.Fatalf("expected ErrReflectUnsupportedType, got %v", err)
	}
}

// customDate implements IsZero() bool to demonstrate omitzero with a custom type.
type customDate struct {
	Year  int
	Month int
	Day   int
}

func (d customDate) IsZero() bool { return d.Year == 0 && d.Month == 0 && d.Day == 0 }

type omitZeroStruct struct {
	Name    string     `vdf:"name,omitzero"`
	Created customDate `vdf:"created,omitzero"`
	Score   uint32     `vdf:"score,omitzero"`
}

func TestMarshalOmitZeroCustomMethod(t *testing.T) {
	t.Parallel()

	v := omitZeroStruct{Name: "test", Created: customDate{}, Score: 0}
	doc, err := Marshal("root", v)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	root := doc.Roots[0]
	if root.First("name") == nil {
		t.Fatal("name should be present")
	}
	if root.First("created") != nil {
		t.Fatal("created should be omitted: IsZero() == true")
	}
	if root.First("score") != nil {
		t.Fatal("score should be omitted: zero uint32")
	}
}

func TestMarshalOmitZeroNonZeroCustomMethod(t *testing.T) {
	t.Parallel()

	v := omitZeroStruct{
		Name:    "test",
		Created: customDate{Year: 2026, Month: 6, Day: 24},
		Score:   42,
	}
	doc, err := Marshal("root", v)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	root := doc.Roots[0]
	if root.First("created") == nil {
		t.Fatal("created should be present: IsZero() == false")
	}
	if root.First("score") == nil {
		t.Fatal("score should be present")
	}
}

func TestMarshalOmitEmptyVsOmitZero(t *testing.T) {
	t.Parallel()

	type comparison struct {
		A customDate `vdf:"a,omitempty"` // reflect.Value.IsZero() only
		B customDate `vdf:"b,omitzero"`  // calls customDate.IsZero() first
	}

	doc, err := Marshal("root", comparison{A: customDate{}, B: customDate{}})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	root := doc.Roots[0]
	if root.First("a") != nil {
		t.Fatal("a (omitempty) should be omitted")
	}
	if root.First("b") != nil {
		t.Fatal("b (omitzero) should be omitted")
	}
}
