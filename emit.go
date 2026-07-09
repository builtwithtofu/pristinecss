package pristinecss

import "bytes"

// Style injects whitespace choices into Emit. The zero value is minified.
type Style struct{ Indent, Newline, AfterColon, AfterComma, BeforeBrace, AfterBrace []byte }

// Minified is the zero-whitespace output style.
var Minified Style

// Pretty emits two-space indentation and readable separators.
var Pretty = Style{Indent: []byte("  "), Newline: []byte("\n"), AfterColon: []byte(" "), AfterComma: []byte(" "), BeforeBrace: []byte(" "), AfterBrace: []byte("\n")}

// EmitOptions configures Sheet.Emit.
type EmitOptions struct {
	Style    Style
	Verbatim bool
	Filter   func(Cursor) bool
	OnRule   func(id NodeID, lo, hi int)
}

// Emit appends CSS for the sheet to dst. It performs no allocation when dst has capacity.
func (s *Sheet) Emit(dst []byte, o EmitOptions) []byte {
	if s == nil || len(s.nodes) == 0 {
		return dst
	}
	root := s.nodes[0]
	if root.Flags&FlagTombstone != 0 {
		return dst
	}
	if root.Flags&FlagScratch != 0 {
		return append(dst, s.nodeBytes(root)...)
	}
	if o.Verbatim && !s.mutated && o.Filter == nil {
		return append(dst, s.src...)
	}
	return s.emitChildren(dst, 1, NodeID(len(s.nodes)), o, 0, true)
}

func (s *Sheet) emitChildren(dst []byte, start, end NodeID, o EmitOptions, depth int, top bool) []byte {
	for id := start; id < end; id += NodeID(s.nodes[id].Sub) {
		n := s.nodes[id]
		if n.Flags&FlagTombstone != 0 {
			continue
		}
		if n.Flags&FlagUnwrap != 0 {
			if b, ok := s.blockChild(id); ok {
				dst = s.emitChildren(dst, b+1, b+NodeID(s.nodes[b].Sub), o, depth, top)
			}
			continue
		}
		if o.Filter != nil && !o.Filter(Cursor{s, id}) {
			continue
		}
		lo := len(dst)
		dst = s.emitNode(dst, id, o, depth)
		if top && o.OnRule != nil && (n.Kind == KindStyleRule || n.Kind.isAtRule()) {
			o.OnRule(id, lo, len(dst))
		}
	}
	return dst
}

func (s *Sheet) emitNode(dst []byte, id NodeID, o EmitOptions, depth int) []byte {
	n := s.nodes[id]
	if n.Flags&FlagScratch != 0 || n.Kind == KindValRaw && n.Sub > 1 {
		return append(dst, s.nodeBytes(n)...)
	}
	switch {
	case n.Kind == KindComment:
		b := s.nodeBytes(n)
		if len(b) >= 3 && bytes.HasPrefix(b, []byte("/*!")) || len(o.Style.Newline) != 0 || o.Verbatim {
			return append(dst, b...)
		}
		return dst
	case n.Kind == KindStyleRule || n.Kind == KindKeyframeBlock || n.Kind == KindMarginBox:
		block, ok := s.blockChild(id)
		if !ok {
			return minifyAppend(dst, s.nodeBytes(n), o.Style)
		}
		dst = appendIndent(dst, o.Style, depth)
		if n.Kind == KindMarginBox {
			dst = append(dst, s.atName(n)...)
			dst = append(dst, o.Style.BeforeBrace...)
			dst = append(dst, '{')
			dst = append(dst, o.Style.Newline...)
			dst = s.emitChildren(dst, block+1, block+NodeID(s.nodes[block].Sub), o, depth+1, false)
			dst = trimTrailingSemicolon(dst)
			dst = appendIndent(dst, o.Style, depth)
			dst = append(dst, '}')
			dst = append(dst, o.Style.AfterBrace...)
			return dst
		}
		dst = s.emitHeadChildren(dst, id, block, o)
		dst = append(dst, o.Style.BeforeBrace...)
		dst = append(dst, '{')
		dst = append(dst, o.Style.Newline...)
		dst = s.emitChildren(dst, block+1, block+NodeID(s.nodes[block].Sub), o, depth+1, false)
		dst = trimTrailingSemicolon(dst)
		dst = appendIndent(dst, o.Style, depth)
		dst = append(dst, '}')
		dst = append(dst, o.Style.AfterBrace...)
		return dst
	case n.Kind.isAtRule():
		block, ok := s.blockChild(id)
		dst = appendIndent(dst, o.Style, depth)
		dst = append(dst, s.atName(n)...)
		headStart := id + 1
		if ok {
			headEnd := block
			dst = s.emitPreludeChildren(dst, headStart, headEnd, o)
		} else {
			dst = s.emitPreludeChildren(dst, headStart, id+NodeID(n.Sub), o)
		}
		if ok {
			dst = append(dst, o.Style.BeforeBrace...)
			dst = append(dst, '{')
			dst = append(dst, o.Style.Newline...)
			dst = s.emitChildren(dst, block+1, block+NodeID(s.nodes[block].Sub), o, depth+1, false)
			dst = trimTrailingSemicolon(dst)
			dst = appendIndent(dst, o.Style, depth)
			dst = append(dst, '}')
			dst = append(dst, o.Style.AfterBrace...)
		} else {
			if len(dst) == 0 || dst[len(dst)-1] != ';' {
				dst = append(dst, ';')
			}
			dst = append(dst, o.Style.Newline...)
		}
		return dst
	case n.Kind == KindBlock:
		dst = append(dst, '{')
		dst = s.emitChildren(dst, id+1, id+NodeID(n.Sub), o, depth+1, false)
		dst = trimTrailingSemicolon(dst)
		return append(dst, '}')
	case n.Kind == KindDeclaration:
		dst = appendIndent(dst, o.Style, depth)
		dst = s.emitDeclaration(dst, id, o)
		dst = trimTrailingSemicolon(dst)
		dst = append(dst, ';')
		dst = append(dst, o.Style.Newline...)
		return dst
	case n.Kind == KindValFunction:
		dst = append(dst, s.nodeBytes(Node{Lo: n.Lo, Hi: n.Lo + uint32(n.Aux)})...)
		dst = append(dst, '(')
		dst = s.emitDelimitedChildren(dst, id+1, id+NodeID(n.Sub), o, valueSep)
		dst = append(dst, ')')
		return dst
	case n.Kind == KindSelPseudo && n.Sub > 1:
		nameEnd := n.Lo + uint32(n.Aux)
		dst = append(dst, s.src[n.Lo:nameEnd]...)
		dst = append(dst, '(')
		dst = s.emitDelimitedChildren(dst, id+1, id+NodeID(n.Sub), o, selectorSep)
		dst = append(dst, ')')
		return dst
	case (n.Kind == KindMediaQuery || n.Kind == KindContainerQuery || n.Kind == KindSupportsCond) && n.Sub > 1 && s.hasChangesInSub(id):
		return s.emitSourceWithChildren(dst, id, o, depth)
	case n.Kind == KindSelCombinator:
		b := s.nodeBytes(n)
		if isAllSpace(b) {
			return append(dst, ' ')
		}
		dst = append(dst, b...)
		if len(b) == 1 && b[0] == ',' {
			dst = append(dst, o.Style.AfterComma...)
		}
		return dst
	default:
		return append(dst, s.nodeBytes(n)...)
	}
}

func (s *Sheet) emitSourceWithChildren(dst []byte, id NodeID, o EmitOptions, depth int) []byte {
	n := s.nodes[id]
	parentLo, parentHi, ok := sourceSpanFrom(s.scratch, n)
	if !ok || int(parentHi) > len(s.src) {
		return dst
	}
	cursor := parentLo
	end := id + NodeID(n.Sub)
	for child := id + 1; child < end; child += NodeID(s.nodes[child].Sub) {
		childLo, childHi, childOK := sourceSpanFrom(s.scratch, s.nodes[child])
		if !childOK || childLo < cursor || childHi > parentHi {
			return append(dst, s.src[parentLo:parentHi]...)
		}
		dst = append(dst, s.src[cursor:childLo]...)
		if s.nodes[child].Flags&FlagTombstone == 0 {
			dst = s.emitNode(dst, child, o, depth)
		}
		cursor = childHi
	}
	return append(dst, s.src[cursor:parentHi]...)
}

type sepMode uint8

const (
	selectorSep sepMode = iota
	valueSep
	preludeSep
)

func (s *Sheet) emitHeadChildren(dst []byte, id, block NodeID, o EmitOptions) []byte {
	return s.emitDelimitedChildren(dst, id+1, block, o, selectorSep)
}

func (s *Sheet) emitPreludeChildren(dst []byte, start, end NodeID, o EmitOptions) []byte {
	if start >= end {
		return dst
	}
	dst = append(dst, ' ')
	return s.emitDelimitedChildren(dst, start, end, o, preludeSep)
}

func (s *Sheet) emitDeclaration(dst []byte, id NodeID, o EmitOptions) []byte {
	n := s.nodes[id]
	dst = append(dst, s.src[n.Lo:n.Lo+uint32(n.Aux)]...)
	dst = append(dst, ':')
	dst = append(dst, o.Style.AfterColon...)
	dst = s.emitDelimitedChildren(dst, id+1, id+NodeID(n.Sub), o, valueSep)
	if n.Flags&FlagImportant != 0 {
		if len(dst) > 0 && needsSpaceBefore(dst, '!') {
			dst = append(dst, ' ')
		}
		dst = append(dst, "!important"...)
	}
	return dst
}

func (s *Sheet) emitDelimitedChildren(dst []byte, start, end NodeID, o EmitOptions, mode sepMode) []byte {
	var prev *Node
	for id := start; id < end; id += NodeID(s.nodes[id].Sub) {
		n := s.nodes[id]
		if n.Flags&FlagTombstone != 0 {
			continue
		}
		if prev != nil {
			dst = s.emitBetween(dst, *prev, n, o, mode)
		}
		before := len(dst)
		dst = s.emitNode(dst, id, o, 0)
		if mode == valueSep && len(dst) > before && dst[len(dst)-1] == ',' {
			dst = append(dst, o.Style.AfterComma...)
		}
		prev = &s.nodes[id]
	}
	return dst
}

func (s *Sheet) emitBetween(dst []byte, prev, next Node, o EmitOptions, mode sepMode) []byte {
	_, prevHi, prevOK := sourceSpanFrom(s.scratch, prev)
	nextLo, _, nextOK := sourceSpanFrom(s.scratch, next)
	separated := prevOK && nextOK && nextLo > prevHi
	if mode == selectorSep {
		if prev.Kind == KindSelCombinator || next.Kind == KindSelCombinator {
			return dst
		}
		if separated {
			return append(dst, ' ')
		}
		return dst
	}
	if mode == preludeSep {
		if (prev.Kind == KindLayerName && next.Kind == KindLayerName) || (prev.Kind == KindMediaQuery && next.Kind == KindMediaQuery) || (prev.Kind == KindContainerQuery && next.Kind == KindContainerQuery) {
			dst = append(dst, ',')
			return append(dst, o.Style.AfterComma...)
		}
		if separated {
			return append(dst, ' ')
		}
		return dst
	}
	if separated {
		return append(dst, ' ')
	}
	return dst
}

func (s *Sheet) atName(n Node) []byte {
	end := n.Lo + 1 + uint32(n.Aux)
	if int(end) <= len(s.src) {
		return s.src[n.Lo:end]
	}
	return []byte(n.Kind.String())
}
func firstByte(b []byte) byte {
	if len(b) == 0 {
		return 0
	}
	return b[0]
}
func isAllSpace(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	for _, c := range b {
		if !isCSSSpace(c) {
			return false
		}
	}
	return true
}

func (s *Sheet) nodeBytes(n Node) []byte {
	buf := s.src
	if n.Flags&FlagScratch != 0 {
		buf = s.scratch
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

func (s *Sheet) hasChangesInSub(id NodeID) bool {
	end := id + NodeID(s.nodes[id].Sub)
	for i := id + 1; i < end; i++ {
		if s.nodes[i].Flags&(FlagTombstone|FlagUnwrap|FlagScratch) != 0 {
			return true
		}
	}
	return false
}

func minifyAppend(dst, src []byte, style Style) []byte {
	sp := false
	for i := 0; i < len(src); i++ {
		c := src[i]
		if c == '/' && i+1 < len(src) && src[i+1] == '*' {
			bang := i+2 < len(src) && src[i+2] == '!'
			j := i + 2
			for j+1 < len(src) && !(src[j] == '*' && src[j+1] == '/') {
				j++
			}
			if bang {
				if sp && needsSpaceBefore(dst, '/') {
					dst = append(dst, ' ')
				}
				dst = append(dst, src[i:min(j+2, len(src))]...)
				sp = false
			}
			i = min(j+1, len(src)-1)
			continue
		}
		if isCSSSpace(c) {
			sp = true
			continue
		}
		if c == ':' {
			dst = append(dst, ':')
			dst = append(dst, style.AfterColon...)
			sp = false
			continue
		}
		if c == ',' {
			dst = append(dst, ',')
			dst = append(dst, style.AfterComma...)
			sp = false
			continue
		}
		if sp && needsSpaceBefore(dst, c) {
			dst = append(dst, ' ')
		}
		dst = append(dst, c)
		sp = false
	}
	return trimTrailingSemicolon(dst)
}

func isCSSSpace(c byte) bool { return c == ' ' || c == '\n' || c == '\t' || c == '\r' || c == '\f' }
func cssIdent(c byte) bool {
	return ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') || ('0' <= c && c <= '9') || c == '-' || c == '_' || c >= 0x80
}
func needsSpaceBefore(dst []byte, c byte) bool {
	if len(dst) == 0 {
		return false
	}
	p := dst[len(dst)-1]
	return cssIdent(p) && cssIdent(c)
}
func trimTrailingSemicolon(dst []byte) []byte {
	i := len(dst)
	for i > 0 && isCSSSpace(dst[i-1]) {
		i--
	}
	if i > 0 && dst[i-1] == ';' {
		copy(dst[i-1:], dst[i:])
		return dst[:len(dst)-1]
	}
	return dst
}
func appendIndent(dst []byte, st Style, depth int) []byte {
	if len(st.Newline) == 0 || len(st.Indent) == 0 {
		return dst
	}
	for i := 0; i < depth; i++ {
		dst = append(dst, st.Indent...)
	}
	return dst
}
