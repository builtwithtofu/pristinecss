package pristinecss

import (
	"bytes"

	"github.com/builtwithtofu/pristinecss/internal/lexer"
)

func (p *parser) parseDeclaration() {
	name := p.advance()
	id := p.beginAt(KindDeclaration, name.Start)
	n := &p.s.nodes[id]
	n.Aux = uint16(name.End - name.Start)
	if bytes.HasPrefix(p.src[name.Start:name.End], []byte("--")) {
		n.Flags |= FlagCustom
	}
	if p.cur().Type != lexer.Colon {
		p.err(p.cur(), "expected ':' after declaration name, found %s", p.cur().Type)
		p.finish(id, name.End)
		return
	}
	p.advance()
	valueStart := p.pos
	custom := n.Flags&FlagCustom != 0
	valueEnd := p.scanDeclarationValue()
	important := false
	if !custom {
		last := p.previousNonComment(valueEnd)
		bang := p.previousNonComment(last)
		if last >= valueStart && bang >= valueStart && p.toks[bang].Type == lexer.Exclamation && bytes.EqualFold(p.toks[last].Literal(p.src), []byte("important")) && p.valueTokenIsTopLevel(valueStart, bang) {
			important = true
			valueEnd = bang
		}
	}
	if important {
		p.s.nodes[id].Flags |= FlagImportant
	}
	if p.s.nodes[id].Flags&FlagCustom != 0 {
		lo, hi := name.End, name.End
		if valueStart < p.pos {
			lo = p.toks[valueStart].Start
			hi = p.toks[p.pos-1].End
		}
		p.leaf(KindValRaw, lo, hi, 0, 0)
	} else {
		p.emitValueParts(valueStart, valueEnd)
	}
	hi := name.End
	if p.pos > valueStart {
		hi = p.toks[p.pos-1].End
	}
	if p.cur().Type == lexer.Semicolon {
		hi = p.advance().End
	}
	p.finish(id, hi)
}

func (p *parser) scanDeclarationValue() int {
	parenDepth, bracketDepth, braceDepth := 0, 0, 0
	for p.cur().Type != lexer.EOF {
		t := p.cur().Type
		if t == lexer.Semicolon && parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 {
			break
		}
		if t == lexer.RBrace && braceDepth == 0 {
			break
		}
		switch t {
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
		case lexer.LBrace:
			braceDepth++
		case lexer.RBrace:
			if braceDepth > 0 {
				braceDepth--
			}
		}
		p.advance()
	}
	return p.pos
}

func (p *parser) previousNonComment(end int) int {
	for end--; end >= 0; end-- {
		if p.toks[end].Type != lexer.Comment {
			return end
		}
	}
	return -1
}

func (p *parser) valueTokenIsTopLevel(start, target int) bool {
	parenDepth, bracketDepth, braceDepth := 0, 0, 0
	for i := start; i < target; i++ {
		switch p.toks[i].Type {
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
		case lexer.LBrace:
			braceDepth++
		case lexer.RBrace:
			if braceDepth > 0 {
				braceDepth--
			}
		}
	}
	return parenDepth == 0 && bracketDepth == 0 && braceDepth == 0
}

func (p *parser) emitValueParts(start, end int) {
	for i := start; i < end; i++ {
		t := p.toks[i]
		switch t.Type {
		case lexer.Ident:
			if i+1 < end && p.toks[i+1].Type == lexer.LParen {
				fid := p.beginAt(KindValFunction, t.Start)
				p.s.nodes[fid].Aux = uint16(t.End - t.Start)
				i += 2
				childStart := i
				depth := 1
				for i < end && depth > 0 {
					if p.toks[i].Type == lexer.LParen {
						depth++
					} else if p.toks[i].Type == lexer.RParen {
						depth--
					}
					if depth > 0 {
						i++
					}
				}
				if depth > 0 {
					p.err(p.toks[childStart-1], "expected ')' to close value function")
				}
				p.emitValueParts(childStart, i)
				hi := p.toks[childStart-1].End
				if i > childStart {
					hi = p.toks[i-1].End
				}
				if i < end {
					hi = p.toks[i].End
				}
				p.finish(fid, hi)
			} else {
				p.leaf(KindValBasic, t.Start, t.End, 0, 0)
			}
		case lexer.String:
			flags := Flags(0)
			if len(p.src[t.Start:t.End]) > 0 && p.src[t.Start] == '\'' {
				flags = FlagSingleQuote
			}
			p.leaf(KindValString, t.Start, t.End, 0, flags)
		case lexer.Comment:
			p.leaf(KindComment, t.Start, t.End, 0, 0)
		default:
			p.leaf(KindValBasic, t.Start, t.End, 0, 0)
		}
	}
}
