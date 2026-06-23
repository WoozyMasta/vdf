# vdf

This project provides a high performance implementation of
Valve Data Format (VDF) text and binary formats.  
It is built around `io.Reader` and `io.Writer` for streaming workloads,
keeps an explicit AST (`Document` and `Node`) as the canonical model,
preserves node order and duplicate keys,
and includes low allocation byte slice encode paths.

## Reading VDF

Use `ParseString`, `ParseBytes`, or `NewDecoder` depending on your input source.

```go
doc, err := vdf.ParseString(`"root" { "name" "srv" }`)
if err != nil {
    return err
}

root := doc.Roots[0]
name := root.First("name")
```

Use auto format detection when input may be text or binary:

```go
doc, err := vdf.ParseAuto(data)
if err != nil {
    return err
}
```

For file inputs, use `ParseFile` with optional options or convenience wrappers:
`ParseTextFile` and `ParseAutoFile`.

## Writing VDF

For full document encode, use `WriteString`, `AppendText`, `AppendBinary`, or `NewEncoder`.

```go
out, err := vdf.WriteString(doc)
if err != nil {
    return err
}
```

For binary output with low allocations:

```go
bin, err := vdf.AppendBinary(nil, doc, vdf.EncodeOptions{
    Format: vdf.FormatBinary,
})
if err != nil {
    return err
}
```

If strict AST checks are required before encoding:

```go
enc := vdf.NewEncoder(w, vdf.EncodeOptions{
    Format:   vdf.FormatText,
    Validate: true,
})
err := enc.EncodeDocument(doc)
```

For file output, use `WriteFile` with optional options or convenience wrappers:
`WriteTextFile` and `WriteBinaryFile`.

## Building a VDF document

### Fluent Builder

`NewBuilder` provides a safe, ordered API for constructing documents:

```go
doc, err := vdf.NewBuilder("settings").
    Set("name", "demo").
    SetUint32("port", 2302).
    Object("auth", func(b *vdf.Builder) {
        b.Set("token", "abc123")
    }).
    Document()
```

### Manual construction

```go
doc := vdf.NewDocumentWithFormat(vdf.FormatText)
root := vdf.NewObjectNode("settings")
root.Add(vdf.NewStringNode("name", "demo"))
root.Add(vdf.NewUint32Node("port", 2302))
doc.AddRoot(root)
```

`NodeObject` keeps ordered children and allows duplicate keys.
This matches real VDF behavior.

## Reflection API

Use `Marshal` and `Unmarshal` to convert between Go structs and VDF documents.

```go
type Server struct {
    Name string `vdf:"name"`
    Port uint32 `vdf:"port"`
}

// Encode
doc, err := vdf.Marshal("Server", Server{Name: "game-1", Port: 2302})

// Decode
var s Server
err = vdf.Unmarshal(doc, "Server", &s)
```

### Struct tags

Tag | Meaning
--- | -------
`vdf:"name"` | Use `name` as the VDF key
`vdf:"-"` | Skip this field
`vdf:",omitempty"` | Omit if `reflect.Value.IsZero()` is true
`vdf:",omitzero"` | Omit if `IsZero() bool` is true, else reflect zero check
`vdf:",inline"` | Hoist struct fields into the parent object
`vdf:",repeated"` | Map `[]T` to multiple sibling nodes with the same key
`vdf:",indexed"` | Map `[]T` to a child object with keys `"0"`, `"1"`, ...

Fields implementing `encoding.TextMarshaler` / `TextUnmarshaler`
are handled automatically.

## Streaming traversal

Two event-based APIs are available.  
Both return events of type `EventType`:
`EventDocumentStart`, `EventObjectStart`, `EventObjectEnd`, `EventString`,
`EventUint32`, `EventDocumentEnd`.

### WalkEvents - AST-based traversal

Decodes the full document into an AST on the first call,
then traverses it in DFS order.
Use when you need the parsed document available after iteration.

```go
dec := vdf.NewDecoder(r, vdf.DecodeOptions{Format: vdf.FormatAuto})
for {
    ev, err := dec.WalkEvents()
    if err != nil {
        break
    }
    _ = ev
}
```

### NextEvent - true streaming

Yields events directly from the reader without building an AST.
Suitable for large inputs where keeping
the full document in memory is undesirable.

```go
dec := vdf.NewDecoder(r, vdf.DecodeOptions{Format: vdf.FormatAuto})
for {
    ev, err := dec.NextEvent()
    if err != nil {
        break
    }
    _ = ev
}
```

`WalkEvents` and `NextEvent` are mutually exclusive
on the same `Decoder` instance.
Mixing them returns `ErrInvalidNodeState`.

## Security limits

Use `DecodeOptions` to cap memory usage when parsing untrusted input:

```go
doc, err := vdf.ParseBytes(data, vdf.DecodeOptions{
    Format:         vdf.FormatAuto,
    MaxDepth:       32,
    MaxNodes:       10_000,
    MaxStringBytes: 4096, // ceiling for both keys and values
    MaxKeyBytes:    256,  // per-key limit (overrides MaxStringBytes keys)
    MaxValueBytes:  4096, // per-value limit (overrides MaxStringBytes values)
})
```

All limits use 0 to mean unlimited (the default).

## Compatibility notes

* **Text format** - keys and values may be quoted or unquoted.
  Escape sequences `\\`, `\"`, `\n`, `\t`, `\r` are supported.
  Line comments (`// ...`) are supported;
  block comments (`/* ... */`) are not.
* **Binary format** - little-endian uint32, null-terminated C strings.
  Type bytes:
  * `0x00` object start
  * `0x01` string
  * `0x02` uint32
  * `0x08` object end
* **Auto-detection** - `FormatAuto` peeks up to 64 bytes to detect the format.
  Files shorter than 64 bytes are handled correctly.
* **Duplicate keys** - the AST preserves duplicate keys in source order.
  Use `Node.All(key)` to retrieve all matches.
  `Strict: true` rejects duplicates at parse time.
* **Map conversions**:
  * `ToMapLossy` applies last-write-wins for duplicate keys;
  * `ToMapStrict` returns an error.
  * `FromMapSorted` builds a document with lexicographically ordered keys
    for deterministic output.
