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
  (`Sheet`). Mixing the two is what made today's binary 2.7 times the size of
  its CSS.
- This replaces today's AST frame (`Marshal`/`Load`) and the IR
  (`CompileIR`/`FilterIR`).

## How it is used

### Build: CSS in, binary out

```go
bin, diags := pristinecss.ToBinary(css) // bin is []byte
for _, d := range diags {
    fmt.Println(d) // "app.css:12:3: unknown property 'colr'"
}
```

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

One call covers plain output and lowering:

```go
css := app.CSS(dst, caps)                   // minified, lowered for caps
css = app.CSS(dst, caps, pristinecss.Pretty)
css = app.CSS(dst, pristinecss.Modern)      // no lowering; stored ready-made
src := app.Source()                         // the exact file the author wrote
```

`dst` lets callers reuse a buffer, so a warm server allocates nothing.

### Live server

```go
func serveCSS(w http.ResponseWriter, r *http.Request) {
    caps := pristinecss.CapsFromBits(handshakeBits(r)) // handshake is yours
    w.Header().Set("Content-Type", "text/css")
    w.Write(app.CSS(nil, caps))
}
```

For caching, `app.Key(caps)` gives the sheet's hash plus only the capabilities
this sheet cares about. A sheet that never uses `oklch()` has the same key for
browsers that differ only in colour support, so they share a cache entry.

### Ahead of time

```go
css := app.CSS(nil, pristinecss.Target("chrome 49"))
```

Or from the command line:

```sh
pristinecss build app.css --target "chrome 49" -o app.chrome49.css
```

### What cannot be lowered

```go
for _, d := range app.Unlowerable(pristinecss.Target("safari 12")) {
    fmt.Println(d) // "app.css:40:10: oklch() with var() can't become rgb()"
}
```

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

| Ready to use | Bootstrap | All 22 |
|---|---:|---:|
| Parse the CSS text | 932 µs | 13.8 ms |
| Today's `Load` | 320 µs | 4.8 ms |
| Trusted open, no copy | 0.03 µs | 0.001 ms |

| Index design (all 22) | Size | Walk every rule | Minified out |
|---|---:|---:|---:|
| Today's binary | 11.0 MB | 7.0 ms | 5.4 ms |
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
2. How far back should presets go? IE6 is the worst case, not a promise.
3. How is the capability list built? One idea: from the public Baseline data, so
   it updates when Baseline moves.
4. How does a browser report what it can do? Not fleshed out yet. Polyfill.io is
   the model to study, but decide by what the browser reports, not its user
   agent. One idea: each capability ships with the check a browser runs to
   report it, so the caller's handshake just runs our list and sends back bits.

## Known bug in today's binary

The AST frame's checksum covers only the CSS text, not the parsed structure. A
changed node passes `Load` and changes the output. Step 2 fixes it by replacing
the format.

## Steps

Each step ends with something we can measure or use.

0. **Clear the decks.** Finish and merge the broken-CSS recovery work. Commit the
   docs.
1. **Check the risk first.** Measure how long Go takes to compile very large
   string constants. If it's slow, rethink "constant in code" before building
   on it.
2. **Style v1, no lowering.** `ToBinary`, `Open`, `Load`, `Source()` and
   `CSS(dst, Modern)` in minified and pretty forms, using the spans design. The
   checksum covers everything. Replaces `Marshal`/`Load`. Proof: every corpus
   sheet round-trips exactly, output matches today's emitter, and it beats
   today's numbers.
3. **Capabilities and first lowerings.** `Caps`, a few capability bits, per-rule
   capability notes, `CSS(dst, caps)` lowering `inset` and `oklch()`/`oklab()`,
   `Unlowerable`, `Key(caps)` and two or three hand-written `Target` presets.
   Proof: Bootstrap lowers in tens of microseconds with no allocations once warm.
4. **Decoded values.** Colours, lengths and keywords decoded at build time;
   property names as numbers; lowering uses them. Proof: the dense 1 MB case
   drops well below 5.8 ms.
5. **Native reader.** `Rules()`, `Selector()`, `Decls()` and typed values, shaped
   by Clay's answers.
6. **CLI.** `pristinecss build --target`, and a command that outputs the binary.
7. **Capabilities at scale.** The capability list from Baseline data; more
   lowerings one at a time (nesting, logical properties and so on); the
   handshake model.
8. **Remove the old IR** once step 3 covers what it did.

Steps 2 to 4 are the core. Steps 5 to 7 can go in any order after that.

## Questions for Clay

1. Does the TUI want pristinecss at run time, reading `Rules()` and typed values?
   Or does Clay's compiler read them and generate its own style structs, so the
   TUI never loads a binary at all?
2. What shape suits Clay's matching: rules in source order, or grouped by
   selector part (class, element, ID)?
3. Which values does the TUI need decoded: colours, lengths, keywords, others?
4. For web, does Clay want one binary per component, or one per app?
