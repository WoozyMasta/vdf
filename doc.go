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

NextEvent provides traversal events over the decoded document:

	event, err := dec.NextEvent()

# Encode API

Use Encoder for stream-oriented output to io.Writer:

	enc := vdf.NewEncoder(w, vdf.EncodeOptions{Format: vdf.FormatText})
	err := enc.EncodeDocument(doc)

Manual streaming methods are available for incremental writing:
StartObject, WriteString, WriteUint32, EndObject, Close.

# Fast paths

AppendText and AppendBinary append encoded output directly into destination
byte slices to reduce allocations on hot paths.

# Validation

Document.Validate can be called explicitly when strict AST checks are required.
For performance, encoding does not force full validation unless
EncodeOptions.Validate is set to true.
*/
package vdf
