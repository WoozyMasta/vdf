# vdf

[vdf] is a Go library for working with Valve Data Format (VDF),
the configuration and data format used by Valve games and tools.
It reads, edits, and writes VDF while keeping the original key order
and duplicate keys in an ordered document tree.
This makes it useful for game tools and config converters
that need to preserve details ordinary maps cannot represent.

See the [Valve VDF reference][vdf-ref] for format background
and the official [Steamworks IGA docs][iga]
for a VDF-based KeyValues document and examples.

The package supports VDF text and a documented subset of the binary format,
as well as mapping between VDF documents and Go structs.

## When to use it

Use `vdf` when a Go program needs to inspect
or update existing Valve data, generate configuration files,
or exchange VDF with another tool while preserving ordering and repeated keys.

## Installation

Add the dependency with:

```sh
go get github.com/woozymasta/vdf@latest
```

## Use

Read a value from VDF text:

```go
func read() error {
    doc, err := vdf.ParseString(`"server" { "name" "dayz" }`)
    if err != nil {
        return err
    }

    fmt.Println(*doc.Roots[0].First("name").StringValue)
    return nil
}
```

Build a document and write it as VDF text:

```go
func write() error {
    doc, err := vdf.NewBuilder("server").Set("name", "dayz").Document()
    if err != nil {
        return err
    }

    text, err := vdf.WriteString(doc)
    if err != nil {
        return err
    }

    fmt.Print(text)
    return nil
}
```

## Struct mapping

`vdf` maps exported Go struct fields to VDF keys.
The `vdf` tag controls the key name and how a field is represented:

Tag | Meaning
--- | ---
`vdf:"name"` | Use `name` as the VDF key.
`vdf:"-"` | Skip the field.
`vdf:",omitempty"` | Omit the field when its Go value is zero.
`vdf:",omitzero"` | Omit if `IsZero()` is true; else if the value is zero.
`vdf:",inline"` | Place the struct's fields in the parent object.
`vdf:",repeated"` | Write a slice as repeated sibling keys.
`vdf:",indexed"` | Encode a slice under numeric object keys (`"0"`, `"1"`, ...).

Indexed slices decode by key index, regardless of source order.
Keys must be canonical non-negative integers forming contiguous range `0..N-1`.
Fields implementing `encoding.TextMarshaler` or `encoding.TextUnmarshaler`
use those interfaces for scalar conversion.

## Compatibility

* Text keys and values may be quoted or unquoted.
  Supported escapes are `\\`, `\"`, `\n`, `\t`, and `\r`.
  Line comments are supported; block comments are not.
* Binary support covers objects (`0x00`), strings (`0x01`),
  and `uint32` values (`0x02`, little-endian).
  Strings are NUL-terminated and documents end with `0x08`.
  Other Valve binary types are not supported.
* Decode `FormatAuto` selects binary when the first byte
  is `0x00`, `0x01`, `0x02`, or `0x08`;
  empty input and other first bytes are treated as text.
  Encode `FormatAuto` follows the document's text or binary format,
  and uses text when the document format is also automatic.
* Documents preserve duplicate keys and their order.
  Strict decoding rejects duplicates.

<!-- links -->

[vdf]: https://pkg.go.dev/github.com/woozymasta/vdf
[vdf-ref]: https://developer.valvesoftware.com/wiki/VDF
[iga]: https://partner.steamgames.com/doc/features/steam_controller/iga_file
