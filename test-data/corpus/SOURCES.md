# Extended CSS parser corpus

Fetched/generated and licenses verified on **2026-10-23 (UTC)**. Third-party CSS is unmodified, including all headers; only CSS is included, not referenced fonts, icons, scripts, images or source maps. Files were extracted from the pinned npm tarballs; the URLs below identify the package files. Full package notices are in `licenses/`; they do not license pristinecss itself.

| File | Version | Pinned source | License | License evidence / local notice | Coverage |
| --- | --- | --- | --- | --- | --- |
| `tailwind-4.1.18.css` | 4.1.18 | Generated using [tailwindcss](https://cdn.jsdelivr.net/npm/tailwindcss@4.1.18/index.css) and `@tailwindcss/cli@4.1.18`; see command below | MIT | [tailwindcss LICENSE](https://cdn.jsdelivr.net/npm/tailwindcss@4.1.18/LICENSE), [CLI LICENSE](https://cdn.jsdelivr.net/npm/@tailwindcss/cli@4.1.18/LICENSE) / [notice](licenses/tailwind-4.1.18.txt) | `@layer`, `@property`, `color-mix()`, `oklch()`, utility variants |
| `missing-1.3.0.css` | 1.3.0 | https://cdn.jsdelivr.net/npm/missing.css@1.3.0/dist/missing.css | BSD-2-Clause | [LICENSE](https://cdn.jsdelivr.net/npm/missing.css@1.3.0/LICENSE) / [notice](licenses/missing-1.3.0.txt) | Shipped native nesting: `&`, nested `@supports`/`@media`, declarations after nested rules |
| `bootstrap-3.4.1.min.css` | 3.4.1 | https://cdn.jsdelivr.net/npm/bootstrap@3.4.1/dist/css/bootstrap.min.css | MIT | [LICENSE](https://cdn.jsdelivr.net/npm/bootstrap@3.4.1/LICENSE) / [notice](licenses/bootstrap-3.4.1.txt) | One-line minified production body, `filter:progid:DXImageTransform...` |
| `pure-0.6.0.min.css` | 0.6.0 | https://cdn.jsdelivr.net/npm/purecss@0.6.0/build/pure-min.css | BSD-3-Clause; bundled Normalize.css is MIT | [LICENSE.md](https://cdn.jsdelivr.net/npm/purecss@0.6.0/LICENSE.md) / [notice](licenses/pure-0.6.0.txt) | Minified production CSS, `*zoom`, `*display`, escaped selectors |
| `yui-console-3.5.1.min.css` | 3.5.1 | https://cdn.jsdelivr.net/npm/yui@3.5.1/assets/skins/sam/console.css | BSD-3-Clause | [project LICENSE.md at v3.5.1](https://raw.githubusercontent.com/yui/yui3/v3.5.1/LICENSE.md) / [notice](licenses/yui-3.5.1.txt) | One-line production body, `_height:32px` IE6 hack |
| `dijit-1.10.4.css` | 1.10.4 | https://cdn.jsdelivr.net/npm/dijit@1.10.4/themes/dijit.css | BSD-3-Clause (chosen from BSD or AFL-2.1) | [LICENSE](https://cdn.jsdelivr.net/npm/dijit@1.10.4/LICENSE) / [notice](licenses/dijit-1.10.4.txt) | Real shipped `expression(eval((document.documentElement\|\|document.body).scrollTop))` IE workaround; not minified |
| `fontawesome-free-6.7.2.css` | 6.7.2 | https://cdn.jsdelivr.net/npm/@fortawesome/fontawesome-free@6.7.2/css/all.css | MIT (CSS only) | [LICENSE.txt, Code section](https://cdn.jsdelivr.net/npm/@fortawesome/fontawesome-free@6.7.2/LICENSE.txt) / [notice](licenses/fontawesome-free-6.7.2.txt) | Font faces, escaped glyph values, custom properties, animation; no font/icon files copied |
| `animate-3.7.2.css` | 3.7.2 | https://cdn.jsdelivr.net/npm/animate.css@3.7.2/animate.css | MIT | [LICENSE](https://cdn.jsdelivr.net/npm/animate.css@3.7.2/LICENSE) / [notice](licenses/animate-3.7.2.txt) | Vendor-prefixed keyframes and transforms; deliberately not v4+ |
| `generated-data-uri.css` | generator v1 (checked in here) | Local `generate-data-uri.py`, no upstream URL | Original generated fixture; no third-party material | No external license needed: both PNG pixel arrays are generated here, not copied artwork | Large base64 PNGs in quoted and unquoted `url(data:...)` |

## Tailwind reproduction

Run from the repository root (Node/npm required):

```sh
npm install --registry=https://registry.npmjs.org --prefix test-data/corpus/tailwind --no-save --package-lock=false @tailwindcss/cli@4.1.18 tailwindcss@4.1.18
test-data/corpus/tailwind/node_modules/.bin/tailwindcss --input test-data/corpus/tailwind/input.css --output test-data/corpus/tailwind-4.1.18.css
```

Input: `tailwind/input.css` and `tailwind/utilities.html`. `source(none)` and the explicit `@source` restrict scanning to that HTML, independent of other repository files. The output is intentionally not flattened/minified, retaining native nested variants. Do not check in `node_modules/`.

## Synthetic data URI reproduction

```sh
python3 test-data/corpus/generate-data-uri.py
```

This is a **synthetic** sheet, not a real site's CSS. The generator creates 128×128 and 256×256 RGB PNGs from a fixed integer sequence and PNG scanline bytes. Stored (compression level 0) image data makes the base64 payloads large. No external image, font, icon, JavaScript or artwork is embedded. The two rules exercise quoted and unquoted data URLs.

## Checks

All top-level `.css` files in both this directory and `../frameworks/` are exercised by the parse/invariant, allocation-budget and structural round-trip tests. Generator inputs are in a subdirectory so they are not standalone fixtures.
