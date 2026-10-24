package pristinecss

import "github.com/builtwithtofu/pristinecss/internal/lexer"

func (p *parser) emitSelectorParts(start, end int) {
	for end > start && p.toks[end-1].Type == lexer.Comment {
		end--
	}
	var prevEnd uint32 // end of the last component; 0 until there is one
	space := -1
	for i := start; i < end; i++ {
		t := p.toks[i]
		if prevEnd != 0 && t.Type != lexer.Comment && t.Start > prevEnd && sourceHasSpace(p.src[prevEnd:t.Start]) && t.Type != lexer.Comma && t.Type != lexer.RBrace && t.Type != lexer.LBrace {
			if space != len(p.s.nodes)-1 {
				lo := prevEnd + uint32(sourceSpaceIndex(p.src[prevEnd:t.Start]))
				space = int(p.leaf(KindSelCombinator, lo, lo+1, 0, 0))
			}
		}
		switch t.Type {
		case lexer.Dot:
			// Comments are not tokens, so ".", a comment and a name are still a class.
			if j := p.nextNonComment(i+1, end); j < end && p.toks[j].Type == lexer.Ident {
				p.leaf(KindSelClass, t.Start, p.toks[j].End, 0, 0)
				prevEnd = p.toks[j].End
				i = j
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
			if j := p.nextNonComment(i+1, end); j < end && p.toks[j].Type == lexer.Ident {
				name := p.toks[j]
				if j+1 < end && p.toks[j+1].Type == lexer.LParen && p.toks[j+1].Start == name.End {
					pid := p.beginAt(KindSelPseudo, t.Start)
					p.s.nodes[pid].Aux = uint16(name.End - t.Start)
					last, closed := p.componentEnd(j+1, end, false)
					i = j + 2
					argStart := i
					i = last
					if closed {
						i--
					} else {
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
					i = j
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
		case lexer.LParen:
			last, _ := p.componentEnd(i, end, false)
			p.leaf(KindPreludeRaw, t.Start, p.toks[last-1].End, 0, 0)
			prevEnd = p.toks[last-1].End
			i = last - 1
		case lexer.LBracket:
			last, closed := p.componentEnd(i, end, false)
			lo, hi := t.Start, p.toks[last-1].End
			i = last - 1
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
		case lexer.Comment:
			continue
		default:
			p.leaf(KindPreludeRaw, t.Start, t.End, 0, 0)
			prevEnd = t.End
		}
	}
}

func pseudoHasSelectorArgs(name []byte) bool {
	return asciiEqual(name, "has") || asciiEqual(name, "host") || asciiEqual(name, "host-context") || asciiEqual(name, "is") || asciiEqual(name, "not") || asciiEqual(name, "slotted") || asciiEqual(name, "where")
}
