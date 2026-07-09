package pristinecss
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
