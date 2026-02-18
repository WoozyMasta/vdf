package vdf

import (
	"errors"
	"testing"
)

func TestNodeConstructorsAndLookup(t *testing.T) {
	t.Parallel()

	obj := NewObjectNode("root")
	obj.Add(NewStringNode("dup", "one"))
	obj.Add(NewStringNode("dup", "two"))
	obj.Add(NewUint32Node("id", 7))

	if obj.Kind != NodeObject {
		t.Fatalf("object kind = %v, want %v", obj.Kind, NodeObject)
	}

	first := obj.First("dup")
	if first == nil || first.StringValue == nil || *first.StringValue != "one" {
		t.Fatalf("First(dup) mismatch: %+v", first)
	}

	all := obj.All("dup")
	if len(all) != 2 {
		t.Fatalf("All(dup) len = %d, want 2", len(all))
	}

	if all[1].StringValue == nil || *all[1].StringValue != "two" {
		t.Fatalf("All(dup)[1] mismatch: %+v", all[1])
	}
}

func TestDocumentValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		doc     *Document
		wantErr error
	}{
		{
			name: "valid document",
			doc: func() *Document {
				d := NewDocumentWithFormat(FormatText)
				root := NewObjectNode("root")
				root.Add(NewStringNode("name", "test"))
				d.AddRoot(root)
				return d
			}(),
		},
		{
			name: "invalid mixed node state",
			doc: func() *Document {
				d := NewDocument()
				v := "x"
				d.AddRoot(&Node{Key: "bad", Kind: NodeObject, StringValue: &v})
				return d
			}(),
			wantErr: ErrInvalidNodeState,
		},
		{
			name: "cyclic node",
			doc: func() *Document {
				d := NewDocument()
				r := NewObjectNode("root")
				r.Add(r)
				d.AddRoot(r)
				return d
			}(),
			wantErr: ErrInvalidNodeState,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := tt.doc.Validate()
			if tt.wantErr == nil && err != nil {
				t.Fatalf("Validate() unexpected error: %v", err)
			}

			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() error = %v, want errors.Is(_, %v)", err, tt.wantErr)
			}
		})
	}
}

func TestToMapStrictAndLossy(t *testing.T) {
	t.Parallel()

	doc := NewDocument()
	root := NewObjectNode("root")
	root.Add(NewStringNode("dup", "first"))
	root.Add(NewStringNode("dup", "second"))
	doc.AddRoot(root)

	if _, err := doc.ToMapStrict(); !errors.Is(err, ErrDuplicateKeyInStrictMode) {
		t.Fatalf("ToMapStrict() error = %v, want duplicate key error", err)
	}

	lossy := doc.ToMapLossy()
	rootVal, ok := lossy["root"].(Map)
	if !ok {
		t.Fatalf("lossy root type = %T, want Map", lossy["root"])
	}

	if got := rootVal["dup"]; got != "second" {
		t.Fatalf("lossy duplicate value = %#v, want %#v", got, "second")
	}
}

func TestFromMap(t *testing.T) {
	t.Parallel()

	doc, err := FromMap("root", Map{
		"name": "value",
		"id":   5,
		"sub": Map{
			"flag": uint32(1),
		},
	})
	if err != nil {
		t.Fatalf("FromMap() returned error: %v", err)
	}

	if err := doc.Validate(); err != nil {
		t.Fatalf("Validate() returned error: %v", err)
	}

	root := doc.Roots[0]
	if root.Kind != NodeObject {
		t.Fatalf("root kind = %v, want %v", root.Kind, NodeObject)
	}

	if _, err := FromMap("root", Map{"bad": true}); !errors.Is(err, ErrUnsupportedMapValueType) {
		t.Fatalf("FromMap(unsupported) error = %v, want ErrUnsupportedMapValueType", err)
	}
}
