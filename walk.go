package pristinecss

import "iter"

// Action controls Walk traversal after a Cursor is visited.
type Action uint8

const (
	// Continue descends into the current node's children.
	Continue Action = iota
	// SkipChildren skips the current node's subtree.
	SkipChildren
	// Stop stops the walk immediately.
	Stop
)

// Cursor is an ergonomic read-only handle to one node in a Sheet.
type Cursor struct {
	s  *Sheet
	id NodeID
}

// ID returns the node ID.
func (c Cursor) ID() NodeID { return c.id }

// Kind returns the node kind.
func (c Cursor) Kind() Kind { return c.s.nodes[c.id].Kind }

// Flags returns the node flags.
func (c Cursor) Flags() Flags { return c.s.nodes[c.id].Flags }

// Aux returns the node's per-kind auxiliary payload.
func (c Cursor) Aux() uint16 { return c.s.nodes[c.id].Aux }

// Span returns the node byte span.
func (c Cursor) Span() (lo, hi uint32) { n := c.s.nodes[c.id]; return n.Lo, n.Hi }

// Text returns the node's source or scratch bytes.
func (c Cursor) Text() []byte {
	n := c.s.nodes[c.id]
	buf := c.s.src
	if n.Flags&FlagScratch != 0 {
		buf = c.s.scratch
	}
	if int(n.Lo) > len(buf) || n.Hi < n.Lo {
		return nil
	}
	hi := int(n.Hi)
	if hi > len(buf) {
		hi = len(buf)
	}
	return buf[n.Lo:uint32(hi)]
}

// Name returns the leading Aux bytes of Text, used by declarations and named constructs.
func (c Cursor) Name() []byte {
	n := c.s.nodes[c.id]
	if n.Kind == KindAtGeneric && n.Flags&FlagScratch == 0 {
		lo := n.Lo + 1
		hi := lo + uint32(n.Aux)
		if int(hi) <= len(c.s.src) {
			return c.s.src[lo:hi]
		}
	}
	txt := c.Text()
	if int(c.Aux()) > len(txt) {
		return txt
	}
	return txt[:c.Aux()]
}

// PropertyName returns a declaration property name.
func (c Cursor) PropertyName() []byte {
	if c.Kind() != KindDeclaration {
		return nil
	}
	return c.Name()
}

// FunctionName returns a value function name.
func (c Cursor) FunctionName() []byte {
	if c.Kind() != KindValFunction {
		return nil
	}
	return c.Name()
}

// FirstChild returns the first physical child cursor.
func (c Cursor) FirstChild() (Cursor, bool) {
	n := c.s.nodes[c.id]
	if n.Sub <= 1 || n.Kind.isOpaque() {
		return Cursor{}, false
	}
	return Cursor{c.s, c.id + 1}, true
}

// NextSibling returns the next physical sibling cursor if it is still within the sheet.
func (c Cursor) NextSibling() (Cursor, bool) {
	parent, ok := c.s.parentOf(c.id)
	if !ok {
		return Cursor{}, false
	}
	next := c.id + NodeID(c.s.nodes[c.id].Sub)
	if next >= parent+NodeID(c.s.nodes[parent].Sub) {
		return Cursor{}, false
	}
	return Cursor{c.s, next}, true
}

// Children returns the node's physical children in pre-order sibling order.
func (c Cursor) Children() iter.Seq[Cursor] {
	return func(yield func(Cursor) bool) {
		n := c.s.nodes[c.id]
		if n.Sub <= 1 || n.Kind.isOpaque() {
			return
		}
		end := c.id + NodeID(n.Sub)
		for id := c.id + 1; id < end; id += NodeID(c.s.nodes[id].Sub) {
			if !yield(Cursor{c.s, id}) {
				return
			}
		}
	}
}

// Walk visits every visible node in document order. Tombstoned subtrees are skipped;
// unwrapped containers present their block children at the parent's level.
func (s *Sheet) Walk(fn func(Cursor) Action) {
	if s == nil || len(s.nodes) == 0 {
		return
	}
	stop := false
	var visitChildren func(NodeID, NodeID)
	visitChildren = func(start, end NodeID) {
		for id := start; id < end && !stop; id += NodeID(s.nodes[id].Sub) {
			n := s.nodes[id]
			if n.Flags&FlagTombstone != 0 {
				continue
			}
			if n.Flags&FlagUnwrap != 0 {
				if b, ok := s.blockChild(id); ok {
					bn := s.nodes[b]
					visitChildren(b+1, b+NodeID(bn.Sub))
				}
				continue
			}
			a := fn(Cursor{s, id})
			switch a {
			case Stop:
				stop = true
			case SkipChildren:
				continue
			default:
				if !n.Kind.isOpaque() && n.Sub > 1 {
					visitChildren(id+1, id+NodeID(n.Sub))
				}
			}
		}
	}
	visitChildren(1, NodeID(len(s.nodes)))
}

// All returns every visible node with the requested kind.
func (s *Sheet) All(k Kind) iter.Seq[Cursor] {
	return func(yield func(Cursor) bool) {
		s.Walk(func(c Cursor) Action {
			if c.Kind() == k && !yield(c) {
				return Stop
			}
			return Continue
		})
	}
}

// Parents builds an O(n) sidecar mapping every node to its physical parent.
func (s *Sheet) Parents() []NodeID {
	parents := make([]NodeID, len(s.nodes))
	for i := range parents {
		parents[i] = NoNode
	}
	var rec func(NodeID)
	rec = func(id NodeID) {
		n := s.nodes[id]
		if n.Kind.isOpaque() {
			return
		}
		end := id + NodeID(n.Sub)
		for c := id + 1; c < end; c += NodeID(s.nodes[c].Sub) {
			parents[c] = id
			rec(c)
		}
	}
	if len(s.nodes) > 0 {
		rec(0)
	}
	return parents
}

func (s *Sheet) parentOf(id NodeID) (NodeID, bool) {
	if s == nil || id == 0 || int(id) >= len(s.nodes) {
		return NoNode, false
	}
	parent := NoNode
	for current := NodeID(0); current < id; {
		n := s.nodes[current]
		if id < current+NodeID(n.Sub) && !n.Kind.isOpaque() {
			parent = current
			current++
		} else {
			current += NodeID(n.Sub)
		}
	}
	return parent, parent != NoNode
}

func (s *Sheet) blockChild(id NodeID) (NodeID, bool) {
	n := s.nodes[id]
	end := id + NodeID(n.Sub)
	for c := id + 1; c < end; c += NodeID(s.nodes[c].Sub) {
		if s.nodes[c].Kind == KindBlock {
			return c, true
		}
	}
	return NoNode, false
}
