package pristinecss

import "github.com/builtwithtofu/pristinecss/internal/lexer"

func (p *parser) emitSelectorParts(start, end int) {
	var prevEnd uint32
	for i := start; i < end; i++ {
		t := p.toks[i]
		if i > start && t.Start > prevEnd && t.Type != lexer.Comma && t.Type != lexer.RBrace && t.Type != lexer.LBrace {
			p.leaf(KindSelCombinator, prevEnd, t.Start, 0, 0)
		}
		switch t.Type {
		case lexer.Dot:
			if i+1 < end && p.toks[i+1].Type == lexer.Ident {
				p.leaf(KindSelClass, t.Start, p.toks[i+1].End, 0, 0)
				prevEnd = p.toks[i+1].End
				i++
			} else {
				p.leaf(KindSelCombinator, t.Start, t.End, 0, 0)
				prevEnd = t.End
			}
		case lexer.Hash:
			if i+1 < end && p.toks[i+1].Type == lexer.Ident {
				p.leaf(KindSelID, t.Start, p.toks[i+1].End, 0, 0)
				prevEnd = p.toks[i+1].End
				i++
			} else {
				p.leaf(KindSelID, t.Start, t.End, 0, 0)
				prevEnd = t.End
			}
		case lexer.Color:
			p.leaf(KindSelID, t.Start, t.End, 0, 0)
			prevEnd = t.End
		case lexer.Colon, lexer.DblColon:
			if i+1 < end && p.toks[i+1].Type == lexer.Ident {
				name := p.toks[i+1]
				if i+2 < end && p.toks[i+2].Type == lexer.LParen {
					pid := p.beginAt(KindSelPseudo, t.Start)
					p.s.nodes[pid].Aux = uint16(name.End - t.Start)
					i += 3
					argStart := i
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
						p.err(p.toks[argStart-1], "expected ')' to close functional pseudo")
					}
					if pseudoHasSelectorArgs(name.Literal(p.src)) {
						p.emitSelectorParts(argStart, i)
					} else if argStart < i {
						p.leaf(KindPreludeRaw, p.toks[argStart].Start, p.toks[i-1].End, 0, 0)
					}
					hi := p.toks[argStart-1].End
					if i > argStart {
						hi = p.toks[i-1].End
					}
					if i < end {
						hi = p.toks[i].End
					}
					p.finish(pid, hi)
					prevEnd = hi
				} else {
					p.leaf(KindSelPseudo, t.Start, name.End, uint16(name.End-t.Start), 0)
					prevEnd = name.End
					i++
				}
			} else {
				p.leaf(KindSelPseudo, t.Start, t.End, 0, 0)
				prevEnd = t.End
			}
		case lexer.Ident:
			if i+1 < end && p.toks[i+1].Type == lexer.Pipe && (i+2 >= end || p.toks[i+2].Type != lexer.Pipe) {
				p.leaf(KindSelNamespace, t.Start, p.toks[i+1].End, 0, 0)
				prevEnd = p.toks[i+1].End
				i++
			} else {
				p.leaf(KindSelElement, t.Start, t.End, 0, 0)
				prevEnd = t.End
			}
		case lexer.LBracket:
			lo := t.Start
			hi := t.End
			closed := false
			for i+1 < end {
				i++
				hi = p.toks[i].End
				if p.toks[i].Type == lexer.RBracket {
					closed = true
					break
				}
			}
			if !closed {
				p.err(t, "expected ']' to close attribute selector")
			}
			p.leaf(KindSelAttr, lo, hi, 0, 0)
			prevEnd = hi
		case lexer.Greater, lexer.Plus, lexer.Tilde, lexer.Comma:
			p.leaf(KindSelCombinator, t.Start, t.End, 0, 0)
			prevEnd = t.End
		case lexer.Pipe:
			if i+1 < end && p.toks[i+1].Type == lexer.Pipe {
				p.leaf(KindSelCombinator, t.Start, p.toks[i+1].End, 0, 0)
				prevEnd = p.toks[i+1].End
				i++
			} else {
				p.leaf(KindSelNamespace, t.Start, t.End, 0, 0)
				prevEnd = t.End
			}
		case lexer.Asterisk:
			if i+1 < end && p.toks[i+1].Type == lexer.Pipe {
				p.leaf(KindSelNamespace, t.Start, p.toks[i+1].End, 0, 0)
				prevEnd = p.toks[i+1].End
				i++
			} else {
				p.leaf(KindSelUniversal, t.Start, t.End, 0, 0)
				prevEnd = t.End
			}
		case lexer.Ampersand:
			p.leaf(KindSelNesting, t.Start, t.End, 0, 0)
			prevEnd = t.End
		default:
			prevEnd = t.End
		}
	}
}

func pseudoHasSelectorArgs(name []byte) bool {
	return asciiEqual(name, "has") || asciiEqual(name, "host") || asciiEqual(name, "host-context") || asciiEqual(name, "is") || asciiEqual(name, "not") || asciiEqual(name, "slotted") || asciiEqual(name, "where")
}
