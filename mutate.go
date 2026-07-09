package pristinecss

import "encoding/binary"

const scratchHeaderLen = 8

// Delete marks a node and its subtree as logically deleted. NodeIDs remain stable until Compact.
func (s *Sheet) Delete(id NodeID) {
	if !s.valid(id) {
		return
	}
	s.nodes[id].Flags |= FlagTombstone
	s.mutated = true
}

// Unwrap makes a container transparent during traversal and emission. NodeIDs remain stable until Compact.
func (s *Sheet) Unwrap(id NodeID) {
	if !s.valid(id) {
		return
	}
	if id == 0 {
		return
	}
	k := s.nodes[id].Kind
	if kinds[k].class == classLeaf || kinds[k].class == classDeclaration {
		panic("pristinecss: cannot unwrap non-container kind " + k.String())
	}
	s.nodes[id].Flags |= FlagUnwrap
	s.mutated = true
}

// ReplaceRaw replaces a node's presentation with raw bytes in sheet-owned scratch storage.
// The node's Kind and Sub are preserved, so rule callbacks, sibling arithmetic, and held
// NodeIDs remain stable until Compact.
func (s *Sheet) ReplaceRaw(id NodeID, text []byte) {
	if !s.valid(id) {
		return
	}
	n := s.nodes[id]
	originLo, originHi, ok := sourceSpanFrom(s.scratch, n)
	if !ok {
		return
	}
	lo, hi := appendReplacement(&s.scratch, originLo, originHi, text)
	n.Flags |= FlagScratch
	n.Lo, n.Hi = lo, hi
	s.nodes[id] = n
	s.mutated = true
}

// Compact rebuilds the node array and scratch storage, dropping tombstones, superseded
// replacements, and raw-node descendants while physically splicing unwrapped block children.
// It returns an old-to-new ID remap; NodeIDs are stable across all other mutations and
// unstable after Compact.
func (s *Sheet) Compact() []NodeID {
	remap := make([]NodeID, len(s.nodes))
	for i := range remap {
		remap[i] = NoNode
	}
	oldNodes, oldScratch := s.nodes, s.scratch
	if len(oldNodes) > 0 && oldNodes[0].Flags&FlagTombstone != 0 {
		s.nodes = []Node{{Kind: KindStylesheet, Lo: 0, Hi: uint32(len(s.src)), Sub: 1}}
		s.scratch = nil
		return remap
	}
	out := make([]Node, 0, len(oldNodes))
	var newScratch []byte
	var copyNode func(NodeID) NodeID
	copyNode = func(id NodeID) NodeID {
		n := oldNodes[id]
		if n.Flags&FlagTombstone != 0 {
			return NoNode
		}
		if n.Flags&FlagUnwrap != 0 {
			if b, ok := s.blockChildIn(oldNodes, id); ok {
				first := NodeID(len(out))
				for c := b + 1; c < b+NodeID(oldNodes[b].Sub); c += NodeID(oldNodes[c].Sub) {
					copyNode(c)
				}
				return first
			}
			return NoNode
		}
		newID := NodeID(len(out))
		remap[id] = newID
		out = append(out, n)
		out[newID].Flags &^= FlagUnwrap
		if n.Flags&FlagScratch != 0 {
			originLo, originHi, _ := sourceSpanFrom(oldScratch, n)
			lo, hi := appendReplacement(&newScratch, originLo, originHi, oldScratch[n.Lo:n.Hi])
			out[newID].Lo, out[newID].Hi, out[newID].Sub = lo, hi, 1
			return newID
		}
		if n.Kind == KindValRaw {
			out[newID].Sub = 1
			return newID
		}
		end := id + NodeID(n.Sub)
		for c := id + 1; c < end; c += NodeID(oldNodes[c].Sub) {
			copyNode(c)
		}
		out[newID].Sub = uint32(len(out)) - uint32(newID)
		if out[newID].Sub == 0 {
			out[newID].Sub = 1
		}
		return newID
	}
	if len(oldNodes) > 0 {
		copyNode(0)
	}
	s.nodes, s.scratch = out, newScratch
	return remap
}

// Extract copies a contiguous subtree into a new Sheet that borrows the same source buffer.
func (s *Sheet) Extract(id NodeID) *Sheet {
	if !s.valid(id) {
		return &Sheet{}
	}
	n := s.nodes[id]
	base, hi, ok := sourceSpanFrom(s.scratch, n)
	if !ok {
		return &Sheet{}
	}
	child := append([]Node(nil), s.nodes[id:id+NodeID(n.Sub)]...)
	var childScratch []byte
	for i := range child {
		original := s.nodes[id+NodeID(i)]
		if original.Flags&FlagScratch != 0 {
			originLo, originHi, _ := sourceSpanFrom(s.scratch, original)
			if originLo >= base {
				originLo -= base
				originHi -= base
			}
			lo, replacementHi := appendReplacement(&childScratch, originLo, originHi, s.scratch[original.Lo:original.Hi])
			child[i].Lo, child[i].Hi = lo, replacementHi
		} else if child[i].Lo >= base {
			child[i].Lo -= base
			child[i].Hi -= base
		}
	}
	rootHi := uint32(0)
	if hi >= base {
		rootHi = hi - base
	}
	nodes := make([]Node, 1, len(child)+1)
	nodes[0] = Node{Kind: KindStylesheet, Lo: 0, Hi: rootHi, Sub: uint32(len(child) + 1)}
	nodes = append(nodes, child...)
	src := s.src[base:hi]
	return &Sheet{src: src, nodes: nodes, scratch: childScratch, mutated: s.mutated || len(childScratch) != 0}
}

// ExtractOwned copies a subtree and its referenced bytes so it can outlive the source buffer.
func (s *Sheet) ExtractOwned(id NodeID) *Sheet {
	ex := s.Extract(id)
	ex.src = append([]byte(nil), ex.src...)
	ex.scratch = append([]byte(nil), ex.scratch...)
	return ex
}

func appendReplacement(dst *[]byte, originLo, originHi uint32, text []byte) (uint32, uint32) {
	start := len(*dst)
	*dst = append(*dst, make([]byte, scratchHeaderLen)...)
	binary.LittleEndian.PutUint32((*dst)[start:], originLo)
	binary.LittleEndian.PutUint32((*dst)[start+4:], originHi)
	lo := uint32(len(*dst))
	*dst = append(*dst, text...)
	return lo, uint32(len(*dst))
}

func sourceSpanFrom(scratch []byte, n Node) (uint32, uint32, bool) {
	if n.Flags&FlagScratch == 0 {
		return n.Lo, n.Hi, n.Hi >= n.Lo
	}
	if n.Lo < scratchHeaderLen || n.Hi < n.Lo || int(n.Hi) > len(scratch) {
		return 0, 0, false
	}
	header := n.Lo - scratchHeaderLen
	lo := binary.LittleEndian.Uint32(scratch[header:])
	hi := binary.LittleEndian.Uint32(scratch[header+4:])
	return lo, hi, hi >= lo
}

func (s *Sheet) valid(id NodeID) bool { return s != nil && int(id) < len(s.nodes) }
func (s *Sheet) blockChildIn(nodes []Node, id NodeID) (NodeID, bool) {
	n := nodes[id]
	end := id + NodeID(n.Sub)
	for c := id + 1; c < end; c += NodeID(nodes[c].Sub) {
		if nodes[c].Kind == KindBlock {
			return c, true
		}
	}
	return NoNode, false
}
