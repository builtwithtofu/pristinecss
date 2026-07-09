package pristinecss

import "github.com/builtwithtofu/pristinecss/internal/lexer"

func (p *parser) parseAtRule(depth int) {
	at := p.advance()
	name := p.cur()
	if name.Type != lexer.Ident {
		p.err(at, "expected at-rule name after '@', found %s", name.Type)
		return
	}
	p.advance()
	k := atKind(name.Literal(p.src))
	if k == KindAtGeneric && isMarginBoxName(name.Literal(p.src)) {
		p.parseMarginBox(at, name, depth)
		return
	}
	id := p.beginAt(k, at.Start)
	p.s.nodes[id].Aux = uint16(name.End - name.Start)
	preludeLo := name.End
	preludeStart := p.pos
	parenDepth, bracketDepth := 0, 0
	for p.cur().Type != lexer.EOF && p.cur().Type != lexer.Semicolon && p.cur().Type != lexer.LBrace && p.cur().Type != lexer.RBrace {
		switch p.cur().Type {
		case lexer.LParen:
			parenDepth++
		case lexer.RParen:
			if parenDepth > 0 {
				parenDepth--
			}
		case lexer.LBracket:
			bracketDepth++
		case lexer.RBracket:
			if bracketDepth > 0 {
				bracketDepth--
			}
		}
		p.advance()
	}
	if parenDepth > 0 || bracketDepth > 0 {
		p.err(p.cur(), "expected closing ')' or ']' before at-rule prelude ended")
	}
	if p.pos > preludeStart {
		p.emitPrelude(k, preludeStart, p.pos, preludeLo)
	}
	if p.cur().Type == lexer.LBrace {
		var hi uint32
		if k == KindAtKeyframes {
			hi = p.parseKeyframesBlock(depth + 1)
		} else {
			hi = p.parseBlock(depth + 1)
		}
		p.finish(id, hi)
		return
	}
	hi := name.End
	if p.pos > preludeStart {
		hi = p.toks[p.pos-1].End
	}
	if p.cur().Type == lexer.Semicolon {
		hi = p.advance().End
	}
	p.finish(id, hi)
}

func (p *parser) emitPrelude(k Kind, start, end int, fallbackLo uint32) {
	lo := fallbackLo
	hi := fallbackLo
	if start < end {
		lo = p.toks[start].Start
		hi = p.toks[end-1].End
	}
	switch k {
	case KindAtMedia:
		p.emitQueryList(KindMediaQuery, KindMediaFeature, start, end)
	case KindAtContainer:
		p.emitQueryList(KindContainerQuery, KindContainerFeature, start, end)
	case KindAtSupports:
		id := p.beginAt(KindSupportsCond, lo)
		p.emitParenFeatures(KindMediaFeature, start, end)
		p.finish(id, hi)
	case KindAtLayer:
		segStart := start
		for i := start; i <= end; i++ {
			if i == end || p.toks[i].Type == lexer.Comma {
				if segStart < i {
					p.leaf(KindLayerName, p.toks[segStart].Start, p.toks[i-1].End, 0, 0)
				}
				segStart = i + 1
			}
		}
	case KindAtFontFace, KindAtFontPaletteValues:
		p.leaf(KindFontFamilyName, lo, hi, 0, 0)
	default:
		p.leaf(KindPreludeRaw, lo, hi, 0, 0)
	}
}

func (p *parser) emitQueryList(queryKind, featureKind Kind, start, end int) {
	segStart, depth := start, 0
	for i := start; i <= end; i++ {
		if i < end {
			if p.toks[i].Type == lexer.LParen {
				depth++
			} else if p.toks[i].Type == lexer.RParen && depth > 0 {
				depth--
			}
		}
		if i == end || (depth == 0 && p.toks[i].Type == lexer.Comma) {
			if segStart < i {
				id := p.beginAt(queryKind, p.toks[segStart].Start)
				p.emitParenFeatures(featureKind, segStart, i)
				p.finish(id, p.toks[i-1].End)
			}
			segStart = i + 1
		}
	}
}

func (p *parser) emitParenFeatures(kind Kind, start, end int) {
	for i := start; i < end; i++ {
		if p.toks[i].Type != lexer.LParen {
			continue
		}
		lo := p.toks[i].Start
		depth := 1
		for i++; i < end && depth > 0; i++ {
			if p.toks[i].Type == lexer.LParen {
				depth++
			} else if p.toks[i].Type == lexer.RParen {
				depth--
			}
		}
		hi := p.toks[min(i-1, end-1)].End
		p.leaf(kind, lo, hi, 0, 0)
	}
}

func (p *parser) parseKeyframesBlock(depth int) uint32 {
	open := p.advance()
	blockID := p.beginAt(KindBlock, open.Start)
	for p.cur().Type != lexer.EOF && p.cur().Type != lexer.RBrace {
		if p.cur().Type == lexer.Semicolon || p.cur().Type == lexer.Comment {
			p.advance()
			continue
		}
		kid := p.beginAt(KindKeyframeBlock, p.cur().Start)
		stopStart := p.pos
		for p.cur().Type != lexer.EOF && p.cur().Type != lexer.LBrace && p.cur().Type != lexer.RBrace {
			p.advance()
		}
		p.emitValueParts(stopStart, p.pos)
		hi := p.cur().End
		if p.cur().Type == lexer.LBrace {
			hi = p.parseBlock(depth + 1)
		}
		p.finish(kid, hi)
	}
	hi := open.End
	if p.cur().Type == lexer.RBrace {
		hi = p.advance().End
	} else {
		p.err(p.cur(), "expected '}' to close keyframes block, found %s", p.cur().Type)
	}
	p.finish(blockID, hi)
	return hi
}

func (p *parser) parseMarginBox(at, name lexer.Token, depth int) {
	id := p.beginAt(KindMarginBox, at.Start)
	p.s.nodes[id].Aux = uint16(name.End - name.Start)
	if p.cur().Type == lexer.LBrace {
		hi := p.parseBlock(depth + 1)
		p.finish(id, hi)
		return
	}
	p.finish(id, name.End)
}

func isMarginBoxName(name []byte) bool {
	switch {
	case asciiEqual(name, "top-left"), asciiEqual(name, "top-center"), asciiEqual(name, "top-right"), asciiEqual(name, "bottom-left"), asciiEqual(name, "bottom-center"), asciiEqual(name, "bottom-right"), asciiEqual(name, "left-top"), asciiEqual(name, "left-middle"), asciiEqual(name, "left-bottom"), asciiEqual(name, "right-top"), asciiEqual(name, "right-middle"), asciiEqual(name, "right-bottom"), asciiEqual(name, "top-left-corner"), asciiEqual(name, "top-right-corner"), asciiEqual(name, "bottom-left-corner"), asciiEqual(name, "bottom-right-corner"):
		return true
	}
	return false
}
