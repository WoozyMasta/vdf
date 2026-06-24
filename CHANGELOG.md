<!-- markdownlint-disable MD024 -->
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog][],
and this project adheres to [Semantic Versioning][].

<!--
## Unreleased

### Added
### Changed
### Removed
-->

## [0.2.1][] - 2026-06-24

### Fixed

* `NextEvent` now returns `ErrUnexpectedEOFInObject`
  when EOF is reached with one or more unclosed objects,
  matching the behavior of `DecodeDocument`
* `NextEvent` now enforces `MaxNodes` and `Strict` duplicate-key detection,
  previously ignored in the streaming path;
  adds `nodeCount` counter and per-scope key sets to `streamState`
* `Encoder.Close` now validates that all objects are closed for text format,
  not only for binary; unclosed objects return `ErrInvalidNodeState`
* Text lexer `readQuotedString`, `readUnquotedString`
  and binary decoder `readNullTerminatedString` now enforce byte-length limits
  during reading rather than after the full string is accumulated,
  preventing unbounded memory growth on malformed input without a delimiter

[0.2.1]: https://github.com/WoozyMasta/vdf/compare/v0.2.0...v0.2.1

## [0.2.0][] - 2026-06-24

### Added

* `DecodeOptions.MaxKeyBytes`, `MaxValueBytes`, `MaxStringBytes` -
  byte length limits for keys and string values;
  enforced in both text and binary decoders
* `Marshal(root, v)` and `Unmarshal(doc, root, out)`
  reflection API for converting Go structs to/from VDF documents
* Struct tag options: `omitempty`, `omitzero`, `inline`, `repeated`, `indexed`
* `NewBuilder` fluent ordered API for constructing documents
  without manual AST manipulation; `Set`, `SetUint32`, `Object`, `Document`
* `FromMapSorted` like `FromMap` but keys are sorted lexicographically
* `WalkEvents` canonical name for DFS event traversal over a decoded AST

### Changed

* `NextEvent` now does true one-pass streaming instead of AST traversal;
  use `WalkEvents` for the old behavior
* `EncodeText` ~2x faster, 6x fewer allocations:
  `fmt.Fprintf` replaced with direct `io.WriteString` calls;
  escaping writes directly to the writer

[0.2.0]: https://github.com/WoozyMasta/vdf/compare/v0.1.1...v0.2.0

## [0.1.1][] - 2026-02-18

### Added

* File wrapper API

[0.1.1]: https://github.com/WoozyMasta/vdf/compare/v0.1.0...v0.1.1

## [0.1.0][] - 2026-02-18

### Added

* First public release

[0.1.0]: https://github.com/WoozyMasta/vdf/tree/v0.1.0

<!--links-->
[Keep a Changelog]: https://keepachangelog.com/en/1.1.0/
[Semantic Versioning]: https://semver.org/spec/v2.0.0.html
