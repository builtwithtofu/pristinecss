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
		if last >= valueStart && bang >= valueStart && p.toks[bang].Type == lexer.Exclamation && p.toks[last].Type == lexer.Ident && cssNameEqual(string(p.toks[last].Literal(p.src)), "important") && p.valueTokenIsTopLevel(valueStart, bang) {
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
	for p.cur().Type != lexer.EOF {
		switch p.cur().Type {
		case lexer.Semicolon, lexer.RBrace:
			return p.pos
		case lexer.LParen, lexer.LBracket, lexer.LBrace:
			p.pos, _ = p.componentEnd(p.pos, len(p.toks)-1, true)
		default:
			p.advance()
		}
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

func (p *parser) nextNonComment(i, end int) int {
	for i < end && p.toks[i].Type == lexer.Comment {
		i++
	}
	return i
}

func (p *parser) valueTokenIsTopLevel(start, target int) bool {
	for i := start; i < target; i++ {
		switch p.toks[i].Type {
		case lexer.LParen, lexer.LBracket, lexer.LBrace:
			last, _ := p.componentEnd(i, len(p.toks)-1, true)
			if last > target {
				return false
			}
			i = last - 1
		}
	}
	return true
}

func (p *parser) emitValueParts(start, end int) {
	for i := start; i < end; i++ {
		t := p.toks[i]
		switch t.Type {
		case lexer.Ident:
			if i+1 < end && p.toks[i+1].Type == lexer.LParen && p.toks[i+1].Start == t.End {
				fid := p.beginAt(KindValFunction, t.Start)
				p.s.nodes[fid].Aux = uint16(t.End - t.Start)
				last, closed := p.componentEnd(i+1, end, false)
				childEnd := last
				if closed {
					childEnd--
				} else {
					p.err(p.toks[i+1], "expected ')' to close value function")
				}
				p.emitValueParts(i+2, childEnd)
				p.finish(fid, p.toks[last-1].End)
				i = last - 1
			} else {
				p.leaf(KindValBasic, t.Start, t.End, 0, 0)
			}
		case lexer.LParen, lexer.LBracket, lexer.LBrace:
			last, _ := p.componentEnd(i, end, false)
			p.leaf(KindValRaw, t.Start, p.toks[last-1].End, 0, 0)
			i = last - 1
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

func (p *parser) componentEnd(start, end int, ruleBoundary bool) (int, bool) {
	var stackBuf [16]lexer.Type
	stack := stackBuf[:0]
	braces := 0
	for i := start; i < end; i++ {
		if ruleBoundary && p.toks[i].Type == lexer.RBrace && braces == 0 {
			return i, false
		}
		switch p.toks[i].Type {
		case lexer.LParen:
			stack = append(stack, lexer.RParen)
		case lexer.LBracket:
			stack = append(stack, lexer.RBracket)
		case lexer.LBrace:
			stack = append(stack, lexer.RBrace)
			braces++
		default:
			if len(stack) > 0 && p.toks[i].Type == stack[len(stack)-1] {
				if p.toks[i].Type == lexer.RBrace {
					braces--
				}
				stack = stack[:len(stack)-1]
				if len(stack) == 0 {
					return i + 1, true
				}
			}
		}
	}
	return end, false
}
