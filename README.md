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

## CLI

Build with `go build -o pristinecss ./cmd/pristinecss`, then run:

```sh
pristinecss raise old.css --to baseline-2024 > modern.css
pristinecss lower app.css --target "chrome 49" > chrome49.css
```

CSS goes to stdout, pretty-printed by default; `--minified` gives minified output. Changes, lowering gaps, gamut mappings and recoverable parse errors go to stderr. Parse errors do not fail the command; usage and file read errors do. Use `-` instead of a file to read stdin. `--to baseline-YYYY` selects `Baseline(YYYY)`. Targets use caniuse-style IDs (for example `chrome`, `ios_saf`, `safari`) and `BrowserID.At` versions, including `"ios_saf 15.2-15.3"` and `"safari TP"`. Rewrites are exactly those provided by the library.

## Ownership and concurrency

`Parse` and `ParseInto` borrow their source buffer: the caller must not mutate or free it while the `Sheet` or any `Extract`ed child lives. Use `ExtractOwned` when a result must outlive the original bytes. Binary `Style` values are immutable and safe to share across goroutines. `Load` copies its input; `Open` borrows an immutable string.

A sheet with no mutator ever called is safe for unlimited concurrent readers. Mutators (`Delete`, `Unwrap`, `ReplaceRaw`, `Compact`) require exclusive access.

## Variants are data

Emission takes an `EmitStyle` value instead of a formatter interface. `Minified` is the zero-whitespace style; `Pretty` injects indentation and separator bytes. `EmitOptions.Filter` skips block children in O(1).

## Embedding

`ToBinary(css, Options{})` returns a read-only spans binary and a `Report` containing parse errors, raising changes, lowering gaps, gamut mappings, variant count and binary size (`Bytes`). It retains the exact source and precomputes minified and pretty output. Generate a Go string constant with `fmt.Fprintf(w, "const App = %q\n", bin)`.

```go
app := pristinecss.Open(styles.App) // trusted binary constant; no validation or copy
out := app.CSS(dst, pristinecss.Modern) // minified; reuses dst
pretty := app.CSS(dst[:0], pristinecss.Modern, pristinecss.Pretty)
source := app.Source() // exact original CSS
```

Use `Load(bytes)` for untrusted binaries: it checks the binary's structure and a checksum over all of it, and owns a copy. The checksum catches accidental damage, not deliberate tampering. `Open` is only for binaries built by your own program. `CSS` accepts only the `Minified` and `Pretty` presets, not custom whitespace settings.

### Build-time raising

```go
bin, report := pristinecss.ToBinary(css, pristinecss.Options{
    Raise: pristinecss.Baseline(2024),
})
```

`Raise` drops `-webkit-`, `-moz-`, `-ms-` and `-o-` declarations only when a later, nonempty unprefixed declaration in the same block overrides their importance and its capability is supported at the floor. A prefixed `!important` requires an unprefixed `!important`. Unknown properties, unsupported capabilities and replacements with parser errors stay as written. Limitation: an unprefixed value that our parser accepts but the browser rejects at parse time can still cause an unsafe drop; raising does not validate property-specific value grammars. `Report.Changes` lists every drop with original source byte positions (`Lo`, exclusive `Hi`), capability and reason. `Source()` stays exact.

Inside keyframe blocks, unprefixed `!important` declarations do not count as replacements: browsers ignore those declarations.

Raising runs before lowering. Every stored output starts from the raised CSS. A lower target lacking a dropped declaration's capability gets a `Report.Gaps` entry if BCD records its browser/version reading that specific prefixed property, including compatibility aliases. For a capability-only target, a gap is reported if any recorded browser could read the prefix without supporting the unprefixed feature. Prefix support is build-time browser data, never capability bits. Zero `Options{}` leaves the binary and output byte-identical.

### Build-time lowering

```go
bin, report := pristinecss.ToBinary(css, pristinecss.Options{
    Lower: []pristinecss.Target{
        pristinecss.Chrome.At("49"),
        pristinecss.Safari.At("12"),
        pristinecss.Firefox.At("60"),
    },
})
```

`Lower` accepts `Target` values, retaining each browser's `BrowserID`, original `Version`, and `Caps`. Generated IDs include `Chrome`, `Edge`, `Firefox`, `Safari`, `IOSSafari`, `ChromeAndroid`, `FirefoxAndroid`, `Samsung`, `Opera`, `OperaAndroid`, and `IE`. `.At(version)` accepts the same versions as `Browser`, including ranges, ESR suffixes and `TP`. Use `Target{Caps: caps}` for a capability-only target. Browser-specific facts such as prefix support belong in per-browser data, not capability bits.

The build stores one complete output per distinct projected `Target.Caps`, plus the unlowered modern output. Empty `Lower` disables lowering and retains the original single-output binary format. With zero `Raise`, it preserves the original binary byte for byte. `Report.Gaps` and `Report.Mappings` retain the full requested `Target`, even when different identities share an output. Only exact duplicate targets share diagnostics. Request-time callers still use `Browser(Safari, "17")` or handshake `Caps` with `CSS` and `Key`.

`CSS(dst, caps)` copies the richest stored output whose required capabilities are present in `caps`. If none fits, it copies the most-lowered output. Richness counts the projected target's capabilities. Among equally rich safe variants, prefer more retained native colour capabilities (`oklch`/`oklab`), then P3 output over sRGB. Remaining ties use the lowest projected capability bitmask, independent of target order. `Key(caps)` identifies the binary and selected variant, ignoring capabilities that do not change the selection.

The first rewrites expand `inset` to four sides and convert literal `oklch()`/`oklab()` into `color(display-p3 ...)` when the target has `CapDisplayP3`, or legacy `rgb()`/`rgba()` with integer RGB channels otherwise. Alpha is preserved.

Inset expansion validates the whole declaration, including priority. Only one to four lengths, percentages or `auto`, or a single CSS-wide keyword, are expanded. Literal `calc()` arithmetic is type-checked; unsupported math functions, substitutions and invalid values remain unchanged with a positioned gap. The parser recognises escaped, ASCII case-insensitive `important` names even without lowering; rewrites use its `FlagImportant` metadata.

In-gamut colours are `Identical` apart from serialization rounding. Out-of-gamut colours use [CSS Color 4's local-MINDE gamut mapping](https://www.w3.org/TR/css-color-4/#gamut-mapping): preserve OKLCh lightness and hue while binary-searching chroma, with deltaEOK JND 0.02. Mapping is `Close` fidelity. Zero `MinFidelity` defaults to `Close`, so mapping needs no extra opt-in. Explicit `Identical` skips colours that need mapping and reports those sites as fidelity gaps.

`Report.Mappings` records each mapped colour's source byte positions, capability, requested target and destination gamut. Mappings are `Close` by definition; `Changes` stays reserved for raising. `var()`, `calc()`, `none` and relative colour syntax remain gaps, as do top-level `var()`, `env()` and `attr()` substitutions in `inset`. Repeated targets do not repeat diagnostics. Unresolved expressions do not veto the target's other lowerings. Colours kept because of a fidelity gap still require their original capability when selecting a safe variant.

`CapDisplayP3` is bit 170. Unlike the broader `CapColorFunction`, it includes Safari's P3-only support from 10.1 (iOS 10.3); Chrome supports it from 111 and Firefox from 113, according to [BCD's `css.types.color.color`](https://raw.githubusercontent.com/mdn/browser-compat-data/v8.1.7/css/types/color.json). `Key` reuses the stored CRC-32C and allocates nothing; it is not a cryptographic identity.

## Architecture

PristineCSS provides reusable CSS mechanisms rather than application policy. Consumers
decide how to partition inputs, interpret custom at-rules, scope selectors, assign
capability bits, construct variants, and deliver assets.

The design philosophy is recorded in [`docs/adr/`](docs/adr/).

## Performance snapshot

On this machine, bootstrap.css parses warm via `ParseInto` in about 0.49ms with zero allocations, and minified emit runs in about 0.93ms with zero allocations when the destination buffer has capacity.
