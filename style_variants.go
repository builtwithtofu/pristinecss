package pristinecss

import (
	"bytes"
	"cmp"
	"encoding/binary"
	"math/bits"
	"slices"
	"strings"
)

// PCS2 keeps the PCS1 header, index, source and modern outputs. Before the
// checksum it adds a u32 variant count, then entries holding required Caps
// (four u64s), four u32s (minified offset/length, pretty offset/length), and
// a u32 selection score (richness, native colour support, then P3 support).
// Entry zero refers to the existing modern output; other outputs follow the
// table contiguously. PCS1 is still emitted when no extra variants are needed.
const (
	variantHeaderLen = 4
	variantEntryLen  = 52
)

// Key identifies this binary and the output selected for a capability set.
// Browsers selecting the same variant share a key.
type Key struct {
	Hash    uint32 // Stored CRC-32C of the binary; not a cryptographic identity.
	Variant uint32
}

func (s *Style) Key(caps Caps) Key {
	key := Key{Hash: styleU32(s.blob, len(s.blob)-4)}
	if s.blob[3] == '2' {
		table := styleHeaderLen + int(styleU32(s.blob, 12)) + int(styleU32(s.blob, 4)) + int(styleU32(s.blob, 16)) + int(styleU32(s.blob, 20))
		key.Variant = s.variant(caps, table)
	}
	return key
}

// Safe variants prefer richness, then native colour support, then P3.
// The table's canonical capability-mask order breaks any remaining ties.
// Unresolved expressions remain reported gaps; fidelity skips require their Cap.
func (s *Style) variant(caps Caps, table int) uint32 {
	var best, fallback uint32
	bestRank, lowRank := -1, 257*8
	for i := uint32(0); i < styleU32(s.blob, table); i++ {
		entry := table + variantHeaderLen + int(i)*variantEntryLen
		safe, rank := true, int(styleU32(s.blob, entry+48))
		for word := range 4 {
			need := uint64(styleU32(s.blob, entry+word*8)) | uint64(styleU32(s.blob, entry+word*8+4))<<32
			safe = safe && need&^caps.bits[word] == 0
		}
		if safe && rank > bestRank {
			best, bestRank = i, rank
		}
		if rank < lowRank {
			fallback, lowRank = i, rank
		}
	}
	if bestRank < 0 {
		return fallback
	}
	return best
}

func buildVariants(out, css []byte, spans []styleSpan, o Options, report *Report) []byte {
	source := string(css)
	if len(report.Changes) != 0 {
		// Blanking deletions preserves original coordinates for diagnostics.
		css = bytes.Clone(css)
		for _, change := range report.Changes {
			for i := change.Lo; i < change.Hi; i++ {
				css[i] = ' '
			}
		}
	}
	src := string(css)
	var original Caps
	for cap := Cap(0); cap < 256; cap++ {
		if r, ok := rewrites[cap]; !ok || r.lower == nil {
			continue
		}
		if scanLower(src, spans, cap, o.MinFidelity, Caps{}).used {
			original.bits[cap/64] |= uint64(1) << (cap % 64)
		}
	}
	used := original
	if original.Has(CapOklch) || original.Has(CapOklab) {
		used.bits[CapDisplayP3/64] |= uint64(1) << (CapDisplayP3 % 64)
	}
	targets := []Caps{used}
	seen := make(map[Target]bool)
	for _, target := range o.Lower {
		if seen[target] {
			continue
		}
		seen[target] = true
		for _, change := range report.Changes {
			if target.Caps.Has(change.Cap) {
				continue
			}
			name, _, _ := strings.Cut(source[change.Lo:change.Hi], ":")
			name = strings.TrimSpace(name)
			if targetReadPrefix(name, change.Cap, target) {
				report.Gaps = append(report.Gaps, Gap{
					Lo: change.Lo, Hi: change.Hi, Cap: change.Cap, Target: target,
					What: "dropped prefix cannot be restored: no lowering for this capability",
				})
			}
		}
		var projected Caps
		for word := range 4 {
			projected.bits[word] = target.Caps.bits[word] & used.bits[word]
		}
		for cap := Cap(0); cap < 256; cap++ {
			if !original.Has(cap) || target.Caps.Has(cap) {
				continue
			}
			w := scanLower(src, spans, cap, o.MinFidelity, target.Caps)
			for _, gap := range w.gaps {
				gap.Target = target
				report.Gaps = append(report.Gaps, gap)
			}
			for _, mapping := range w.mappings {
				mapping.Target = target
				report.Mappings = append(report.Mappings, mapping)
			}
		}
		if !slices.Contains(targets, projected) {
			targets = append(targets, projected)
		}
	}
	slices.SortStableFunc(report.Gaps, func(a, b Gap) int { return cmp.Compare(a.Lo, b.Lo) })
	slices.SortStableFunc(report.Mappings, func(a, b Mapping) int { return cmp.Compare(a.Lo, b.Lo) })
	slices.SortFunc(targets[1:], func(a, b Caps) int {
		for word := 3; word >= 0; word-- {
			if order := cmp.Compare(a.bits[word], b.bits[word]); order != 0 {
				return order
			}
		}
		return 0
	})
	report.Variants = len(targets)
	if len(targets) == 1 {
		return out
	}
	out[3] = '2'
	table := len(out)
	out = binary.LittleEndian.AppendUint32(out, uint32(len(targets)))
	out = append(out, make([]byte, len(targets)*variantEntryLen)...)
	miniOff := styleHeaderLen + int(binary.LittleEndian.Uint32(out[12:])) + len(css)
	miniLen := int(binary.LittleEndian.Uint32(out[16:]))
	prettyLen := int(binary.LittleEndian.Uint32(out[20:]))
	putVariant(out, table+variantHeaderLen, original, used, miniOff, miniLen, miniOff+miniLen, prettyLen)
	for i, target := range targets[1:] {
		text, currentSpans := src, spans
		var needed Caps
		for word := range 4 {
			needed.bits[word] = target.bits[word] & original.bits[word]
		}
		for cap := Cap(0); cap < 256; cap++ {
			if !original.Has(cap) || target.Has(cap) {
				continue
			}
			w := scanLower(text, currentSpans, cap, o.MinFidelity, target)
			if w.fidelityGap {
				needed.bits[cap/64] |= uint64(1) << (cap % 64)
			}
			if len(w.edits) != 0 {
				if (cap == CapOklch || cap == CapOklab) && target.Has(CapDisplayP3) {
					needed.bits[CapDisplayP3/64] |= uint64(1) << (CapDisplayP3 % 64)
				}
				changed := w.emit(text)
				sheet, _ := Parse(changed)
				currentSpans = styleSpans(sheet)
				text = string(changed)
			}
		}
		sheet, _ := Parse([]byte(text))
		miniOff := len(out)
		out = sheet.Emit(out, EmitOptions{Style: Minified})
		prettyOff := len(out)
		out = sheet.Emit(out, EmitOptions{Style: Pretty})
		putVariant(out, table+variantHeaderLen+(i+1)*variantEntryLen, needed, target, miniOff, prettyOff-miniOff, prettyOff, len(out)-prettyOff)
	}
	return out
}

func putVariant(out []byte, entry int, needed, projected Caps, miniOff, miniLen, prettyOff, prettyLen int) {
	rank := 0
	for _, word := range projected.bits {
		rank += bits.OnesCount64(word)
	}
	// Colour preference fits in three bits: two native capabilities and P3.
	colour := 0
	for _, cap := range [...]Cap{CapOklch, CapOklab} {
		if needed.Has(cap) {
			colour += 2
		}
	}
	if needed.Has(CapDisplayP3) {
		colour++
	}
	binary.LittleEndian.PutUint32(out[entry+48:], uint32(rank*8+colour))
	for word, n := range needed.bits {
		binary.LittleEndian.PutUint64(out[entry+word*8:], n)
	}
	for i, n := range [...]int{miniOff, miniLen, prettyOff, prettyLen} {
		binary.LittleEndian.PutUint32(out[entry+32+i*4:], uint32(n))
	}
}

func validVariants(b []byte, table, end int) bool {
	if end-table < variantHeaderLen {
		return false
	}
	count := uint64(binary.LittleEndian.Uint32(b[table:]))
	if count < 2 || count*variantEntryLen > uint64(end-table-variantHeaderLen) {
		return false
	}
	pos := uint64(table) + variantHeaderLen + count*variantEntryLen
	miniOff := uint64(styleHeaderLen) + uint64(binary.LittleEndian.Uint32(b[12:])) + uint64(binary.LittleEndian.Uint32(b[4:]))
	miniLen, prettyLen := binary.LittleEndian.Uint32(b[16:]), binary.LittleEndian.Uint32(b[20:])
	for i := uint64(0); i < count; i++ {
		entry := uint64(table) + variantHeaderLen + i*variantEntryLen + 32
		for format := range 2 {
			off := uint64(binary.LittleEndian.Uint32(b[entry+uint64(format*8):]))
			size := uint64(binary.LittleEndian.Uint32(b[entry+uint64(format*8)+4:]))
			if i == 0 {
				wantOff, wantLen := miniOff, uint64(miniLen)
				if format == 1 {
					wantOff, wantLen = miniOff+uint64(miniLen), uint64(prettyLen)
				}
				if off != wantOff || size != wantLen {
					return false
				}
			} else {
				if off != pos || size > uint64(end)-pos {
					return false
				}
				pos += size
			}
		}
	}
	return pos == uint64(end)
}
