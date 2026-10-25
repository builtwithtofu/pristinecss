# Research: the lowerings pristinecss could do

Status: research for discussion. Nothing here is built yet.

## The goal

A lowering rewrites CSS so that a browser without a feature still renders the
page. pristinecss will lower per browser, from what that browser reports it
supports (see [binary-style.md](binary-style.md)).

The aim is that the page **looks the same** in every browser, as far as CSS
allows. A replacement that merely parses is not enough. So every row below says
how close the lowered page gets, not just whether a rewrite exists.

The list is not limited to what PostCSS or Lightning CSS do. It includes future
CSS that few or no browsers ship yet. Lowering is CSS-only for now. Where the
same look needs JavaScript, the row says **needs JS shim** and what the shim
would do. Shims are a possible future direction, not part of this work.

### Fidelity

- **identical**: the old browser renders the same pixels and the same cascade.
- **close**: the same look in normal use; differences at the edges (rounding,
  rule order, rare states).
- **approximate**: looks right in common cases; a named visible difference.
- **needs JS shim**: CSS alone can't get the look; a script could.
- **not possible**: neither CSS nor a reasonable shim gets the look. Report it.

A row can have two grades, for example "identical for constants, not possible
for `var()`".

### What a lowering needs

- **decl**: only the one declaration.
- **rule**: the whole rule (other declarations, or the selector).
- **sheet**: the whole sheet (reordering, selector rewriting, definitions
  elsewhere).
- **decoded**: decoded values, such as colour maths or constant folding. This is
  step 4 in binary-style.md.
- **HTML**: knowledge of the page's markup, which a stylesheet alone doesn't
  have.

### Evidence tags

- **[S]** from `support.json` (web-features 3.44.0 and @mdn/browser-compat-data
  8.1.7). All browser versions and Baseline cells come from here. Nothing else.
- **[M]** measured: a researcher ran the tool, or I re-ran it for this doc.
- **[D]** read in a tool's docs or a spec.
- **[I]** inferred: reasoning from the spec or from tool output.
- **[G]** from memory, not checked.

**No fidelity grade has been checked in a real browser.** Every grade is [I]
unless a row says otherwise. Rendering old and new browsers side by side is the
main thing still to do.

### Reading the support columns

`Chr / Edge / FF / Saf / IE` is the first version with full, unprefixed support
for that BCD key [S]. A prefixed or older-named form is in brackets, for example
`9 (-webkit- 7)`. A dash means no full support is recorded. iOS Safari is in
`support.json` too; it tracks desktop Safari closely and is left out of the
tables to keep them readable. Baseline is the web-features status: "high 2025
(low 2023)" means widely available in 2025, newly available in 2023. "no" means
not Baseline.

Each cell is the `compat[].browsers.<browser>.first_full` field for the named
BCD key, read from `support.json` with a short node script.

### Tools

- **LC** Lightning CSS 1.33.0. **PPE** postcss-preset-env 11.6.1. **ESB**
  esbuild 0.28.2. **AP** Autoprefixer 10.6.1. All four were run on 75 test
  cases by the haiku researcher; outputs were in `/tmp/haiku-scratch/` [M].
- Docs: [Lightning CSS transpilation](https://lightningcss.dev/transpilation.html),
  [preset-env features](https://preset-env.cssdb.org/features/),
  [cssdb](https://cssdb.org/), [esbuild CSS](https://esbuild.github.io/content-types/),
  [Autoprefixer](https://github.com/postcss/autoprefixer).

### Detecting support

`CSS.supports()` exists from Chrome 28, Edge 12, Firefox 22, Safari 9, and never
in IE (haiku.md §10, BCD `api.CSS.supports_static`). So IE needs a fixed table.
At-rules (`@layer`, `@container`, `@scope`, `@property`, media range syntax)
have no `CSS.supports` test; detect them with JS (`matchMedia`, `'registerProperty' in CSS`)
or a table [I]. `CSS.supports` says a value parses, not that it behaves right [D, luna.md].

## Colours

| Feature | Example → lowered | Chr / Edge / FF / Saf / IE | Baseline | Fidelity | Closest technique | Needs | Done by today | Detect |
|---|---|---|---|---|---|---|---|---|
| Hex with alpha | `#ff000080` → `rgba(255,0,0,.5)` | 62 / 79 / 49 / 10 / – | high 2022 (low 2020) | identical where `rgba()` exists; approximate in IE 6–8 | rgba(); for IE 6–8 a solid colour pre-blended on an assumed background [I] | decl, decoded | LC, PPE, ESB [M] | `CSS.supports('color','#ff000080')` |
| Space-separated `rgb()`/`hsl()`, slash alpha | `rgb(255 0 0 / 50%)` → `rgba(255,0,0,.5)` | 65 / 79 / 52 / 12.1 / – | high 2022 (low 2020) | identical | comma syntax | decl | LC, PPE [M] | `CSS.supports('color','rgb(255 0 0 / 50%)')` |
| `hwb()` | `hwb(120 0% 0%)` → `#0f0` | 101 / 101 / 96 / 15 / – | high 2024 (low 2022) | identical to 8-bit rounding; not possible with `var()` | convert to sRGB | decl, decoded | LC, PPE, ESB [M] | `CSS.supports('color','hwb(120 0% 0%)')` |
| `lab()` / `lch()` | `lab(50% 40 59.5)` → `#bf5700` then the original | 111 / 111 / 113 / 15 / – | high 2025 (low 2023) | close in sRGB gamut; approximate outside it (clipped) | sRGB fallback, original kept after it | decl, decoded | LC, PPE (adds a P3 branch), ESB [M] | `CSS.supports('color','lab(50% 40 59.5)')` |
| `oklab()` / `oklch()` | `oklch(70% .1 200)` → `#40b1b7`, then `lab(…)`, then the original | 111 / 111 / 113 / 15.4 / – | high 2025 (low 2023) | as lab | sRGB, then lab() for Safari 15–15.3 | decl, decoded | LC, PPE, ESB [M] | `CSS.supports('color','oklch(70% .1 200)')` |
| `color()` (display-p3 etc.) | `color(display-p3 1 0 0)` → `#ff0f0e` | 111 / 111 / 113 / 15 / – | high 2025 (low 2023) | close in sRGB; approximate outside (tools disagree, see bugs) | gamut-map to sRGB | decl, decoded | LC, PPE, ESB [M] | `CSS.supports('color','color(display-p3 1 0 0)')` |
| `color-mix()` | `color-mix(in srgb, red 50%, blue)` → `#800080` | 111 / 111 / 113 / 16.2 / – | high 2025 (low 2023) | identical for constants; not possible with `var()`, `currentColor` | evaluate at build time | decl, decoded | LC, PPE [M]; ESB keeps [M] | `CSS.supports('color','color-mix(in srgb, red, blue)')` |
| Relative colour | `rgb(from red r g b / .5)` → `rgba(255,0,0,.5)` | 122 / 122 / 128 / 18 / – | low 2024 | identical for constants; not possible with runtime sources | evaluate at build time | decl, decoded | LC, PPE [M] | `CSS.supports('color','rgb(from red r g b)')` [I] |
| `alpha()` | `alpha(from red / .5)` → `rgba(255,0,0,.5)` [I] | 152 / 152 / 155 / 27 / – | low 2026 | identical for constants [I] | evaluate | decl, decoded | not measured | `CSS.supports('color','alpha(from red / .5)')` [I] |
| `light-dark()` | `light-dark(#000,#fff)` → custom-property toggles set by `prefers-color-scheme` and `color-scheme` | 123 / 123 / 120 / 17.5 / – | low 2024 | close where custom properties exist (49/15/31/9.1) and the sheet sets `color-scheme`; approximate otherwise (OS preference only, or light only) | PPE's toggle variables | sheet | PPE (partial) [M]; LC output is broken (see bugs) | `CSS.supports('color','light-dark(#000,#fff)')` |
| `light-dark()` for images | `light-dark(url(a), url(b))` | 150 / 150 / 150 / 27 / – | low 2026 | as above [I] | as above | sheet | none found | none tested |
| `currentColor` | `border-color:currentColor` | 1 / 12 / 1.5 / 4 / 9 | high 2018 (low 2015) | close for borders in IE 6–8 by leaving `border-color` unset (its initial value is the text colour) [I]; not possible in general | omit, or copy `color` | rule, sometimes sheet | none [M] | `CSS.supports('color','currentColor')` |
| `transparent` | keep the keyword | `css.types.color.transparent`: 1 / 12 / 1 / 1 / 3; `named-color.transparent`: IE 9 | high 2018 (low 2015) | identical: no lowering needed for `background`/`border` | leave it alone | – | LC rewrites to `rgba(0,0,0,0)` even for IE 6 (see bugs) | `CSS.supports('color','transparent')` |
| System colours | `ButtonText` → fixed hex | 1 / 12 / 1 / 1 / 3 | high 2018 (low 2015) | approximate: loses OS theme and forced colours | don't lower; keep the keyword | – | none [M] | `CSS.supports('color','ButtonText')` |
| `rebeccapurple` | `rebeccapurple` → `#663399` | 38 / 12 / 33 / 9 / 11 | high 2018 (low 2015) | identical | hex | decl | PPE plugin exists [D] | `CSS.supports('color','rebeccapurple')` |
| `contrast-color()` | see Future CSS | | | | | | | |

## Selectors

| Feature | Example → lowered | Chr / Edge / FF / Saf / IE | Baseline | Fidelity | Closest technique | Needs | Done by today | Detect |
|---|---|---|---|---|---|---|---|---|
| Nesting, `&` | `.a{color:red;& .b{color:blue}}` → `.a{color:red}.a .b{color:blue}` | 120 / 120 / 117 / 17.2 / – | high 2026 (low 2023) | identical for flat nesting; close where flattening changes rule order | flatten; nested `@media` bubbles out | sheet | LC, PPE, ESB [M] | `CSS.supports('selector(&)')` [I] |
| Nesting, declarations after nested rules | `.a{& .b{…} color:red}` | 130 / 130 / 132 / 18.2 / – (`CSSNestedDeclarations`) | as above | identical [I] | flatten for browsers between the two rows | sheet | as above | by version table [I] |
| `:is()` | `:is(.a,.b) .c` → `.a .c,.b .c` | 88 / 88 / 78 / 14 / – (`:-webkit-any()` Chr 12; `:-moz-any()` FF 4; `:matches()` Saf 9) | high 2023 (low 2021) | close: same matches, specificity can differ | expand the list; for Safari 9–13 use `:matches()`, which takes combinators | rule | PPE expands; LC uses `-webkit-any`/`-moz-any`, and keeps `:is()` for Safari 10 [M] | `CSS.supports('selector(:is(a))')` |
| `:where()` | `:where(.a,.b) p` → `.a p,.b p` | 88 / 88 / 78 / 14 / – | high 2023 (low 2021) | approximate: the expansion gains specificity; `:where()` has none | expand, then check whether specificity changes any winner in the sheet [I] | sheet | none [M] | `CSS.supports('selector(:where(a))')` |
| `:not()` with a list | `.a:not(.b,.c)` → `.a:not(.b):not(.c)` | 88 / 88 / 84 / 9 / – (single argument: 1 / 12 / 1 / 3.1 / 9) | high 2023 (low 2021) | identical matches; specificity is higher [I] | chain single `:not()`s | rule | PPE [M]; LC emits `:not(:is(…))`, invalid for IE [M] | `CSS.supports('selector(:not(a,b))')` |
| `:has()` | `.card:has(img)` | 105 / 105 / 121 / 15.4 / – | high 2026 (low 2023) | needs JS shim | see JS shim section | HTML | PPE has a JS-assisted plugin (css-has-pseudo) [D] | `CSS.supports('selector(:has(a))')` |
| `:focus-visible` | `:focus-visible{…}` → `:focus{…}` | 86 / 86 / 85 (`:-moz-focusring` 4) / 15.4 / – | high 2024 (low 2022) | approximate in CSS (ring also shows on click); close with a JS shim | `:focus`, or `:-moz-focusring` in old Firefox | rule | none [M] | `CSS.supports('selector(:focus-visible)')` |
| `:focus-within` | `.f:focus-within` | 60 / 79 / 52 / 10.1 / – | high 2022 (low 2020) | needs JS shim | see JS shim section | HTML | PPE plugin (JS) [D] | `CSS.supports('selector(:focus-within)')` |
| `:any-link` | `.a:any-link` → `.a:link,.a:visited` | 65 / 79 / 50 / 9 / – (`-webkit-` Chr 1, Saf 3; `-moz-` FF 1) | high 2022 (low 2020) | identical | `:link,:visited` | rule | PPE; LC prefixes only [M] | `CSS.supports('selector(:any-link)')` |
| `:dir()` | `.a:dir(rtl)` → `[dir="rtl"] .a` | 120 / 120 / 49 / 16.4 / – | high 2026 (low 2023) | approximate: sees only an explicit `dir` attribute, not `dir=auto` or CSS `direction` | `[dir]` attribute selectors | rule | PPE `[dir]`; LC language list wrapped in `:is()`, invalid for IE [M] | `CSS.supports('selector(:dir(rtl))')` |
| `:lang()` with a list | `:lang(en,fr)` → `:lang(en),:lang(fr)` | – / – / 114 / 9 / – (single argument: 1 / 12 / 1 / 3.1 / 8) | high 2018 (low 2015), for the single form | identical | split into a list | rule | LC uses `:is()` [M]; PPE leaves it [M] | `CSS.supports('selector(:lang(en,fr))')` |
| `::marker` | `li::marker{color:red}` | 86 / 86 / 68 / – (11.1 partial: `color` and `font-size` only) / – | no | approximate: unstyled default marker | `list-style-type`; or `li::before` with `counter()` [I] | rule, HTML for the `::before` form | none [M] | `CSS.supports('selector(::marker)')` |
| `::placeholder` | → `::-webkit-input-placeholder`, `::-moz-placeholder`, `:-ms-input-placeholder` | 57 / 79 / 51 / 10.1 / – (prefixed Chr 6, FF 19, Saf 5) | high 2022 (low 2020) | identical per engine | one rule per prefix (a bad prefix drops the whole rule) | rule | AP, LC [M] | `CSS.supports('selector(::placeholder)')` |
| `:placeholder-shown` | → `:-ms-input-placeholder` (IE 10) | 47 / 79 / 51 / 9 / – | high 2022 (low 2020) | close [I] | prefix | rule | not measured | `CSS.supports('selector(:placeholder-shown)')` |
| `[a="x" i]` | `[type="a" i]` → `[type="a"],[type="A"]` | 49 / 79 / 47 / 9 / – | high 2022 (low 2020) | identical for short ASCII values; not possible for long ones (2ⁿ forms) | enumerate case forms | rule | PPE [M] | `CSS.supports('selector([a="x" i])')` |
| `::before` and friends, double colon | `p::before` → `p:before` | 1 / 12 / 1.5 / 4 / 9 (`:before` IE 8) | high 2018 (low 2015) | identical | single-colon form for IE 8 | rule | not measured | by version table |
| `:nth-child(… of S)` | see Future CSS | 111 / 111 / 113 / 9 / – | high 2025 (low 2023) | | | | | |
| `:user-valid`, `:user-invalid` | `:user-invalid` → `:-moz-ui-invalid` | 119 / 119 / 88 (`:-moz-ui-valid` 4) / 16.5 / – | high 2026 (low 2023) | close in old Firefox; approximate elsewhere (`:invalid` shows before the user types) | prefix, or `:invalid` | rule | not measured | `CSS.supports('selector(:user-invalid)')` |
| `:fullscreen` | → `:-webkit-full-screen`, `:-ms-fullscreen` | 71 / 12 / 64 / 16.4 / – (prefixed Chr 15, FF 9, Saf 6, IE 11) | no | identical per engine [I] | prefixes, one rule each | rule | AP [G] | `CSS.supports('selector(:fullscreen)')` |
| `:autofill` | → `:-webkit-autofill` | 110 / 110 / – (`-webkit-` 86) / 15 / – | no | identical per engine [I] | prefix | rule | not measured | `CSS.supports('selector(:autofill)')` |

## Values and units

| Feature | Example → lowered | Chr / Edge / FF / Saf / IE | Baseline | Fidelity | Closest technique | Needs | Done by today | Detect |
|---|---|---|---|---|---|---|---|---|
| `calc()` | `calc(1px + 2px)` → `3px` | 26 / 12 / 16 / 7 / 9 (`-webkit-` Chr 19, Saf 6; `-moz-` FF 4) | high 2018 (low 2015) | identical for constants; approximate for mixed `%`/`px` in IE ≤ 8 (one fixed value) | fold constants; prefix for old WebKit | decl, decoded | LC folds constants; nobody lowers mixed units [M] | `CSS.supports('width','calc(1px + 2%)')` |
| `min()` / `max()` / `clamp()` | `clamp(1rem, 2vw, 3rem)` → a fixed value plus `@media` steps | min/max: 79 / 79 / 75 / 11.1 / –; clamp: 79 / 79 / 75 / 13.1 / – | high 2023 (low 2020) | identical for constants; close with many `@media` steps (stepped, not smooth); approximate with one value | fold, or a viewport ladder | decl, decoded; sheet for the ladder | LC folds constants; PPE turns clamp into `max(min())` [M] | `CSS.supports('width','clamp(1px,2px,3px)')` |
| `round()` / `mod()` / `rem()` | `round(10px,5px)` → `10px` | 125 / 125 / 118 / 15.4 / – | low 2024 | identical for constants; not possible with `var()` | fold | decl, decoded | LC, PPE [M] | `CSS.supports('width','round(10px,5px)')` |
| Trig (`sin()` …) | `sin(30deg)` → `.5` | 111 / 111 / 108 / 15.4 / – | high 2025 (low 2023) | identical for constants | fold | decl, decoded | LC, PPE [M] | `CSS.supports('width','calc(1px * sin(30deg))')` |
| `pow()` / `sqrt()` / `log()` / `exp()` | `pow(2,3)` → `8` | 120 / 120 / 118 / 15.4 / – | high 2026 (low 2023) | identical for constants | fold | decl, decoded | LC [D]; PPE plugin [D] | `CSS.supports('width','calc(1px * pow(2,3))')` [I] |
| `abs()` / `sign()` | `abs(-2px)` → `2px` | 138 / 138 / 118 / 15.4 / – | low 2025 | identical for constants | fold | decl, decoded | PPE plugin [D] | `CSS.supports('width','abs(-1px)')` [I] |
| `vw` / `vh` / `vmin` / `vmax` | keep, or a px value per `@media` step | 20 / 12 / 19 / 6 / 9 | high 2020 (low 2017) | approximate (close with a ladder) | `@media` ladder | sheet | none [M] | `CSS.supports('width','1vw')` |
| `dvh` / `svh` / `lvh`, `vi`/`vb` | `100dvh` → `100vh` then `100dvh` | 108 / 108 / 101 / 15.4 / – | high 2025 (low 2022) | identical on desktop; approximate on mobile (the toolbar changes `vh`) | `vh` fallback first | decl | none [M] | `CSS.supports('height','100dvh')` |
| `rem` | `1rem` → `16px` | 4 / 12 / 3.6 / 5 / 9 | high 2018 (low 2015) | identical if the root font size is set in px in the sheet and the user hasn't changed theirs; approximate otherwise (ignores user font size) | px fallback | sheet, decoded | not measured | `CSS.supports('width','1rem')` |
| `ch`, `ic` | `20ch` → `10em` | ch: 27 / 12 / 1 / 7 / 9; ic: 106 / 106 / 97 / 15.4 / – | ch high 2018; ic high 2025 | approximate: depends on the font | em ratio | decl | none [M] | `CSS.supports('width','1ch')` |
| `lh` / `rlh` | `2lh` → `calc(2 * 1.5em)` when `line-height:1.5` is known | lh: 109 / 109 / 120 / 16.4 / –; rlh: 111 / 111 / 120 / 16.4 / – | high 2026 (low 2023) | identical when line-height is known for that element; approximate otherwise | multiply out | rule or sheet, decoded | none [M] | `CSS.supports('height','1lh')` |
| `cap`, `rcap`, `rex`, `ric` | `1cap` → `.7em` | cap: 118 / 118 / 97 / 17.2 / – | cap high 2026; others low 2026 | approximate: font metrics | em ratio for a known font [I] | decl, font metrics | none found | `CSS.supports('height','1cap')` |
| Container units (`cqw` …) | `50cqi` | 105 / 105 / 110 / 16 / – | high 2025 (low 2023) | approximate with `vw` (exact only for full-width containers); needs JS shim for the real thing | `vw` fallback | decl | none [M] | `CSS.supports('width','1cqw')` |
| `image-set()` | `image-set(url(a.png) 1x, url(b.png) 2x)` → `url(a.png)`, then `-webkit-image-set(…)`, then the original | 113 / 113 / 89 / 17 / – (`-webkit-` Chr 113 [S, as recorded], FF 90, Saf 17) | high 2026 (low 2023) | close: with a `min-resolution` `@media` branch, 2x screens get the 2x file too | 1x fallback plus a resolution `@media` | rule, sheet | PPE (with `@media`), LC and AP (prefix) [M] | `CSS.supports('background-image','image-set(url(a.png) 1x)')` |
| Gradients, prefixed and old syntax | `linear-gradient(to right,…)` → `-webkit-linear-gradient(left,…)` | 26 / 12 / 16 / 7 / 10 (`-webkit-` Chr 10, Saf 5.1) | high 2018 (low 2015) | identical in old WebKit (direction keywords invert) [G]; IE 6–9: approximate with `filter: progid:DXImageTransform.Microsoft.gradient` (two stops only) [G] | prefix with old direction syntax | decl, decoded | AP [D] | `CSS.supports('background-image','linear-gradient(red,blue)')` |
| Double-position colour stops | `linear-gradient(red 10% 20%, blue)` → `red 10%, red 20%` | 71 / 79 / 64 / 12.1 / – | high 2018 (low 2015) | identical | repeat the stop | decl | LC, PPE [M] | `CSS.supports('background-image','linear-gradient(red 10% 20%, blue)')` |
| `conic-gradient()` | `conic-gradient(red, blue)` | 69 / 79 / 83 / 12.1 / – | high 2023 (low 2020) | approximate with linear slices; an image is identical but not CSS-only | first colour as solid fallback, or slices | decoded | none [M] | `CSS.supports('background-image','conic-gradient(red,blue)')` |
| Gradient interpolation (`in oklab`) | `linear-gradient(in oklab, red, blue)` → extra sRGB stops | 111 / 111 / 127 / 16.2 / – | low 2024 | close with enough extra stops [I] | sample the curve into sRGB stops | decl, decoded | none [M] | `CSS.supports('background-image','linear-gradient(in oklab, red, blue)')` [I] |
| Logical keywords | `text-align:start` → `left`, plus `[dir=rtl]` → `right` | `text-align:start`: 1 / 79 / 1 / 3.1 / –; `float:inline-start`: 118 / 118 / 55 / 15 / – | high 2024 (low 2021) | close with a `[dir=rtl]` pair; approximate with a language list | LTR rule plus RTL rule | sheet | LC (language list); PPE LTR only, wrong in RTL [M] | `CSS.supports('float','inline-start')` |
| `env()` | `env(safe-area-inset-top)` → `0` fallback | 69 / 79 / 65 / 11.1 / – | high 2022 (low 2020) | identical on devices with no notch; approximate on notched phones | constant fallback before | decl | none [M] | `CSS.supports('top','env(safe-area-inset-top)')` |
| `var()` with values known at build time | `--gap:8px; margin:var(--gap)` → `margin:8px` | 49 / 15 / 31 / 9.1 / – | high 2019 (low 2017) | identical when every value is static and the cascade is resolvable; not possible when JS or user input changes it | substitute at build time | sheet, HTML when the property is set on different elements | PPE `custom-properties` [D]; not measured | `CSS.supports('color','var(--x)')` |
| `system-ui` font | `font-family:system-ui` → `system-ui,-apple-system,"Segoe UI",Roboto,…` | 56 / 79 / 92 / 11 / – (`-apple-system` FF 43, Saf 9) | high 2024 (low 2021) | close: right font on each OS [I] | font stack | decl | LC [D] | `CSS.supports('font-family','system-ui')` |

## Properties

| Feature | Example → lowered | Chr / Edge / FF / Saf / IE | Baseline | Fidelity | Closest technique | Needs | Done by today | Detect |
|---|---|---|---|---|---|---|---|---|
| `inset` | `inset:0` → `top:0;right:0;bottom:0;left:0` | 87 / 87 / 66 / 14.1 / – | high 2024 (low 2021) | identical | expand | decl | LC, PPE, ESB [M] | `CSS.supports('inset','0')` |
| Logical box properties | `margin-inline:1px 2px` → LTR `margin-left:1px;margin-right:2px`, plus a `[dir=rtl]` rule with them swapped | shorthands 87 / 87 / 66 / 14.1 / –; longhands `margin-inline-start` 69 / 79 / 41 / 12.1 / – (`-webkit-margin-start` Chr 2, Saf 3; `-moz-` FF 3) | high 2024 (low 2021) | close for horizontal writing with a `[dir]` pair; approximate in vertical writing modes | LTR/RTL pair; old prefixed longhands | sheet | LC (language list in `:is()`; IE drops the rule) [M]; PPE LTR only [M] | `CSS.supports('margin-inline','1px')` |
| Logical radius, size, border | `border-start-start-radius`, `inline-size` → physical | radius: 89 / 89 / 66 / 15 / – | high 2024 (low 2021) | as above | as above | sheet | LC [D] | `CSS.supports('inline-size','1px')` |
| `gap` in flexbox | `display:flex;gap:1rem` → margins on children | 84 / 84 / 63 / 14.1 / – | high 2023 (low 2021) | approximate: wrapping and outer edges differ; negative-margin wrapper is closer but needs HTML | `> * + *` margins | rule | none [M] | `CSS.supports('gap','1px')` is true in grid-only browsers, so it can't tell; use a version table [D, luna.md] |
| `gap` in grid | `gap:10px` → `grid-gap:10px` | 66 / 16 / 61 / 12 / – (`grid-gap` Chr 57, FF 52, Saf 10.1) | high 2020 (low 2017) | identical | old `grid-gap` name; IE gutter tracks (see Layout) | decl | AP [M] | by version table |
| `place-*` | `place-items:center` → `align-items:center;justify-items:center` | items/self: 59 / 79 / 45 / 11 / –; content: 59 / 79 / 45 / 9 / – | high 2018 (low 2015) | identical | expand | decl | LC, PPE [M] | `CSS.supports('place-items','center')` |
| `aspect-ratio` | `aspect-ratio:16/9` → `padding-top:56.25%` on a pseudo-element | 88 / 88 / 89 / 15 / – | high 2024 (low 2021) | close for empty boxes and media with set width; approximate when content can grow | `::before{padding-top:…;float:left}` plus a clearfix [G] | rule | none [M] | `CSS.supports('aspect-ratio','16 / 9')` |
| `text-decoration` with colour/style/thickness | `underline dotted red` → longhands, plus `-webkit-text-decoration` | colour+style in shorthand: 57 / 79 / 6 / 26.2 / – (`-webkit-` Saf 8); thickness: 87 / 87 / 70 / 26.2 / – | high 2018 (low 2015) | identical with the prefix in Safari 8–26.1; approximate before (plain underline in text colour) | `-webkit-` prefix; plain line first | decl | PPE adds a plain fallback [M]; LC docs say it expands [D] | `CSS.supports('text-decoration','underline dotted red')` |
| `overflow: clip` | → `overflow:hidden` | 90 / 90 / 81 / 16 / – (`-moz-hidden-unscrollable` FF 1.5) | high 2025 (low 2022) | approximate: `hidden` makes a scroll container, which breaks `position:sticky` children | `hidden` | decl | none [M] | `CSS.supports('overflow','clip')` |
| Two-value `overflow` | `overflow:hidden scroll` → `overflow-x`/`overflow-y` | 68 / 79 / 61 / 13.1 / – | high 2022 (low 2020) | identical | expand | decl | LC [D], PPE plugin [D] | `CSS.supports('overflow','hidden scroll')` |
| `translate` / `rotate` / `scale` | `translate:10px 20px` → `transform:translate(10px,20px)` | 104 / 104 / 72 / 14.1 / – | high 2025 (low 2022) | identical for static values if spec order is kept (translate, rotate, scale, then the `transform` list); approximate when they are animated separately | merge into `transform` in spec order | rule; sheet if `transform` is set by another rule on the same element | LC merges in the wrong order and drops values (see bugs) [M] | `CSS.supports('translate','1px')` |
| `transform` (2D/3D) prefixes | `transform:rotate(5deg)` → `-webkit-`, `-ms-transform` | 36 / 12 / 16 / 9 / 10 (`-webkit-` Saf 3.1; `-ms-` IE 9 [G]) | high 2018 (low 2015) | identical per engine; IE 6–8 approximate (`filter: progid:…Matrix` [G]) | prefix | decl | AP, LC [M] | `CSS.supports('transform','rotate(1deg)')` |
| `opacity` | `opacity:.5` → plus `filter:alpha(opacity=50)` for IE 6–8 [G] | 1 / 12 / 1 / 2 / 9 | high 2018 (low 2015) | close in IE 6–8 (the element needs layout, `zoom:1`) [G] | IE filter | decl | not measured | `CSS.supports('opacity','.5')` |
| Opacity as a percentage | `opacity:50%` → `.5` | 78 / 79 / 70 / 13.1 / – | high 2018 (low 2015) | identical | number | decl | LC, PPE [M] | `CSS.supports('opacity','50%')` |
| `clip-path` | `clip-path:circle(50%)` → `-webkit-clip-path` | 55 / 79 / 3.5 / 9.1 / – (`-webkit-` Chr 23, Saf 7); shapes: Chr 23, FF 54 | high 2023 (low 2021) | identical with the prefix; approximate with `border-radius` for circles; not possible for `polygon()` without clip-path | prefix; `border-radius` | decl | LC, AP [M] | `CSS.supports('clip-path','circle(50%)')` |
| `mask` | `mask:url(m.svg)` → `-webkit-mask` first | 120 / 12 / 53 / 15.4 / – (`-webkit-` Chr 1, Saf 3.1) | high 2026 (low 2023) | identical with the prefix; not possible where masks don't exist (shows unmasked) | prefix | decl | LC, AP, ESB [M] | `CSS.supports('mask-image','url(m.svg)')` |
| `backdrop-filter` | `backdrop-filter:blur(8px)` → `-webkit-backdrop-filter` first | 76 / 79 / 103 / 18 / – (`-webkit-` Edge 17, Saf 9) | low 2024 | identical with the prefix; not possible without it; a semi-opaque background is the usual approximation | prefix; author-supplied fallback background | decl | LC, AP [M] | `CSS.supports('backdrop-filter','blur(2px)')` |
| `user-select` | `user-select:none` → `-webkit-` first (and `-moz-`, `-ms-` for old targets) | 54 / 79 / 69 / preview / – (`-webkit-` Saf 3, still needed; `-moz-` FF 1; `-ms-` IE 10) | no | identical per engine | prefixes | decl | LC, AP, ESB [M] | `CSS.supports('-webkit-user-select','none')` |
| `appearance` | `appearance:none` → `-webkit-`, `-moz-` | 84 / 84 / 80 / 15.4 / – (`-webkit-` Chr 1, FF 64, Saf 3) | high 2024 (low 2022) | identical per engine; native controls still differ by OS | prefixes | decl | AP; LC adds `-ms-appearance`, which no IE had (see bugs) [M] | `CSS.supports('appearance','none')` |
| `background-clip: text` | → `-webkit-background-clip:text` | 120 / 15 / – (49 partial) / 14 / – (prefix-only before Chr 120, Saf 14) | no | identical with the prefix; not possible without it (text shows its fallback colour) | prefix, with a solid `color` before | decl | not measured | `CSS.supports('background-clip','text')` |
| `color-scheme` | keep | 81 / 81 / 96 / 13 / – | high 2024 (low 2022) | approximate: old browsers keep light form controls | none | – | none | `CSS.supports('color-scheme','light dark')` |
| `text-wrap: balance` / `pretty` | keep | balance: 114 / 114 / 121 / 17.5 / –; pretty: 117 / 117 / – / 26 / – | balance low 2024; pretty no | approximate: normal wrapping | none in CSS | – | none | `CSS.supports('text-wrap','balance')` |
| `hyphens` | → `-webkit-hyphens`, `-ms-hyphens` | 55 / 79 / 43 / 17 / – (`-webkit-` Saf 5.1; `-ms-` IE 10) | high 2026 (low 2023) | identical per engine | prefixes | decl | AP [G] | `CSS.supports('hyphens','auto')` |
| `box-sizing`, `border-radius`, `box-shadow`, `background-size` | → `-webkit-`/`-moz-` forms | box-sizing 10 / 12 / 29 / 5.1 / 8; radius 4 / 12 / 4 / 5 / 9 | high 2018 (low 2015) | identical in engines that had the prefix; IE 6–8 no radius or shadow (approximate) | prefixes | decl | AP [M] | by version table |
| `line-clamp` | `line-clamp:3` → `display:-webkit-box;-webkit-box-orient:vertical;-webkit-line-clamp:3;overflow:hidden` | – / – / – / 18.2 / – (`-webkit-` Chr 6, FF 68, Saf 5) | no | identical where the prefix works [I] | the `-webkit-box` recipe | rule | not measured | `CSS.supports('-webkit-line-clamp','3')` |
| `print-color-adjust` | → `-webkit-print-color-adjust` | 136 / 136 / 97 / 15.4 / – (`-webkit-` Chr 17, Saf 6) | low 2025 | identical per engine | prefix | decl | not measured | `CSS.supports('print-color-adjust','exact')` |
| `display: flow-root` | → clearfix `::after` [G] | 58 / 79 / 53 / 13 / – | high 2022 (low 2020) | close (margin collapse differs) [I] | clearfix | rule | none found | `CSS.supports('display','flow-root')` |
| Two-value `display` | `display:block flex` → `display:flex` | 115 / 115 / 70 / 15 / – | high 2026 (low 2023) | identical | one-keyword form | decl | LC [D] | `CSS.supports('display','block flex')` |

## At-rules

| Feature | Example → lowered | Chr / Edge / FF / Saf / IE | Baseline | Fidelity | Closest technique | Needs | Done by today | Detect |
|---|---|---|---|---|---|---|---|---|
| `@layer` | `@layer base,ui; @layer ui{.b{…}} @layer base{.a{…}}` → rules sorted into layer order, unlayered rules last | 99 / 99 / 97 / 15.4 / – | high 2024 (low 2022) | identical when no selector wins across layers by specificity; close when the lowering raises specificity to keep layer order [I]; `!important` reverses layer order and needs its own pass | reorder; bump specificity with `:not(#\#)` tricks where needed [I] | sheet | PPE unwraps only, losing order [M]; LC, ESB keep [M] | no CSS test; version table |
| `@container` (size) | `@container (min-width:400px)` | 105 / 105 / 110 / 16 / – | high 2025 (low 2023) | needs JS shim; `@media` on the viewport is approximate | see JS shim section | HTML | LC rewrites syntax only [M] | `CSS.supports('container-type','inline-size')` [I] |
| `@container style()` | see Future CSS | 111 / 111 / 151 / 18 / – | low 2026 | | | | | |
| Media range syntax | `@media (width >= 600px)` → `(min-width: 600px)` | 104 / 104 / 102 / 16.4 / – | high 2025 (low 2023) | identical; strict `>` needs a small offset such as `.02px`, which is close [I] | `min-`/`max-` | decl (the prelude) | LC, PPE [M] | `matchMedia('(width >= 1px)').media !== 'not all'` [D, luna.md] |
| `or` in media queries | `@media (a) or (b)` → `@media (a), (b)` | 104 / 104 / 64 / 16.4 / – | high 2018 (low 2015) for media queries | identical [I] | comma list | prelude | not measured | `matchMedia` |
| Newer media features | `prefers-color-scheme` 76 / 79 / 67 / 12.1 / –; `prefers-reduced-motion` 74 / 79 / 63 / 10.1 / –; `dynamic-range` 98 / 98 / 100 / 13.1 / –; `color-gamut` 58 / 79 / 110 / 10 / – | see left | high 2022–2025 | approximate: old browsers skip the block and show the base rule | keep the block; make sure a sensible base rule exists | sheet | none | `matchMedia(…)` |
| `@custom-media` | `@custom-media --sm (width < 40em); @media (--sm)` → `@media (width < 40em)` | – / – / – / – / – (Firefox behind a flag) | no | identical | inline the definition | sheet | PPE; LC with `drafts.customMedia` [M] | none: no browser ships it |
| `@supports` | keep | 28 / 12 / 22 / 9 / – | high 2018 (low 2015) | identical when the base rule outside is the fallback | evaluate at build time for a known target, and drop dead branches [I] | sheet | none | n/a |
| `@supports selector()` | `@supports selector(:has(a)){…}` → evaluate for the target | 83 / 83 / 69 / 14.1 / – | high 2018 (low 2015) | identical [I] | build-time evaluation, since pristinecss knows the target | sheet | none [M] | `CSS.supports('selector(a)')` |
| `@property` | `@property --x{syntax:'<length>';initial-value:0;inherits:false}` → `:root{--x:0}` | 85 / 85 / 128 / 16.4 / – | low 2024 | close for static use; not possible for animating the property (typed interpolation) and for `inherits:false` across elements | declare the initial value on `:root` | sheet | none; both keep it [M] | `'registerProperty' in CSS` |
| `@scope` | `@scope (.card){.t{…}}` → `.card .t{…}` | 118 / 118 / 146 / 17.4 / – (Safari: full 17.4–18.6, partial 26.0–26.3, full again 26.4) | low 2026 | close for flat scopes; approximate with `to (…)` limits and proximity | prefix selectors, wrap the root in `:where()` to keep specificity; `:not()` for limits | rule | postcss-at-scope [D, search only] | no CSS test; version table |
| `@starting-style` | keep | 117 / 117 / 129 / 17.5 / – | low 2024 | the final look is identical; the entry transition needs a JS shim | none | – | none | `CSS.supports('transition-behavior','allow-discrete')` [I] |
| `@font-face` formats | `format("woff2")` only → add woff, ttf, eot entries | woff2 Chr 36 / Edge 14 / FF 39 / Saf 10 / – [haiku.md, BCD]; `src` 4 / 12 / 3.5 / 3.1 / 6 | high 2019 (low 2016) | identical when the files exist; the files are a build step, not CSS | extra `src` entries | sheet + assets | none | `@supports font-format(woff2)`: 108 / 108 / 106 / 17 / – [S]; none for older browsers |
| `@font-face` `format()` keywords and `tech()` | `format(woff2)` → `format("woff2")` | keywords: 108 / 108 / 105 / 4 / –; `tech()`: 108 / 108 / 107 / 17 / – | high 2019 (low 2016) | identical (quoted strings) | quote it; drop `tech()` entries the target can't use | decl | PPE font-format plugin [D] | as above |
| `@import` with `layer()` / `supports()` | `@import "a.css" layer(x)` → inline into `@layer x{…}` | `layer()` 99 / 99 / 97 / 15.4 / –; `supports()` 122 / 122 / 115 / 17.5 / – | high 2024 / high 2018 | identical once inlined and the wrappers are lowered | bundle | sheet + files | LC bundler [D] | none |
| `@namespace` | keep | 1 / 12 / 1 / 1 / 9 | high 2018 (low 2015) | identical in HTML pages | drop for IE ≤ 8 when only the default HTML namespace is used [I] | sheet | none | n/a |
| `@counter-style` | `@counter-style thumbs{…}` → `list-style-type` string or `::before` counter | 91 / 91 / 33 / 17 / – | high 2026 (low 2023) | close for simple cyclic/symbolic styles [I] | `::before{content:counter(…)}` | sheet | none found | none |
| Nested media queries | `@media screen{@media (min-width:1px){…}}` → `@media screen and (min-width:1px)` | 26 / 12 / 11 / 7 / – | high 2018 (low 2015) | identical | merge preludes | sheet | LC [G] | none |

## Layout

| Feature | Example → lowered | Chr / Edge / FF / Saf / IE | Baseline | Fidelity | Closest technique | Needs | Done by today | Detect |
|---|---|---|---|---|---|---|---|---|
| Flexbox, older syntaxes | `display:flex` → `-webkit-box` (2009), `-ms-flexbox` (IE 10), `-webkit-flex` | 29 / 12 / 20 / 9 / – (`-webkit-` Chr 21, Saf 7; IE 11 partial; `-ms-flexbox` IE 10); `flex-wrap` FF 28 | high 2018 (low 2015) | identical with `-webkit-`/`-ms-` 2012 syntax for common cases [I]; approximate with 2009 `-webkit-box` (no wrap) | property and value map | rule | AP [M] | `CSS.supports('display','flex')` |
| Grid for IE 10–11 | `grid-template-columns:1fr 1fr;gap:10px` → `display:-ms-grid;-ms-grid-columns:1fr 10px 1fr` plus child placement | 57 / 16 / 52 / 10.1 / – (`-ms-` IE 10, Edge 12) | high 2020 (low 2017) | close for explicit grids and named areas; not possible for auto-placement, `auto-fill`, `minmax` edge cases | gutter tracks, explicit `-ms-grid-row/column` on children | sheet, HTML for auto-placed children | AP with `grid: 'autoplace'` [M] | version table (no `CSS.supports` in IE) |
| Subgrid | `grid-template-columns:subgrid` | 117 / 117 / 71 / 16 / – | high 2026 (low 2023) | approximate: copy the parent's track list [I] | repeat tracks | sheet, HTML | none found | `CSS.supports('grid-template-columns','subgrid')` |
| Multi-column | `columns:2` → `-webkit-columns`, `-moz-columns` | 50 / 12 / 52 / 9 / 10 (`-webkit-` Chr 50 [S as recorded], Saf 3; `-moz-` FF 9) | high 2019 (low 2017) | identical per engine; one column where unsupported (approximate) | prefixes | decl | AP [M] | `CSS.supports('columns','2')` |
| `position: sticky` | → `position:-webkit-sticky` first | 56 / 16 / 32 / 13 / – (`-webkit-` Saf 7) | high 2022 (low 2019) | identical with the prefix; approximate without (stays in flow, doesn't stick) | prefix; `relative` fallback | decl | AP, ESB, LC [M] | `CSS.supports('position','sticky')` |
| `display: contents` | keep | 65 / 79 / 37 / 11.1 / – | no | not possible in CSS (the box can't vanish) | none | – | none | `CSS.supports('display','contents')` |

## Prefixes in general

Prefixes are the cheapest lowerings: one declaration, identical where the
prefix exists. They matter more than their age suggests. `support.json` shows
several still needed by current browsers [S]:

- `-webkit-user-select`: Safari has unprefixed `user-select` only in preview.
- `-webkit-text-decoration` for colour and style in the shorthand: Safari before
  26.2.
- `-webkit-backdrop-filter`: Safari before 18.
- `-webkit-mask`: Chrome before 120, Safari before 15.4.
- `-webkit-background-clip:text`: Chrome before 120, Safari before 14.
- `-webkit-line-clamp`: everything except Safari 18.2+.
- `-webkit-print-color-adjust`: Chrome before 136.

Rules for emitting them [I]:

- Prefixed declaration first, standard one last, so the standard wins where
  both parse.
- Prefixed selectors need one rule each. One unknown pseudo in a list drops the
  whole rule.
- Prefixes come from BCD's `prefix` and `alternative_name` statements, which
  `support.json` keeps per key. That is a ready-made source for a prefix table,
  so pristinecss doesn't need Autoprefixer's data.

## Where the researchers disagreed

| Topic | haiku.md | luna.md | What I found |
|---|---|---|---|
| Browser versions | from BCD | from memory | Settled: every version here is from `support.json`. Luna was wrong on several: nesting (said Chr 112 / Saf 16.5; that is the CSSOM API, the `&` selector is 120 / 17.2), `:dir()` (FF 49, not 118), `:lang()` lists (Chrome has never shipped them; luna said 120), `image-set()` (FF 89, not 119), double-position stops (FF 64, not 52), sticky (Saf 13 unprefixed, not 6.1). |
| `@scope` in Safari | 26.4 | 17.4 | Both partly right. Full 17.4–18.6, removed in 26, partial 26.0–26.3, full again in 26.4 [S]. Support isn't always "from version N on"; the capability model has to allow gaps. |
| esbuild and nesting | "ESB keeps" | "limited nesting" | Settled against haiku. I re-ran esbuild 0.28.2: `.a{color:red;& .b{color:blue}}` → `.a{color:red}.a .b{color:#00f}` for `chrome50` and for haiku's own target list [M]. |
| `transparent` in IE | IE 3 | IE 9 | Both are in BCD. `css.types.color.transparent` says IE 3; `css.types.color.named-color.transparent` says IE 9 [S]. The usual reading is that IE 6–8 accepted it for `background` and `border` (CSS 2) but not for `color` [G]. Not settled without an IE. |
| `currentColor` | fully, when `color` is in the same rule | not generally | Luna is right in general [I]: another rule can set `color` on the same element (`:hover`, a more specific selector), so copying the value is wrong. A narrow identical case exists for borders (see table). |
| `light-dark()` | approximate (OS only) | partial | Depends on the technique. PPE's toggle variables follow `color-scheme` set in the sheet, so close in browsers with custom properties [D, future.md]. Not rendered, so not settled. |
| `:focus-visible` | approximate with `:focus` | JS only | Both: CSS gives approximate, a shim gives close. |
| Logical properties | rule pair needed | partial | Agree. Haiku measured that PPE drops RTL [M]. |
| `calc()` | fold constants | not lowerable | Agree once split: constants fold identically; mixed units only need lowering for IE ≤ 8. |
| Display-P3 to sRGB | LC/ESB `#ff0f0e`, PPE `rgb(255,52,40)` | – | Not settled. LC looks like clipping, PPE like chroma reduction [I]. CSS Color 4 defines a gamut-mapping method; checking which tool follows it is open. |

## Future CSS

Same columns. "Chr / Edge / FF / Saf" from `support.json`; "preview" means only
in a preview build. Where `support.json` has no record, the source is named.

| Feature | Example → lowered | Chr / Edge / FF / Saf / IE | Baseline | Fidelity | Closest technique | Needs | Done by today | Detect |
|---|---|---|---|---|---|---|---|---|
| `if()` with `media()` / `supports()` | `color:if(media(width>600px):red;else:blue)` → `color:blue;@media (min-width:600.02px){color:red}` (offset for strict `>` [I]) | 137 / 137 / – / – / – | no | identical [I] | fallback first, then a conditional block | rule | postcss-if-function [search only, README 404] | `CSS.supports('color','if(else:red)')` [I] |
| `if()` with `style()` | `if(style(--theme:dark):white;else:black)` | as above | no | close when every value of `--theme` is in the sheet (selector rules) [I]; otherwise needs JS shim | rewrite to rules on the elements that set `--theme` | sheet | none | as above |
| `@when` / `@else` | `@when media(a){…} @else {…}` → `@media (a){…} @media not (a){…}` | no data in `support.json`; MDN says no browser [D, future-luna.md] | no | identical for `media()`/`supports()` [I] | each branch negates the earlier ones | rule | none found | none |
| `@function` | `@function --neg(--v){result:calc(-1*var(--v))}` | 139 / 139 / – / preview / – | no | identical for constant arguments (inline the result); close with `var()` arguments via helper custom properties | inline, or PPE's generated variables | sheet, decoded | postcss-custom-functions, with limits [D] | none yet |
| `@mixin` / `@apply` | `@apply --card;` → the mixin's declarations | no BCD data [S: `no_bcd_data`] | no | identical for static bodies [I] | expand in place | sheet | postcss-mixins (old syntax, not checked) [D] | none |
| `@custom-media` | see At-rules | – (Firefox flag) | no | identical | inline | sheet | PPE, LC [M] | none |
| Typed `attr()` | `width:attr(data-w type(<length>))` → `[data-w="3px"]{width:3px}` | 133 / 133 / 155 / preview / – | no | identical for values known from the HTML; needs JS shim when attributes change | one rule per value | HTML | none found | `CSS.supports('width','attr(data-w type(<length>))')` [I] |
| `sibling-index()` / `sibling-count()` | `transition-delay:calc(sibling-index()*50ms)` → `:nth-child(1){…}` … `:nth-child(N){…}` | 138 / 138 / 154 / 26.2 / – | low 2026 | identical up to a chosen N; wrong past it | emit N rules (N from the HTML or a limit) | rule; HTML for N | none found | `CSS.supports('order','sibling-index()')` [I] |
| `:nth-child(… of S)` | `:nth-child(2 of .x)` | 111 / 111 / 113 / 9 / – | high 2025 (low 2023) | close for known markup; not possible without it | classes or positions from the HTML | HTML | none found | `CSS.supports('selector(:nth-child(1 of a))')` |
| `random()` | `rotate:random(0deg,20deg)` → `rotate:7.3deg` | 156 / 156 / preview / 26.2 / – | no | approximate: one fixed value, no per-element randomness | seeded build-time value | decl | postcss-random-function [D] | `CSS.supports('width','random(1px,2px)')` [I] |
| `progress()` | `progress(50 from 0 to 200)` → `.25` | 138 / 138 / 155 / 26 / – | low 2026 | identical for constants; approximate with `var()` | fold | decl, decoded | none found | `CSS.supports('opacity','progress(1 from 0 to 2)')` [I] |
| `contrast-color()` | `contrast-color(#123)` → `white` | 147 / 147 / 146 / 26 / – | low 2026 | identical for constants if the spec's rule is followed; not possible with `var()` | evaluate | decl, decoded | postcss-contrast-color-function [D] | `CSS.supports('color','contrast-color(red)')` |
| `color-contrast()` | | no BCD data [S] | no | as above, once specified | evaluate | decl, decoded | none | none |
| `color-mix()` with many colours | `color-mix(in srgb, red, blue, lime)` | – / – / 150 / 27 / – | no | identical for constants [I] | evaluate | decl, decoded | none found | `CSS.supports('color','color-mix(red,blue,lime)')` [I] |
| `@container style()` | `@container style(--theme:dark){…}` | 111 / 111 / 151 / 18 / – | low 2026 | close when the values are set by known selectors in the sheet [I]; needs JS shim otherwise | descendant selectors under the setter | sheet | none | no CSS test |
| Anchor positioning | `position-anchor:--tip; position-area:top` | `anchor-name`: 125 / 125 / 147 / 26 / –; feature as a whole: 157 / 157 / – / 27 / – | no | needs JS shim | see JS shim section | HTML, layout | OddBird polyfill (JS) [search] | `CSS.supports('anchor-name','--a')` |
| `field-sizing: content` | `textarea{field-sizing:content}` | 123 / 123 / 152 / 26.2 / – | low 2026 | needs JS shim for live text; a min-height is approximate | grid mirror (HTML change) | HTML | none | `CSS.supports('field-sizing','content')` |
| `text-box-trim` | `text-box:trim-both cap alphabetic` → negative margins | 133 / 133 / 154 / 18.2 / – | no | approximate: needs font metrics; wrong if the fallback font loads | margins from the font's metrics | decl, font metrics | none | `CSS.supports('text-box-trim','trim-both')` |
| `corner-shape` | `corner-shape:squircle` → keep `border-radius` | 139 / 139 / preview / preview / – | no | approximate: round corners instead of the shape | `border-radius`; `clip-path` for bevel cuts borders and shadows | decl | none | `CSS.supports('corner-shape','squircle')` |
| `interpolate-size` / `calc-size()` | `interpolate-size:allow-keywords` | 129 / 129 / – / – / – | no | needs JS shim; `max-height` transition is approximate | `max-height` | rule | none | `CSS.supports('interpolate-size','allow-keywords')` |
| Scroll-driven animations | `animation-timeline:scroll()` | 115 / 115 / 159 / 26 / –; feature as a whole: no full record | no | needs JS shim; the end state without animation is the CSS fallback | none | – | scroll-timeline polyfill (JS) [search] | `CSS.supports('animation-timeline','scroll()')` |
| View transitions | `view-transition-name:hero` | name: 111 / 111 / 144 / 18 / –; cross-document `@view-transition`: 126 / 126 / – / 18.2 / – | low 2025 (same-document) | needs JS shim, or skip (pages change with no animation) | none | – | none | `CSS.supports('view-transition-name','a')` |
| Masonry / `grid-lanes` | `display:grid-lanes` | – / – / – / 26.4 / – | no | needs JS shim; multi-column is approximate (order runs down columns) | `columns` | rule | none | `CSS.supports('display','grid-lanes')` |
| `reading-flow` | `reading-flow:grid-rows` | 137 / 137 / – / – / – | no | visually identical (it only changes focus order); focus order needs the HTML reordered | none in CSS | HTML | none | `CSS.supports('reading-flow','grid-rows')` |
| `transition-behavior: allow-discrete` | `transition:display .2s allow-discrete` | 117 / 117 / – (129 partial) / 17.4 / – | low 2024 | the end state is identical; the transition is lost | drop it | decl | none | `CSS.supports('transition-behavior','allow-discrete')` |

### Shortlist, merged

Both future researchers ranked by value to authors and how well a CSS-only
lowering keeps the look. Merged, with the reason in one line each [I]:

1. **`if()` with `media()`/`supports()`, `@when`/`@else`, `@custom-media`.**
   Identical, CSS-only, and only a parser plus a prelude rewrite. Both put this
   at or near the top.
2. **`@function` and `@mixin`.** High author value. Identical for constant
   arguments and static bodies. Specs still moving, so ship behind a flag.
3. **Constant folding of new functions:** `progress()`, `contrast-color()`,
   many-colour `color-mix()`, `alpha()`. Same machinery as the colour and math
   work in the main tables.
4. **`@container style()` and `if(style())`** where the sheet holds every value.
   Close, sheet-level.
5. **HTML-aware lowerings:** typed `attr()`, `sibling-index()`/`sibling-count()`,
   `:nth-child(of S)`. Identical, but only with the markup. They suit Clay,
   which generates both the HTML and the CSS.
6. **Approximations, opt-in only:** `text-box-trim`, `corner-shape`, `random()`,
   `field-sizing`.
7. **JS shim only:** anchor positioning, scroll-driven animations, view
   transitions, masonry, `interpolate-size`. Report them; don't fake them.

## Needs a JS shim

Each shim would ship as a capability the handshake reports, so the server sends
the shim only to browsers that lack the feature [I].

| Feature | What the shim would do |
|---|---|
| `:has()` | Watch the DOM (MutationObserver). For each `:has(X)` in the sheet, add a class to elements that have a matching descendant. The lowering rewrites `:has(X)` to that class. Re-run on changes. |
| `:focus-within` | Listen to `focusin`/`focusout` on the document; add a class to every ancestor of the focused element. |
| `:focus-visible` | Track whether the last input was a keyboard; add a class on focus only then. |
| Container size queries and units | ResizeObserver on each container; set classes for each query threshold, and `--cqw`-style custom properties for units. The lowering rewrites `@container` blocks to class selectors. |
| `@container style()`, `if(style())` (dynamic values) | Read the computed custom property on containers and toggle classes. |
| Anchor positioning | Find each anchor and positioned box; read both rectangles; compute the position; check overflow against the viewport and pick a `position-try` fallback. Re-run on scroll, resize and content changes. |
| `@starting-style` | When an element is inserted or shown, apply the starting style, force a style flush, then remove it so the transition runs. |
| Scroll-driven animations | Map scroll offset (or an element's position in view) to animation progress, and drive the animation with the Web Animations API. |
| View transitions | Snapshot the old state, swap the DOM, snapshot the new state, animate between them. Heavy; may not be worth it. |
| Masonry / `grid-lanes` | Measure items and place each in the shortest column; redo on resize and image load. Keep focus order sane. |
| `interpolate-size` / `calc-size()` | Measure the content height, animate to that pixel value, then set `auto` again. |
| `field-sizing: content` | On input, size the field to its content (copy into a hidden mirror, or set height from `scrollHeight`). |
| Typed `attr()`, `sibling-index()` with changing DOM | Copy attributes or indices into custom properties on each element, and keep them updated. |

## Not possible

When a lowering can't keep the look, the rule from binary-style.md applies:
**never a silent gap**. `Report.Gaps` should say the rule, the target, and what
happened instead. Suggested report kinds [I]:

- **ignored**: the browser skips it and the base style shows. Examples:
  `text-wrap: balance`, `color-scheme`, newer media features, `::marker`
  styling, `backdrop-filter` without a prefix, `mask` where masks don't exist,
  `display: contents`.
- **approximated**: a named difference. Examples: `:focus-visible` as `:focus`,
  `overflow:clip` as `hidden`, `dvh` as `vh`, out-of-gamut colours clipped,
  flex `gap` as margins, `:where()` gaining specificity, `light-dark()` by OS
  preference only.
- **needs runtime**: the value depends on things the sheet can't know. Examples:
  `var()` changed by JS, `color-mix()` or relative colour with `var()`,
  `@property` animation, mixed-unit `calc()` in IE ≤ 8.
- **needs shim**: listed above. Report the shim's name, so the caller can choose
  to serve it.
- **needs HTML**: lowerable if the caller hands over the markup (typed `attr()`,
  `sibling-index()`, `:nth-child(of S)`, `:has()` for static pages).

Approximations should be opt-in per target, as luna.md and future-luna.md both
argue: "silent approximation would contradict 'looks the same'".

## Bugs in the incumbent tools

These are where pristinecss can do better. Each was seen in tool output. Items
1–5 I re-ran today with the same Lightning CSS 1.33.0 install (scripts in
`/tmp/haiku-tools/repro-transform.cjs` and `/tmp/lowerings-q/`) [M].

1. **LC drops `translate` when `transform` is in the same rule.** Target Chrome
   60: `.a{translate:10px 20px;transform:skewX(10deg)}` → `.a{transform:skewX(10deg)}`.
   The translate is gone. Re-run: same output.
2. **LC merges transforms in the wrong order.** Target IE 6:
   `.a{transform:skewX(10deg);rotate:45deg}` → `transform: skewX(10deg) rotate3d(0,0,1,45deg)`.
   The spec applies translate, rotate, scale, then the `transform` list
   ([css-transforms-2 §6](https://drafts.csswg.org/css-transforms-2/#individual-transforms)),
   so rotate should come first. Re-run: same output.
3. **LC emits `light-dark()` as two `var()`s side by side** with no definitions:
   `color:var(--lightningcss-light,#000)var(--lightningcss-dark,#fff)` (target IE
   11). Without the variables defined elsewhere, this isn't a valid colour.
   Re-run: same output.
4. **LC uses `:is()` in lowerings for IE**, which has no `:is()`. `.a:dir(rtl)`
   → `.a:is(:lang(ae),…)`; `:lang(en,fr)` → `:is(:lang(en),:lang(fr))`;
   `margin-inline:1px 2px` → two rules guarded by `:is(:lang(…))`. IE drops a
   rule with an unknown pseudo-class [I], so in IE 11 the margins vanish
   entirely. Re-run: same output for all three.
5. **LC rewrites `transparent` to `rgba(0,0,0,0)` for IE 6.**
   `background-color:transparent` → `rgba(0, 0, 0, 0)`. IE 6–8 have no
   `rgba()` [G], and BCD's `css.types.color.transparent` says IE 3 [S], so this
   likely turns a working rule into a broken one. Not checked in an IE.
6. **LC adds `-ms-appearance` for IE 11.** BCD records no IE support for
   `appearance` in any form [S]. Harmless but wasted bytes. Re-run: same output.
7. **LC keeps `:is()` with combinators for Safari 10**, unlowered. BCD shows
   Safari 9–13 had `:matches()`, which takes combinators, while
   `:-webkit-any()` does not ([S], BCD note "Doesn't support combinators").
   pristinecss could emit `:matches()`. Re-run: `.a:is(.b .c,.d)` unchanged.
8. **PPE drops RTL for logical properties.** `margin-inline:1px 2px` →
   `margin-left:1px;margin-right:2px` only; `text-align:start` → `left` only.
   RTL pages get mirrored margins wrong [M, haiku.md].
9. **PPE doesn't lower `:lang(en,fr)`** for IE 6, and Chrome has never shipped
   the list form [S], so the rule fails in every Chromium browser today
   [M, haiku.md].
10. **PPE unwraps `@layer`** without reordering, so layer precedence is lost
    [M, haiku.md].
11. **Tools disagree on Display-P3 → sRGB**: LC and ESB `#ff0f0e`, PPE
    `rgb(255,52,40)` for `color(display-p3 1 0 0)` [M, haiku.md].
12. **ESB emits `-moz-user-select:-moz-none`** for `user-select:none` [M,
    haiku.md]. Whether `-moz-none` is right for old Firefox wasn't checked [G].

## Suggested first lowerings

A suggestion for the maintainer to decide. Ranked by value to authors, ease,
and fidelity. binary-style.md's steps build the machinery (capabilities, opt-in
lowering, opt-in raising, the CLI) with `inset` and `oklch()`/`oklab()` as the
first rewrites; everything else here is the deferred "more rewrites" work, in
this order.

**First, one declaration each.** All identical or close, and they prove the
capability bits and `Report.Gaps`:

1. `inset` → four sides. Identical, one declaration. Already planned.
2. `oklch()`/`oklab()` → sRGB, with `lab()` in between for Safari 15–15.3.
   Already planned. Brings in a gamut-mapping choice (bug 11), so decide it
   here and test against the CSS Color 4 method.
3. The rest of the colour family on the same code: `lab()`/`lch()`, `hwb()`,
   `color()`, space-separated `rgb()`/`hsl()`, hex alpha. Cheap once 2 exists.
4. `place-*`, two-value `overflow`, opacity percentages, double-position
   gradient stops, two-value `display`. Identical, one declaration each, and
   they exercise "expand one declaration into several".
5. Media range syntax → `min-`/`max-`. Identical for `>=` and `<=`; strict `>`
   and `<` need a small offset such as `.02px`, which is only close. First
   lowering of an at-rule prelude.

**Then constant folding, once decoded values exist.** Constant folding for `calc()`, `min()`/`max()`/
`clamp()`, `round()`/`mod()`/`rem()`, trig, `pow()`, `abs()`/`sign()`, and
`color-mix()`/relative colour with constant inputs. These need decoded values,
which are deferred in binary-style.md, and they are identical for constants.

**Then more lowerings one at a time.** Roughly in this order:

1. **Prefixes from BCD data**, starting with the ones current browsers still
   need (list under "Prefixes in general"). High value, identical, and the
   data is the same MDN data `internal/capgen` already reads.
2. **`:lang()` lists and `:not()` lists.** Rule-level. `:lang()` lists are
   identical. `:not()` list expansion raises specificity: `.a:not(.b,.c)` is
   (0,2,0) but `.a:not(.b):not(.c)` is (0,3,0), so `.a:not(.b,.c){color:red}
   .a.d{color:blue}` turns an `a d` element red instead of blue. It needs a
   sheet-wide specificity check, or a gap where a conflict exists. `:lang()`
   lists help every Chromium browser today.
3. **Nesting.** Sheet-level, identical for flat cases, high author value. Two
   capability bits: `&` (Chr 120) and declarations after nested rules (Chr
   130).
4. **Individual transforms**, done right (bugs 1–2).
5. **Logical properties and keywords with a `[dir=rtl]` pair**, valid in IE
   (bugs 4 and 8).
6. **`:is()`/`:any-link`/`::placeholder`**, using `:matches()` for Safari
   9–13 (bug 7).
7. **`@custom-media`, `if()` with `media()`/`supports()`, `@when`/`@else`,
   `@supports` evaluated per target.** Future CSS with identical CSS-only
   lowerings.
8. **`@layer`**, with a specificity check. Hardest of the identical ones.

Later, behind opt-in: the approximations (`clamp()` ladders, flex `gap`,
`aspect-ratio` padding, `:focus-visible` as `:focus`), then HTML-aware
lowerings if Clay wants them, then shims if that direction is taken.

## Still to check

- **Render it.** No fidelity grade has been checked in a browser. The
  approximations and every IE row need side-by-side screenshots.
- IE claims marked [G]: `filter:alpha`, IE gradient filters, `rgba()` in IE
  6–8, single-colon pseudo-elements, the `transparent` split.
- Which gamut-mapping method each tool uses, and which pristinecss should use.
- `CSS.supports` strings marked [I] haven't been run in any browser.
- `@when`/`@else` and `@mixin` have no BCD data; ship status is from MDN and
  search only.
- The `if()`, `@scope` and anchor polyfill tools were found by search only.

## Sources

- `/Users/adavies/.cache/pcss-research/support.json`: web-features 3.44.0 and
  @mdn/browser-compat-data 8.1.7, regenerated by `regenerate-support.mjs` in the
  same folder. Every version and Baseline cell.
- `/Users/adavies/.cache/pcss-research/haiku.md`: tool runs, fidelity ratings,
  bugs.
- `/Users/adavies/.cache/pcss-research/luna.md`: tool docs and detection notes
  (its versions are not used).
- `/Users/adavies/.cache/pcss-research/future.md` and `future-luna.md`: future
  CSS, tools and shortlists.
- Specs: [css-transforms-2](https://drafts.csswg.org/css-transforms-2/#individual-transforms),
  [css-values-5](https://drafts.csswg.org/css-values-5/),
  [css-mixins-1](https://drafts.csswg.org/css-mixins-1/),
  [css-conditional-5](https://drafts.csswg.org/css-conditional-5/),
  [css-anchor-position-1](https://drafts.csswg.org/css-anchor-position-1/),
  [css-cascade-6](https://drafts.csswg.org/css-cascade-6/).
- Tools: [Lightning CSS transpilation](https://lightningcss.dev/transpilation.html),
  [preset-env features](https://preset-env.cssdb.org/features/),
  [postcss-custom-functions](https://github.com/csstools/postcss-plugins/tree/main/plugins/postcss-custom-functions),
  [postcss-light-dark-function](https://github.com/csstools/postcss-plugins/tree/main/plugins/postcss-light-dark-function),
  [Autoprefixer](https://github.com/postcss/autoprefixer),
  [esbuild CSS](https://esbuild.github.io/content-types/).
