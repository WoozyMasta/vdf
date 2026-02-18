package vdf

import (
	"reflect"
	"testing"
)

func TestStructureTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		typ     reflect.Type
		field   string
		jsonTag string
		yamlTag string
	}{
		{typ: reflect.TypeOf(Node{}), field: "Key", jsonTag: "key", yamlTag: "key"},
		{typ: reflect.TypeOf(Node{}), field: "Kind", jsonTag: "kind", yamlTag: "kind"},
		{typ: reflect.TypeOf(Node{}), field: "StringValue", jsonTag: "string_value,omitempty", yamlTag: "string_value,omitempty"},
		{typ: reflect.TypeOf(Node{}), field: "Uint32Value", jsonTag: "uint32_value,omitempty", yamlTag: "uint32_value,omitempty"},
		{typ: reflect.TypeOf(Node{}), field: "Children", jsonTag: "children,omitempty", yamlTag: "children,omitempty"},
		{typ: reflect.TypeOf(Document{}), field: "Format", jsonTag: "format,omitempty", yamlTag: "format,omitempty"},
		{typ: reflect.TypeOf(Document{}), field: "Roots", jsonTag: "roots,omitempty", yamlTag: "roots,omitempty"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.typ.Name()+"."+tt.field, func(t *testing.T) {
			field, ok := tt.typ.FieldByName(tt.field)
			if !ok {
				t.Fatalf("field %s not found in %s", tt.field, tt.typ.Name())
			}

			if got := field.Tag.Get("json"); got != tt.jsonTag {
				t.Fatalf("json tag mismatch: got %q, want %q", got, tt.jsonTag)
			}

			if got := field.Tag.Get("yaml"); got != tt.yamlTag {
				t.Fatalf("yaml tag mismatch: got %q, want %q", got, tt.yamlTag)
			}
		})
	}
}
