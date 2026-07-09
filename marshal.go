package pristinecss

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
)

const frameVersion uint16 = 1
const frameHeaderLen = 28

const (
	frameAST byte = iota
	frameIR
)

// IRMasks carries feature masks for one emitted top-level rule.
type IRMasks struct{ Required, Fallback [4]uint64 }

// IRRule is one serve-time rule record over minified CSS bytes.
type IRRule struct {
	Required, Fallback [4]uint64
	Lo, Hi             uint32
}

// IR is a loaded serve-time stylesheet representation.
type IR struct {
	Rules []IRRule
	CSS   []byte
}

// Marshal serializes the sheet as a versioned AST frame. If includeSrc is false, Load
// must be called with the same byte-identical source buffer.
func (s *Sheet) Marshal(includeSrc bool) []byte {
	srcLen := 0
	if includeSrc {
		srcLen = len(s.src)
	}
	nodeBytes := len(s.nodes) * 16
	out := make([]byte, frameHeaderLen+nodeBytes+len(s.scratch)+srcLen)
	writeHeader(out, frameAST, uint32(len(s.nodes)), uint32(len(s.scratch)), uint32(srcLen), checksum(s.src))
	off := frameHeaderLen
	for _, n := range s.nodes {
		out[off] = byte(n.Kind)
		out[off+1] = byte(n.Flags)
		binary.LittleEndian.PutUint16(out[off+2:], n.Aux)
		binary.LittleEndian.PutUint32(out[off+4:], n.Lo)
		binary.LittleEndian.PutUint32(out[off+8:], n.Hi)
		binary.LittleEndian.PutUint32(out[off+12:], n.Sub)
		off += 16
	}
	copy(out[off:], s.scratch)
	off += len(s.scratch)
	if includeSrc {
		copy(out[off:], s.src)
	}
	return out
}

// Load reconstructs a Sheet from a versioned AST frame.
//
// The source buffer is borrowed: the caller must not mutate or free it while the Sheet
// (or any Extracted child) lives. ExtractOwned and Marshal(includeSrc=true) are the
// escape hatches.
func Load(blob []byte, src []byte) (*Sheet, error) {
	kind, a, b, c, sum, err := readHeader(blob)
	if err != nil {
		return nil, err
	}
	if kind != frameAST {
		return nil, errors.New("pristinecss: frame is not an AST blob")
	}
	need := uint64(frameHeaderLen) + uint64(a)*16 + uint64(b) + uint64(c)
	if need != uint64(len(blob)) {
		return nil, errors.New("pristinecss: malformed or truncated AST blob")
	}
	off := frameHeaderLen
	nodes := make([]Node, int(a))
	for i := range nodes {
		nodes[i] = Node{Kind: Kind(blob[off]), Flags: Flags(blob[off+1]), Aux: binary.LittleEndian.Uint16(blob[off+2:]), Lo: binary.LittleEndian.Uint32(blob[off+4:]), Hi: binary.LittleEndian.Uint32(blob[off+8:]), Sub: binary.LittleEndian.Uint32(blob[off+12:])}
		off += 16
	}
	scratch := append([]byte(nil), blob[off:off+int(b)]...)
	off += int(b)
	if c != 0 {
		src = append([]byte(nil), blob[off:off+int(c)]...)
	} else if uint64(len(src)) == 0 && sum != checksum(nil) {
		return nil, errors.New("pristinecss: AST blob omits source; pass the byte-identical source to Load")
	}
	if checksum(src) != sum {
		return nil, errors.New("pristinecss: source does not match AST blob checksum; pass the byte-identical source or regenerate with includeSrc=true")
	}
	mutated, err := validateASTNodes(nodes, src, scratch)
	if err != nil {
		return nil, err
	}
	return &Sheet{src: src, nodes: nodes, scratch: scratch, mutated: mutated}, nil
}

// CompileIR emits minified CSS and records top-level rule spans with optional masks.
func (s *Sheet) CompileIR(masks []IRMasks) ([]byte, error) {
	var rules []IRRule
	css := s.Emit(make([]byte, 0, len(s.src)), EmitOptions{OnRule: func(id NodeID, lo, hi int) {
		var m IRMasks
		if len(masks) > len(rules) {
			m = masks[len(rules)]
		}
		rules = append(rules, IRRule{Required: m.Required, Fallback: m.Fallback, Lo: uint32(lo), Hi: uint32(hi)})
	}})
	out := make([]byte, frameHeaderLen+len(rules)*72+len(css))
	writeHeader(out, frameIR, uint32(len(rules)), uint32(len(css)), 0, checksum(css))
	off := frameHeaderLen
	for _, r := range rules {
		for _, v := range r.Required {
			binary.LittleEndian.PutUint64(out[off:], v)
			off += 8
		}
		for _, v := range r.Fallback {
			binary.LittleEndian.PutUint64(out[off:], v)
			off += 8
		}
		binary.LittleEndian.PutUint32(out[off:], r.Lo)
		binary.LittleEndian.PutUint32(out[off+4:], r.Hi)
		off += 8
	}
	copy(out[off:], css)
	return out, nil
}

// LoadIR reconstructs an IR frame.
func LoadIR(blob []byte) (IR, error) {
	kind, count, cssLen, _, sum, err := readHeader(blob)
	if err != nil {
		return IR{}, err
	}
	if kind != frameIR {
		return IR{}, errors.New("pristinecss: frame is not an IR blob")
	}
	need := uint64(frameHeaderLen) + uint64(count)*72 + uint64(cssLen)
	if need != uint64(len(blob)) {
		return IR{}, errors.New("pristinecss: malformed or truncated IR blob")
	}
	off := frameHeaderLen
	ir := IR{Rules: make([]IRRule, int(count))}
	for i := range ir.Rules {
		for j := 0; j < 4; j++ {
			ir.Rules[i].Required[j] = binary.LittleEndian.Uint64(blob[off:])
			off += 8
		}
		for j := 0; j < 4; j++ {
			ir.Rules[i].Fallback[j] = binary.LittleEndian.Uint64(blob[off:])
			off += 8
		}
		ir.Rules[i].Lo = binary.LittleEndian.Uint32(blob[off:])
		ir.Rules[i].Hi = binary.LittleEndian.Uint32(blob[off+4:])
		off += 8
	}
	ir.CSS = append([]byte(nil), blob[off:off+int(cssLen)]...)
	if checksum(ir.CSS) != sum {
		return IR{}, errors.New("pristinecss: IR CSS checksum mismatch; regenerate the blob")
	}
	for i, rule := range ir.Rules {
		if rule.Lo > rule.Hi || uint64(rule.Hi) > uint64(len(ir.CSS)) {
			return IR{}, fmt.Errorf("pristinecss: IR rule %d has invalid span [%d:%d] for %d CSS bytes", i, rule.Lo, rule.Hi, len(ir.CSS))
		}
	}
	return ir, nil
}

// FilterIR appends rules whose masks are satisfied by capMask.
func FilterIR(ir IR, capMask [4]uint64, dst []byte) []byte {
	for _, r := range ir.Rules {
		if r.Lo > r.Hi || uint64(r.Hi) > uint64(len(ir.CSS)) {
			continue
		}
		ok := true
		for i := 0; i < 4; i++ {
			if r.Required[i]&^capMask[i] != 0 || r.Fallback[i]&capMask[i] != 0 {
				ok = false
				break
			}
		}
		if ok {
			dst = append(dst, ir.CSS[r.Lo:r.Hi]...)
		}
	}
	return dst
}

func validateASTNodes(nodes []Node, src, scratch []byte) (bool, error) {
	if len(nodes) == 0 {
		return false, errors.New("pristinecss: AST frame has no stylesheet root")
	}
	if nodes[0].Kind != KindStylesheet || nodes[0].Sub != uint32(len(nodes)) {
		return false, errors.New("pristinecss: AST frame has an invalid stylesheet root")
	}
	const knownFlags = FlagTombstone | FlagUnwrap | FlagScratch | FlagImportant | FlagCustom | FlagSingleQuote
	type ancestor struct{ id, end int }
	stack := make([]ancestor, 0, 16)
	mutated := len(scratch) != 0
	for i, node := range nodes {
		for len(stack) > 0 && stack[len(stack)-1].end == i {
			stack = stack[:len(stack)-1]
		}
		if i > 0 && len(stack) == 0 {
			return false, fmt.Errorf("pristinecss: AST node %d is outside the stylesheet tree", i)
		}
		if node.Kind >= kindCount {
			return false, fmt.Errorf("pristinecss: AST node %d has invalid kind %d", i, node.Kind)
		}
		if node.Flags&^knownFlags != 0 {
			return false, fmt.Errorf("pristinecss: AST node %d has unknown flags %#x", i, node.Flags)
		}
		if node.Sub == 0 || uint64(i)+uint64(node.Sub) > uint64(len(nodes)) {
			return false, fmt.Errorf("pristinecss: AST node %d has invalid subtree size %d", i, node.Sub)
		}
		end := i + int(node.Sub)
		if len(stack) > 0 && end > stack[len(stack)-1].end {
			return false, fmt.Errorf("pristinecss: AST node %d subtree escapes its parent", i)
		}
		if node.Kind.isOpaque() && node.Sub != 1 {
			return false, fmt.Errorf("pristinecss: opaque AST node %d has children", i)
		}
		lo, hi, ok := sourceSpanFrom(scratch, node)
		if !ok || uint64(hi) > uint64(len(src)) {
			return false, fmt.Errorf("pristinecss: AST node %d has an invalid source span", i)
		}
		if node.Flags&FlagScratch != 0 {
			if uint64(node.Hi) > uint64(len(scratch)) {
				return false, fmt.Errorf("pristinecss: AST node %d has an invalid replacement span", i)
			}
			mutated = true
		}
		if len(stack) > 0 {
			parent := nodes[stack[len(stack)-1].id]
			parentLo, parentHi, _ := sourceSpanFrom(scratch, parent)
			if lo < parentLo || hi > parentHi {
				return false, fmt.Errorf("pristinecss: AST node %d span escapes its parent", i)
			}
		}
		if node.Flags&(FlagTombstone|FlagUnwrap) != 0 {
			mutated = true
		}
		if node.Flags&FlagScratch == 0 {
			usesAux := node.Kind == KindDeclaration || node.Kind == KindValFunction || node.Kind == KindSelPseudo || node.Kind == KindMarginBox || node.Kind.isAtRule()
			prefix := uint64(0)
			if node.Kind == KindMarginBox || node.Kind.isAtRule() {
				prefix = 1
			}
			if usesAux && uint64(node.Lo)+prefix+uint64(node.Aux) > uint64(node.Hi) {
				return false, fmt.Errorf("pristinecss: AST node %d has invalid auxiliary length", i)
			}
		}
		if node.Sub > 1 {
			stack = append(stack, ancestor{id: i, end: end})
		}
	}
	for len(stack) > 0 && stack[len(stack)-1].end == len(nodes) {
		stack = stack[:len(stack)-1]
	}
	if len(stack) != 0 {
		return false, errors.New("pristinecss: AST frame has an incomplete tree")
	}
	return mutated, nil
}

func writeHeader(out []byte, kind byte, a, b, c uint32, sum uint64) {
	copy(out, "PCSS")
	binary.LittleEndian.PutUint16(out[4:], frameVersion)
	out[6] = kind
	binary.LittleEndian.PutUint32(out[8:], a)
	binary.LittleEndian.PutUint32(out[12:], b)
	binary.LittleEndian.PutUint32(out[16:], c)
	binary.LittleEndian.PutUint64(out[20:], sum)
}
func readHeader(blob []byte) (kind byte, a, b, c uint32, sum uint64, err error) {
	if len(blob) < frameHeaderLen || string(blob[:4]) != "PCSS" {
		err = errors.New("pristinecss: not a PCSS blob")
		return
	}
	v := binary.LittleEndian.Uint16(blob[4:])
	if v > frameVersion {
		err = fmt.Errorf("blob version %d, this pristinecss supports ≤%d — regenerate the blob with the version that ships in your build, or upgrade the library", v, frameVersion)
		return
	}
	kind = blob[6]
	a = binary.LittleEndian.Uint32(blob[8:])
	b = binary.LittleEndian.Uint32(blob[12:])
	c = binary.LittleEndian.Uint32(blob[16:])
	sum = binary.LittleEndian.Uint64(blob[20:])
	return
}
func checksum(b []byte) uint64 { h := fnv.New64a(); _, _ = h.Write(b); return h.Sum64() }
