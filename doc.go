// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Maxim Levchenko (WoozyMasta)
// Source: github.com/woozymasta/vdf

/*
Package vdf implements a parser and encoder for Valve Data Format (VDF)
in text and binary variants.

# Data model

The canonical model is an AST:

  - Document is a full file with ordered root nodes.
  - NodeObject keeps ordered children and allows duplicate keys.
  - NodeString and NodeUint32 are scalar leaves.

This preserves VDF semantics that are commonly lost in map-based APIs
(ordering and duplicate keys).

# Decode API

Use Decoder for stream-oriented decoding from io.Reader:

	dec := vdf.NewDecoder(r, vdf.DecodeOptions{Format: vdf.FormatAuto})
	doc, err := dec.DecodeDocument()

For byte slices and strings use ParseBytes and ParseString.
For file paths use ParseFile with optional DecodeOptions,
or ParseTextFile/ParseAutoFile.

WalkEvents decodes the full document into an AST on the first call,
then returns DFS traversal events on subsequent calls.
Use it when you need the document available after iteration:

	event, err := dec.WalkEvents()

NextEvent is a true streaming decoder:
it reads and yields events one at a time without building an AST,
making it suitable for large inputs.
WalkEvents and NextEvent are mutually exclusive on a single Decoder instance.

# Security limits

DecodeOptions.MaxDepth and MaxNodes bound recursion and node count.
MaxKeyBytes, MaxValueBytes, and MaxStringBytes bound string lengths
for keys and values respectively. All limits use 0 to mean unlimited.

# Encode API

Use Encoder for stream-oriented output to io.Writer:

	enc := vdf.NewEncoder(w, vdf.EncodeOptions{Format: vdf.FormatText})
	err := enc.EncodeDocument(doc)

Manual streaming methods are available for incremental writing:
StartObject, WriteString, WriteUint32, EndObject, Close.
For file output use WriteFile with optional EncodeOptions,
or WriteTextFile/WriteBinaryFile.

# Fast paths

AppendText and AppendBinary append encoded output directly
into destination byte slices to reduce allocations on hot paths.

# Builder

NewBuilder provides a fluent, ordered API
for constructing Documents without manual AST manipulation:

	doc, err := vdf.NewBuilder("root").
		Set("key", "value").
		Object("child", func(b *vdf.Builder) { b.SetUint32("n", 1) }).
		Document()

# Reflection API

Marshal and Unmarshal convert between Go structs
and VDF Documents using struct field tags of the form vdf:"name,option":

	type S struct {
		Name string `vdf:"name"`
		Port uint32 `vdf:"port"`
	}
	doc, _ := vdf.Marshal("Server", S{Name: "x", Port: 2302})
	var s S
	_ = vdf.Unmarshal(doc, "Server", &s)

Supported options: omitempty, inline, repeated, indexed.
Fields implementing encoding.TextMarshaler/TextUnmarshaler are handled automatically.

# Validation

Document.Validate can be called explicitly when strict AST checks are required.
For performance, encoding does not force full validation unless
EncodeOptions.Validate is set to true.
*/
package vdf
