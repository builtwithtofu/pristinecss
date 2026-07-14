package pristinecss

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/builtwithtofu/pristinecss/internal/lexer"
)

// Parse lexes and parses CSS into a flat Sheet.
//
// The source buffer is borrowed: the caller must not mutate or free it while the Sheet
// (or any extracted child) lives. Owned extraction and source-inclusive serialization
// are the escape hatches.
func Parse(src []byte) (*Sheet, []ParseError) {
	s := &Sheet{}
	errs := parseInto(src, s, false)
	return s, errs
}

// ParseInto resets s, then lexes and parses CSS into it while reusing retained buffers.
//
// The source buffer is borrowed: the caller must not mutate or free it while the Sheet
// (or any extracted child) lives. Owned extraction and source-inclusive serialization
// are the escape hatches.
func ParseInto(src []byte, s *Sheet) []ParseError {
	return parseInto(src, s, true)
}

func parseInto(src []byte, s *Sheet, cacheSource bool) []ParseError {
	if s == nil {
		panic("pristinecss: ParseInto called with nil Sheet")
	}
	s.src = src
	s.scratch = s.scratch[:0]
	s.lines = s.lines[:0]
	s.lineOnce = sync.Once{}
	s.errs = s.errs[:0]
	s.mutated = false
	if !cacheSource || !bytes.Equal(src, s.tokSrc) {
		s.toks = lexer.LexInto(src, s.toks)
		if cacheSource {
			s.tokSrc = append(s.tokSrc[:0], src...)
		}
	}
	// Framework census on 2026-07-08: selector/value components as full nodes stay below
	// the token count for all framework files with this concrete granularity, because raw
	// preludes and declaration values coalesce token runs unless a function/tree boundary is
	// useful. The parser sizes nodes to len(tokens)+1 and rejects packing components into Aux.
	need := len(s.toks) + 1
	if cap(s.nodes) < need {
		s.nodes = make([]Node, 0, need)
	} else {
		s.nodes = s.nodes[:0]
	}
	p := parser{src: src, toks: s.toks, s: s, maxDepth: 128}
	root := p.beginAt(KindStylesheet, 0)
	p.parseList(lexer.EOF, 0)
	p.finish(root, uint32(len(src)))
	resolveParseErrorPositions(s.src, s.errs)
	return s.errs
}

type parser struct {
	src      []byte
	toks     []lexer.Token
	pos      int
	s        *Sheet
	maxDepth int
}

func (p *parser) cur() lexer.Token {
	if p.pos >= len(p.toks) {
		return lexer.Token{Type: lexer.EOF, Start: uint32(len(p.src)), End: uint32(len(p.src))}
	}
	return p.toks[p.pos]
}
func (p *parser) peek(n int) lexer.Token {
	i := p.pos + n
	if i >= len(p.toks) {
		return lexer.Token{Type: lexer.EOF, Start: uint32(len(p.src)), End: uint32(len(p.src))}
	}
	return p.toks[i]
}
func (p *parser) advance() lexer.Token {
	t := p.cur()
	if p.pos < len(p.toks) {
		p.pos++
	}
	return t
}
func (p *parser) begin(k Kind) NodeID { return p.beginAt(k, p.cur().Start) }
func (p *parser) beginAt(k Kind, lo uint32) NodeID {
	id := NodeID(len(p.s.nodes))
	p.s.nodes = append(p.s.nodes, Node{Kind: k, Lo: lo})
	return id
}
func (p *parser) finish(id NodeID, hi uint32) {
	n := &p.s.nodes[id]
	n.Hi = hi
	n.Sub = uint32(len(p.s.nodes)) - uint32(id)
	if n.Sub == 0 {
		n.Sub = 1
	}
}
func (p *parser) leaf(k Kind, lo, hi uint32, aux uint16, flags Flags) NodeID {
	id := NodeID(len(p.s.nodes))
	p.s.nodes = append(p.s.nodes, Node{Kind: k, Flags: flags, Aux: aux, Lo: lo, Hi: hi, Sub: 1})
	return id
}
func (p *parser) err(tok lexer.Token, msg string, args ...any) {
	p.s.errs = append(p.s.errs, ParseError{Message: fmt.Sprintf(msg, args...), Offset: tok.Start})
}

func (p *parser) parseList(until lexer.Type, depth int) {
	for p.cur().Type != lexer.EOF && p.cur().Type != until {
		start := p.pos
		switch p.cur().Type {
		case lexer.At:
			p.parseAtRule(depth)
		case lexer.Comment:
			t := p.advance()
			p.leaf(KindComment, t.Start, t.End, 0, 0)
		case lexer.CDO, lexer.CDC, lexer.Semicolon:
			p.advance()
		case lexer.RBrace:
			p.err(p.cur(), "expected rule before unexpected '}'")
			p.advance()
		case lexer.Illegal:
			p.err(p.cur(), "expected rule, at-rule, or declaration; found invalid token")
			p.advance()
		default:
			p.parseRuleOrDeclaration(depth)
		}
		if p.pos == start {
			p.advance()
		}
	}
}

func (p *parser) skipBalanced(stop1, stop2 lexer.Type) uint32 {
	depthParen, depthBracket := 0, 0
	end := p.cur().End
	for p.cur().Type != lexer.EOF {
		t := p.cur()
		if depthParen == 0 && depthBracket == 0 && (t.Type == stop1 || t.Type == stop2) {
			return end
		}
		switch t.Type {
		case lexer.LParen:
			depthParen++
		case lexer.RParen:
			if depthParen > 0 {
				depthParen--
			}
		case lexer.LBracket:
			depthBracket++
		case lexer.RBracket:
			if depthBracket > 0 {
				depthBracket--
			}
		}
		end = t.End
		p.advance()
	}
	return end
}

func isIdentLike(t lexer.Token) bool {
	return t.Type == lexer.Ident || t.Type == lexer.Minus || t.Type == lexer.Asterisk
}
