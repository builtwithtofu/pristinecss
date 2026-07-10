# ADR 0002: Borrowed source and bounded mutation

## Status

Accepted

## Context

Most syntax text already exists in the input buffer. Copying it into every parsed node
would increase parse cost and retained memory, while unrestricted structural mutation
would undermine stable IDs, contiguous storage, and concurrent read safety.

The library nevertheless needs transformations that can remove rules, expose the
contents of wrapper rules, and replace selected syntax without rebuilding the complete
tree after every edit.

## Decision

A `Sheet` borrows its source buffer and owns its node and scratch buffers.

- Callers must not modify or reuse source bytes while the Sheet or a borrowing extract
  is live.
- Node text is a read-only view into source or Sheet-owned scratch bytes.
- A Sheet that has never been mutated is safe for concurrent readers.
- Every mutator requires exclusive access; the library does not add locks around a
  Sheet.
- `Delete` tombstones a node and its complete subtree.
- `Unwrap` makes a container transparent in the visible tree and emission.
- `ReplaceRaw` redirects a node's presentation to Sheet-owned scratch bytes while
  preserving its kind, subtree length, and `NodeID`.
- These bounded mutations do not move nodes, so IDs and sidecars remain stable.
- `Compact` is the explicit phase boundary that rebuilds the physical tree, removes
  tombstones and superseded descendants, applies unwraps, and compacts scratch storage.
  It returns an old-to-new ID remap because previous IDs are no longer valid.
- `Extract` creates an independent node array that still borrows source bytes.
  `ExtractOwned` also copies referenced bytes and can outlive the original source.

The visible tree used by `Walk`, `All`, and `Emit` applies tombstone and unwrap state.
Physical cursor navigation and `Nodes` continue to describe the stored pre-order tree.

## Consequences

- Parsing avoids a source copy and replacement storage grows only with actual edits.
- Common filtering and substitution transforms are O(1) at mutation time.
- Immutable sheets can be retained and shared without synchronization or GC pointer
  scanning in the node array.
- Mutation accumulates debt: deleted nodes, replaced descendants, and superseded scratch
  bytes remain until compaction.
- `Compact` is O(n) and invalidates held IDs; consumers must rebuild or remap sidecars at
  that boundary.
- Returned byte slices and the `Nodes` slice are views, not mutable ownership transfers.
  Writing through them violates Sheet invariants and concurrency guarantees.
- A consumer needing unrestricted insertion or reordering should construct fresh CSS
  rather than rely on compatibility layers around the bounded mutation model.

## Related docs

- ADR 0001 defines the flat pre-order representation.
- ADR 0003 defines owned and source-bound serialized forms.
- `README.md` states the caller-facing ownership contract.
