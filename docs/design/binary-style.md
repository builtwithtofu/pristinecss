# Design draft: one binary, many uses

Status: draft for discussion. Names in code are placeholders.

## The idea

Authors write modern CSS once. pristinecss turns it into a binary that holds what
the CSS means. Anything can load that binary and get CSS that fits where it is
going: any browser, a fixed target, or a native app with no DOM. Loading is free
and lowering takes microseconds.

The binary lives in code. It is never sent anywhere. What gets sent, and gzipped,
is the CSS that comes out of it.

## What the library does, and what it leaves to others

pristinecss:

- turns CSS into a binary, with diagnostics
- opens a binary, trusted (a constant in code) or checked (anything else)
- writes CSS out, lowered for a set of capabilities
- says at build time what cannot be lowered for a target
- hands typed values to programs that are not browsers
- owns the capability list: names, bits and presets

The caller:

- decides where the binary is stored: a Go constant, a file, memory
- finds out what a browser can do (the handshake)
- serves the CSS and caches it
- matches rules to elements, if it is not a browser

## Rules we keep

- Never a silent gap. If something can't be lowered for a target, say which rule,
  for which target, and what happened instead.
- The binary is read-only (`Style`). Tools that change CSS use the editable tree
  (`Sheet`). Mixing the two is what made the old AST frame 2.7 times the size
  of its CSS.
- The spans binary is the only binary format. Step 2 removed the old AST frame
  and the IR.

## How it is used

### Build: CSS in, binary out

```go
func ToBinary(css []byte, o Options) ([]byte, Report)

type Options struct {
    Raise       Caps     // opt-in: rewrite up to this floor; zero keeps CSS as written
    Lower       []Caps   // opt-in: the targets to support; empty means no lowering
    MinFidelity Fidelity // skip rewrites less faithful than this; default Identical
}

type Fidelity uint8 // Identical, Close, Approximate

type Report struct {
    Errors   []ParseError
    Changes  []Change // raises made
    Gaps     []Gap    // what can't be lowered for each target
    Variants int      // finished outputs stored
}

type Change struct {
    Lo, Hi uint32
    Cap    Cap
    What   string
}
```

`Raise` opts in to rewriting old CSS to a modern floor. Zero options keep CSS
as written. Raising drops prefixes Baseline does not need, removes old hacks
such as `*zoom` and `\9`, collapses fallback pairs, and upgrades old syntax.
`Lower` opts in to lowering, browserslist-style: list the browsers to support,
such as `[]Caps{Browser("chrome", "49"), Browser("safari", "12")}`. The build
does all the lowering. It cuts each target down to the capabilities this sheet
uses, drops duplicates, and stores one finished output per remaining target plus
the modern output, so there are at most `len(Lower)+1`. A slightly bigger binary
buys a request that does nothing but copy.

`MinFidelity` is a build-time choice only. `Source()` always returns the exact
original CSS.

The caller decides what to do with the bytes. Clay writes them into the
component it generates. Someone else might write a `.go` file:

```go
fmt.Fprintf(w, "package styles\n\nconst App = %q\n", bin)
```
### Open it

```go
var app = pristinecss.Open(styles.App)   // trusted constant: costs nothing
app, err := pristinecss.Load(fileBytes)  // checked: for anything else
```

`Style` is read-only. One copy is shared by every request and goroutine.

### Get CSS out

Raise and lower when building the binary, then pick per request. Each browser
gets the richest stored output it can safely use:

```go
css := app.CSS(dst, caps) // minified, lowered for caps
css = app.CSS(dst, caps, pristinecss.Pretty)
src := app.Source()       // the exact file the author wrote
```

`dst` lets callers reuse a buffer, so a warm server allocates nothing. `CSS`
picks the richest stored variant whose capabilities are all in `caps`. When none
fits, such as an unknown browser, it uses the most-lowered variant.

### Live server

```go
func serveCSS(w http.ResponseWriter, r *http.Request) {
    caps := pristinecss.CapsFromBits(handshakeBits(r)) // handshake is yours
    w.Header().Set("Content-Type", "text/css")
    w.Write(app.CSS(nil, caps))
}
```

For caching, `app.Key(caps)` names the variant `CSS` would pick, so browsers
that get the same output share a key. It does no hashing per request.

### Ahead of time

Build with that one target in `Lower`, then:

```go
css := app.CSS(nil, pristinecss.Browser("chrome", "49"))
```

Or from the command line:

```sh
pristinecss lower app.css --target "chrome 49"
```

### What cannot be lowered

`Report.Gaps` lists what cannot be lowered for each target, at build time,
before anything ships. A gap is never silent; it says where, for which target,
and what happened instead.

### Native apps

The app does the matching. pristinecss hands over rules and decoded values:

```go
for rule := range app.Rules() {
    sel := rule.Selector() // parsed: classes, IDs, pseudos, combinators
    for d := range rule.Decls() {
        switch d.Prop() { // a number, not a string compare
        case prop.Color:
            if c, ok := d.Value().Color(); ok { /* use c.R, c.G, c.B */ }
        case prop.PaddingLeft:
            if l, ok := d.Value().Length(); ok { /* use l.N, l.Unit */ }
        }
    }
}
```
### Tools

Linters, formatters and editors change CSS, so they keep today's editable tree
(`Parse`, `Walk`, `Delete`, `Emit`). When they are done they call `ToBinary` on
the result.

## Capabilities and rewrites

Capabilities are numbered, append-only, and cover the features pristinecss can
raise, lower or report on. The list is generated from web-features and MDN
data. Browser IDs use caniuse-style names, including `chrome`, `firefox`,
`safari`, `ios_saf`, `and_chr`, `edge`, `ie` and `samsung`.

```go
type Cap uint16
type Caps struct{ bits [4]uint64 }

func (Caps) Has(Cap) bool
func (Caps) Bits() [4]uint64
func CapsFromBits([4]uint64) Caps

var Modern Caps
func Baseline(year int) Caps
func Browser(id BrowserID, version string) Caps
```

`Caps` is a comparable value with no pointers, so it can be a map key. `Modern`
means everything is supported, so nothing is lowered.

Serving uses `CSS(dst, caps, opts...)` and `Key(caps)`; gaps come from the
build `Report`. Rewrites are internal and never exported. One table keyed by `Cap`
holds each rewrite's fidelity, raise function and lower function.

The files are `caps.go`, `caps_gen.go`, `style.go`, `rewrite.go`,
`rewrite_<area>.go`, and `internal/capgen`.

Raising is opt-in at build time. The CLI is:

```sh
pristinecss raise old.css --to baseline-2024
pristinecss lower app.css --target "chrome 49"
```

## What the binary holds

Size is cheap because the binary lives in code. Speed is what counts. So the
binary carries work done at build time:

- the original CSS, so `Source()` is exact
- an index over rules and declarations
- values decoded into numbers: colours, lengths, keywords
- for each rule, the capabilities it needs, so lowering skips the rest
- the plain minified output, ready-made

## Evidence so far

Prototypes, Apple M5, all 22 sheets in `test-data/`, run one at a time.

Step 2 as built, Bootstrap: checked `Load` 171 µs, trusted `Open` 0.0003 µs,
against 1,036 µs to parse the CSS text in the same run.

| Ready to use | Bootstrap | All 22 |
|---|---:|---:|
| Parse the CSS text | 932 µs | 13.8 ms |
| Old AST frame's `Load` | 320 µs | 4.8 ms |
| Trusted open, no copy | 0.03 µs | 0.001 ms |

| Index design (all 22) | Size | Walk every rule | Minified out |
|---|---:|---:|---:|
| Old AST frame | 11.0 MB | 7.0 ms | 5.4 ms |
| Spans: CSS + index | 5.8 MB | 2.6 ms | 0.75 ms |

| Lowering `inset` and `oklch()` | Size | Lowered | Time |
|---|---:|---:|---:|
| Bootstrap | 280 KB | few | 46 µs |
| DaisyUI | 1.09 MB | 368 | 400 µs |
| Dense synthetic | 1 MB | 40,208 | 5.8 ms |

Not yet measured: decoded values (the main fix for dense lowering), and how long
Go takes to compile very large string constants.

## Open questions

1. Is `Lower` the only transform, or can users plug in their own? Plugins would
   slow the microsecond path.
2. How far back should browser presets go? IE6 is the worst case, not a promise.
3. How does a browser report what it can do? The caller owns the handshake. The
   handshake model still needs deciding: Polyfill.io is one model to study, but
   decide by what the browser reports, not its user agent.
## Steps

Each step ends with something we can measure or use.

Done:

- Clear the decks: the broken-CSS recovery work and these docs landed.
- Check the risk: a large string constant builds in under 2 seconds and costs
  nothing at startup. A `[]byte{...}` literal is the form to avoid.
- Style v1: `ToBinary`, `Open`, `Load`, `Source()` and `CSS(dst, Modern)` on the
  spans binary, replacing the old AST frame and the IR.

Next:

1. **Benchmark budget.** Time prototype lowering and raising on the spans
   binary for every corpus sheet. Those times are the budget steps 3 and 4 must
   keep or beat.
2. **Capabilities.** Generate the append-only capability list and provide
   `Caps`, `Baseline` and `Browser`.
3. **Opt-in lowering.** `Options.Lower` stores one finished output per target,
   starting with `inset` and `oklch()`/`oklab()`; `CSS(dst, caps)` picks one.
   Include `Report.Gaps` and `Key(caps)`; never leave a silent
   gap.
4. **Opt-in raise.** `ToBinary(css, Options{Raise: ...})` raises CSS to a modern
   floor and reports every change. `Source()` stays exact.
5. **CLI.** Add `pristinecss raise` and `pristinecss lower`.

Deferred: more rewrites from `lowerings.md`, decoded values, the native reader,
JS shims, and rendering in real browsers to prove fidelity.

## Questions for Clay

1. Does the TUI want pristinecss at run time, reading `Rules()` and typed values?
   Or does Clay's compiler read them and generate its own style structs, so the
   TUI never loads a binary at all?
2. What shape suits Clay's matching: rules in source order, or grouped by
   selector part (class, element, ID)?
3. Which values does the TUI need decoded: colours, lengths, keywords, others?
4. For web, does Clay want one binary per component, or one per app?
