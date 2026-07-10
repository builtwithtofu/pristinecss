# ADR 0001: Flat pre-order AST

## Status

Accepted

## Context

PristineCSS must support parsing, inspection, transformation, emission, and embedding
without making callers pay for a pointer-heavy object graph. Its common operations walk
stylesheets in document order, inspect all fields of each visited node, skip whole
subtrees, and emit contiguous output. Parsed sheets should also remain cheap for the Go
garbage collector to retain and safe for concurrent readers when immutable.

A conventional typed tree would make insertion and parent navigation convenient, but it
would add pointers, interface values, child slices, allocation bookkeeping, and a custom
serialization traversal to the hot representation.

## Decision

The AST is one array of fixed-size `Node` records in pre-order.

- `Node` is 16 bytes and pointer-free.
- `NodeID` is the node's dense `uint32` array index; node 0 is the stylesheet root.
- Every node stores its complete subtree length in `Sub`, including itself.
- A node's first physical child immediately follows it. Its next physical sibling is
  `id + Sub`.
- Syntax payload is primarily represented by byte spans into source or scratch storage.
- `Kind` is one flat enum, including specific at-rule kinds and a generic at-rule kind.
- Parent maps, kind indexes, and other analysis structures are optional sidecars rather
  than permanent fields on every node.
- Tree-wide operations use array scans and subtree jumps instead of recursive object
  navigation.

The public API exposes cursors and iterators for ergonomic reading while preserving the
array as the canonical representation.

## Consequences

- Sequential walks and emission have strong locality, and skipped subtrees cost one
  index addition.
- The node array contains no pointers for the garbage collector to scan.
- Nodes can be serialized field-for-field into a stable frame representation.
- Holding a `NodeID` is cheaper and more portable than holding an object pointer.
- Parent lookup is not O(1) unless a caller builds the `Parents` sidecar.
- Kind-filtered queries are scans unless a caller builds an index.
- Insertion and reordering are intentionally second-class. Consumers that need extensive
  constructive rewriting should build new output or reparse a generated fragment rather
  than turn the core representation into a general mutable object graph.
- The compact fields impose representation limits: source offsets, subtree lengths, and
  node counts fit in `uint32`; auxiliary lengths fit in `uint16`. Parsing and loading must
  reject data that cannot be represented rather than truncate it.

## Related docs

- ADR 0002 defines ownership and mutation over this representation.
- ADR 0003 defines its serialized AST and emitted IR forms.
- `CONTEXT.md` defines physical and visible tree terminology.
