// SPDX-License-Identifier: MIT
// Copyright (c) 2026 Maxim Levchenko (WoozyMasta)
// Source: github.com/woozymasta/vdf

/*
Package vdf reads and writes Valve Data Format (VDF) text and supported binary forms.
Its [Document] and [Node] tree preserves source order and duplicate keys.
The package requires Go 1.22.0 or newer and supports binary object, string, and uint32 nodes.

See [Valve VDF reference]
for format background and the official [Steamworks IGA docs]
for a VDF-based KeyValues document and examples.

Use [ParseString] or [ParseBytes] for in-memory input and [NewDecoder] for an [io.Reader].
Use [WriteString], [AppendText], [AppendBinary], or [NewEncoder] for output.
[NewBuilder] and [Marshal] provide higher-level document creation;
[Unmarshal] converts a document into a Go struct.

Struct mapping in [Marshal] and [Unmarshal] uses `vdf` tags:
  - `name` selects the VDF key; `-` skips the field.
  - `omitempty` omits Go zero values.
  - `omitzero` omits when `IsZero() bool` returns true, or otherwise for Go zero values.
  - `inline` places a struct's fields in its parent object.
  - `repeated` writes slice elements as repeated keys.
  - `indexed` writes slices under contiguous numeric keys (`"0"`, `"1"`, ...).

Decoding with [FormatAuto] checks the first byte for a supported binary type;
empty input and other first bytes are treated as text.
Encoding with [FormatAuto] follows [Document.Format]
when it is text or binary and otherwise uses text.
Binary documents end with the `0x08` terminator.

[DecodeOptions.MaxDepth] limits nested object scopes:
a root object has depth 1, and scalar leaves do not increase object depth.
[DecodeOptions.Strict] rejects duplicate keys.
Reflection's `indexed` slice option uses canonical, contiguous numeric keys.
[Decoder.WalkEvents] builds the document tree before yielding events,
while [Decoder.NextEvent] streams events directly;
those modes are exclusive on one decoder.

A single [Encoder] writes one logical document.
Manual binary output is finalized by [Encoder.Close].

[Valve VDF reference]: https://developer.valvesoftware.com/wiki/VDF
[Steamworks IGA docs]: https://partner.steamgames.com/doc/features/steam_controller/iga_file
*/
package vdf
