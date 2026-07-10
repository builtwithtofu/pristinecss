# ADR 0003: Versioned embeddable AST and IR frames

## Status

Accepted

## Context

Some consumers do all CSS work at runtime, while others parse and transform CSS during
generation or build steps and need the result as static program data. Repeating parsing
when the input and transformation are already known wastes startup or request time.

The reusable artifacts have two distinct shapes:

- a parsed tree for later inspection or transformation; and
- emitted CSS divided into independently selectable top-level rules.

Neither artifact should depend on Go object pointers, implementation-specific encoders,
or sidecar files.

## Decision

PristineCSS defines a versioned binary frame with `PCSS` magic and separate AST and IR
kinds.

AST frames contain the flat node records and scratch storage. They can either:

- include source bytes and be self-contained; or
- omit source bytes and carry a checksum that binds the frame to byte-identical source
  supplied to `Load`.

IR frames contain minified CSS and one record for each emitted top-level rule. Each
record contains:

- its byte span in the emitted CSS;
- an opaque required capability mask; and
- an opaque fallback capability mask.

`CompileIR` accepts masks chosen by the caller and records rule spans while emitting.
`FilterIR` selects a rule when all required bits are present and all fallback bits are
absent. PristineCSS defines this selection algebra but does not assign domain meanings to
individual bits.

Loaders validate frame lengths, kinds, versions, checksums, node invariants, and rule
spans before returning data. A loader rejects frames from newer unsupported versions
with an error that directs the caller to regenerate or upgrade.

## Consequences

- Generated programs can use `go:embed` for parsed sheets or request-time lookup data
  without reparsing CSS.
- The same parser and emitter support runtime libraries, generators, static analyzers,
  and other consumers without prescribing when work occurs.
- AST frames preserve transformation capability; IR frames minimize selection-time
  work. Consumers choose the artifact appropriate to their boundary.
- Omitting source bytes reduces duplication but requires the exact original source at
  load time.
- Frame bytes are a compatibility contract. Layout or semantic changes require version
  handling rather than silent reinterpretation.
- Capability-bit assignment, feature detection, and variant construction remain caller
  policy. If a consumer persists a bit schema independently of the frame version, that
  schema must carry its own identity and compatibility rules.
- Combining multiple IR payloads requires rebuilding rule offsets and frame metadata;
  complete frame byte strings are not directly concatenable.

## Related docs

- ADR 0001 makes direct node serialization practical.
- ADR 0002 defines source borrowing and owned extraction.
- ADR 0004 defines the boundary between reusable mechanisms and consumer policy.
