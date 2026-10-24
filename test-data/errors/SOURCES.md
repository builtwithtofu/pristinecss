# Error-recovery inputs

License texts and inputs verified from the repositories at the pinned commits below. Only CSS strings are imported: no browser harness, expected results, images, fonts or scripts. JSON files here are **extracted input arrays**, not byte-for-byte copies of upstream JSON/HTML. Original strings, order and duplicates are retained. Local notices do not license pristinecss.

## Simon Sapin's css-parsing-tests — CC0 / public-domain dedication

Requested repository https://github.com/SimonSapin/css-parsing-tests redirects to CourtBouillon/css-parsing-tests. Pin: `203ce36bffd617db7f118c551e32794561fb273d`.

The repository's own [LICENSE at that pin](https://raw.githubusercontent.com/SimonSapin/css-parsing-tests/203ce36bffd617db7f118c551e32794561fb273d/LICENSE) says “Written in 2013 by Simon Sapin” and dedicates copyright and related rights to the public domain worldwide, referring to CC0. Complete text: [licenses/css-parsing-tests.txt](licenses/css-parsing-tests.txt).

The [upstream README](https://raw.githubusercontent.com/SimonSapin/css-parsing-tests/203ce36bffd617db7f118c551e32794561fb273d/README.rst) defines an alternating input/expected-output array. For each file below, extract zero-based elements `0, 2, 4, ...` into a JSON array. No filtering by pristinecss's results; valid controls are retained.

| Local file | Pinned upstream source | Inputs | Test-time adaptation |
| --- | --- | ---: | --- |
| `css-parsing-tests/stylesheet.json` | [stylesheet.json](https://raw.githubusercontent.com/SimonSapin/css-parsing-tests/203ce36bffd617db7f118c551e32794561fb273d/stylesheet.json) | 16 | None |
| `css-parsing-tests/rule_list.json` | [rule_list.json](https://raw.githubusercontent.com/SimonSapin/css-parsing-tests/203ce36bffd617db7f118c551e32794561fb273d/rule_list.json) | 15 | None; exercised through the stylesheet entry point |
| `css-parsing-tests/declaration_list.json` | [declaration_list.json](https://raw.githubusercontent.com/SimonSapin/css-parsing-tests/203ce36bffd617db7f118c551e32794561fb273d/declaration_list.json) | 10 | Wrap in `a{` and `}` |

All three files use the same verified root LICENSE above. Coverage includes invalid rule/declaration starts, unmatched brackets/parentheses, missing closing braces, malformed at-keywords, and CDO/CDC controls. This is a syntax/recovery subset, not the entire upstream suite (color grammars and byte-encoding tests are not imported).

## web-platform-tests — BSD-3-Clause

Repository https://github.com/web-platform-tests/wpt. Pin: `5af3c135c9c97381733866150a7ada27d1029c50`.

The repository's own [LICENSE.md at that pin](https://raw.githubusercontent.com/web-platform-tests/wpt/5af3c135c9c97381733866150a7ada27d1029c50/LICENSE.md) is headed “The 3-Clause BSD License”, copyright web-platform-tests contributors. Complete text: [licenses/wpt.txt](licenses/wpt.txt). Each selected file was read at that pin; none contains a separate license overriding the root notice.

| Local file | Pinned upstream source | Extraction | Inputs | Test-time adaptation |
| --- | --- | --- | ---: | --- |
| `wpt/parse-invalid-input.json` | [css/css-syntax/parse-invalid-input.html](https://raw.githubusercontent.com/web-platform-tests/wpt/5af3c135c9c97381733866150a7ada27d1029c50/css/css-syntax/parse-invalid-input.html) | First string in each `invalidStyles` row | 8 | Prefix `a{`, no closing brace: retain EOF inside an inline declaration list |
| `wpt/unclosed-url-at-eof.json` | [css/css-syntax/unclosed-url-at-eof.html](https://raw.githubusercontent.com/web-platform-tests/wpt/5af3c135c9c97381733866150a7ada27d1029c50/css/css-syntax/unclosed-url-at-eof.html) | Four `style` attribute values, in document order | 4 | Prefix `a{`, no closing brace, preserving URL EOF |
| `wpt/trailing-braces.json` | [css/css-syntax/trailing-braces.html](https://raw.githubusercontent.com/web-platform-tests/wpt/5af3c135c9c97381733866150a7ada27d1029c50/css/css-syntax/trailing-braces.html) | Text inside `<style id=style>`, including surrounding newlines | 1 | None |

All three files use the verified BSD license above. Closed URL controls are deliberately retained. These are inputs only, not WPT conformance results. In particular, EOF auto-closing can be valid CSS grammar while still causing syntax parse errors; the WPT titles calling unclosed URLs valid are not requirements that pristinecss return zero diagnostics.

## Generated and handwritten inputs

`damageSheet` in `error_corpus_test.go` generates eight variants for **every top-level CSS file** in `test-data/frameworks/` and `test-data/corpus/`. Those original files and their license notices remain unchanged; see their own SOURCES.md files. Seed: `0x5eedc55 XOR FNV-1a-64(filename)`, using a local `math/rand` source, independent of map iteration and test order.

Lexed token boundaries select one seeded location per variant: cut immediately after a block opener, opening quote, comment opener or `url(`; inject `}`, `;` or junk between top-level rules; inject junk after a property colon inside a declaration (an identifier following `{` or `;`). Missing string/comment/URL constructs get a small open synthetic tail, named `*-tail`, rather than pretending the original contains them. No damaged third-party sheet is checked in. A lone top-level semicolon is intentionally a recovery/control input even though pristinecss ignores it without error.

`recoverySpotChecks` and `errorRegressionInputs` are original local fixtures in new Go test files, not third-party imports. Small survival checks follow [CSS Syntax error handling](https://drafts.csswg.org/css-syntax-3/#error-handling) and [string-token newline handling](https://drafts.csswg.org/css-syntax-3/#consume-string-token): declaration errors end at semicolons, bad strings at newlines, and a balanced unknown at-rule cannot consume following rules. They assert the trailing `after` rule survives; they do not require arbitrary trailing text after an unclosed comment/string/block to survive.

## What expected-output comparison would require (out of scope)

For css-parsing-tests: retain upstream expected arrays, provide each specified parser entry point (not just stylesheet Parse), and normalize pristinecss's selector/value nodes and error representation to upstream's component-value/declaration/rule model. Byte-stream cases additionally need the prescribed encoding detection and decoding. Grammar-specific color/An+B tests need those grammar parsers, not merely syntax parsing.

For WPT: run the original HTML/JS under a browser harness, or port assertions to a CSSOM-compatible adapter including property grammar validation and serialization. This library's best-effort AST/emission and ParseErrors are not a browser's `CSSStyleDeclaration`; prefixing an inline list with a style rule is deliberately only an input adaptation, not an equivalent conformance oracle.
