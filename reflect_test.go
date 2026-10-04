// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Maxim Levchenko (WoozyMasta)
// Source: github.com/woozymasta/vdf

package vdf

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"
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

type indexedItem struct {
	Name string `vdf:"name"`
}

type indexedNestedStruct struct {
	Items []indexedItem `vdf:"items,indexed"`
}

// pointerTextMarshaler implements TextMarshaler only through its pointer type.
type pointerTextMarshaler struct {
	Value string // Value serialized by MarshalText.
}

// MarshalText returns the wrapped value with a pointer-specific prefix.
func (v *pointerTextMarshaler) MarshalText() ([]byte, error) {
	return []byte("pointer:" + v.Value), nil
}

// pointerTextMarshalerContainer exercises addressability across field shapes.
type pointerTextMarshalerContainer struct {
	Field    pointerTextMarshaler            `vdf:"field"`             // Value field.
	Map      map[string]pointerTextMarshaler `vdf:"values"`            // Map values.
	Repeated []pointerTextMarshaler          `vdf:"repeated,repeated"` // Repeated slice elements.
	Indexed  []pointerTextMarshaler          `vdf:"indexed,indexed"`   // Indexed slice elements.
	Omitted  *pointerTextMarshaler           `vdf:"omitted"`           // Nil pointer omission.
}

// mutatingPointerTextMarshaler checks that fallback addressability uses a copy.
type mutatingPointerTextMarshaler struct {
	Value string // Value serialized before the receiver mutates it.
}

// MarshalText mutates the receiver to expose accidental writes to map values.
func (v *mutatingPointerTextMarshaler) MarshalText() ([]byte, error) {
	text := []byte(v.Value)
	v.Value = "mutated"
	return text, nil
}

// mutatingMapContainer supplies an unaddressable map value to Marshal.
type mutatingMapContainer struct {
	Values map[string]mutatingPointerTextMarshaler `vdf:"values"` // Input map.
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

func TestMarshalPointerTextMarshalerAddressability(t *testing.T) {
	t.Parallel()

	value := pointerTextMarshalerContainer{
		Field:    pointerTextMarshaler{Value: "field"},
		Map:      map[string]pointerTextMarshaler{"entry": {Value: "map"}},
		Repeated: []pointerTextMarshaler{{Value: "first"}, {Value: "second"}},
		Indexed:  []pointerTextMarshaler{{Value: "zero"}, {Value: "one"}},
	}
	byValue, err := Marshal("root", value)
	if err != nil {
		t.Fatalf("Marshal(value): %v", err)
	}
	byPointer, err := Marshal("root", &value)
	if err != nil {
		t.Fatalf("Marshal(pointer): %v", err)
	}
	if !reflect.DeepEqual(byValue, byPointer) {
		t.Fatalf("Marshal(value) = %#v, Marshal(pointer) = %#v", byValue, byPointer)
	}

	root := byValue.Roots[0]
	assertString := func(key, want string) {
		t.Helper()
		node := root.First(key)
		if node == nil || node.StringValue == nil || *node.StringValue != want {
			t.Fatalf("%s = %#v, want %q", key, node, want)
		}
	}
	assertString("field", "pointer:field")
	if node := root.First("values"); node == nil ||
		node.First("entry") == nil ||
		node.First("entry").StringValue == nil ||
		*node.First("entry").StringValue != "pointer:map" {
		t.Fatalf("map value node = %#v, want pointer:map", node)
	}

	repeated := root.All("repeated")
	if len(repeated) != 2 ||
		repeated[0].StringValue == nil ||
		*repeated[0].StringValue != "pointer:first" ||
		repeated[1].StringValue == nil ||
		*repeated[1].StringValue != "pointer:second" {
		t.Fatalf("repeated values = %#v, want [pointer:first pointer:second]", repeated)
	}

	indexed := root.First("indexed")
	if indexed == nil ||
		indexed.First("0") == nil ||
		indexed.First("0").StringValue == nil ||
		*indexed.First("0").StringValue != "pointer:zero" ||
		indexed.First("1") == nil || indexed.First("1").StringValue == nil ||
		*indexed.First("1").StringValue != "pointer:one" {
		t.Fatalf("indexed values = %#v, want pointer:zero and pointer:one", indexed)
	}
	if root.First("omitted") != nil {
		t.Fatal("nil pointer TextMarshaler field should be omitted")
	}
}

func TestMarshalPointerTextMarshalerDoesNotMutateMapValue(t *testing.T) {
	t.Parallel()

	value := mutatingMapContainer{
		Values: map[string]mutatingPointerTextMarshaler{"entry": {Value: "original"}},
	}
	doc, err := Marshal("root", value)
	if err != nil {
		t.Fatalf("Marshal(): %v", err)
	}
	if got := value.Values["entry"].Value; got != "original" {
		t.Fatalf("map value after Marshal = %q, want unchanged value", got)
	}
	values := doc.Roots[0].First("values")
	if values == nil {
		t.Fatal("values map node missing")
	}
	node := values.First("entry")
	if node == nil || node.StringValue == nil || *node.StringValue != "original" {
		t.Fatalf("map value node = %#v, want original", node)
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

func TestUnmarshalIndexedOutOfOrder(t *testing.T) {
	t.Parallel()

	doc, err := ParseString(`"root" { "items" { "1" "second" "0" "first" } }`)
	if err != nil {
		t.Fatalf("ParseString(): %v", err)
	}

	var value indexedStruct
	if err := Unmarshal(doc, "root", &value); err != nil {
		t.Fatalf("Unmarshal(): %v", err)
	}
	if !reflect.DeepEqual(value.Items, []string{"first", "second"}) {
		t.Fatalf("Items = %v, want [first second]", value.Items)
	}
}

func TestUnmarshalIndexedRejectsMalformedKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		items string
	}{
		{name: "gap", items: `"0" "first" "2" "third"`},
		{name: "duplicate index", items: `"0" "first" "0" "again"`},
		{name: "non-numeric", items: `"zero" "first"`},
		{name: "negative", items: `"-1" "first"`},
		{name: "leading zero", items: `"00" "first"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			doc, err := ParseString(`"root" { "items" { ` + test.items + ` } }`)
			if err != nil {
				t.Fatalf("ParseString(): %v", err)
			}
			var value indexedStruct
			if err := Unmarshal(doc, "root", &value); !errors.Is(err, ErrReflectFieldMismatch) {
				t.Fatalf("Unmarshal(): error = %v, want ErrReflectFieldMismatch", err)
			}
		})
	}
}

func TestUnmarshalIndexedStructElementsAndRoundtrip(t *testing.T) {
	t.Parallel()

	input := `"root" { "items" { "1" { "name" "second" } "0" { "name" "first" } } }`
	doc, err := ParseString(input)
	if err != nil {
		t.Fatalf("ParseString(): %v", err)
	}
	var nested indexedNestedStruct
	if err := Unmarshal(doc, "root", &nested); err != nil {
		t.Fatalf("Unmarshal(nested): %v", err)
	}
	wantNested := []indexedItem{{Name: "first"}, {Name: "second"}}
	if !reflect.DeepEqual(nested.Items, wantNested) {
		t.Fatalf("nested Items = %#v, want %#v", nested.Items, wantNested)
	}

	want := indexedStruct{Items: []string{"alpha", "beta", "gamma"}}
	encoded, err := Marshal("root", want)
	if err != nil {
		t.Fatalf("Marshal(): %v", err)
	}
	var got indexedStruct
	if err := Unmarshal(encoded, "root", &got); err != nil {
		t.Fatalf("Unmarshal(roundtrip): %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("roundtrip = %#v, want %#v", got, want)
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

func TestUnmarshalIntegerBoundaries(t *testing.T) {
	t.Parallel()

	types := []reflect.Type{
		reflect.TypeFor[uint8](),
		reflect.TypeFor[uint16](),
		reflect.TypeFor[uint32](),
		reflect.TypeFor[uint](),
		reflect.TypeFor[uint64](),
		reflect.TypeFor[int8](),
		reflect.TypeFor[int16](),
		reflect.TypeFor[int32](),
		reflect.TypeFor[int](),
		reflect.TypeFor[int64](),
	}

	for _, fieldType := range types {
		t.Run(fieldType.String(), func(t *testing.T) {
			t.Parallel()

			min, max := integerBounds(fieldType)
			value, err := unmarshalReflectValue(t, fieldType, NewStringNode("value", max.String()))
			if err != nil {
				t.Fatalf("Unmarshal(max string): %v", err)
			}
			if fieldType.Kind() >= reflect.Int && fieldType.Kind() <= reflect.Int64 {
				if value.Int() != max.Int64() {
					t.Fatalf("max string = %d, want %s", value.Int(), max)
				}
			} else if value.Uint() != max.Uint64() {
				t.Fatalf("max string = %d, want %s", value.Uint(), max)
			}
			if fieldType.Kind() >= reflect.Uint && fieldType.Kind() <= reflect.Uint64 {
				value, err = unmarshalReflectValue(t, fieldType, NewStringNode("value", "0"))
				if err != nil || value.Uint() != 0 {
					t.Fatalf("Unmarshal(unsigned zero) = (%v, %v), want (0, nil)", value, err)
				}
			}

			overflow := new(big.Int).Add(max, big.NewInt(1))
			if _, err := unmarshalReflectValue(
				t, fieldType,
				NewStringNode("value", overflow.String()),
			); !errors.Is(err, ErrReflectFieldMismatch) {
				t.Errorf(
					"Unmarshal(overflow string %s): error = %v, want ErrReflectFieldMismatch",
					overflow, err)
			}

			if fieldType.Kind() >= reflect.Int && fieldType.Kind() <= reflect.Int64 {
				value, err = unmarshalReflectValue(t, fieldType, NewStringNode("value", min.String()))
				if err != nil {
					t.Fatalf("Unmarshal(min string): %v", err)
				}
				if value.Int() != min.Int64() {
					t.Fatalf("min string = %d, want %s", value.Int(), min)
				}

				underflow := new(big.Int).Sub(min, big.NewInt(1))
				if _, err := unmarshalReflectValue(
					t, fieldType,
					NewStringNode("value", underflow.String()),
				); !errors.Is(err, ErrReflectFieldMismatch) {
					t.Errorf(
						"Unmarshal(underflow string %s): error = %v, want ErrReflectFieldMismatch",
						underflow, err)
				}
			}

			maxUint32 := new(big.Int).SetUint64(uint64(^uint32(0)))
			nodeValue := max
			if nodeValue.Cmp(maxUint32) > 0 {
				nodeValue = maxUint32
			}
			nodeUint := uint32(nodeValue.Uint64())
			value, err = unmarshalReflectValue(t, fieldType, NewUint32Node("value", nodeUint))
			if err != nil {
				t.Fatalf("Unmarshal(NodeUint32 %d): %v", nodeUint, err)
			}
			if fieldType.Kind() >= reflect.Int && fieldType.Kind() <= reflect.Int64 {
				if value.Int() != int64(nodeUint) {
					t.Fatalf("NodeUint32 value = %d, want %d", value.Int(), nodeUint)
				}
			} else if value.Uint() != uint64(nodeUint) {
				t.Fatalf("NodeUint32 value = %d, want %d", value.Uint(), nodeUint)
			}

			if max.Cmp(maxUint32) < 0 {
				nodeOverflow := uint32(overflow.Uint64())
				if _, err := unmarshalReflectValue(
					t, fieldType,
					NewUint32Node("value", nodeOverflow),
				); !errors.Is(err, ErrReflectFieldMismatch) {
					t.Errorf(
						"Unmarshal(overflow NodeUint32 %d): error = %v, want ErrReflectFieldMismatch",
						nodeOverflow, err)
				}
			}
		})
	}
}

func TestUnmarshalRejectsInvalidDocumentAndScalarNodes(t *testing.T) {
	t.Parallel()

	var out struct {
		Text  string `vdf:"text"`
		Count uint32 `vdf:"count"`
	}
	if err := Unmarshal(nil, "root", &out); !errors.Is(err, ErrInvalidNodeState) {
		t.Fatalf("Unmarshal(nil): error = %v, want ErrInvalidNodeState", err)
	}

	tests := []struct {
		name string
		node *Node
	}{
		{name: "nil string payload", node: &Node{Key: "text", Kind: NodeString}},
		{name: "nil uint32 payload", node: &Node{Key: "count", Kind: NodeUint32}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			doc := NewDocument()
			root := NewObjectNode("root")
			root.Add(test.node)
			doc.AddRoot(root)
			var value struct {
				Text  string `vdf:"text"`
				Count uint32 `vdf:"count"`
			}
			if err := Unmarshal(doc, "root", &value); !errors.Is(err, ErrInvalidNodeState) {
				t.Fatalf("Unmarshal(): error = %v, want ErrInvalidNodeState", err)
			}
		})
	}
}

func TestUnmarshalBoolValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value string
		want  bool
		valid bool
	}{
		{value: "1", want: true, valid: true},
		{value: "TRUE", want: true, valid: true},
		{value: "0", valid: true},
		{value: "False", valid: true},
		{value: "yes"},
		{value: "2"},
	}
	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			doc, err := ParseString(`"root" { "value" "` + test.value + `" }`)
			if err != nil {
				t.Fatalf("ParseString(): %v", err)
			}
			var out struct {
				Value bool `vdf:"value"`
			}
			err = Unmarshal(doc, "root", &out)
			if !test.valid {
				if !errors.Is(err, ErrReflectFieldMismatch) {
					t.Fatalf("Unmarshal(): error = %v, want ErrReflectFieldMismatch", err)
				}
				return
			}
			if err != nil || out.Value != test.want {
				t.Fatalf("Unmarshal() = (%v, %v), want (%v, nil)", out.Value, err, test.want)
			}
		})
	}
}

// integerBounds returns the exact signed or unsigned range for an integer type.
func integerBounds(fieldType reflect.Type) (*big.Int, *big.Int) {
	bits := uint(fieldType.Bits())
	one := big.NewInt(1)
	max := new(big.Int).Lsh(new(big.Int).Set(one), bits)
	max.Sub(max, one)
	min := big.NewInt(0)

	if fieldType.Kind() >= reflect.Int && fieldType.Kind() <= reflect.Int64 {
		half := new(big.Int).Lsh(new(big.Int).Set(one), bits-1)
		max.Sub(half, one)
		min.Neg(half)
	}

	return min, max
}

// unmarshalReflectValue decodes one node into a dynamic struct field type.
func unmarshalReflectValue(t *testing.T, fieldType reflect.Type, node *Node) (reflect.Value, error) {
	t.Helper()
	doc := NewDocument()
	root := NewObjectNode("root")
	root.Add(node)
	doc.AddRoot(root)

	structType := reflect.StructOf([]reflect.StructField{{
		Name: "Value",
		Type: fieldType,
		Tag:  `vdf:"value"`,
	}})
	out := reflect.New(structType)
	err := Unmarshal(doc, "root", out.Interface())
	return out.Elem().Field(0), err
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
