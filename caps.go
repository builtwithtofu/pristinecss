package pristinecss

import "strings"

// Cap is one thing a browser may support. Numbers are append-only; see
// internal/capgen/capabilities.tsv for the stable wire assignments and sources.
type Cap uint16

// Caps is a comparable, pointer-free set of 256 capabilities.
// The zero value describes no support. Unknown handshake bits are preserved.
type Caps struct{ bits [4]uint64 }

// Has reports whether bit x is set. Values outside the 256-bit set return false.
func (c Caps) Has(x Cap) bool {
	return x < 256 && c.bits[x/64]&(uint64(1)<<(x%64)) != 0
}

// Bits returns the handshake representation, least-significant bit first in each word.
func (c Caps) Bits() [4]uint64 { return c.bits }

// CapsFromBits constructs a set without discarding unknown bits.
func CapsFromBits(b [4]uint64) Caps { return Caps{bits: b} }

// Modern sets every bit and requests CSS without lowering.
var Modern = CapsFromBits([4]uint64{^uint64(0), ^uint64(0), ^uint64(0), ^uint64(0)})

// BrowserID uses caniuse-style agent names: chrome, firefox, safari, ios_saf,
// and_chr, and_ff, edge, ie, samsung, opera and op_mob.
type BrowserID string

// Baseline returns capabilities whose Baseline low dates fall in or before year.
// Explicit BCD keys must all be Baseline; whole-feature rows use the feature date.
// Missing dates never qualify; bounded dates use their stated year.
func Baseline(year int) Caps {
	var c Caps
	for x, low := range capBaseline {
		if low != 0 && int(low) <= year {
			c.bits[x/64] |= uint64(1) << (x % 64)
		}
	}
	return c
}

// Browser returns full, unprefixed, unflagged support recorded in the pinned
// compatibility data. Versions are one to four decimal components (0..65535),
// compared numerically with missing components zero.
// Unknown IDs and other malformed versions (including preview) return an empty
// set. Future numeric versions follow recorded open intervals, not a prediction
// of future support. A BCD introduction bounded by ≤ is accepted only from the
// stated bound; unknown introduction dates and preview-only support do not qualify.
func Browser(id BrowserID, version string) Caps {
	v, ok := capVersion(version)
	if !ok {
		return Caps{}
	}
	var c Caps
	for _, s := range capBrowsers[id] {
		if v >= s.from && (s.to == 0 || v < s.to) {
			c.bits[s.cap/64] |= uint64(1) << (s.cap % 64)
		}
	}
	return c
}

type capSupport struct {
	cap      Cap
	from, to uint64 // packed four-component versions; to is exclusive, zero is unbounded
}

func capVersion(s string) (uint64, bool) {
	var v uint64
	for i := range 4 {
		part, rest, more := strings.Cut(s, ".")
		if part == "" {
			return 0, false
		}
		var n uint64
		for j := range len(part) {
			if part[j] < '0' || part[j] > '9' {
				return 0, false
			}
			n = n*10 + uint64(part[j]-'0')
			if n > 65535 {
				return 0, false
			}
		}
		v |= n << (48 - 16*i)
		if !more {
			return v, true
		}
		s = rest
	}
	return 0, false
}
