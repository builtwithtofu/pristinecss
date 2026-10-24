# Broken-CSS verification

## Recovery-fix verification (2026-10-24)

**Measured:** the base with the legacy-hack fix had 58 unstable inputs; after the recovery fixes, `go test ./...` and `go test -race ./...` pass, and `go vet ./...` exits 0. No existing assertion was changed, skipped or removed. Three reduced recovery cases and selector/replacement/valid-value controls were added. The older measurements below describe the original test-only change, not the fixed parser.

Final proof commands and output:

```text
$ go test ./...
ok  github.com/builtwithtofu/pristinecss  1.764s
ok  github.com/builtwithtofu/pristinecss/internal/lexer  0.236s
$ go test -race ./...
ok  github.com/builtwithtofu/pristinecss  53.177s
ok  github.com/builtwithtofu/pristinecss/internal/lexer  3.764s
$ go vet ./...
(no output; exit 0)
$ go test -run '^$' -fuzz '^FuzzParse$' -fuzztime=5m -parallel=1 .
fuzz: elapsed: 2s, gathering baseline coverage: 235/235 completed, now fuzzing with 1 workers
fuzz: elapsed: 5m1s, execs: 27386 (0/sec), new interesting: 5 (total: 240)
PASS
ok  github.com/builtwithtofu/pristinecss  301.690s
```

**Measured fuzz discovery:** a fuzz run reduced an instability to `{url("\n0` (one actual newline). CSS Syntax 3 tokenizes `url("` as a function followed by an ordinary string, so an unescaped newline ends the string and the rules after it survive. The input is kept as a native fuzz regression and a named corpus case.

**Measured legacy-hack comparison:** the damaged `milligram.css/open-string` and `water.css/open-string` cases are stable at `9941a02d` and unstable after `89f43f0c`. Both end in `input[type='`. The new raw-statement emitter appended `;` inside that open string, adding another on every reparse. Both are stable with the recovery fix. The comparison used the same seeded cut selection on the exact old revisions via a temporary Go test overlay; the overlay was removed afterwards. Reduced case: `input[type='`.

**Recovery choices:** EOF strings/URLs get explicit token terminators before parent punctuation; bad strings retain their newline boundary; malformed non-adjacent at-names are kept as qualified-rule/raw-statement data, not truncated at-rule names. Comment removal does not introduce descendant combinators or merge value tokens. These follow the relevant [CSS Syntax tokenization and recovery rules](https://drafts.csswg.org/css-syntax-3/), without claiming complete CSS conformance. Valid selectors and declarations remain structured. Raw declaration replacements share statement separator handling without rewriting their supplied bytes.

No timing benchmarks were run for the recovery fixes.

## Historical test-only baseline

## Measured outcome

Added test-time damaged versions of **all 22 sheets** (8 each), **54 standard-suite inputs**, five survival spot checks, and 13 reduced regression/control cases. **248 inputs total; 142 return ParseErrors; 84 fail emit stability.** No parser/emitter production files or existing tests were changed. Failures remain failing.

Measured by `go test -run '^TestErrorCorpus$' .`, and repeated in `go test ./...`:

| Input set | Inputs | With ParseErrors | Panic failures | Offset bounds failures | Offset order failures | Unstable emit | Lost trailing rule |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Damaged frameworks (13 sheets) | 104 | 66 | 0 | 0 | 0 | 28 | — |
| Damaged corpus (9 sheets) | 72 | 45 | 0 | 0 | 0 | 33 | — |
| css-parsing-tests | 41 | 14 | 0 | 0 | 0 | 5 | — |
| WPT CSS syntax | 13 | 11 | 0 | 0 | 0 | 5 | — |
| Handwritten survival cases | 5 | 1 | 0 | 0 | 0 | 2 | 0 / 5 |
| Reduced regressions and controls | 13 | 5 | 0 | 0 | 0 | 11 | — |
| **Total** | **248** | **142** | **0** | **0** | **0** | **84** | **0 / 5** |

Counts are per input, not unique bytes or individual diagnostics. “With ParseErrors” counts the **first parse only**. Bounds/order checks cover both original and emitted-source parses. Bounds include EOF: `0 <= Offset <= len(source)`. Panic checking covers Parse, Emit and reparse. Survival is only asserted on small cases with a known recovery boundary, not on arbitrary truncations where a string/comment/block can legitimately consume the rest of the sheet. Fuzz uses the same checks and recognizes those exact survival seeds.

Zero errors is not itself a test failure: standard inputs retain valid controls, standalone semicolons are ignored, and some truncations are handled without diagnostics. Every input is checked regardless. No out-of-order offsets were observed, so this run did not reproduce the slow restart path in position resolution.

## Measured reduced reproducers

Run `go test -run '^TestBrokenCSSRegressions$' -v .`. Every failing row below is kept in that test and also included in TestErrorCorpus, FuzzParse and the benchmarks. These are small isolated reproducers, not a claim of globally byte-minimal strings. Related rows may share a root cause; **11 reproducers does not mean 11 distinct root causes**.

**Expected for every row:** second emit equals first emit byte-for-byte. The first-emit column is an observation, not a golden expected serialization: a fix may change both emits as long as they stabilize. `\n` denotes one actual newline in the bad-string input.

| Case | Input | First emit (observed) | Second emit (observed) |
| --- | --- | --- | --- |
| Open string | `a{x:"` | `a{x:"}` | `a{x:"}}` |
| Open URL | `a{x:url(` | `a{x:url(}` | `a{x:url(}}` |
| Malformed at-name | `@ x;` | `@ ;` | empty |
| Dangling at-rule block contents | `@a{x` | `@a{x}` | `@a{x}}` |
| Missing colon | `a{x y;z:0}` | `a{x yz:0}` | `a{x:;yz:0}` |
| Function before stripped comment | `a{x:f() /*x*/;}` | `a{x:f() }` | `a{x:f()}` |
| Star property hack | `a{*x:y;z:0;w:0}` | `a{*x:yz:0;w:0}` | `a{*x:yz:0w:0}` |
| Hash property hack | `a{#x:y;z:0;w:0}` | `a{#x:yz:0;w:0}` | `a{#x:yz:0w:0}` |
| Bad string newline | `a{x:"a\n;y:0}b{}` | `a{x:"a;y:0}b{}` | `a{x:"a;y:0}b{}}` |
| Double colon at EOF | `a{x::y` | `a{x::y}` | `a{x::y}}` |
| Unclosed components | `@a{[(0` | `@a{[(0}` | `@a{[(0}}` |

Measured original error counts for these rows, in order: **1, 1, 0, 1, 0, 0, 0, 0, 0, 1, 2**. Returning no diagnostic on several malformed inputs is an additional observation, not a newly asserted complete CSS conformance requirement.

Two controls remain in the regression table: `a{x:var(--x, "/") }` is byte-stable; `a{x::y}` is also byte-stable but emits `a{x::y}}`, whose reparse reports an unexpected closing brace. The brief's stability property alone does not reject that latter output. No expected-output/conformance oracle was added.

The bootstrap failures reduce to the stripped-comment whitespace case, not necessarily the injected damage. The star/hash cases overlap the parent corpus's existing legacy-hack failures. This is **inferred grouping**, not a claim that every failed large sheet was delta-debugged to a unique cause. All large failures remain independently reproducible by their TestErrorCorpus subtest names.

## Full-suite proof

`go test ./...` exited **1**. It ran the new checks and all fuzz seeds as ordinary tests. The existing TestRoundTripFrameworks failures on Dijit, Pure and YUI are still present, as documented in `../corpus/RESULTS.md`.

Final run, with repeated per-input diagnostics omitted from the output below (the command was piped through a read-only output filter, with `set -o pipefail` to preserve the test exit status):

```text
--- FAIL: TestErrorCorpus (0.25s)
    error_corpus_test.go:241: set | inputs | with errors | panic | bounds | order | unstable | lost rule
    error_corpus_test.go:244: damaged-frameworks | 104 | 66 | 0 | 0 | 0 | 28 | 0
    error_corpus_test.go:244: damaged-corpus | 72 | 45 | 0 | 0 | 0 | 33 | 0
    error_corpus_test.go:244: css-parsing-tests | 41 | 14 | 0 | 0 | 0 | 5 | 0
    error_corpus_test.go:244: wpt | 13 | 11 | 0 | 0 | 0 | 5 | 0
    error_corpus_test.go:244: handwritten | 5 | 1 | 0 | 0 | 0 | 2 | 0
    error_corpus_test.go:244: regressions | 13 | 5 | 0 | 0 | 0 | 11 | 0
--- FAIL: TestBrokenCSSRegressions (0.00s)
--- FAIL: TestRoundTripFrameworks (0.07s)
--- FAIL: FuzzParse (0.26s)
FAIL
FAIL    github.com/builtwithtofu/pristinecss  1.106s
ok      github.com/builtwithtofu/pristinecss/internal/lexer (cached)
FAIL
```

Representative full diagnostic from the preceding unfiltered `go test ./...` run:

```text
--- FAIL: TestBrokenCSSRegressions/open-string (0.00s)
    error_regression_test.go:33: input="a{x:\"" errors=[{expected '}' to close block, found EOF 5 0 5}]; first="a{x:\"}" reparse_errors=[{expected '}' to close block, found EOF 6 0 6}]; second="a{x:\"}}"
    error_regression_test.go:35: [unstable emit at byte 6: first="a{x:\"}" second="a{x:\"}}"]
```

## Native fuzz attempt — blocked before mutation

Command (one worker to respect the shared machine):

```sh
go test -run '^$' -fuzz '^FuzzParse$' -fuzztime=5m -parallel=1 .
```

Final output, exit **1**:

```text
fuzz: elapsed: 0s, gathering baseline coverage: 0/227 completed
failure while testing seed corpus entry: FuzzParse/seed#1
fuzz: elapsed: 0s, gathering baseline coverage: 1/227 completed
--- FAIL: FuzzParse (0.35s)
    --- FAIL: FuzzParse (0.00s)
        error_corpus_test.go:261: unstable emit at byte 98345: first="(--bs-breadcrumb-divider, \"/\") }.breadcrumb-item.active{color:var(--bs-breadcrumb-item-active-color)}.paginati" second="(--bs-breadcrumb-divider, \"/\")}.breadcrumb-item.active{color:var(--bs-breadcrumb-item-active-color)}.paginatio"

FAIL
exit status 1
FAIL    github.com/builtwithtofu/pristinecss  0.587s
```

227 is Go's deduplicated baseline count; 248 cases were added as seeds. The first failing seed is `damaged-frameworks/bootstrap.css/open-string`. Its first mismatch is the existing comment/whitespace issue, reduced above to `a{x:f() /*x*/;}`. An earlier attempt before adding reduced cases likewise stopped on this seed (214 deduplicated entries then).

**No five-minute mutation run completed and no mutation discoveries are claimed.** Native Go fuzzing rejects failing baseline seeds before mutation. Removing those seeds, ignoring the failing property, or fixing the parser would contradict the brief. Rerun the same command after the production fixes land. Ordinary `go test ./...` does execute all 248 seed cases and reports their failures.

## Determinism and benchmark proof

```text
$ go test -run '^TestErrorCorpusDeterministic$' -count=2 .
ok      github.com/builtwithtofu/pristinecss  0.245s
```

This compares names and exact bytes of every generated/input case across repeated generations. `damageSheet` clones buffers before insertion; third-party files remain unchanged in the working-copy diff.

New benchmarks live in `error_bench_test.go`; `bench_test.go` was not edited. Parse only, no Emit, no ParseInto token-cache shortcut. All 248 benchmark subcases were smoke-run once:

```sh
go test -run '^$' -bench '^BenchmarkParseBroken$' -benchtime=1x -count=1 .
```

Output excerpts (exit **0**):

```text
goos: darwin
goarch: arm64
pkg: github.com/builtwithtofu/pristinecss
cpu: Apple M5
BenchmarkParseBroken/damaged-frameworks/bootstrap.css/open-block-10  1  408583 ns/op  213.25 MB/s  624536 B/op  9 allocs/op
BenchmarkParseBroken/damaged-frameworks/daisyui.css/stray-brace-10    1  4225333 ns/op  257.62 MB/s  9152408 B/op  9 allocs/op
BenchmarkParseBroken/regressions/unclosed-components-10             1  2916 ns/op  2.06 MB/s  2408 B/op  11 allocs/op
PASS
ok      github.com/builtwithtofu/pristinecss  0.219s
```

These single-iteration measurements only prove the benchmarks run; they are not a performance comparison or reliable throughput estimate. No long benchmark comparison was run.

## License evidence

Every imported file, extraction recipe, pinned upstream URL, verified repository LICENSE and local complete notice is listed in [SOURCES.md](SOURCES.md). css-parsing-tests: CC0/public-domain dedication. WPT: BSD-3-Clause. No non-permissive fixture was imported. No network is needed to run the tests.

## Deviations

- The requested five-minute fuzz run cannot proceed past failing baseline seeds with all required checks intact; it was attempted and stopped by Go, not skipped or reported as passing.
- Imported a focused syntax/recovery subset of both standard suites, with valid controls retained; no claim of whole-suite conformance. Inputs are extracted JSON arrays rather than entire HTML/JS harnesses. Expected-output comparison requirements are documented in SOURCES.md.
- Where a sheet lacks a requested string/comment/URL construct, its variant gets an explicitly named synthetic open tail. Existing constructs are truncated at seeded token boundaries.
