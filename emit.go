package pristinecss

import (
	"bytes"
	"slices"

	"github.com/builtwithtofu/pristinecss/internal/lexer"
)

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
		if n.Flags&FlagScratch != 0 && len(s.nodeBytes(n)) == 0 {
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
		if n.Kind == KindRawStatement || n.Kind == KindDeclaration {
			dst = appendIndent(dst, o.Style, depth)
			switch {
			case n.Flags&FlagScratch != 0:
				dst = append(dst, s.nodeBytes(n)...)
			case n.Kind == KindRawStatement:
				dst = s.emitLeaf(dst, n)
			default:
				dst = s.emitDeclaration(dst, id, o)
			}
			dst = appendSemicolon(dst)
			dst = append(dst, o.Style.Newline...)
		} else {
			dst = s.emitNode(dst, id, o, depth)
		}
		if top && o.OnRule != nil && (n.Kind == KindStyleRule || n.Kind.isAtRule() || n.Kind == KindRawStatement) {
			o.OnRule(id, lo, len(dst))
		}
	}
	return dst
}

func (s *Sheet) emitNode(dst []byte, id NodeID, o EmitOptions, depth int) []byte {
	n := s.nodes[id]
	if n.Flags&FlagScratch != 0 {
		return append(dst, s.nodeBytes(n)...)
	}
	switch {
	case n.Kind == KindComment:
		b := s.nodeBytes(n)
		if len(b) >= 3 && bytes.HasPrefix(b, []byte("/*!")) || len(o.Style.Newline) != 0 || o.Verbatim {
			return s.emitLeaf(dst, n)
		}
		return dst
	case n.Kind == KindStyleRule || n.Kind == KindKeyframeBlock || n.Kind == KindMarginBox:
		block, ok := s.blockChild(id)
		if !ok {
			return minifyAppend(dst, s.nodeBytes(n), o.Style)
		}
		dst = appendIndent(dst, o.Style, depth)
		if n.Kind == KindMarginBox {
			dst = s.emitLeaf(dst, Node{Lo: n.Lo, Hi: n.Lo + 1 + uint32(n.Aux)})
			dst = appendSeparator(dst, o.Style.BeforeBrace)
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
		dst = appendSeparator(dst, o.Style.BeforeBrace)
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
		dst = s.emitLeaf(dst, Node{Lo: n.Lo, Hi: n.Lo + 1 + uint32(n.Aux)})
		headStart := id + 1
		if ok {
			headEnd := block
			dst = s.emitPreludeChildren(dst, headStart, headEnd, o)
		} else {
			dst = s.emitPreludeChildren(dst, headStart, id+NodeID(n.Sub), o)
		}
		if ok {
			dst = appendSeparator(dst, o.Style.BeforeBrace)
			dst = append(dst, '{')
			dst = append(dst, o.Style.Newline...)
			dst = s.emitChildren(dst, block+1, block+NodeID(s.nodes[block].Sub), o, depth+1, false)
			dst = trimTrailingSemicolon(dst)
			dst = appendIndent(dst, o.Style, depth)
			dst = append(dst, '}')
			dst = append(dst, o.Style.AfterBrace...)
		} else {
			dst = appendSemicolon(dst)
			dst = append(dst, o.Style.Newline...)
		}
		return dst
	case n.Kind == KindBlock:
		dst = append(dst, '{')
		dst = s.emitChildren(dst, id+1, id+NodeID(n.Sub), o, depth+1, false)
		dst = trimTrailingSemicolon(dst)
		return append(dst, '}')
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
			return appendSeparator(dst, []byte(" "))
		}
		dst = append(dst, b...)
		if len(b) == 1 && b[0] == ',' {
			dst = append(dst, o.Style.AfterComma...)
		}
		return dst
	default:
		return s.emitLeaf(dst, n)
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
	dst = appendSeparator(dst, []byte(" "))
	return s.emitDelimitedChildren(dst, start, end, o, preludeSep)
}

func (s *Sheet) emitDeclaration(dst []byte, id NodeID, o EmitOptions) []byte {
	n := s.nodes[id]
	dst = s.emitLeaf(dst, Node{Lo: n.Lo, Hi: n.Lo + uint32(n.Aux)})
	dst = append(dst, ':')
	dst = append(dst, o.Style.AfterColon...)
	value := len(dst)
	dst = s.emitDelimitedChildren(dst, id+1, id+NodeID(n.Sub), o, valueSep)
	if len(o.Style.AfterColon) == 0 && len(dst) > value && dst[value] == ':' {
		dst = slices.Insert(dst, value, ' ')
	}
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
	prevStart := 0
	for id := start; id < end; id += NodeID(s.nodes[id].Sub) {
		n := s.nodes[id]
		if n.Flags&FlagTombstone != 0 {
			continue
		}
		if n.Kind == KindComment && !o.Verbatim && len(o.Style.Newline) == 0 && !bytes.HasPrefix(s.nodeBytes(n), []byte("/*!")) {
			continue
		}
		if prev != nil {
			dst = s.emitBetween(dst, prevStart, *prev, n, o, mode)
		}
		before := len(dst)
		prevStart = before
		dst = s.emitNode(dst, id, o, 0)
		if mode == valueSep && len(dst) > before && dst[len(dst)-1] == ',' {
			dst = append(dst, o.Style.AfterComma...)
		}
		prev = &s.nodes[id]
	}
	return dst
}

// prevStart is where prev's own output begins in dst.
func (s *Sheet) emitBetween(dst []byte, prevStart int, prev, next Node, o EmitOptions, mode sepMode) []byte {
	_, prevHi, prevOK := sourceSpanFrom(s.scratch, prev)
	nextLo, _, nextOK := sourceSpanFrom(s.scratch, next)
	separated := prevOK && nextOK && nextLo > prevHi
	if mode == selectorSep {
		if prev.Kind == KindSelCombinator || next.Kind == KindSelCombinator {
			return dst
		}
		if separated && sourceHasSpace(s.src[prevHi:nextLo]) {
			return appendSeparator(dst, []byte(" "))
		}
		if separated && (tokensMerge(dst, prevStart, s.nodeBytes(next)) || formsCDO(dst, s.nodeBytes(next))) {
			return append(dst, "/**/"...)
		}
		return dst
	}
	if mode == preludeSep {
		if (prev.Kind == KindLayerName && next.Kind == KindLayerName) || (prev.Kind == KindMediaQuery && next.Kind == KindMediaQuery) || (prev.Kind == KindContainerQuery && next.Kind == KindContainerQuery) {
			dst = append(dst, ',')
			return append(dst, o.Style.AfterComma...)
		}
		if separated {
			return appendSeparator(dst, []byte(" "))
		}
		return dst
	}
	if separated {
		return appendSeparator(dst, []byte(" "))
	}
	return dst
}

// tokensMerge reports whether right, written straight after dst[from:], would
// lex differently. right is staged in dst's spare capacity, so Emit stays
// allocation-free with a warm buffer.
func tokensMerge(dst []byte, from int, right []byte) bool {
	left := dst[from:]
	if len(left) > 0 && len(right) > 0 && (left[len(left)-1] == '@' || left[len(left)-1] == '#') && (cssIdent(right[0]) || right[0] == '\\') {
		return true
	}
	joined := append(dst, right...)
	return !lexer.Splits(joined[from:], len(left))
}

// formsCDO reports whether right, written straight after left, could complete
// a "<!--" token. A CDO can span more than two nodes, so left is the output so far.
func formsCDO(left, right []byte) bool {
	const cdo = "<!--"
	for k := 1; k < len(cdo); k++ {
		m := min(len(cdo)-k, len(right))
		if m > 0 && bytes.HasSuffix(left, []byte(cdo[:k])) && string(right[:m]) == cdo[k:k+m] {
			return true
		}
	}
	return false
}

func appendSeparator(dst, separator []byte) []byte {
	if escapeOpen(dst) {
		// This space only ends the escape; the separator still has to follow.
		dst = append(dst, ' ')
	} else if endsWithSpace(dst) {
		return dst
	}
	return append(dst, separator...)
}
func sourceHasSpace(src []byte) bool { return sourceSpaceIndex(src) >= 0 }

func sourceSpaceIndex(src []byte) int {
	for i := 0; i < len(src); i++ {
		if i+1 < len(src) && src[i] == '/' && src[i+1] == '*' {
			i += 2
			for i+1 < len(src) && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			i++
		} else if isCSSSpace(src[i]) {
			return i
		}
	}
	return -1
}

// Close EOF-terminated tokens before adding punctuation belonging to their parent.
// A bad string keeps its newline so it cannot swallow the following statement.
func (s *Sheet) emitLeaf(dst []byte, n Node) []byte {
	src := s.nodeBytes(n)
	var quote byte
	urlDepth := 0
	comment := false
	escEnd := -1
	var closerBuf [16]byte
	closers := closerBuf[:0]
	for i := 0; i < len(src); i++ {
		c := src[i]
		dst = append(dst, c)
		if comment {
			if c == '*' && i+1 < len(src) && src[i+1] == '/' {
				i++
				dst = append(dst, '/')
				comment = false
			}
			continue
		}
		if urlDepth > 0 && c != '\\' {
			if c == ')' {
				urlDepth = 0
			}
			continue
		}
		if c == '\\' {
			if i+1 < len(src) {
				end := escapeEnd(src, i)
				// A backslash before a newline is no escape and ends any name.
				if src[i+1] != '\n' && src[i+1] != '\r' && src[i+1] != '\f' {
					escEnd = end
				}
				dst = append(dst, src[i+1:end]...)
				i = end - 1
			} else if quote != 0 {
				dst = dst[:len(dst)-1]
			} else {
				dst = append(dst[:len(dst)-1], "\ufffd"...)
			}
			continue
		}
		if quote != 0 {
			if c == quote || c == '\n' || c == '\r' || c == '\f' {
				quote = 0
			}
			continue
		}
		if urlDepth == 0 {
			switch c {
			case '(':
				closers = append(closers, ')')
			case '[':
				closers = append(closers, ']')
			case '{':
				closers = append(closers, '}')
			case ')', ']', '}':
				if len(closers) > 0 && closers[len(closers)-1] == c {
					closers = closers[:len(closers)-1]
				}
			}
		}
		if c == '"' || c == '\'' {
			quote = c
		} else if urlDepth == 0 && c == '/' && i+1 < len(src) && src[i+1] == '*' {
			comment = true
			dst = append(dst, '*')
			i++
		} else if (c == 'u' || c == 'U') && (i == 0 || !cssIdent(src[i-1])) && i != escEnd && i+3 < len(src) && bytes.EqualFold(src[i:i+4], []byte("url(")) {
			dst = append(dst, src[i+1:i+4]...)
			i += 3
			j := i + 1
			for j < len(src) && isCSSSpace(src[j]) {
				j++
			}
			if j < len(src) && (src[j] == '"' || src[j] == '\'') {
				closers = append(closers, ')')
			} else {
				urlDepth = 1
			}
		}
	}
	if quote != 0 {
		if n.Flags&FlagScratch == 0 && int(n.Hi) < len(s.src) && (s.src[n.Hi] == '\n' || s.src[n.Hi] == '\r' || s.src[n.Hi] == '\f') {
			dst = append(dst, s.src[n.Hi])
		} else {
			dst = append(dst, quote)
		}
	}
	if comment {
		dst = append(dst, '*', '/')
	}
	for range urlDepth {
		dst = append(dst, ')')
	}
	for i := len(closers) - 1; i >= 0; i-- {
		dst = append(dst, closers[i])
	}
	return dst
}

// escapeEnd returns the index after the escape whose backslash is at src[i]:
// up to six hex digits and one whitespace (CRLF as one), or one code unit.
func escapeEnd(src []byte, i int) int {
	j := i + 1
	if !isHexDigit(src[j]) {
		if src[j] == '\r' && j+1 < len(src) && src[j+1] == '\n' {
			return j + 2
		}
		return j + 1
	}
	for j < len(src) && j-i <= 6 && isHexDigit(src[j]) {
		j++
	}
	if j+1 < len(src) && src[j] == '\r' && src[j+1] == '\n' {
		return j + 2
	}
	if j < len(src) && isCSSSpace(src[j]) {
		return j + 1
	}
	return j
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

// appendSemicolon ends a statement unless its own bytes already did. Newlines
// before a ';' are not skipped: one may end a bad string that holds the ';'.
func appendSemicolon(dst []byte) []byte {
	i := len(dst)
	for i > 0 && (dst[i-1] == ' ' || dst[i-1] == '\t') {
		i--
	}
	if i > 0 && dst[i-1] == ';' && !unescapedBackslash(dst[:i-1]) {
		return dst
	}
	return append(dst, ';')
}
func trimTrailingSemicolon(dst []byte) []byte {
	i := len(dst)
	for i > 0 && isCSSSpace(dst[i-1]) {
		i--
	}
	if i > 0 && dst[i-1] == ';' {
		before := dst[:i-1]
		// Without the ';' the whitespace after it could join an escape: as the
		// escape's own whitespace, or as the LF of a CRLF that ends the escape.
		cr := len(before) > 0 && before[len(before)-1] == '\r' && escapeOpen(before[:len(before)-1])
		if unescapedBackslash(before) || i < len(dst) && (escapeOpen(before) || cr) {
			return dst
		}
		copy(dst[i-1:], dst[i:])
		return dst[:len(dst)-1]
	}
	return dst
}

// escapeOpen reports whether whitespace appended to b would belong to an escape.
func escapeOpen(b []byte) bool {
	h := len(b)
	for h > 0 && len(b)-h < 6 && isHexDigit(b[h-1]) {
		h--
	}
	return unescapedBackslash(b[:h])
}

// endsWithSpace reports whether b ends in whitespace that separates tokens.
func endsWithSpace(b []byte) bool {
	n := len(b)
	if n == 0 || !isCSSSpace(b[n-1]) {
		return false
	}
	if n > 1 && b[n-2] == '\r' && b[n-1] == '\n' {
		n--
	}
	return !escapeOpen(b[:n-1])
}
func unescapedBackslash(b []byte) bool {
	n := 0
	for n < len(b) && b[len(b)-1-n] == '\\' {
		n++
	}
	return n%2 == 1
}
func isHexDigit(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F'
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
