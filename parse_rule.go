package pristinecss

import "github.com/builtwithtofu/pristinecss/internal/lexer"

func (p *parser) parseRuleOrDeclaration(depth int) {
	if p.looksLikeDeclaration() {
		p.parseDeclaration()
		return
	}
	p.parseStyleRule(depth)
}

func (p *parser) looksLikeDeclaration() bool {
	if p.cur().Type != lexer.Ident {
		return false
	}
	if len(p.cur().Literal(p.src)) >= 2 && p.src[p.cur().Start] == '-' && p.src[p.cur().Start+1] == '-' {
		return true
	}
	paren, bracket := 0, 0
	seenColon := false
	for i := 1; ; i++ {
		t := p.peek(i)
		switch t.Type {
		case lexer.EOF, lexer.RBrace, lexer.Semicolon:
			return seenColon
		case lexer.LBrace:
			return false
		case lexer.LParen:
			paren++
		case lexer.RParen:
			if paren > 0 {
				paren--
			}
		case lexer.LBracket:
			bracket++
		case lexer.RBracket:
			if bracket > 0 {
				bracket--
			}
		case lexer.Colon:
			if paren == 0 && bracket == 0 {
				seenColon = true
			}
		}
	}
}

func (p *parser) parseStyleRule(depth int) {
	lo := p.cur().Start
	id := p.beginAt(KindStyleRule, lo)
	preludeStart := p.pos
	for p.cur().Type != lexer.EOF && p.cur().Type != lexer.LBrace && p.cur().Type != lexer.Semicolon && p.cur().Type != lexer.RBrace {
		p.advance()
	}
	p.emitSelectorParts(preludeStart, p.pos)
	if p.cur().Type == lexer.LBrace {
		hi := p.parseBlock(depth + 1)
		p.finish(id, hi)
		return
	}
	hi := p.cur().End
	if p.cur().Type == lexer.Semicolon {
		p.advance()
	}
	p.finish(id, hi)
}

func (p *parser) parseBlock(depth int) uint32 {
	open := p.advance()
	id := p.beginAt(KindBlock, open.Start)
	if depth > p.maxDepth {
		p.err(open, "maximum block depth exceeded")
		p.skipBalanced(lexer.RBrace, lexer.EOF)
	}
	p.parseList(lexer.RBrace, depth)
	hi := open.End
	if p.cur().Type == lexer.RBrace {
		hi = p.advance().End
	} else {
		hi = p.cur().Start
		p.err(p.cur(), "expected '}' to close block, found %s", p.cur().Type)
	}
	p.finish(id, hi)
	return hi
}
