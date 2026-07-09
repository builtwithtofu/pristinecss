# pristinecss

pristinecss is a Go library for parsing, inspecting, transforming, emitting, and embedding CSS. It stores each stylesheet as a flat pre-order array of 16-byte pointer-free nodes, so rules and values can be walked by dense `NodeID`, shared between goroutines when immutable, and serialized into generated code.

## Quickstart

```go
src := []byte(`@web { a { color: oklch(70% 0.1 200) } }`)
sheet, errs := pristinecss.Parse(src)
if len(errs) != 0 {
    // best-effort sheet is still returned
}

for c := range sheet.All(pristinecss.KindAtGeneric) {
    if string(c.Name()) == "web" {
        sheet.Unwrap(c.ID())
    }
}

out := sheet.Emit(nil, pristinecss.EmitOptions{Style: pristinecss.Minified})
fmt.Printf("%s\n", out)
```

## Ownership and concurrency

`Parse`, `ParseInto`, and `Load` borrow their source buffer: the caller must not mutate or free it while the `Sheet` or any `Extract`ed child lives. Use `ExtractOwned` or `Marshal(includeSrc=true)` when a result must outlive the original bytes.

A sheet with no mutator ever called is safe for unlimited concurrent readers. Mutators (`Delete`, `Unwrap`, `ReplaceRaw`, `Compact`) require exclusive access.

## Variants are data

Emission takes a `Style` value instead of a formatter interface. `Minified` is the zero-whitespace style; `Pretty` injects indentation and separator bytes. `EmitOptions.Filter` skips block children in O(1), and `EmitOptions.OnRule` reports top-level output spans for IR compilation.

## Embedding

`Marshal` stores an AST frame. `CompileIR` stores minified CSS plus per-rule `[4]uint64` masks for serve-time filtering with `FilterIR`. Both use the same `PCSS` versioned frame and reject newer blob versions with a regeneration-oriented error.

## Performance snapshot

On this machine, bootstrap.css parses warm via `ParseInto` in about 0.49ms with zero allocations, and minified emit runs in about 0.93ms with zero allocations when the destination buffer has capacity.
