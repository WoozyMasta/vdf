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

```go
doc := vdf.NewDocumentWithFormat(vdf.FormatText)
root := vdf.NewObjectNode("settings")
root.Add(vdf.NewStringNode("name", "demo"))
root.Add(vdf.NewUint32Node("port", 2302))
doc.AddRoot(root)
```

`NodeObject` keeps ordered children and allows duplicate keys.
This matches real VDF behavior.

## Streaming traversal

`Decoder.NextEvent` returns a sequence of structural
and leaf events that can be consumed incrementally.

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
