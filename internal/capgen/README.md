# Capability data

`capabilities.tsv` owns stable handshake bit numbers. Append new numbers; never
renumber or reuse them. Keep a retired assignment in place. All 256 bits survive
`CapsFromBits` / `Bits`, including bits unknown to this release. `Modern` sets
all 256 bits.

The list covers the CSS surfaces in `docs/design/lowerings.md`, splitting groups
whose support differs (for example `oklab`/`oklch`, logical properties, font
formats, and newer units). Each row has an explicit wire number, Go name,
web-features ID, and one or more MDN BCD keys. `*` selects the web-feature's
complete `compat_features`; `-` means no data. Explicit keys distinguish a CSS
surface from an entire web-feature with unrelated CSSOM or later subfeatures.
A capability requires the intersection of full-support intervals for its keys.

Baseline uses the latest `status.by_compat_key[k].baseline_low_date` across
explicit keys; any missing date disqualifies the capability. Only `*` rows use
the whole web-feature's date. Bounded dates and introduction versions (`≤`)
use the stated bound conservatively. Prefixes, alternative names, partial
implementations, flags, preview builds and unknown
introduction dates do not count as full browser support. Removal endpoints are
exclusive. A removal in `preview` leaves stable support open-ended.

`CapDisplayP3` is narrower than `CapColorFunction`. Its generator accepts BCD's
explicit Safari 10.1 / iOS 10.3 partial interval whose note says that `display-p3`
and `srgb` are supported. Other partial implementations are still excluded.

Browser IDs are mapped explicitly from MDN names to caniuse-style names in
`main.go`; browsers without a corresponding dataset are unknown, not aliases to
a different engine. Numeric future versions follow recorded open intervals.
The same mapping generates typed `BrowserID` constants (`Chrome`, `Safari`,
`ChromeAndroid`, and the other mapped browsers) for both `Browser` and `.At`.

## Regenerate

From the repository root, with Go, curl, shasum, and tar:

```sh
sh internal/capgen/fetch.sh /path/to/data
go run ./internal/capgen -data /path/to/data
PCSS_CAPS_DATA=/path/to/data go test ./internal/capgen -run TestPinnedRegeneration -v
```

The fetcher pins web-features 3.44.0 and @mdn/browser-compat-data 8.1.7 and
verifies SHA-256 hashes of the npm archives. The generator checks the extracted
`data.json` hashes. Each hash is pinned in one place. Data directories and
archives are not checked in. Generation has no network access or timestamps,
and all output ordering is fixed.

The regular tests need no downloads. The regeneration test runs when
`PCSS_CAPS_DATA` names the extracted releases. To update releases intentionally,
change the pins and hashes in the generator and fetcher, regenerate, and review
the changed support intervals and Baseline dates. Do not reorder assignments.
