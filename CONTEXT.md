# PristineCSS context

This file defines settled project terms. Architectural rationale belongs in
`docs/adr/`; delivery state belongs outside this glossary.

## Terms

**Sheet** — A parsed stylesheet. It owns its node and scratch buffers and borrows its
source bytes.

**Node** — One 16-byte, pointer-free AST record containing a kind, flags, an auxiliary
value, a byte span, and a subtree length.

**NodeID** — A dense index into a Sheet's pre-order node array. Node 0 is the stylesheet
root. IDs remain stable through bounded mutations and become invalid when the Sheet is
compacted; `Compact` returns an old-to-new remap.

**Physical tree** — The complete node array, including tombstoned subtrees and unwrapped
containers. Physical navigation is based on node order and subtree lengths.

**Visible tree** — The logical view presented by `Walk`, `All`, and `Emit`. Tombstoned
subtrees are absent and unwrapped containers expose their block children at the parent
level.

**Source span** — A `Lo:Hi` byte range into the borrowed source buffer.

**Scratch span** — A `Lo:Hi` byte range into Sheet-owned replacement storage, identified
by `FlagScratch`. Its hidden header retains the replaced source span.

**Bounded mutation** — A mutation that marks a subtree deleted, makes a container
transparent, or redirects a node to replacement bytes without moving nodes.

**Compaction** — The explicit O(n) rebuild that removes mutation debt, physically applies
unwraps, compacts replacement bytes, and remaps NodeIDs.

**AST frame** — A versioned `PCSS` serialization of nodes and scratch bytes, with either
embedded source bytes or a checksum binding to caller-supplied source bytes.

**IR frame** — A versioned `PCSS` serialization containing emitted CSS and records for
its top-level rules.

**Required mask** — Capability bits that must all be present for an IR rule to be
selected.

**Fallback mask** — Capability bits that must all be absent for an IR rule to be
selected.

**Structural round-trip** — Parse, emit, and parse again with equivalent syntax kinds,
payload flags, and significant node text; formatting and discardable comments need not
be byte-identical.
