package pristinecss

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
	"strings"
)

// Style is a read-only binary stylesheet, safe to share across goroutines.
// Build one with ToBinary, then use Open or Load.
type Style struct{ blob string }

// The spans format has a little-endian u32 header: magic, source length, event
// count, index length, minified length, pretty length. Sections follow in order:
// index, source, minified, pretty, then CRC-32C over all preceding bytes.
// Index events are kind/flags followed by five uvarints: subtree count, delta
// source start, name length, gap to value, value length.
const styleHeaderLen = 24

var checksumTable = crc32.MakeTable(crc32.Castagnoli)

func checksum(b []byte) uint32 { return crc32.Checksum(b, checksumTable) }

// ToBinary builds a spans binary from CSS, returning Parse's diagnostics.
// Even malformed CSS produces a best-effort binary, just as Parse returns a Sheet.
// The result owns its bytes and does not retain css.
func ToBinary(css []byte) ([]byte, []ParseError) {
	s, diags := Parse(css)
	spans := styleSpans(s)
	var index []byte
	var prev uint32
	for _, e := range spans {
		index = append(index, byte(e.kind), byte(e.flags))
		for _, n := range []uint32{e.sub, e.lo - prev, e.nameEnd - e.lo, e.valueLo - e.nameEnd, e.valueEnd - e.valueLo} {
			index = binary.AppendUvarint(index, uint64(n))
		}
		prev = e.lo
	}

	mini := s.Emit(nil, EmitOptions{Style: Minified})
	pretty := s.Emit(nil, EmitOptions{Style: Pretty})

	out := make([]byte, styleHeaderLen)
	copy(out, "PCS1")
	for i, n := range []int{len(css), len(spans), len(index), len(mini), len(pretty)} {
		binary.LittleEndian.PutUint32(out[4+4*i:], uint32(n))
	}
	out = append(out, index...)
	out = append(out, css...)
	out = append(out, mini...)
	out = append(out, pretty...)
	out = binary.LittleEndian.AppendUint32(out, checksum(out))
	return out, diags
}

// Open opens a trusted binary without validation or copying, in constant time.
// Only use it for binaries your own program built, such as a generated constant.
// Use Load for anything else.
func Open(b string) *Style { return &Style{blob: b} }

// Load checks a spans binary, rejecting corruption and invalid structure.
// It owns an immutable copy: callers may reuse b after Load returns.
// It checks structure and integrity, but cannot prove the stored output matches
// what Sheet.Emit would produce for the stored source.
func Load(b []byte) (*Style, error) {
	bad := errors.New("pristinecss: invalid spans binary; regenerate with ToBinary")
	if len(b) < styleHeaderLen+4 || string(b[:4]) != "PCS1" {
		return nil, bad
	}
	end := len(b) - 4
	if checksum(b[:end]) != binary.LittleEndian.Uint32(b[end:]) {
		return nil, bad
	}
	field := func(off int) uint64 { return uint64(binary.LittleEndian.Uint32(b[off:])) }
	srcLen, count, indexLen, minLen, prettyLen := field(4), field(8), field(12), field(16), field(20)
	if uint64(styleHeaderLen)+indexLen+srcLen+minLen+prettyLen != uint64(end) {
		return nil, bad
	}
	i := styleHeaderLen + int(indexLen)
	if !validStyleIndex(b[styleHeaderLen:i], count, srcLen) {
		return nil, bad
	}
	return Open(string(b)), nil
}

// Source returns the exact original CSS, borrowing immutable binary storage.
func (s *Style) Source() string {
	i := styleHeaderLen + int(styleU32(s.blob, 12))
	return s.blob[i : i+int(styleU32(s.blob, 4))]
}

// CSS appends minified CSS to dst, or pretty CSS when passed Pretty.
// Only the Minified and Pretty presets are supported, not custom EmitStyle
// settings. With enough capacity in dst it allocates nothing. caps must be Modern.
func (s *Style) CSS(dst []byte, caps Caps, opts ...EmitStyle) []byte {
	b := s.blob
	off := styleHeaderLen + int(styleU32(b, 12)) + int(styleU32(b, 4))
	n := int(styleU32(b, 16))
	if len(opts) > 0 && len(opts[len(opts)-1].Newline) != 0 {
		off += n
		n = int(styleU32(b, 20))
	}
	return append(dst, b[off:off+n]...)
}

func styleU32(s string, off int) uint32 {
	return uint32(s[off]) | uint32(s[off+1])<<8 | uint32(s[off+2])<<16 | uint32(s[off+3])<<24
}

func validStyleIndex(b []byte, count, srcLen uint64) bool {
	var lo uint64
	var ends []uint64
	for i := uint64(0); i < count; i++ {
		for len(ends) > 0 && ends[len(ends)-1] == i {
			ends = ends[:len(ends)-1]
		}
		if len(b) < 2 || !styleSelected(Kind(b[0])) || Flags(b[1]) & ^(FlagImportant|FlagCustom) != 0 {
			return false
		}
		kind := Kind(b[0])
		b = b[2:]
		var v [5]uint64
		for j := range v {
			x, n := binary.Uvarint(b)
			if n <= 0 {
				return false
			}
			v[j], b = x, b[n:]
		}
		sub := v[0]
		if sub == 0 || sub > count-i || (len(ends) > 0 && sub > ends[len(ends)-1]-i) {
			return false
		}
		if (kind == KindDeclaration || kind == KindRawStatement) && sub != 1 {
			return false
		}
		// Subtract before adding so even malicious uint64 varints cannot wrap.
		for _, size := range v[1:] {
			if size > srcLen-lo {
				return false
			}
			lo += size
		}
		lo -= v[2] + v[3] + v[4] // Next event's delta is relative to its start.
		if sub > 1 {
			ends = append(ends, i+sub)
		}
	}
	return len(b) == 0
}

type styleSpan struct {
	kind                                Kind
	flags                               Flags
	sub, lo, nameEnd, valueLo, valueEnd uint32
}

func styleSelected(k Kind) bool {
	return k == KindDeclaration || k == KindStyleRule || k.isAtRule() || k == KindKeyframeBlock || k == KindMarginBox || k == KindRawStatement
}

// Selector and value AST nodes are omitted; spans refer to the original source.
func styleSpans(s *Sheet) []styleSpan {
	prefix := make([]uint32, len(s.nodes)+1)
	for i, n := range s.nodes {
		prefix[i+1] = prefix[i]
		if styleSelected(n.Kind) {
			prefix[i+1]++
		}
	}
	var out []styleSpan
	for i, n := range s.nodes {
		if !styleSelected(n.Kind) {
			continue
		}
		e := styleSpan{kind: n.Kind, flags: n.Flags, sub: prefix[i+int(n.Sub)] - prefix[i], lo: n.Lo}
		if n.Kind == KindDeclaration {
			e.nameEnd = n.Lo + uint32(n.Aux)
			if n.Sub > 1 {
				e.valueLo = s.nodes[i+1].Lo
				e.valueEnd = e.valueLo
				for _, child := range s.nodes[i+1 : i+int(n.Sub)] {
					e.valueEnd = max(e.valueEnd, child.Hi)
				}
			} else {
				// Aux avoids confusing escaped colons with the property separator.
				tail := string(s.src[e.nameEnd:n.Hi])
				_, after, found := strings.Cut(tail, ":")
				e.valueLo, e.valueEnd = e.nameEnd, e.nameEnd
				if found {
					start := n.Hi - uint32(len(after))
					value := strings.Trim(strings.TrimSuffix(after, ";"), " \t\r\n\f")
					left := len(after) - len(strings.TrimLeft(after, " \t\r\n\f"))
					e.valueLo = start + uint32(left)
					e.valueEnd = e.valueLo + uint32(len(value))
				}
			}
		} else {
			end := n.Hi
			if block, ok := s.blockChild(NodeID(i)); ok {
				end = s.nodes[block].Lo
			}
			e.nameEnd = n.Lo + uint32(len(strings.TrimRight(string(s.src[n.Lo:end]), " \t\r\n\f")))
			e.valueLo, e.valueEnd = e.nameEnd, e.nameEnd
		}
		out = append(out, e)
	}
	return out
}
