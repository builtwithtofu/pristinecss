# pristinecss

pristinecss is a Go library for parsing, inspecting, transforming, emitting, and embedding CSS. Editable sheets use a flat pre-order array of 16-byte pointer-free nodes. Read-only styles embed the original CSS, a sparse rule/declaration index, and ready-made output in a binary string constant.

## Quickstart

```go
src := []byte(`a { color: oklch(70% 0.1 200) }`)
sheet, errs := pristinecss.Parse(src)
if len(errs) != 0 {
    // best-effort sheet is still returned
}

for c := range sheet.All(pristinecss.KindValFunction) {
    if string(c.FunctionName()) == "oklch" {
        sheet.ReplaceRaw(c.ID(), []byte("rgb(81 174 178)"))
    }
}

out := sheet.Emit(nil, pristinecss.EmitOptions{Style: pristinecss.Minified})
fmt.Printf("%s\n", out)
```

## Ownership and concurrency

`Parse` and `ParseInto` borrow their source buffer: the caller must not mutate or free it while the `Sheet` or any `Extract`ed child lives. Use `ExtractOwned` when a result must outlive the original bytes. Binary `Style` values are immutable and safe to share across goroutines. `Load` copies its input; `Open` borrows an immutable string.

A sheet with no mutator ever called is safe for unlimited concurrent readers. Mutators (`Delete`, `Unwrap`, `ReplaceRaw`, `Compact`) require exclusive access.

## Variants are data

Emission takes an `EmitStyle` value instead of a formatter interface. `Minified` is the zero-whitespace style; `Pretty` injects indentation and separator bytes. `EmitOptions.Filter` skips block children in O(1).

## Embedding

`ToBinary(css)` returns a read-only spans binary and the same diagnostics as `Parse`. It retains the exact source and precomputes minified and pretty output. Generate a Go string constant with `fmt.Fprintf(w, "const App = %q\n", bin)`.

```go
app := pristinecss.Open(styles.App) // trusted binary constant; no validation or copy
out := app.CSS(dst, pristinecss.Modern) // minified; reuses dst
pretty := app.CSS(dst[:0], pristinecss.Modern, pristinecss.Pretty)
source := app.Source() // exact original CSS
```

Use `Load(bytes)` for untrusted binaries: it checks the binary's structure and a checksum over all of it, and owns a copy. The checksum catches accidental damage, not deliberate tampering. `Open` is only for binaries built by your own program. `CSS` accepts only the `Minified` and `Pretty` presets, not custom whitespace settings. Lowering is not implemented yet.

## Architecture

PristineCSS provides reusable CSS mechanisms rather than application policy. Consumers
decide how to partition inputs, interpret custom at-rules, scope selectors, assign
capability bits, construct variants, and deliver assets.

The design philosophy is recorded in [`docs/adr/`](docs/adr/).

## Performance snapshot

On this machine, bootstrap.css parses warm via `ParseInto` in about 0.49ms with zero allocations, and minified emit runs in about 0.93ms with zero allocations when the destination buffer has capacity.
